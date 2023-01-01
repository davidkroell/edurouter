package edurouter

//go:generate mockgen -destination ./internal/mocks/mock_internet_layer_strategy.go -package mocks github.com/davidkroell/edurouter InternetLayerStrategy

type InternetLayerStrategy interface {
	GetHandler(ipProto IPProtocol) (TransportLayerHandler, error)
}

type InternetLayerStrategyImpl struct {
	icmpHandler *IcmpHandler
}

func NewInternetLayerStrategy(icmpHandler *IcmpHandler) *InternetLayerStrategyImpl {
	return &InternetLayerStrategyImpl{icmpHandler: icmpHandler}
}

func (l *InternetLayerStrategyImpl) GetHandler(ipProto IPProtocol) (TransportLayerHandler, error) {
	switch ipProto {
	case IPProtocolICMPv4:
		return l.icmpHandler, nil
	default:
		return nil, ErrNoInternetLayerHandler
	}
}
