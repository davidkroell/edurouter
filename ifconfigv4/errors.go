package ifconfigv4

import "errors"

var (
	DropPduError                = errors.New("no action for given PDU found. dropping it")
	NoLinkLayerHandlerError     = errors.New("no link layer handler for given etherType found")
	UnsupportedArpProtocolError = errors.New("unsupported ARP version. requires ethernet+IPv4")

	HardwareAddrSizeError = errors.New("hardware address must be 6 byte")
	IPAddrSizeError       = errors.New("ip address must be 4 byte")
	CIDRMaskError         = errors.New("CIDR mask must be between 0 and 32")
)
