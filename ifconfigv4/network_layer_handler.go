package ifconfigv4

import (
	"encoding/binary"
)

type NetworkLayerResultPdu interface {
	MarshalBinary() ([]byte, error)
	TargetIPAddr() []byte
	SenderIPAddr() []byte
}

type NetworkLayerHandler struct {
	// TODO routing table
}

type icmpResult struct {
	icmpType     uint8
	code         uint8
	checksum     uint16
	restOfHeader [4]byte
}

func (i icmpResult) MarshalBinary() ([]byte, error) {
	b := make([]byte, 8)

	b[0] = i.icmpType
	b[1] = i.code
	binary.BigEndian.PutUint16(b[2:4], i.checksum)
	copy(b[4:8], i.restOfHeader[:4])
	return b, nil
}

func (nll *NetworkLayerHandler) Handle(packet *Ipv4Pdu) (NetworkLayerResultPdu, error) {
	switch packet.innerProto {
	case 1:
		// innerProto 1 = ICMP
		icmpType := packet.payload[0]

		if icmpType == 8 {
			icmpBinary, _ := icmpResult{
				icmpType:     0, // echo reply
				code:         0,
				checksum:     0,
				restOfHeader: [4]byte{},
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

	return nil, DropPduError
}
