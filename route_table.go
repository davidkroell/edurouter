package edurouter

import (
	"bytes"
	"encoding/binary"
	"net"
	"sort"
	"sync"
)

type RouteType uint8

const (
	LinkLocalRouteType RouteType = 0
	StaticRouteType    RouteType = 1
)

func (r RouteType) String() string {
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
	RouteType    RouteType
	DstNet       net.IPNet
	OutInterface *InterfaceConfig
	NextHop      *net.IP
}

func (ri *RouteInfo) Validate() error {
	isNetworkAddr := bytes.Equal(ri.DstNet.IP.Mask(ri.DstNet.Mask), ri.DstNet.IP)
	if !isNetworkAddr {
		return ErrNotANetworkAddress
	}

	if ri.RouteType == LinkLocalRouteType {
		if ri.NextHop != nil {
			return ErrLinkLocalRouteShouldNotHaveNextHop
		} else {
			return nil
		}
	}

	// check if OutInterface and NextHop are on same network by masking both with the
	// OutInterface subnet mask
	if ri.OutInterface == nil ||
		ri.NextHop == nil ||
		!bytes.Equal(ri.OutInterface.Addr.IP.Mask(ri.OutInterface.Addr.Mask),
			ri.NextHop.Mask(ri.OutInterface.Addr.Mask)) {
		return ErrNextHopNotOnLinkLocalNetwork
	}

	return nil
}

type RouteTable struct {
	configuredRoutes []RouteInfo
	mu               sync.RWMutex
}

func NewRouteTable() *RouteTable {
	return &RouteTable{
		configuredRoutes: make([]RouteInfo, 0),
		mu:               sync.RWMutex{},
	}
}

func (table *RouteTable) AddRoute(config RouteInfo) error {
	err := config.Validate()
	if err != nil {
		return err
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	table.configuredRoutes = append(table.configuredRoutes, config)

	// sort slice by most exact match
	sort.SliceStable(table.configuredRoutes, func(i, j int) bool {
		netIPi := binary.BigEndian.Uint32(table.configuredRoutes[i].DstNet.IP)
		netIPj := binary.BigEndian.Uint32(table.configuredRoutes[j].DstNet.IP)
		netMaskIPi := binary.BigEndian.Uint32(table.configuredRoutes[i].DstNet.Mask)
		netMaskIPj := binary.BigEndian.Uint32(table.configuredRoutes[j].DstNet.Mask)

		return table.configuredRoutes[i].RouteType < table.configuredRoutes[j].RouteType || // ascending by route type (pseudo-metric)
			// ascending by IP
			netIPi < netIPj ||
			// descending by netMask: means more exact match
			netMaskIPi > netMaskIPj
	})
	return nil
}

func (table *RouteTable) GetRoutes() []RouteInfo {
	table.mu.RLock()
	defer table.mu.RUnlock()

	r := make([]RouteInfo, len(table.configuredRoutes))

	copy(r, table.configuredRoutes)
	return r
}

func (table *RouteTable) DeleteRouteAtIndex(index uint) {
	table.mu.Lock()
	defer table.mu.Unlock()

	if int(index) >= len(table.configuredRoutes) {
		return
	}

	// ensure ordering to skip sorting afterwards
	table.configuredRoutes = append(table.configuredRoutes[:index], table.configuredRoutes[index+1:]...)
}

func (table *RouteTable) getRouteInfoForPacket(ip *IPv4Pdu) (*RouteInfo, error) {
	table.mu.RLock()
	defer table.mu.RUnlock()

	for _, ri := range table.configuredRoutes {
		if bytes.Equal(ip.DstIP.Mask(ri.DstNet.Mask), ri.DstNet.IP) {
			// dst ip of the packet is inside this configured route table entries destination network
			return &ri, nil
		}
	}

	return nil, ErrNoRoute
}

func (table *RouteTable) RoutePacket(ip *IPv4Pdu) (*IPv4Pdu, *RouteInfo, error) {
	ri, err := table.getRouteInfoForPacket(ip)

	if err != nil {
		return nil, nil, err
	}

	ip.TTL--

	if ip.TTL == 0 {
		// time to live ended, dropping
		return nil, nil, ErrDropPdu
	}

	return ip, ri, nil
}
