package edurouter

import (
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"net"
)

type ARPWriter interface {
	SendArpRequest(ip net.IP) error
}

type ARPv4Writer struct {
	ifconfig *InterfaceConfig
	c        net.PacketConn
}

func NewARPv4Writer(ifconfig *InterfaceConfig) *ARPv4Writer {
	return &ARPv4Writer{ifconfig: ifconfig}
}

func (a *ARPv4Writer) Initialize(c net.PacketConn) {
	a.c = c
}

func (a *ARPv4Writer) SendArpRequest(ip net.IP) error {
	if a.c == nil {
		return ErrARPPacketConn
	}

	req := ARPv4Pdu{
		HTYPE:           1,
		PTYPE:           ethernet.EtherTypeIPv4,
		HLEN:            HardwareAddrLen,
		PLEN:            net.IPv4len,
		Operation:       ARPOperationRequest,
		SrcHardwareAddr: *a.ifconfig.HardwareAddr,
		SrcProtoAddr:    a.ifconfig.Addr.IP,
		DstHardwareAddr: emptyHardwareAddr,
		DstProtoAddr:    ip,
	}

	bin, err := req.MarshalBinary()
	if err != nil {
		return err
	}

	frame := ethernet.Frame{
		Destination: ethernet.Broadcast,
		Source:      req.SrcHardwareAddr,
		EtherType:   ethernet.EtherTypeARP,
		Payload:     bin,
	}

	frameBinary, err := frame.MarshalBinary()

	if err != nil {
		return err
	}

	_, err = a.c.WriteTo(frameBinary, &raw.Addr{HardwareAddr: ethernet.Broadcast})
	return err
}
