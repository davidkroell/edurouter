package edurouter

import (
	"context"
)

type IcmpHandler struct {
	supplierCh chan *IPv4Pdu
	publishCh  chan<- *IPv4Pdu
}

func NewIcmpHandler(publishCh chan<- *IPv4Pdu) *IcmpHandler {
	return &IcmpHandler{
		supplierCh: make(chan *IPv4Pdu, 128),
		publishCh:  publishCh,
	}
}

func (i *IcmpHandler) SupplierC() chan<- *IPv4Pdu {
	return i.supplierCh
}

func (i *IcmpHandler) RunHandler(ctx context.Context) {
	go i.runHandler(ctx)
}

func (i *IcmpHandler) runHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case inPkg := <-i.supplierCh:
			outPdu, err := i.handle(inPkg)

			if err != nil {
				// TODO error handling
				continue
			}

			i.publishCh <- outPdu
		}
	}
}

func (i *IcmpHandler) handle(packet *IPv4Pdu) (*IPv4Pdu, error) {
	var icmpPacket ICMPPacket

	err := (&icmpPacket).UnmarshalBinary(packet.Payload)

	if err != nil {
		return nil, err
	}

	if icmpPacket.IcmpType == IcmpTypeEchoRequest {
		icmpPacket.MakeResponse()

		// never returns an error
		icmpBinary, _ := icmpPacket.MarshalBinary()

		return NewIPv4Pdu(packet.DstIP, packet.SrcIP, IPProtocolICMPv4, icmpBinary), nil
	}

	return nil, ErrDropPdu
}
