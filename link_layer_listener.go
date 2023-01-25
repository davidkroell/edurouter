package edurouter

import (
	"bytes"
	"context"
	"github.com/mdlayher/ethernet"
	"log"
	"net"
)

type LinkLayerListener struct {
	interfaces    []*InterfaceConfig
	strategy      *LinkLayerStrategy
	fromHandlerCh chan *ethernet.Frame
}

func NewLinkLayerListener(interfaces ...*InterfaceConfig) *LinkLayerListener {
	routeTable := NewRouteTable()

	for _, i := range interfaces {
		routeTable.AddRoute(RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   i.Addr.IP.Mask(i.Addr.Mask),
				Mask: i.Addr.Mask,
			},
			OutInterface: i,
		})
	}

	routeTable.AddRoute(RouteInfo{
		RouteType: StaticRouteType,
		DstNet: net.IPNet{
			IP:   net.IP{0, 0, 0, 0},
			Mask: net.CIDRMask(0, 32),
		},
		OutInterface: interfaces[0],
		NextHop:      &net.IP{192, 168, 0, 1},
	})

	fromHandlerCh := make(chan *ethernet.Frame, 128)

	arpHandler := NewARPv4LinkLayerHandler(fromHandlerCh)
	arpHandler.RunHandler(context.TODO()) // TODO do not run in ctor

	ipv4Handler := NewIPv4LinkLayerHandler(fromHandlerCh, NewInternetLayerHandler(NewInternetLayerStrategy(&IcmpHandler{}), routeTable))
	ipv4Handler.RunHandler(context.TODO()) // TODO do not run in ctor

	return &LinkLayerListener{
		interfaces:    interfaces,
		fromHandlerCh: fromHandlerCh,
		strategy: NewLinkLayerStrategy(map[ethernet.EtherType]LinkLayerHandler{
			ethernet.EtherTypeARP:  arpHandler,
			ethernet.EtherTypeIPv4: ipv4Handler,
		}),
	}
}

type FrameFromInterface struct {
	Frame       *ethernet.Frame
	InInterface *InterfaceConfig
}

func (listener *LinkLayerListener) ListenAndServe(ctx context.Context) {

	fromInterfaceCh := make(chan FrameFromInterface)
	supportedEtherTypes := listener.strategy.GetSupportedEtherTypes()

	for _, iface := range listener.interfaces {
		iface.SetupAndListen(ctx, supportedEtherTypes, fromInterfaceCh)
	}

	// read frames from supplier channel
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-fromInterfaceCh:
			handler, err := listener.strategy.GetHandler(f.Frame.EtherType)
			if err != nil {
				continue
			}

			handler.SupplierC() <- f
			if err == ErrDropPdu || err != nil {
				continue
			}
		case frame := <-listener.fromHandlerCh:
			var err error

			for _, iface := range listener.interfaces {
				if bytes.Equal(*iface.HardwareAddr, frame.Source) {
					err = iface.WriteFrame(frame)
					break
				}
			}

			if err != nil {
				log.Printf("failed to write ethernet frame: %v\n", err)
				continue
			}
		}
	}
}
