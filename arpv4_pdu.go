package edurouter

import (
	"bytes"
	"encoding/binary"
	"github.com/mdlayher/ethernet"
	"io"
	"net"
)

type ARPOperation uint16

const (
	ARPOperationRequest  ARPOperation = 1
	ARPOperationResponse ARPOperation = 2
	HTYPEEthernet                     = 1
)

var emptyHardwareAddr = []byte{
	0x0, 0x0, 0x0,
	0x0, 0x0, 0x0,
}

type ARPv4Pdu struct {
	HTYPE           uint16
	PTYPE           ethernet.EtherType
	HLEN            uint8
	PLEN            uint8
	Operation       ARPOperation
	SrcHardwareAddr []byte
	SrcProtoAddr    []byte
	DstHardwareAddr []byte
	DstProtoAddr    []byte
}

func (a *ARPv4Pdu) IsEthernetAndIPv4() bool {
	if a.HTYPE != HTYPEEthernet {
		// not ethernet
		return false
	}

	if a.PTYPE != ethernet.EtherTypeARP && a.PTYPE != ethernet.EtherTypeIPv4 {
		// not IPv4
		return false
	}

	if a.HLEN != HardwareAddrLen {
		// MAC's are 6 bytes
		return false
	}

	if a.PLEN != net.IPv4len {
		// IP's are 4 bytes
		return false
	}

	return true
}

func (a *ARPv4Pdu) IsArpRequestForConfig(config *InterfaceConfig) bool {
	if a.Operation != ARPOperationRequest {
		// not request
		return false
	}

	if !bytes.Equal(a.DstHardwareAddr, emptyHardwareAddr) {
		// something went wrong, should be empty
		return false
	}

	if !bytes.Equal(a.DstProtoAddr, config.Addr.IP) {
		// targetAddr should be the same
		return false
	}
	return true
}

func (a *ARPv4Pdu) IsArpResponse() bool {
	return a.Operation == ARPOperationResponse
}

func (a *ARPv4Pdu) BuildARPResponseWithConfig(config *InterfaceConfig) *ARPv4Pdu {
	return &ARPv4Pdu{
		HTYPE:     a.HTYPE,
		PTYPE:     a.PTYPE,
		HLEN:      a.HLEN,
		PLEN:      a.PLEN,
		Operation: ARPOperationResponse,

		// provide configured mac as src
		SrcHardwareAddr: *config.HardwareAddr,
		SrcProtoAddr:    config.Addr.IP,

		// flip original sender to target
		DstHardwareAddr: a.SrcHardwareAddr,
		DstProtoAddr:    a.SrcProtoAddr,
	}
}

func (a *ARPv4Pdu) MarshalBinary() ([]byte, error) {
	b := make([]byte, 28)

	b[0] = byte(a.HTYPE >> 8)
	b[1] = byte(a.HTYPE)
	b[2] = byte(a.PTYPE >> 8)
	b[3] = byte(a.PTYPE)
	b[4] = a.HLEN
	b[5] = a.PLEN
	b[6] = byte(a.Operation >> 8)
	b[7] = byte(a.Operation)

	copy(b[8:], a.SrcHardwareAddr)
	copy(b[14:], a.SrcProtoAddr)
	copy(b[18:], a.DstHardwareAddr)
	copy(b[24:], a.DstProtoAddr)

	return b, nil
}

func (a *ARPv4Pdu) UnmarshalBinary(payload []byte) error {
	if len(payload) < 27 {
		return io.ErrUnexpectedEOF
	}

	a.HTYPE = binary.BigEndian.Uint16(payload[0:2])
	a.PTYPE = ethernet.EtherType(binary.BigEndian.Uint16(payload[2:4]))
	a.HLEN = payload[4]
	a.PLEN = payload[5]
	a.Operation = ARPOperation(binary.BigEndian.Uint16(payload[6:8]))
	a.SrcHardwareAddr = payload[8:14]
	a.SrcProtoAddr = payload[14:18]
	a.DstHardwareAddr = payload[18:24]
	a.DstProtoAddr = payload[24:28]

	return nil
}
