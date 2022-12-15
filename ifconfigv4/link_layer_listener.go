package ifconfigv4

import (
	"context"
	"encoding/binary"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type LinkLayerListener struct {
	ifconfig  *InterfaceConfig
	strategy  linkLayerStrategy
	arp4Table *arp4Table
}

func NewListener(ifconfig *InterfaceConfig) *LinkLayerListener {
	arpTable := &arp4Table{
		ifconfig:     ifconfig,
		ipv4ToMacMap: map[uint32][]byte{},
		arpWriter: arpWriter{
			ifconfig: ifconfig,
		},
		mu: sync.Mutex{},
	}

	return &LinkLayerListener{
		ifconfig: ifconfig,
		strategy: linkLayerStrategy{
			arpHandler: &arpv4LinkLayerHandler{
				ifconfig:  ifconfig,
				arp4Table: arpTable,
			},
			ipv4Handler: &ipv4LinkLayerHandler{
				ifconfig:  ifconfig,
				arp4Table: arpTable,
				nextHandler: &InternetLayerHandler{
					ifconfig: ifconfig,
					internetLayerStrategy: &internetLayerStrategy{
						icmpHandler: &icmpHandler{},
					},
				},
			},
		},
		arp4Table: arpTable,
	}
}

func (d *LinkLayerListener) setArpResolverConn(conn net.PacketConn) {
	d.arp4Table.arpWriter.c = conn
}

func (d *LinkLayerListener) ListenAndServe(ctx context.Context) {
	// Select the interface to use for Ethernet traffic
	ifi, err := net.InterfaceByName(d.ifconfig.InterfaceName)
	if err != nil {
		log.Fatalf("failed to open interface: %v", err)
	}

	// map real hardware and IP addresses
	d.ifconfig.SetRealHardwareAddr(ifi.HardwareAddr)
	ifAddresses, err := ifi.Addrs()
	for _, ipAddr := range ifAddresses {
		if strings.Contains(ipAddr.String(), ".") {
			// is IPv4
			d.ifconfig.SetRealIP(ipAddr)
			break
		}
	}

	frameChan := make(chan ethernet.Frame)
	connections := map[ethernet.EtherType]net.PacketConn{}

	supportedEtherTypes := d.strategy.GetSupportedEtherTypes()
	for _, etherType := range supportedEtherTypes {
		conn, err := raw.ListenPacket(ifi, uint16(etherType), nil)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		if etherType == ethernet.EtherTypeARP {
			d.setArpResolverConn(conn)
		}

		connections[etherType] = conn
		go readFramesFromConn(ctx, ifi.MTU, conn, frameChan)
	}

	defer func() {
		for _, v := range connections {
			v.Close()
		}
	}()

	// read frames from supplier channel
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-frameChan:
			handler, err := d.strategy.GetHandler(f.EtherType)
			if err != nil {
				continue
			}

			ethernetResponse, err := handler.Handle(&f)
			if err == ErrDropPdu {
				continue
			}

			if err != nil {
				continue
			}

			frameBinary, err := ethernetResponse.MarshalBinary()
			if err != nil {
				log.Printf("failed to marshal Payload to binary: %v", err)
				continue
			}

			// TODO select correct interface based on SenderHardwareAddr
			//  this is required for routing, where a packet enters the router at a different
			//  interface than where it's leaving
			_, err = connections[f.EtherType].WriteTo(frameBinary, &raw.Addr{
				HardwareAddr: ethernetResponse.Destination,
			})

			if err != nil {
				log.Printf("failed to write ethernet frame: %v\n", err)
				continue
			}
		}
	}
}

type arpWriter struct {
	ifconfig *InterfaceConfig
	c        net.PacketConn
}

func (a *arpWriter) SendArpRequest(ipAddr []byte) {
	if a.c == nil {
		return
	}

	req := arpv4Pdu{
		hardwareType:       1,
		protoType:          ethernet.EtherTypeARP,
		hardwareLen:        hardwareAddrSize,
		protoLen:           ipAddrSize,
		operation:          arpOperationRequest,
		senderHardwareAddr: a.ifconfig.HardwareAddr,
		senderProtoAddr:    a.ifconfig.Addr.IP,
		targetHardwareAddr: emptyHardwareAddr,
		targetProtoAddr:    ipAddr,
	}

	bin, err := req.MarshalBinary()
	if err != nil {
		panic(err) // TODO proper error handling
	}

	_, err = a.c.WriteTo(bin, &raw.Addr{HardwareAddr: ethernet.Broadcast})
	if err != nil {
		panic(err) // TODO proper error handling
	}
}

type arp4Table struct {
	ifconfig     *InterfaceConfig
	ipv4ToMacMap map[uint32][]byte
	arpWriter    arpWriter
	mu           sync.Mutex
}

func (a *arp4Table) Store(ipAddr, macAddr []byte) {
	if len(macAddr) != hardwareAddrSize {
		return
	}

	if len(ipAddr) != ipAddrSize {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	ipv4NumFormat := binary.BigEndian.Uint32(ipAddr)
	a.ipv4ToMacMap[ipv4NumFormat] = macAddr
}

func (a *arp4Table) Resolve(ipAddr net.IP) ([]byte, error) {
	if len(ipAddr) != ipAddrSize {
		return nil, ErrNotAnIPv4Address
	}

	a.mu.Lock()

	ipv4NumFormat := binary.BigEndian.Uint32(ipAddr)
	if k, ok := a.ipv4ToMacMap[ipv4NumFormat]; ok {
		a.mu.Unlock()
		return k, nil
	}
	a.mu.Unlock()

	// Send ARP
	a.arpWriter.SendArpRequest(ipAddr)

	// Wait for result
	t := time.NewTimer(time.Millisecond * 10)
	defer t.Stop()
	for {
		<-t.C
		a.mu.Lock()
		if k, ok := a.ipv4ToMacMap[ipv4NumFormat]; ok {
			a.mu.Unlock()
			return k, nil
		}
		a.mu.Unlock()
	}
}

func readFramesFromConn(ctx context.Context, mtu int, conn net.PacketConn, outChan chan<- ethernet.Frame) {
	// TODO implement context close

	// Accept frames up to interface's MTU in size
	b := make([]byte, mtu)
	var f ethernet.Frame

	// Keep reading frames
	for {
		n, _, err := conn.ReadFrom(b)
		if err != nil {
			log.Printf("failed to receive message: %v\n", err)
			continue
		}

		// Unpack Ethernet frame into Go representation.
		if err := (&f).UnmarshalBinary(b[:n]); err != nil {
			log.Printf("failed to unmarshal ethernet frame: %v\n", err)
			continue
		}

		outChan <- f
	}
}
