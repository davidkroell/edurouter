package edurouter

import (
	"context"
	"github.com/mdlayher/ethernet"
)

type ARPv4LinkLayerHandler struct {
	supplierCh chan FrameFromInterface
	publishCh  chan<- *ethernet.Frame
}

func (llh *ARPv4LinkLayerHandler) SupplierC() chan<- FrameFromInterface {
	return llh.supplierCh
}

func NewARPv4LinkLayerHandler(publishCh chan<- *ethernet.Frame) *ARPv4LinkLayerHandler {
	return &ARPv4LinkLayerHandler{
		supplierCh: make(chan FrameFromInterface, 128),
		publishCh:  publishCh,
	}
}

func (llh *ARPv4LinkLayerHandler) RunHandler(ctx context.Context) {
	go llh.runHandler(ctx)
}

func (llh *ARPv4LinkLayerHandler) runHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-llh.supplierCh:
			var packet ARPv4Pdu

			// ARP logic
			err := (&packet).UnmarshalBinary(f.Frame.Payload)
			if err != nil {
				continue
			}

			if !packet.IsEthernetAndIPv4() {
				continue
			}

			if packet.IsArpResponse() {
				f.InInterface.ArpTable.Store(packet.SrcProtoAddr, packet.SrcHardwareAddr)
				continue
			}

			if packet.IsArpRequestForConfig(f.InInterface) {
				arpResponse := packet.BuildARPResponseWithConfig(f.InInterface)

				arpBinary, err := arpResponse.MarshalBinary()
				if err != nil {
					continue
				}

				llh.publishCh <- &ethernet.Frame{
					Destination: f.Frame.Source,
					Source:      *f.InInterface.HardwareAddr,
					EtherType:   ethernet.EtherTypeARP,
					Payload:     arpBinary,
				}
			}
		}
	}
}
