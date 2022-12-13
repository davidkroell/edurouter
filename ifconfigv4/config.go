package ifconfigv4

import "net"

const (
	ipAddrSize       = 4
	hardwareAddrSize = 6
)

type InterfaceConfig struct {
	InterfaceName string
	HardwareAddr  []byte
	Addr          *net.IPNet
	RealIPAddr    *net.IPNet
}

func NewInterfaceConfig(name string, addr *net.IPNet) (*InterfaceConfig, error) {
	if addr.IP.To4() == nil {
		return nil, ErrNotAnIPv4Address
	}
	addr.IP = addr.IP.To4()

	return &InterfaceConfig{
		InterfaceName: name,
		Addr:          addr,
	}, nil
}

func (i *InterfaceConfig) SetRealIP(addr net.Addr) {
	i.RealIPAddr = addr.(*net.IPNet)
	i.RealIPAddr.IP = i.RealIPAddr.IP.To4()
}

func (i *InterfaceConfig) SetRealHardwareAddr(addr net.HardwareAddr) {
	i.HardwareAddr = addr
}
