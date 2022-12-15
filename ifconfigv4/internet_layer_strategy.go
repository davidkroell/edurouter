package ifconfigv4

type internetLayerStrategy struct {
	icmpHandler *icmpHandler
}

type TransportLayerHandler interface {
	Handle(*IPv4Pdu) (*IPv4Pdu, error)
}

type icmpHandler struct{}

func (i *icmpHandler) Handle(packet *IPv4Pdu) (*IPv4Pdu, error) {
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

func (l *internetLayerStrategy) GetHandler(ipProto IPProtocol) (TransportLayerHandler, error) {
	switch ipProto {
	case IPProtocolICMPv4:
		return l.icmpHandler, nil
	default:
		return nil, ErrNoInternetLayerHandler
	}
}
