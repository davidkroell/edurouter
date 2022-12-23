package edurouter

import (
	"encoding/binary"
	"net"
	"sync"
	"time"
)

type ARPv4Table struct {
	ifconfig     *InterfaceConfig
	ipv4ToMacMap map[uint32]net.HardwareAddr
	arpWriter    ARPWriter
	mu           sync.Mutex
}

func NewARPv4Table(ifconfig *InterfaceConfig, arpWriter ARPWriter) *ARPv4Table {
	return &ARPv4Table{
		ifconfig:     ifconfig,
		ipv4ToMacMap: make(map[uint32]net.HardwareAddr),
		arpWriter:    arpWriter,
		mu:           sync.Mutex{}}
}

func (a *ARPv4Table) Store(ipAddr, macAddr []byte) error {
	if len(macAddr) != HardwareAddrLen {
		return ErrNotAnMACHardwareAddress
	}

	if len(ipAddr) != net.IPv4len {
		return ErrNotAnIPv4Address
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	ipv4NumFormat := binary.BigEndian.Uint32(ipAddr)
	a.ipv4ToMacMap[ipv4NumFormat] = macAddr
	return nil
}

func (a *ARPv4Table) Resolve(ipAddr net.IP) ([]byte, error) {
	if len(ipAddr) != net.IPv4len {
		return nil, ErrNotAnIPv4Address
	}

	ipv4NumFormat := binary.BigEndian.Uint32(ipAddr)

	const numArpRequests = 10
	const checkIntervalMillis = 10

	for i := 0; i < numArpRequests*checkIntervalMillis; i++ {
		hwAddr, found := a.resolveFromCache(ipv4NumFormat)
		if found {
			return hwAddr, nil
		}

		time.Sleep(time.Millisecond * checkIntervalMillis)

		// every 100 milliseconds
		if i%numArpRequests == 0 {
			// Send ARP
			err := a.arpWriter.SendArpRequest(ipAddr)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, ErrARPTimeout
}

func (a *ARPv4Table) resolveFromCache(ipv4NumFormat uint32) ([]byte, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if k, ok := a.ipv4ToMacMap[ipv4NumFormat]; ok {
		return k, true
	}
	return nil, false
}
