package ifconfigv4

import (
	"bytes"
	"encoding/binary"
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

type LinkLayerResultPdu interface {
	SenderHardwareAddr() []byte
	TargetHardwareAddr() []byte
	MarshalBinary() ([]byte, error)
}

type arpPacket struct {
	hardwareType       uint16
	protoType          uint16
	hardwareLen        uint8
	protoLen           uint8
	operation          arpOperation
	senderHardwareAddr []byte
	senderProtoAddr    []byte
	targetHardwareAddr []byte
	targetProtoAddr    []byte
}

func (a *arpPacket) SenderHardwareAddr() []byte {
	return a.senderHardwareAddr
}

func (a *arpPacket) TargetHardwareAddr() []byte {
	return a.targetHardwareAddr
}

func (a *arpPacket) isEthernetAndIPv4() bool {
	if a.hardwareType != 1 {
		// not ethernet
		return false
	}

	if a.protoType != ipv4EtherType {
		// not IPv4
		return false
	}

	if a.hardwareLen != hardwareAddrSize {
		// MAC's are 6 bytes
		return false
	}

	if a.protoLen != ipAddrSize {
		// IP's are 4 bytes
		return false
	}

	return true
}

func (a *arpPacket) isArpRequestForConfig(config *InterfaceConfig) bool {
	if a.operation != arpOperationRequest {
		// not request
		return false
	}

	if !bytes.Equal(a.targetHardwareAddr, emptyHardwareAddr) {
		// something went wrong, should be empty
		return false
	}

	if !bytes.Equal(a.targetProtoAddr, config.IPAddr) {
		// targetAddr should be the same
		return false
	}
	return true
}

func (a *arpPacket) buildArpResponseWithConfig(config *InterfaceConfig) *arpPacket {
	return &arpPacket{
		hardwareType: a.hardwareType,
		protoType:    a.protoType,
		hardwareLen:  a.hardwareLen,
		protoLen:     a.protoLen,
		operation:    arpOperationResponse,

		// provide configured mac as sender
		senderHardwareAddr: config.HardwareAddr,
		senderProtoAddr:    config.IPAddr,

		// flip original sender to target
		targetHardwareAddr: a.senderHardwareAddr,
		targetProtoAddr:    a.senderProtoAddr,
	}
}

func (a *arpPacket) MarshalBinary() ([]byte, error) {
	b := make([]byte, 28)

	b[0] = byte(a.hardwareType >> 8)
	b[1] = byte(a.hardwareType)
	b[2] = byte(a.protoType >> 8)
	b[3] = byte(a.protoType)
	b[4] = a.hardwareLen
	b[5] = a.protoLen
	b[6] = byte(a.operation >> 8)
	b[7] = byte(a.operation)

	copy(b[8:], a.senderHardwareAddr)
	copy(b[14:], a.senderProtoAddr)
	copy(b[18:], a.targetHardwareAddr)
	copy(b[24:], a.targetProtoAddr)

	return b, nil
}

func (a *arpPacket) UnmarshalBinary(payload []byte) error {
	if len(payload) < 27 {
		return io.ErrUnexpectedEOF
	}

	a.hardwareType = binary.BigEndian.Uint16(payload[0:2])
	a.protoType = binary.BigEndian.Uint16(payload[2:4])
	a.hardwareLen = payload[4]
	a.protoLen = payload[5]
	a.operation = arpOperation(binary.BigEndian.Uint16(payload[6:8]))
	a.senderHardwareAddr = payload[8:14]
	a.senderProtoAddr = payload[14:18]
	a.targetHardwareAddr = payload[18:24]
	a.targetProtoAddr = payload[24:28]

	return nil
}
