package edurouter

import (
	"context"
	"crypto/rand"
	"log"
	"net"
	"time"
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

func (i *IcmpHandler) Ping(dstIP net.IP, numPings uint16) {
	dstIP = dstIP.To4()

	for seq := uint16(1); seq <= numPings; seq++ {
		icmpPacket := ICMPPacket{
			IcmpType: IcmpTypeEchoRequest,
			Id:       200 + seq,
			Seq:      seq,
			Data:     make([]byte, 48, 48),
		}

		_, _ = rand.Read(icmpPacket.Data)

		// never returns an error
		icmpBinary, _ := icmpPacket.MarshalBinary()

		ipPdu := NewIPv4Pdu(nil, dstIP, IPProtocolICMPv4, icmpBinary)

		i.publishCh <- ipPdu

		time.Sleep(time.Second)
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

			if err == ErrDropPdu {
				continue
			}
			if err != nil {
				log.Printf("error during icmp handling: %v\n", err)
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
	if icmpPacket.IcmpType == IcmpTypeEchoReply {
		log.Printf("64 bytes from %s: icmp_seq=%d, ttl=%d\n", packet.SrcIP.String(), icmpPacket.Seq, packet.TTL)
	}

	return nil, ErrDropPdu
}
