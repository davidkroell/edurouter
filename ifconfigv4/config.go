package ifconfigv4

import (
	"errors"
)

const (
	ipv4EtherType    = 0x0800
	arpEtherType     = 0x0806
	ipAddrSize       = 4
	hardwareAddrSize = 6
)

type InterfaceConfig struct {
	InterfaceName string
	HardwareAddr  []byte
	IPAddr        []byte
	CIDRMask      uint8
}

func NewInterfaceConfig(name string, macAddr []byte, ipAddr []byte, cidrMask uint8) (*InterfaceConfig, error) {
	if len(macAddr) != hardwareAddrSize {
		return nil, errors.New("hardware address must be 6 byte")
	}

	if len(ipAddr) != ipAddrSize {
		return nil, errors.New("ip must be 4 byte")
	}

	if cidrMask > 32 {
		return nil, errors.New("CIDR mask must be between 0 and 32")
	}

	return &InterfaceConfig{
		InterfaceName: name,
		HardwareAddr:  macAddr,
		IPAddr:        ipAddr,
		CIDRMask:      cidrMask,
	}, nil
}
