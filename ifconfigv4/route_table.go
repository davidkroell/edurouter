package ifconfigv4

import (
	"bytes"
	"encoding/binary"
	"net"
	"sort"
	"sync"
)

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

type RouteInfo struct {
	RouteType    routeType
	DstNet       net.IPNet
	OutInterface *InterfaceConfig
	NextHop      *net.IP
}

type RouteTable struct {
	configuredRoutes []RouteInfo
	mu               sync.Mutex
}

func (table *RouteTable) AddRoute(config RouteInfo) {
	table.mu.Lock()
	defer table.mu.Unlock()

	table.configuredRoutes = append(table.configuredRoutes, config)

	sort.Slice(table.configuredRoutes, func(i, j int) bool {

		netIPi := binary.BigEndian.Uint32(table.configuredRoutes[i].DstNet.IP)
		netIPj := binary.BigEndian.Uint32(table.configuredRoutes[j].DstNet.IP)
		netMaskIPi := binary.BigEndian.Uint32(table.configuredRoutes[i].DstNet.Mask)
		netMaskIPj := binary.BigEndian.Uint32(table.configuredRoutes[j].DstNet.Mask)

		return table.configuredRoutes[i].RouteType < table.configuredRoutes[j].RouteType &&
			netIPi < netIPj &&
			netMaskIPi < netMaskIPj
	})
}

func (table *RouteTable) getRouteInfoForPacket(ip *IPv4Pdu) (*RouteInfo, error) {
	table.mu.Lock()
	defer table.mu.Unlock()

	for _, ri := range table.configuredRoutes {
		if bytes.Equal(ip.DstIP.Mask(ri.DstNet.Mask), ri.DstNet.IP) {
			// dst ip of the packet is inside this configured route table entries destination network
			return &ri, nil
		}
	}

	return nil, ErrDropPdu
}

func (table *RouteTable) routePacket(ip *IPv4Pdu) (*IPv4Pdu, *RouteInfo, error) {
	ri, err := table.getRouteInfoForPacket(ip)

	if err != nil {
		return nil, nil, err
	}

	ip.TTL--

	return ip, ri, nil
}
