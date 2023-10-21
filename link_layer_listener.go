package edurouter

import (
	"bytes"
	"context"
	"github.com/mdlayher/ethernet"
	"github.com/rs/zerolog/log"
	"net"
)

type handler interface {
	RunHandler(ctx context.Context)
}

type LinkLayerListener struct {
	interfaces         []*InterfaceConfig
	strategy           *LinkLayerStrategy
	toInterfaceChannel chan *ethernet.Frame
	handlers           []handler
	routeTable         *RouteTable
	icmp               *IcmpHandler
}

func NewLinkLayerListener(interfaces ...*InterfaceConfig) *LinkLayerListener {
	routeTable := NewRouteTable()

	for _, i := range interfaces {
		routeTable.MustAddRoute(RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   i.Addr.IP.Mask(i.Addr.Mask),
				Mask: i.Addr.Mask,
			},
			OutInterface: i,
		})
	}

	toInterfaceCh := make(chan *ethernet.Frame, 128)

	arpHandler := NewARPv4LinkLayerHandler(toInterfaceCh)

	ipv4OutputHandler := NewIPv4LinkLayerOutputHandler(toInterfaceCh)

	internetLayerHandler := NewInternetLayerHandler(ipv4OutputHandler.SupplierC(), routeTable)

	icmp := NewIcmpHandler(internetLayerHandler.SupplierLocalC())
	internetLayerStrategy := NewInternetLayerStrategy(icmp)
	internetLayerHandler.SetStrategy(internetLayerStrategy)

	ipv4InputHandler := NewIPv4LinkLayerInputHandler(internetLayerHandler.SupplierC())

	return &LinkLayerListener{
		routeTable:         routeTable,
		icmp:               icmp,
		interfaces:         interfaces,
		toInterfaceChannel: toInterfaceCh,
		strategy: NewLinkLayerStrategy(map[ethernet.EtherType]LinkLayerHandler{
			ethernet.EtherTypeARP:  arpHandler,
			ethernet.EtherTypeIPv4: ipv4InputHandler,
		}),
		handlers: []handler{
			// reverse order
			icmp,

			internetLayerHandler,

			ipv4OutputHandler,
			ipv4InputHandler,

			arpHandler,
		},
	}
}

func (l *LinkLayerListener) RouteTable() *RouteTable {
	return l.routeTable
}

func (l *LinkLayerListener) IcmpPing(ip net.IP, numPings uint16) {
	l.icmp.Ping(ip, numPings)
}

func (listener *LinkLayerListener) ListenAndServe(ctx context.Context) {

	fromInterfaceCh := make(chan FrameIn)
	supportedEtherTypes := listener.strategy.GetSupportedEtherTypes()

	for _, h := range listener.handlers {
		h.RunHandler(ctx)
	}

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
				log.Error().Msgf("error during strategy GetHandler: %v\n", err)
				continue
			}

			handler.SupplierC() <- f
			if err == ErrDropPdu || err != nil {
				continue
			}
		case frame := <-listener.toInterfaceChannel:
			var err error

			for _, iface := range listener.interfaces {
				if bytes.Equal(*iface.HardwareAddr, frame.Source) {
					err = iface.WriteFrame(frame)
					break
				}
			}

			if err != nil {
				log.Error().Msgf("error writing ethernet frame: %v\n", err)
				continue
			}
		}
	}
}
