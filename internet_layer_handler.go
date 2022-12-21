package edurouter

import (
	"bytes"
)

type InternetLayerHandler struct {
	internetLayerStrategy *InternetLayerStrategy
	routeTable            *RouteTable
}

func NewInternetLayerHandler(internetLayerStrategy *InternetLayerStrategy, routeTable *RouteTable) *InternetLayerHandler {
	return &InternetLayerHandler{internetLayerStrategy: internetLayerStrategy, routeTable: routeTable}
}

func (nll *InternetLayerHandler) Handle(packet *IPv4Pdu, ifconfig *InterfaceConfig) (*IPv4Pdu, *RouteInfo, error) {
	if bytes.Equal(packet.DstIP, ifconfig.RealIPAddr.IP) {
		// this packet is for the real interface, not for the simulated one
		return nil, nil, ErrDropPdu
	}

	if bytes.Equal(packet.DstIP, ifconfig.Addr.IP) {
		// this packet has to be handled at the simulated IP address
		packet, err := nll.handleLocal(packet)
		if err != nil {
			return nil, nil, err
		}

		ri, err := nll.routeTable.getRouteInfoForPacket(packet)
		if err != nil {
			return nil, nil, err
		}

		return packet, ri, err
	}

	return nll.routeTable.RoutePacket(packet)
}

func (nll *InternetLayerHandler) handleLocal(packet *IPv4Pdu) (*IPv4Pdu, error) {
	nextHandler, err := nll.internetLayerStrategy.GetHandler(packet.Protocol)
	if err != nil {
		return nil, ErrDropPdu
	}

	resultPacket, err := nextHandler.Handle(packet)

	if err != nil {
		return nil, ErrDropPdu
	}

	return resultPacket, nil
}
