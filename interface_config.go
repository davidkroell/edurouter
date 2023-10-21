package edurouter

import (
	"context"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"github.com/rs/zerolog/log"
	"net"
	"strings"
	"time"
)

const (
	HardwareAddrLen             = 6
	InterfaceConfigFormatString = "interfaceName:IPv4/Mask"
)

type InterfaceConfig struct {
	InterfaceName      string
	HardwareAddr       *net.HardwareAddr
	Addr               *net.IPNet
	RealIPAddr         *net.IPNet
	ArpTable           *ARPv4Table
	managedConnections map[ethernet.EtherType]net.PacketConn
}

func ParseInterfaceConfig(config string) (*InterfaceConfig, error) {
	splitted := strings.Split(config, ":")

	if len(splitted) != 2 {
		return nil, ErrInvalidInterfaceConfigString
	}

	name := splitted[0]
	ip, ipNet, err := net.ParseCIDR(splitted[1])

	if err != nil {
		return nil, err
	}

	ipNet.IP = ip

	return NewInterfaceConfig(name, ipNet)
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

func (i *InterfaceConfig) SetupAndListen(ctx context.Context, supportedEtherTypes []ethernet.EtherType, frameChan chan<- FrameIn) {
	arpWriter := NewARPv4Writer(i)

	i.ArpTable = NewARPv4Table(i, arpWriter)

	// Select the interface to use for Ethernet traffic
	ifi, err := net.InterfaceByName(i.InterfaceName)
	if err != nil {
		log.Error().Msgf("failed to open interface: %v", err)
	}

	// map real hardware and IP addresses
	i.HardwareAddr = &ifi.HardwareAddr
	ifAddresses, err := ifi.Addrs()
	for _, ipAddr := range ifAddresses {
		if strings.Contains(ipAddr.String(), ".") {

			// is IPv4, set real IP
			i.RealIPAddr = ipAddr.(*net.IPNet)
			i.RealIPAddr.IP = i.RealIPAddr.IP.To4()
			break
		}
	}

	i.managedConnections = map[ethernet.EtherType]net.PacketConn{}

	for _, etherType := range supportedEtherTypes {
		conn, err := raw.ListenPacket(ifi, uint16(etherType), nil)
		if err != nil {
			log.Error().Msgf("failed to listen: %v", err)
		}

		if etherType == ethernet.EtherTypeARP {
			// inject ARP PacketConn into arpWriter
			arpWriter.Initialize(conn)
		}

		i.managedConnections[etherType] = conn
		go i.readFramesFromConn(ctx, ifi.MTU, conn, frameChan)
	}
}

func (i *InterfaceConfig) readFramesFromConn(ctx context.Context, mtu int, conn net.PacketConn, outChan chan<- FrameIn) {
	// Accept frames up to interface's MTU in size
	b := make([]byte, mtu)
	var f ethernet.Frame

	// Keep reading frames
	for {
		select {
		case <-ctx.Done():
			_ = conn.Close()
			return
		case <-time.After(time.Nanosecond):
			// fall-through
		}

		n, _, err := conn.ReadFrom(b)
		if err != nil {
			log.Error().Msgf("failed to receive message: %v\n", err)
			continue
		}

		// Unpack Ethernet frame into Go representation.
		if err := (&f).UnmarshalBinary(b[:n]); err != nil {
			log.Error().Msgf("failed to unmarshal ethernet frame: %v\n", err)
			continue
		}

		outChan <- FrameIn{
			Frame:     &f,
			Interface: i,
		}
	}
}

func (i *InterfaceConfig) WriteFrame(f *ethernet.Frame) error {
	frameBinary, err := f.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = i.managedConnections[f.EtherType].WriteTo(frameBinary, &raw.Addr{
		HardwareAddr: f.Destination,
	})

	return err
}
