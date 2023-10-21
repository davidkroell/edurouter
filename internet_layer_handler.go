package edurouter

import (
	"bytes"
	"context"
	"github.com/rs/zerolog/log"
)

type InternetLayerHandler interface {
	RunHandler(ctx context.Context)
	SupplierC() chan *InternetV4PacketIn
}

type Internetv4LayerHandler struct {
	supplierCh      chan *InternetV4PacketIn
	supplierLocalCh chan *IPv4Pdu
	publishCh       chan<- *InternetV4PacketOut

	internetLayerStrategy InternetLayerStrategy
	routeTable            *RouteTable
}

func (h *Internetv4LayerHandler) SupplierC() chan *InternetV4PacketIn {
	return h.supplierCh
}

func (h *Internetv4LayerHandler) SupplierLocalC() chan *IPv4Pdu {
	return h.supplierLocalCh
}

type InternetV4PacketIn struct {
	Packet   *IPv4Pdu
	Ifconfig *InterfaceConfig
}

type InternetV4PacketOut struct {
	Packet    *IPv4Pdu
	RouteInfo *RouteInfo
}

func NewInternetLayerHandler(publishCh chan<- *InternetV4PacketOut, routeTable *RouteTable) *Internetv4LayerHandler {
	return &Internetv4LayerHandler{
		supplierCh:      make(chan *InternetV4PacketIn, 128),
		supplierLocalCh: make(chan *IPv4Pdu, 128),
		publishCh:       publishCh,
		routeTable:      routeTable,
	}
}

func (h *Internetv4LayerHandler) SetStrategy(s InternetLayerStrategy) {
	h.internetLayerStrategy = s
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
			if bytes.Equal(inPkg.Packet.DstIP, inPkg.Ifconfig.RealIPAddr.IP) {
				// this packet is for the real interface, not for the simulated one
				continue
			}

			if bytes.Equal(inPkg.Packet.DstIP, inPkg.Ifconfig.Addr.IP) {
				// this packet has to be handled at the simulated IP address
				err := h.handleLocal(inPkg.Packet)
				if err != nil {
					log.Error().Msgf("error during handleLocal: %v\n", err)
				}
				continue
			}

			outPdu, routeInfo, err := h.routeTable.RoutePacket(*inPkg.Packet)

			if err != nil && err != ErrDropPdu {
				log.Error().Msgf("error during packet routing: %v\n", err)
				continue
			}

			h.publishCh <- &InternetV4PacketOut{
				Packet:    outPdu,
				RouteInfo: routeInfo,
			}

		case inPkg := <-h.supplierLocalCh:
			outPdu, routeInfo, err := h.routeTable.RoutePacket(*inPkg)

			if err != nil && err != ErrDropPdu {
				log.Error().Msgf("error during packet routing: %v\n", err)
				continue
			}

			h.publishCh <- &InternetV4PacketOut{
				Packet:    outPdu,
				RouteInfo: routeInfo,
			}
		}
	}
}

func (h *Internetv4LayerHandler) handleLocal(packet *IPv4Pdu) error {
	nextHandler, err := h.internetLayerStrategy.GetHandler(packet.Protocol)
	if err != nil {
		return err
	}

	ch := nextHandler.SupplierC()
	ch <- packet
	return nil
}
