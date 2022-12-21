package edurouter

import "errors"

var (
	HandledPdu                = errors.New("this pdu is processed. this is intended behaviour")
	ErrDropPdu                = errors.New("no action for given PDU found. dropping it")
	ErrNoLinkLayerHandler     = errors.New("no link layer handler for given etherType found")
	ErrUnsupportedArpProtocol = errors.New("unsupported ARP Version. requires ethernet+IPv4")

	ErrNotAnIPv4Address       = errors.New("ip address it not an IPv4 address")
	ErrNoInternetLayerHandler = errors.New("no internet layer handler for given IPProtocol found")
	ErrARPTimeout             = errors.New("ARP timeout. no MAC found for this IP Address")
	ErrARPPacketConn          = errors.New("outbound PacketConn was nil")
)
