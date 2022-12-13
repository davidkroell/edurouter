package ifconfigv4

import (
	"bytes"
	"encoding/binary"
)

type NetworkLayerResultPdu interface {
	MarshalBinary() ([]byte, error)
	TargetIPAddr() []byte
	SenderIPAddr() []byte
}

type NetworkLayerHandler struct {
	// TODO routing table
	ifconfig *InterfaceConfig
}

type icmpResult struct {
	icmpType uint8
	icmpCode uint8
	checksum uint16
	id       uint16
	seq      uint16
	body     []byte
}

func (i icmpResult) MarshalBinary() ([]byte, error) {
	b := make([]byte, 8+len(i.body))

	b[0] = i.icmpType
	b[1] = i.icmpCode
	binary.BigEndian.PutUint16(b[4:], i.id)
	binary.BigEndian.PutUint16(b[6:], i.seq)

	copy(b[8:], i.body)

	b[2] = 0
	b[3] = 0
	i.checksum = onesComplementChecksum(b)
	binary.BigEndian.PutUint16(b[2:4], i.checksum)

	return b, nil
}

func (nll *NetworkLayerHandler) Handle(packet *Ipv4Pdu) (NetworkLayerResultPdu, error) {
	if bytes.Equal(packet.destinationIp, nll.ifconfig.RealIPAddr.IP) {
		// this packet is for the real interface, not for the simulated one
		return nil, ErrDropPdu
	}

	if bytes.Equal(packet.destinationIp, nll.ifconfig.Addr.IP) {
		return nll.handleLocal(packet)
	}

	return nll.route(packet)
}

func (nll *NetworkLayerHandler) handleLocal(packet *Ipv4Pdu) (NetworkLayerResultPdu, error) {
	// TODO implement clean structure for different protocols
	switch packet.innerProto {
	case 1:
		// innerProto 1 = ICMP
		icmpType := packet.payload[0]

		if icmpType == 8 {
			icmpBinary, _ := icmpResult{
				icmpType: 0, // echo reply
				icmpCode: 0,
				checksum: 0,
				id:       binary.BigEndian.Uint16(packet.payload[4:6]),
				seq:      binary.BigEndian.Uint16(packet.payload[6:8]),
				body:     packet.payload[8:],
			}.MarshalBinary()

			return &Ipv4Pdu{
				version:       4,
				ttl:           64,
				innerProto:    1,
				sourceIp:      packet.destinationIp,
				destinationIp: packet.sourceIp,
				payload:       icmpBinary,
			}, nil
		}
	}

	return nil, ErrDropPdu
}

func (nll *NetworkLayerHandler) route(packet *Ipv4Pdu) (NetworkLayerResultPdu, error) {
	// TODO implement routing of IP packets
	//  check if current application design is appropriate also for routing
	return nil, ErrDropPdu
}
