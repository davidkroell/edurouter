package ifconfigv4

import (
	"encoding/binary"
	"io"
	"net"
)

const (
	defaultIPv4Version = 4
	ipv4IHL            = 5
	defaultIPv4TTL     = 64
	IPv4HeaderLength   = 20
)

type IPProtocol uint8

const (
	IPProtocolICMPv4 IPProtocol = 1
	IPProtocolIPv4   IPProtocol = 4
	IPProtocolTCP    IPProtocol = 6
	IPProtocolUDP    IPProtocol = 17
)

type IPv4Pdu struct {
	Version        uint8
	TOS            uint8
	TotalLength    uint16
	Id             uint16
	Flags          byte
	FragOffset     uint16
	TTL            uint8
	Protocol       IPProtocol
	HeaderChecksum uint16
	SrcIP          net.IP
	DstIP          net.IP
	Payload        []byte
}

func NewIPv4Pdu(srcIp, dstIp net.IP, ipProto IPProtocol, payload []byte) *IPv4Pdu {
	return &IPv4Pdu{
		Version:     defaultIPv4Version,
		TotalLength: IPv4HeaderLength + uint16(len(payload)),
		TTL:         defaultIPv4TTL,
		Protocol:    ipProto,
		SrcIP:       srcIp,
		DstIP:       dstIp,
		Payload:     payload,
	}
}

func (ip *IPv4Pdu) MarshalBinary() ([]byte, error) {
	length := IPv4HeaderLength + uint16(len(ip.Payload))

	b := make([]byte, length)

	b[0] = (ip.Version << 4) | ipv4IHL
	b[1] = ip.TOS

	binary.BigEndian.PutUint16(b[2:4], length)
	binary.BigEndian.PutUint16(b[4:6], ip.Id)

	b[6] = ip.Flags & 0b1110_0000

	b[8] = ip.TTL
	b[9] = uint8(ip.Protocol)

	copy(b[12:16], ip.SrcIP)
	copy(b[16:20], ip.DstIP)

	// Clear checksum bytes
	b[10] = 0
	b[11] = 0
	checksum := onesComplementChecksum(b[:20])
	// write checksum back
	binary.BigEndian.PutUint16(b[10:12], checksum)

	copy(b[20:], ip.Payload)

	return b, nil
}

func (ip *IPv4Pdu) UnmarshalBinary(payload []byte) error {
	if len(payload) < IPv4HeaderLength {
		return io.ErrUnexpectedEOF
	}

	// Version and IHL share first byte
	// select first 4 bits and shift right 4 times is the result
	ip.Version = (payload[0] & 0b1111_0000) >> 4

	// take only last 4 bits
	// ihl is parsed, but not used
	ihl := payload[0] & 0b0000_1111

	ip.TOS = payload[1]

	ip.TotalLength = binary.BigEndian.Uint16(payload[2:4])

	ip.Id = binary.BigEndian.Uint16(payload[4:6])

	// only first three bits
	ip.Flags = payload[6] & 0b1110_0000
	// 5 bits from 6-th byte, full byte from 7th byte
	ip.FragOffset = (uint16((payload[6]&0b0001_1111)>>3) << 8) + uint16(payload[7])

	ip.TTL = payload[8]

	ip.Protocol = IPProtocol(payload[9])
	ip.HeaderChecksum = binary.BigEndian.Uint16(payload[10:12])
	ip.SrcIP = payload[12:16]
	ip.DstIP = payload[16:20]

	// the default starting byte of the Payload without options.
	// options are only rarely used
	payloadStartByte := uint8(IPv4HeaderLength)

	if ihl > 5 {
		// internetHeaderLength is the length of the header in 32-bits words
		// minimal size: 5 -> 5 * 32 = 160 bits = 20 bytes
		// maximal size is 15 (due to length of 4 bits)
		// maximal size: 15 -> 15 * 32 = 480 bits = 60 bytes

		// 32 bits = 4 bytes
		payloadStartByte = ihl * 4
	}

	// rest of the packet is the Payload
	ip.Payload = payload[payloadStartByte:]
	return nil
}
