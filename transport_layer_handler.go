package edurouter

//go:generate mockgen -destination ./internal/mocks/mock_transport_layer_strategy.go -package mocks github.com/davidkroell/edurouter TransportLayerHandler

type TransportLayerHandler interface {
	Handle(*IPv4Pdu) (*IPv4Pdu, error)
}
