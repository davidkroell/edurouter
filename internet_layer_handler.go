package edurouter

import (
	"bytes"
	"context"
)

type InternetLayerHandler interface {
	RunHandler(ctx context.Context)
	SupplierC() chan *InternetV4PacketIn
}

type Internetv4LayerHandler struct {
	supplierCh chan *InternetV4PacketIn
	publishCh  chan<- *InternetV4PacketOut

	internetLayerStrategy InternetLayerStrategy
	routeTable            *RouteTable
}

func (h *Internetv4LayerHandler) SupplierC() chan *InternetV4PacketIn {
	return h.supplierCh
}

type InternetV4PacketIn struct {
	Packet   *IPv4Pdu
	Ifconfig *InterfaceConfig
}

type InternetV4PacketOut struct {
	Packet    *IPv4Pdu
	RouteInfo *RouteInfo
}

func NewInternetLayerHandler(publishCh chan<- *InternetV4PacketOut, internetLayerStrategy InternetLayerStrategy, routeTable *RouteTable) *Internetv4LayerHandler {
	return &Internetv4LayerHandler{
		supplierCh:            make(chan *InternetV4PacketIn, 128),
		publishCh:             publishCh,
		internetLayerStrategy: internetLayerStrategy,
		routeTable:            routeTable,
	}
}

func (h *Internetv4LayerHandler) RunHandler(ctx context.Context) {
	go h.runHandler(ctx)
}

func (h *Internetv4LayerHandler) runHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case inPkg := <-h.supplierCh:
			outPdu, routeInfo, err := h.handlePacket(inPkg.Packet, inPkg.Ifconfig)

			if err != nil {
				// TODO
				continue
			}

			h.publishCh <- &InternetV4PacketOut{
				Packet:    outPdu,
				RouteInfo: routeInfo,
			}
		}
	}
}

func (h *Internetv4LayerHandler) handlePacket(packet *IPv4Pdu, ifconfig *InterfaceConfig) (*IPv4Pdu, *RouteInfo, error) {
	if bytes.Equal(packet.DstIP, ifconfig.RealIPAddr.IP) {
		// this packet is for the real interface, not for the simulated one
		return nil, nil, ErrDropPdu
	}

	if bytes.Equal(packet.DstIP, ifconfig.Addr.IP) {
		// this packet has to be handled at the simulated IP address
		packet, err := h.handleLocal(packet)
		if err != nil {
			return nil, nil, err
		}

		ri, err := h.routeTable.getRouteInfoForPacket(packet)
		if err != nil {
			return nil, nil, err
		}

		return packet, ri, err
	}

	return h.routeTable.RoutePacket(packet)
}

func (h *Internetv4LayerHandler) handleLocal(packet *IPv4Pdu) (*IPv4Pdu, error) {
	nextHandler, err := h.internetLayerStrategy.GetHandler(packet.Protocol)
	if err != nil {
		return nil, ErrDropPdu
	}

	resultPacket, err := nextHandler.Handle(packet)

	if err != nil {
		return nil, ErrDropPdu
	}

	return resultPacket, nil
}
