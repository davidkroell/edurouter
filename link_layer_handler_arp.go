package edurouter

import (
	"context"
	"github.com/mdlayher/ethernet"
)

type ARPv4LinkLayerHandler struct {
	supplierCh chan FrameIn
	publishCh  chan<- *ethernet.Frame
}

func (llh *ARPv4LinkLayerHandler) SupplierC() chan<- FrameIn {
	return llh.supplierCh
}

func NewARPv4LinkLayerHandler(publishCh chan<- *ethernet.Frame) *ARPv4LinkLayerHandler {
	return &ARPv4LinkLayerHandler{
		supplierCh: make(chan FrameIn, 128),
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
				f.Interface.ArpTable.Store(packet.SrcProtoAddr, packet.SrcHardwareAddr)
				continue
			}

			if packet.IsArpRequestForConfig(f.Interface) {
				arpResponse := packet.BuildARPResponseWithConfig(f.Interface)

				arpBinary, err := arpResponse.MarshalBinary()
				if err != nil {
					continue
				}

				llh.publishCh <- &ethernet.Frame{
					Destination: f.Frame.Source,
					Source:      *f.Interface.HardwareAddr,
					EtherType:   ethernet.EtherTypeARP,
					Payload:     arpBinary,
				}
			}
		}
	}
}
