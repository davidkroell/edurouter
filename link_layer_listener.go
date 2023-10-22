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
	fromInterfaceCh    chan FrameIn
	ctx                context.Context
}

func NewLinkLayerListener(interfaces ...*InterfaceConfig) *LinkLayerListener {
	routeTable := NewRouteTable()

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

func (l *LinkLayerListener) AddInterface(iface *InterfaceConfig) {
	iface.SetupAndListen(l.ctx, l.strategy.GetSupportedEtherTypes(), l.fromInterfaceCh)

	l.routeTable.MustAddRoute(RouteInfo{
		RouteType: LinkLocalRouteType,
		DstNet: net.IPNet{
			IP:   iface.Addr.IP.Mask(iface.Addr.Mask),
			Mask: iface.Addr.Mask,
		},
		OutInterface: iface,
	})

	l.interfaces = append(l.interfaces, iface)
}

func (l *LinkLayerListener) ListenAndServe(ctx context.Context) {
	l.ctx = ctx

	l.fromInterfaceCh = make(chan FrameIn)
	supportedEtherTypes := l.strategy.GetSupportedEtherTypes()

	for _, h := range l.handlers {
		h.RunHandler(ctx)
	}

	for _, iface := range l.interfaces {
		iface.SetupAndListen(ctx, supportedEtherTypes, l.fromInterfaceCh)

		l.routeTable.MustAddRoute(RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   iface.Addr.IP.Mask(iface.Addr.Mask),
				Mask: iface.Addr.Mask,
			},
			OutInterface: iface,
		})
	}

	// read frames from supplier channel
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-l.fromInterfaceCh:
			handler, err := l.strategy.GetHandler(f.Frame.EtherType)
			if err != nil {
				log.Error().Msgf("error during strategy GetHandler: %v", err)
				continue
			}

			handler.SupplierC() <- f
			if err == ErrDropPdu || err != nil {
				continue
			}
		case frame := <-l.toInterfaceChannel:
			var err error

			for _, iface := range l.interfaces {
				if bytes.Equal(*iface.HardwareAddr, frame.Source) {
					err = iface.WriteFrame(frame)
					break
				}
			}

			if err != nil {
				log.Error().Msgf("error writing ethernet frame: %v", err)
				continue
			}
		}
	}
}

func (l *LinkLayerListener) Interfaces() []*InterfaceConfig {
	return l.interfaces
}
