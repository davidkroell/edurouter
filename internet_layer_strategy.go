package edurouter

// TODO refactor this with interface
type InternetLayerStrategy struct {
	icmpHandler *icmpHandler
}

func NewInternetLayerStrategy(icmpHandler *icmpHandler) *InternetLayerStrategy {
	return &InternetLayerStrategy{icmpHandler: icmpHandler}
}

func (l *InternetLayerStrategy) GetHandler(ipProto IPProtocol) (TransportLayerHandler, error) {
	switch ipProto {
	case IPProtocolICMPv4:
		return l.icmpHandler, nil
	default:
		return nil, ErrNoInternetLayerHandler
	}
}
