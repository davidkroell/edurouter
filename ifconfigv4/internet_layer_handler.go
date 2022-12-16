package ifconfigv4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

const (
	icmpv4HeaderLength = 8
)

type InternetLayerResultPdu interface {
	MarshalBinary() ([]byte, error)
	SrcIPAddr() net.IP
	DstIPAddr() net.IP
}

type InternetLayerHandler struct {
	// TODO routing table
	internetLayerStrategy *internetLayerStrategy
}

type IcmpType uint8

const (
	IcmpTypeEchoRequest = 8
	IcmpTypeEchoReply   = 0
)

type ICMPPacket struct {
	IcmpType IcmpType
	IcmpCode uint8
	Checksum uint16
	Id       uint16
	Seq      uint16
	Data     []byte
}

func (icmp *ICMPPacket) UnmarshalBinary(data []byte) error {
	if len(data) < icmpv4HeaderLength {
		return io.ErrUnexpectedEOF
	}

	icmp.IcmpType = IcmpType(data[0])
	icmp.IcmpCode = data[1]

	icmp.Id = binary.BigEndian.Uint16(data[4:])
	icmp.Seq = binary.BigEndian.Uint16(data[6:])

	icmp.Data = data[icmpv4HeaderLength:]

	actualChecksum := binary.BigEndian.Uint16(data[2:4])

	data[2] = 0
	data[3] = 0
	// calculate checksum
	expectedChecksum := onesComplementChecksum(data)

	if actualChecksum != expectedChecksum {
		return errors.New("invalid icmp checksum")
	}

	return nil
}

func (icmp *ICMPPacket) MarshalBinary() ([]byte, error) {
	b := make([]byte, icmpv4HeaderLength+len(icmp.Data))

	b[0] = uint8(icmp.IcmpType)
	b[1] = icmp.IcmpCode
	binary.BigEndian.PutUint16(b[4:], icmp.Id)
	binary.BigEndian.PutUint16(b[6:], icmp.Seq)

	copy(b[icmpv4HeaderLength:], icmp.Data)

	b[2] = 0
	b[3] = 0
	icmp.Checksum = onesComplementChecksum(b)
	binary.BigEndian.PutUint16(b[2:4], icmp.Checksum)

	return b, nil
}

func (icmp *ICMPPacket) MakeResponse() {
	icmp.IcmpType = IcmpTypeEchoReply
}

func (nll *InternetLayerHandler) Handle(packet *IPv4Pdu, ifconfig *InterfaceConfig) (InternetLayerResultPdu, error) {
	if bytes.Equal(packet.DstIP, ifconfig.RealIPAddr.IP) {
		// this packet is for the real interface, not for the simulated one
		return nil, ErrDropPdu
	}

	if bytes.Equal(packet.DstIP, ifconfig.Addr.IP) {
		// this packet has to be handled at the simulated IP address
		return nll.handleLocal(packet)
	}

	return nll.route(packet)
}

func (nll *InternetLayerHandler) handleLocal(packet *IPv4Pdu) (InternetLayerResultPdu, error) {
	nextHandler, err := nll.internetLayerStrategy.GetHandler(packet.Protocol)
	if err != nil {
		return nil, ErrDropPdu
	}

	// todo what about the error?
	resultPacket, err := nextHandler.Handle(packet)

	if err != nil {
		return nil, ErrDropPdu
	}

	return resultPacket, nil
}

func (nll *InternetLayerHandler) route(packet *IPv4Pdu) (InternetLayerResultPdu, error) {
	// TODO implement routing of IP packets
	//  check if current application design is appropriate also for routing
	return nil, ErrDropPdu
}
