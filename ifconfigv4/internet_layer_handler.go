package ifconfigv4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sort"
	"sync"
)

const (
	icmpv4HeaderLength = 8
)

type InternetLayerResultPdu interface {
	MarshalBinary() ([]byte, error)
	SrcIPAddr() net.IP
	DstIPAddr() net.IP
}

type InternetLayerHandler struct {
	internetLayerStrategy *internetLayerStrategy
	routeTable            *routeTable
}

type IcmpType uint8

const (
	IcmpTypeEchoRequest = 8
	IcmpTypeEchoReply   = 0
)

type ICMPPacket struct {
	IcmpType IcmpType
	IcmpCode uint8
	Checksum uint16
	Id       uint16
	Seq      uint16
	Data     []byte
}

func (icmp *ICMPPacket) UnmarshalBinary(data []byte) error {
	if len(data) < icmpv4HeaderLength {
		return io.ErrUnexpectedEOF
	}

	icmp.IcmpType = IcmpType(data[0])
	icmp.IcmpCode = data[1]

	icmp.Id = binary.BigEndian.Uint16(data[4:])
	icmp.Seq = binary.BigEndian.Uint16(data[6:])

	icmp.Data = data[icmpv4HeaderLength:]

	actualChecksum := binary.BigEndian.Uint16(data[2:4])

	data[2] = 0
	data[3] = 0
	// calculate checksum
	expectedChecksum := onesComplementChecksum(data)

	if actualChecksum != expectedChecksum {
		return errors.New("invalid icmp checksum")
	}

	return nil
}

func (icmp *ICMPPacket) MarshalBinary() ([]byte, error) {
	b := make([]byte, icmpv4HeaderLength+len(icmp.Data))

	b[0] = uint8(icmp.IcmpType)
	b[1] = icmp.IcmpCode
	binary.BigEndian.PutUint16(b[4:], icmp.Id)
	binary.BigEndian.PutUint16(b[6:], icmp.Seq)

	copy(b[icmpv4HeaderLength:], icmp.Data)

	b[2] = 0
	b[3] = 0
	icmp.Checksum = onesComplementChecksum(b)
	binary.BigEndian.PutUint16(b[2:4], icmp.Checksum)

	return b, nil
}

func (icmp *ICMPPacket) MakeResponse() {
	icmp.IcmpType = IcmpTypeEchoReply
}

func (nll *InternetLayerHandler) Handle(packet *IPv4Pdu, ifconfig *InterfaceConfig) (InternetLayerResultPdu, *InterfaceConfig, error) {
	if bytes.Equal(packet.DstIP, ifconfig.RealIPAddr.IP) {
		// this packet is for the real interface, not for the simulated one
		return nil, nil, ErrDropPdu
	}

	if bytes.Equal(packet.DstIP, ifconfig.Addr.IP) {
		// this packet has to be handled at the simulated IP address
		packet, err := nll.handleLocal(packet)
		return packet, ifconfig, err
	}

	return nll.routeTable.routePacket(packet)
}

func (nll *InternetLayerHandler) handleLocal(packet *IPv4Pdu) (InternetLayerResultPdu, error) {
	nextHandler, err := nll.internetLayerStrategy.GetHandler(packet.Protocol)
	if err != nil {
		return nil, ErrDropPdu
	}

	// todo what about the error?
	resultPacket, err := nextHandler.Handle(packet)

	if err != nil {
		return nil, ErrDropPdu
	}

	return resultPacket, nil
}

type routeType uint8

const (
	LinkLocalRouteType routeType = 0
	StaticRouteType    routeType = 1
)

func (r routeType) String() string {
	switch r {
	case LinkLocalRouteType:
		return "lo"
	case StaticRouteType:
		return "s"
	default:
		return ""
	}
}

type routeConfig struct {
	routeType    routeType
	destination  net.IPNet
	outInterface *InterfaceConfig
}

type routeTable struct {
	routeConfigs []routeConfig
	mu           sync.Mutex
}

func (table *routeTable) addRoute(config routeConfig) {
	table.mu.Lock()
	defer table.mu.Unlock()

	table.routeConfigs = append(table.routeConfigs, config)

	sort.Slice(table.routeConfigs, func(i, j int) bool {

		netIPi := binary.BigEndian.Uint32(table.routeConfigs[i].destination.IP)
		netIPj := binary.BigEndian.Uint32(table.routeConfigs[j].destination.IP)
		netMaskIPi := binary.BigEndian.Uint32(table.routeConfigs[i].destination.Mask)
		netMaskIPj := binary.BigEndian.Uint32(table.routeConfigs[j].destination.Mask)

		return table.routeConfigs[i].routeType < table.routeConfigs[j].routeType &&
			netIPi < netIPj &&
			netMaskIPi < netMaskIPj
	})
}

func (table *routeTable) routePacket(ip *IPv4Pdu) (*IPv4Pdu, *InterfaceConfig, error) {

	for _, rc := range table.routeConfigs {
		if bytes.Equal(ip.DstIP.Mask(rc.destination.Mask), rc.destination.IP) {
			// dst ip of the packet is inside this configured route table entry

			ipv4Result := NewIPv4Pdu(ip.SrcIP, ip.DstIP, ip.Protocol, ip.Payload)
			ipv4Result.TTL = ip.TTL - 1

			return ipv4Result, rc.outInterface, nil
		}
	}

	return nil, nil, ErrDropPdu
}
