package edurouter

import "errors"

var (
	HandledPdu                = errors.New("this pdu is processed. this is intended behaviour")
	ErrDropPdu                = errors.New("no action for given PDU found. dropping it")
	ErrNoLinkLayerHandler     = errors.New("no link layer handler for given etherType found")
	ErrUnsupportedArpProtocol = errors.New("unsupported ARP Version. requires ethernet+IPv4")

	ErrNotAnMACHardwareAddress = errors.New("provided hardware address was no MAC address")

	ErrNotAnIPv4Address       = errors.New("ip address it not an IPv4 address")
	ErrNoInternetLayerHandler = errors.New("no internet layer handler for given IPProtocol found")
	ErrARPTimeout             = errors.New("ARP timeout. no MAC found for this IP Address")
	ErrARPPacketConn          = errors.New("outbound PacketConn was nil")

	ErrNotANetworkAddress                 = errors.New("not a correct network address")
	ErrNextHopNotOnLinkLocalNetwork       = errors.New("next hop is not on local network of the outbound interface")
	ErrLinkLocalRouteShouldNotHaveNextHop = errors.New("a link-local route should not have a next hop address defined")

	ErrInvalidInterfaceConfigString = errors.New("invalid interface config string. malformed input, should have following format: '" + InterfaceConfigFormatString + "'")
)
