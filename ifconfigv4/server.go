package ifconfigv4

import (
	"context"
	"errors"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"log"
	"net"
)

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

func (d *LinkLayerListener) ListenAndServe(ctx context.Context) {
	// Select the interface to use for Ethernet traffic
	ifi, err := net.InterfaceByName(d.ifconfig.InterfaceName)
	if err != nil {
		log.Fatalf("failed to open interface: %v", err)
	}

	frameChan := make(chan ethernet.Frame)
	connections := map[ethernet.EtherType]net.PacketConn{}

	supportedEtherTypes := d.strategy.GetSupportedEtherTypes()
	for _, etherType := range supportedEtherTypes {
		conn, err := raw.ListenPacket(ifi, uint16(etherType), nil)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		defer conn.Close()

		connections[etherType] = conn
		go readFramesFromConn(ctx, ifi.MTU, conn, frameChan)
	}

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

			response, err := handler.Handle(f)
			if err == DropPduError {
				continue
			}

			framePayload, err := response.MarshalBinary()
			if err != nil {
				log.Printf("failed to marshal payload to binary: %v", err)
				continue
			}

			etherResponse := &ethernet.Frame{
				Destination: response.TargetHardwareAddr(),
				Source:      response.SenderHardwareAddr(),
				EtherType:   f.EtherType,
				Payload:     framePayload,
			}

			frameBinary, err := etherResponse.MarshalBinary()
			if err != nil {
				log.Printf("failed to marshal frame to binary: %v", err)
				continue
			}

			addr := &raw.Addr{
				HardwareAddr: response.TargetHardwareAddr(),
			}

			_, err = connections[f.EtherType].WriteTo(frameBinary, addr)

			if err != nil {
				log.Printf("failed to write ethernet frame: %v\n", err)
				continue
			}
		}
	}
}

type LinkLayerListener struct {
	ifconfig *InterfaceConfig
	strategy linkLayerStrategy
}

func NewListener(ifconfig *InterfaceConfig) *LinkLayerListener {
	return &LinkLayerListener{
		ifconfig: ifconfig,
		strategy: linkLayerStrategy{
			arpHandler: &arpLinkLayerHandler{
				ifconfig: ifconfig,
			},
			ipv4Handler: &ipv4LinkLayerHandler{
				ifconfig: ifconfig,
			},
		},
	}
}

type linkLayerStrategy struct {
	arpHandler  *arpLinkLayerHandler
	ipv4Handler *ipv4LinkLayerHandler
}

func (l *linkLayerStrategy) GetHandler(etherType ethernet.EtherType) (LinkLayerHandler, error) {
	switch etherType {
	case arpEtherType:
		return l.arpHandler, nil
		//case ipv4EtherType:
		//	return l.ipv4Handler, nil
	default:
		return nil, NoLinkLayerHandlerError
	}
}

func (l *linkLayerStrategy) GetSupportedEtherTypes() []ethernet.EtherType {
	return []ethernet.EtherType{arpEtherType, ipv4EtherType}
}

type LinkLayerHandler interface {
	Handle(ethernet.Frame) (LinkLayerResultPdu, error)
}

type arpLinkLayerHandler struct {
	ifconfig *InterfaceConfig
}

type ipv4LinkLayerHandler struct {
	ifconfig *InterfaceConfig
}

func (llh *arpLinkLayerHandler) Handle(f ethernet.Frame) (LinkLayerResultPdu, error) {
	var packet arpPacket

	// ARP logic
	err := (&packet).UnmarshalBinary(f.Payload)
	if err != nil {
		log.Printf("failed to unmarshall ethernet frame: %v\n", err)
	}

	if !packet.isEthernetAndIPv4() {
		return nil, errors.New("unsupported ARP version. requires ethernet+IPv4")
	}

	if packet.isArpRequestForConfig(llh.ifconfig) {
		return packet.buildArpResponseWithConfig(llh.ifconfig), nil
	}
	return nil, DropPduError
}

var DropPduError = errors.New("no action for given PDU found. dropping it")
var NoLinkLayerHandlerError = errors.New("no link layer handler for given etherType found")
