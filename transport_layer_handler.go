package edurouter

//go:generate mockgen -destination ./internal/mocks/mock_transport_layer_strategy.go -package mocks github.com/davidkroell/edurouter TransportLayerHandler

type TransportLayerHandler interface {
	SupplierC() chan<- *IPv4Pdu
}
