package ifconfigv4

import (
	"bytes"
	"io"
)

type arpOperation uint16

const (
	arpOperationRequest  arpOperation = 1
	arpOperationResponse arpOperation = 2
)

var emptyHardwareAddr = []byte{
	0x0, 0x0, 0x0,
	0x0, 0x0, 0x0,
}

type arpPacket struct {
	HTYPE              uint16
	PTYPE              uint16
	HLEN               uint8
	PLEN               uint8
	Operation          arpOperation
	SenderHardwareAddr []byte
	SenderProtoAddr    []byte
	TargetHardwareAddr []byte
	TargetProtoAddr    []byte
}

func (a *arpPacket) isEthernetAndIPv4() bool {
	if a.HTYPE != 1 {
		// not ethernet
		return false
	}

	if a.PTYPE != ipv4EtherType {
		// not IPv4
		return false
	}

	if a.HLEN != hardwareAddrSize {
		// MAC's are 6 bytes
		return false
	}

	if a.PLEN != ipAddrSize {
		// IP's are 4 bytes
		return false
	}

	return true
}

func (a *arpPacket) isArpRequestForConfig(config *InterfaceConfig) bool {
	if a.Operation != arpOperationRequest {
		// not request
		return false
	}

	if !bytes.Equal(a.TargetHardwareAddr, emptyHardwareAddr) {
		// something went wrong, should be empty
		return false
	}

	if !bytes.Equal(a.TargetProtoAddr, config.IPAddr) {
		// targetAddr should be the same
		return false
	}
	return true
}

func (a *arpPacket) buildArpResponseWithConfig(config *InterfaceConfig) *arpPacket {
	return &arpPacket{
		HTYPE:     a.HTYPE,
		PTYPE:     a.PTYPE,
		HLEN:      a.HLEN,
		PLEN:      a.PLEN,
		Operation: arpOperationResponse,

		// provide configured mac as sender
		SenderHardwareAddr: config.HardwareAddr,
		SenderProtoAddr:    config.IPAddr,

		// flip original sender to target
		TargetHardwareAddr: a.SenderHardwareAddr,
		TargetProtoAddr:    a.TargetProtoAddr,
	}
}

func (a *arpPacket) MarshallBinary() []byte {
	b := make([]byte, 28)

	b[0] = byte(a.HTYPE >> 8)
	b[1] = byte(a.HTYPE)
	b[2] = byte(a.PTYPE >> 8)
	b[3] = byte(a.PTYPE)
	b[4] = a.HLEN
	b[5] = a.PLEN
	b[6] = byte(a.Operation >> 8)
	b[7] = byte(a.Operation)

	copy(b[8:], a.SenderHardwareAddr)
	copy(b[14:], a.SenderProtoAddr)
	copy(b[18:], a.TargetHardwareAddr)
	copy(b[24:], a.TargetProtoAddr)

	return b
}

func (a *arpPacket) UnmarshalBinary(payload []byte) error {
	if len(payload) < 27 {
		return io.ErrUnexpectedEOF
	}

	a.HTYPE = uint16(payload[0])<<8 | uint16(payload[1])
	a.PTYPE = uint16(payload[2])<<8 | uint16(payload[3])
	a.HLEN = payload[4]
	a.PLEN = payload[5]
	a.Operation = arpOperation(payload[6])<<8 | arpOperation(payload[7])
	a.SenderHardwareAddr = payload[8:14]
	a.SenderProtoAddr = payload[14:18]
	a.TargetHardwareAddr = payload[18:24]
	a.TargetProtoAddr = payload[24:28]

	return nil
}
