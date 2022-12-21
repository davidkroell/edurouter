package edurouter

type TransportLayerHandler interface {
	Handle(*IPv4Pdu) (*IPv4Pdu, error)
}
