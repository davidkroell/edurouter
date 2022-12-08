package ifconfigv4

const (
	ipAddrSize       = 4
	hardwareAddrSize = 6
	maxCidrSize      = 32
)

type InterfaceConfig struct {
	InterfaceName string
	HardwareAddr  []byte
	IPAddr        []byte
	CIDRMask      uint8
}

func NewInterfaceConfig(name string, macAddr []byte, ipAddr []byte, cidrMask uint8) (*InterfaceConfig, error) {
	if len(macAddr) != hardwareAddrSize {
		return nil, HardwareAddrSizeError
	}

	if len(ipAddr) != ipAddrSize {
		return nil, IPAddrSizeError
	}

	if cidrMask > maxCidrSize {
		return nil, CIDRMaskError
	}

	return &InterfaceConfig{
		InterfaceName: name,
		HardwareAddr:  macAddr,
		IPAddr:        ipAddr,
		CIDRMask:      cidrMask,
	}, nil
}
