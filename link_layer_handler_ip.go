package edurouter

import (
	"context"
	"github.com/mdlayher/ethernet"
)

type IPv4LinkLayerHandler struct {
	supplierCh           chan FrameFromInterface
	publishCh            chan<- *ethernet.Frame
	internetLayerHandler *Internetv4LayerHandler
}

func (llh *IPv4LinkLayerHandler) SupplierC() chan<- FrameFromInterface {
	return llh.supplierCh
}

func NewIPv4LinkLayerHandler(publishCh chan<- *ethernet.Frame, internetLayerHandler *Internetv4LayerHandler) *IPv4LinkLayerHandler {
	return &IPv4LinkLayerHandler{
		supplierCh:           make(chan FrameFromInterface, 128),
		publishCh:            publishCh,
		internetLayerHandler: internetLayerHandler,
	}
}

func (llh *IPv4LinkLayerHandler) RunHandler(ctx context.Context) {
	go llh.runHandler(ctx)
}

func (llh *IPv4LinkLayerHandler) runHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case f := <-llh.supplierCh:
			var ipv4Packet IPv4Pdu

			err := (&ipv4Packet).UnmarshalBinary(f.Frame.Payload)
			if err != nil {
				continue
			}

			// TODO handle error
			f.InInterface.ArpTable.Store(ipv4Packet.SrcIP, f.Frame.Source)

			result, routeInfo, err := llh.internetLayerHandler.Handle(&ipv4Packet, f.InInterface)
			if err != nil {
				continue
			}

			framePayload, err := result.MarshalBinary()
			if err != nil {
				continue
			}

			outFrame := &ethernet.Frame{
				Source:    *routeInfo.OutInterface.HardwareAddr,
				EtherType: f.Frame.EtherType,
				Payload:   framePayload,
			}

			if routeInfo.RouteType == LinkLocalRouteType {
				outFrame.Destination, err = routeInfo.OutInterface.ArpTable.Resolve(result.DstIP)
			} else {
				outFrame.Destination, err = routeInfo.OutInterface.ArpTable.Resolve(*routeInfo.NextHop)
			}

			llh.publishCh <- outFrame
		}
	}
}
