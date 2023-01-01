package edurouter

import (
	"bytes"
	"context"
	"github.com/mdlayher/ethernet"
	"log"
	"net"
)

type LinkLayerListener struct {
	interfaces []*InterfaceConfig
	strategy   *LinkLayerStrategy
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

	return &LinkLayerListener{
		interfaces: interfaces,
		strategy: NewLinkLayerStrategy(map[ethernet.EtherType]LinkLayerHandler{
			ethernet.EtherTypeARP:  NewARPv4LinkLayerHandler(),
			ethernet.EtherTypeIPv4: NewIPv4LinkLayerHandler(NewInternetLayerHandler(NewInternetLayerStrategy(&IcmpHandler{}), routeTable)),
		}),
	}
}

type frameFromInterface struct {
	frame       *ethernet.Frame
	inInterface *InterfaceConfig
}

func (listener *LinkLayerListener) ListenAndServe(ctx context.Context) {

	frameChan := make(chan frameFromInterface)
	supportedEtherTypes := listener.strategy.GetSupportedEtherTypes()

	for _, iface := range listener.interfaces {
		iface.SetupAndListen(ctx, supportedEtherTypes, frameChan)
	}

	// read frames from supplier channel
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-frameChan:
			handler, err := listener.strategy.GetHandler(f.frame.EtherType)
			if err != nil {
				continue
			}

			frameToSend, err := handler.Handle(f.frame, f.inInterface)
			if err == ErrDropPdu || err != nil {
				continue
			}

			for _, iface := range listener.interfaces {
				if bytes.Equal(*iface.HardwareAddr, frameToSend.Source) {
					err = iface.WriteFrame(frameToSend)
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
