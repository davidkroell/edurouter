package ifconfigv4

import (
	"encoding/binary"
	"io"
)

const ipv4MarshallingVersion = 4

type Ipv4Pdu struct {
	version        uint8
	dscp           byte
	ecn            byte
	totalLength    uint16
	identification uint16
	flags          byte
	fragmentOffset uint16
	ttl            uint8
	innerProto     uint8
	headerChecksum uint16
	sourceIp       []byte
	destinationIp  []byte
	payload        []byte
}

func (i Ipv4Pdu) TargetIPAddr() []byte {
	return i.destinationIp
}

func (i Ipv4Pdu) SenderIPAddr() []byte {
	return i.sourceIp
}

func (a *Ipv4Pdu) MarshalBinary() ([]byte, error) {
	length := 20 + uint16(len(a.payload))

	b := make([]byte, length) // TODO set correct length

	// TODO extract 5=IHL flag
	b[0] = (ipv4MarshallingVersion << 4) | (5 & 0b0000_1111)
	b[1] = (a.dscp & 0b1111_1100) | (a.ecn & 0b0000_0011)

	binary.BigEndian.PutUint16(b[2:4], length)
	binary.BigEndian.PutUint16(b[4:6], a.identification)

	b[6] = (a.flags & 0b1110_0000)

	// TODO fragment offset
	b[8] = 64           // ttl
	b[9] = a.innerProto // todo set (icmp =1)

	copy(b[12:16], a.sourceIp)
	copy(b[16:20], a.destinationIp)

	checksum := calcChecksum(b[:20])
	binary.BigEndian.PutUint16(b[10:12], checksum)

	copy(b[20:], a.payload)

	return b, nil
}

// from: https://github.com/google/gopacket/blob/master/layers/ip4.go#L158
func calcChecksum(bytes []byte) uint16 {
	// Clear checksum bytes
	bytes[10] = 0
	bytes[11] = 0

	// Compute checksum
	var csum uint32
	for i := 0; i < len(bytes); i += 2 {
		csum += uint32(bytes[i]) << 8
		csum += uint32(bytes[i+1])
	}
	for {
		// Break when sum is less or equals to 0xFFFF
		if csum <= 65535 {
			break
		}
		// Add carry to the sum
		csum = (csum >> 16) + uint32(uint16(csum))
	}
	// Flip all the bits
	return ^uint16(csum)
}

func (a *Ipv4Pdu) UnmarshalBinary(payload []byte) error {
	if len(payload) < 20 {
		return io.ErrUnexpectedEOF
	}

	// version and IHL share first byte
	// select first 4 bits and shift right 4 times is the result
	a.version = (payload[0] & 0b1111_0000) >> 4

	// take only last 4 bits
	// ihl is parsed, but not used
	ihl := payload[0] & 0b0000_1111

	// dscp and ecn share second byte
	// only first 6 bytes
	a.dscp = payload[1] & 0b1111_1100
	a.ecn = payload[1] & 0b0000_0011

	a.totalLength = binary.BigEndian.Uint16(payload[2:4])

	a.identification = binary.BigEndian.Uint16(payload[4:6])

	// only first three bits
	a.flags = payload[6] & 0b1110_0000
	// 5 bits from 6-th byte, full byte from 7th byte
	a.fragmentOffset = (uint16((payload[6]&0b0001_1111)>>3) << 8) + uint16(payload[7])

	a.ttl = payload[8]

	a.innerProto = payload[9]
	a.headerChecksum = binary.BigEndian.Uint16(payload[10:12])
	a.sourceIp = payload[12:16]
	a.destinationIp = payload[16:20]

	// the default starting byte of the payload without options.
	// options are only rarely used
	payloadStartByte := uint8(20)

	if ihl > 5 {
		// internetHeaderLength is the length of the header in 32-bits words
		// minimal size: 5 -> 5 * 32 = 160 bits = 20 bytes
		// maximal size is 15 (due to length of 4 bits)
		// maximal size: 15 -> 15 * 32 = 480 bits = 60 bytes

		// 32 bits = 4 bytes
		payloadStartByte = ihl * 4
	}

	// rest of the packet is the payload
	a.payload = payload[payloadStartByte:]
	return nil
}
