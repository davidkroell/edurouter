package ifconfigv4

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"log"
	"net"
	"sync"
	"time"
)

type LinkLayerListener struct {
	interfaces []*InterfaceConfig
	strategy   linkLayerStrategy
}

func NewListener(interfaces ...*InterfaceConfig) *LinkLayerListener {
	routeTable := &routeTable{
		routeConfigs: nil,
		mu:           sync.Mutex{},
	}

	for _, i := range interfaces {
		routeTable.addRoute(routeConfig{
			routeType: LinkLocalRouteType,
			destination: net.IPNet{
				IP:   i.Addr.IP.Mask(i.Addr.Mask),
				Mask: i.Addr.Mask,
			},
			outInterface: i,
		})
	}

	return &LinkLayerListener{
		interfaces: interfaces,
		strategy: linkLayerStrategy{
			arpHandler: &arpv4LinkLayerHandler{},
			ipv4Handler: &ipv4LinkLayerHandler{
				nextHandler: &InternetLayerHandler{
					internetLayerStrategy: &internetLayerStrategy{
						icmpHandler: &icmpHandler{},
					},
					routeTable: routeTable,
				},
			},
		},
	}
}

type frameFromInterface struct {
	frame       *ethernet.Frame
	inInterface *InterfaceConfig
}

func (listener *LinkLayerListener) ListenAndServe(ctx context.Context) {

	frameChan := make(chan frameFromInterface)
	supportedEtherTypes := listener.strategy.GetSupportedEtherTypes()

	for _, iface := range listener.interfaces {
		iface.SetupAndListen(ctx, supportedEtherTypes, frameChan)
	}

	// read frames from supplier channel
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-frameChan:
			handler, err := listener.strategy.GetHandler(f.frame.EtherType)
			if err != nil {
				continue
			}

			frameToSend, err := handler.Handle(f.frame, f.inInterface)
			if err == ErrDropPdu || err != nil {
				continue
			}

			for _, iface := range listener.interfaces {
				if bytes.Equal(iface.HardwareAddr, frameToSend.Source) {
					err = iface.WriteFrame(frameToSend)
					break
				}
			}

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

func (a *arpWriter) SendArpRequest(ipAddr []byte) error {
	if a.c == nil {
		return errors.New("outbound ARP-connection was nil")
	}

	req := arpv4Pdu{
		hardwareType:       1,
		protoType:          ethernet.EtherTypeIPv4,
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
		return err
	}

	frame := ethernet.Frame{
		Destination: ethernet.Broadcast,
		Source:      req.senderHardwareAddr,
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

	for i := 0; i <= 100; i++ {

		// every 10 milliseconds
		if i%10 == 0 {
			// Send ARP
			err := a.arpWriter.SendArpRequest(ipAddr)
			if err != nil {
				return nil, err
			}
		}

		a.mu.Lock()
		if k, ok := a.ipv4ToMacMap[ipv4NumFormat]; ok {
			a.mu.Unlock()
			return k, nil
		}
		a.mu.Unlock()

		time.Sleep(time.Millisecond * 10)
	}
	return nil, ErrArpTimeout
}
