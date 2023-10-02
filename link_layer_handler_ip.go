package edurouter

import (
	"context"
	"github.com/mdlayher/ethernet"
	"log"
)

type IPv4LinkLayerInputHandler struct {
	supplierCh chan FrameIn
	publishCh  chan<- *InternetV4PacketIn
}

type FrameIn struct {
	Frame     *ethernet.Frame
	Interface *InterfaceConfig
}

type FrameOut struct {
	Frame     *IPv4Pdu
	RouteInfo *RouteInfo
}

func (llh *IPv4LinkLayerInputHandler) SupplierC() chan<- FrameIn {
	return llh.supplierCh
}

func NewIPv4LinkLayerInputHandler(publishCh chan<- *InternetV4PacketIn) *IPv4LinkLayerInputHandler {
	return &IPv4LinkLayerInputHandler{
		supplierCh: make(chan FrameIn, 128),
		publishCh:  publishCh,
	}
}

func (llh *IPv4LinkLayerInputHandler) RunHandler(ctx context.Context) {
	go llh.runHandler(ctx)
}

func (llh *IPv4LinkLayerInputHandler) runHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case f := <-llh.supplierCh:
			var ipv4Packet IPv4Pdu

			err := (&ipv4Packet).UnmarshalBinary(f.Frame.Payload)
			if err != nil {
				log.Printf("error during arp unmarshall: %v\n", err)
				continue
			}

			err = f.Interface.ArpTable.Store(ipv4Packet.SrcIP, f.Frame.Source)
			if err != nil {
				log.Printf("error during arp table store: %v\n", err)
			}

			llh.publishCh <- &InternetV4PacketIn{
				Packet:   &ipv4Packet,
				Ifconfig: f.Interface,
			}
		}
	}
}

type IPv4LinkLayerOutputHandler struct {
	supplierCh chan *InternetV4PacketOut
	publishCh  chan<- *ethernet.Frame
}

func (h *IPv4LinkLayerOutputHandler) SupplierC() chan *InternetV4PacketOut {
	return h.supplierCh
}

func NewIPv4LinkLayerOutputHandler(publishCh chan<- *ethernet.Frame) *IPv4LinkLayerOutputHandler {
	return &IPv4LinkLayerOutputHandler{
		supplierCh: make(chan *InternetV4PacketOut, 128),
		publishCh:  publishCh,
	}
}

func (h *IPv4LinkLayerOutputHandler) RunHandler(ctx context.Context) {
	go h.runHandler(ctx)
}

func (h *IPv4LinkLayerOutputHandler) runHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case pdu := <-h.supplierCh:
			framePayload, err := pdu.Packet.MarshalBinary()
			if err != nil {
				log.Printf("error during ipv4 marshall: %v\n", err)
				continue
			}

			outFrame := &ethernet.Frame{
				Source:    *pdu.RouteInfo.OutInterface.HardwareAddr,
				EtherType: ethernet.EtherTypeIPv4,
				Payload:   framePayload,
			}

			if pdu.RouteInfo.RouteType == LinkLocalRouteType {
				outFrame.Destination, err = pdu.RouteInfo.OutInterface.ArpTable.Resolve(pdu.Packet.DstIP)
			} else {
				outFrame.Destination, err = pdu.RouteInfo.OutInterface.ArpTable.Resolve(*pdu.RouteInfo.NextHop)
			}

			h.publishCh <- outFrame
		}
	}
}
