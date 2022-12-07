package arp

import (
	"bytes"
	"errors"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"log"
	"net"
)

type Config struct {
	InterfaceName string
	MACAddr       []byte
	IPAddr        []byte
}

func NewConfig(interfaceName string, macAddr []byte, ipAddr []byte) (*Config, error) {
	if len(macAddr) != 6 {
		return nil, errors.New("mac must be 6 byte")
	}

	if len(ipAddr) != 4 {
		return nil, errors.New("ip must be 4 byte")
	}

	return &Config{
		InterfaceName: interfaceName,
		MACAddr:       macAddr,
		IPAddr:        ipAddr,
	}, nil
}

func ListenAndServe(arpConfig *Config) {

	// Select the interface to use for Ethernet traffic
	ifi, err := net.InterfaceByName(arpConfig.InterfaceName)
	if err != nil {
		log.Fatalf("failed to open interface: %v", err)
	}

	// choose ARP as EtherType (proto called here)
	c, err := raw.ListenPacket(ifi, 0x0806, nil)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer c.Close()

	// Accept frames up to interface's MTU in size.
	b := make([]byte, ifi.MTU)
	var f ethernet.Frame

	// Keep reading frames.
	for {
		n, _, err := c.ReadFrom(b)
		if err != nil {
			log.Fatalf("failed to receive message: %v", err)
		}

		// Unpack Ethernet frame into Go representation.
		if err := (&f).UnmarshalBinary(b[:n]); err != nil {
			log.Fatalf("failed to unmarshal ethernet frame: %v", err)
		}

		// ARP logic
		packet, err := getArpPacketFromBytes(f.Payload)

		if err != nil {
			log.Println("Error unmarshalling ethernet frame to ARP packet")
			continue
		}

		if !packet.isEthernetAndIPv4Arp() {
			log.Println("Was not ethernet+IPv4")
			continue
		}

		if packet.isArpRequestForConfig(arpConfig) {
			response := packet.buildArpResponseForAddr(arpConfig)

			etherResponse := &ethernet.Frame{
				Destination: response.TargetHardwareAddr,
				Source:      response.SenderHardwareAddr,
				EtherType:   0x0806,
				Payload:     response.MarshallBinary(),
			}

			frameBinary, err := etherResponse.MarshalBinary()
			if err != nil {
				log.Println("error in marshalling frame to binary")
			}

			addr := &raw.Addr{
				HardwareAddr: response.TargetHardwareAddr,
			}

			_, err = c.WriteTo(frameBinary, addr)
			if err != nil {
				log.Println("Error writing ethernet frame to interface")
			}
		}

	}
}

type arpPacket struct {
	HTYPE              uint16
	PTYPE              uint16
	HLEN               uint8
	PLEN               uint8
	Operation          uint16
	SenderHardwareAddr []byte
	SenderProtoAddr    []byte
	TargetHardwareAddr []byte
	TargetProtoAddr    []byte
}

func (a *arpPacket) isEthernetAndIPv4Arp() bool {
	if a.HTYPE != 1 {
		// not ethernet
		return false
	}

	if a.PTYPE != 0x0800 {
		// not IPv4
		return false
	}

	if a.HLEN != 6 {
		// MAC's are 6 bytes
		return false
	}

	if a.PLEN != 4 {
		// IP's are 4 bytes
		return false
	}

	return true
}

var broadcastHardwareAddr = []byte{
	0xff, 0xff, 0xff,
	0xff, 0xff, 0xff,
}
var emptyHardwareAddr = []byte{
	0x0, 0x0, 0x0,
	0x0, 0x0, 0x0,
}

func (a *arpPacket) isArpRequestForConfig(config *Config) bool {
	if a.Operation != 1 {
		// not request
		return false
	}

	if !bytes.Equal(a.TargetHardwareAddr, emptyHardwareAddr) {
		// something went wrong, should be empty
		return false
	}

	if !bytes.Equal(a.TargetProtoAddr, config.IPAddr) {
		// targetAddr should be the same
		return false
	}
	return true
}

func (a *arpPacket) buildArpResponseForAddr(config *Config) *arpPacket {
	return &arpPacket{
		HTYPE:     a.HTYPE,
		PTYPE:     a.PTYPE,
		HLEN:      a.HLEN,
		PLEN:      a.PLEN,
		Operation: 2, // response

		// provide my mac as sender
		SenderHardwareAddr: config.MACAddr,
		SenderProtoAddr:    config.IPAddr,

		// flip original sender to target
		TargetHardwareAddr: a.SenderHardwareAddr,
		TargetProtoAddr:    a.TargetProtoAddr,
	}
}

func (a *arpPacket) MarshallBinary() []byte {
	b := make([]byte, 28)

	b[0] = byte(a.HTYPE >> 8)
	b[1] = byte(a.HTYPE)
	b[2] = byte(a.PTYPE >> 8)
	b[3] = byte(a.PTYPE)
	b[4] = a.HLEN
	b[5] = a.PLEN
	b[6] = byte(a.Operation >> 8)
	b[7] = byte(a.Operation)

	copy(b[8:], a.SenderHardwareAddr)
	copy(b[14:], a.SenderProtoAddr)
	copy(b[18:], a.TargetHardwareAddr)
	copy(b[24:], a.TargetProtoAddr)

	return b
}

func getArpPacketFromBytes(payload []byte) (*arpPacket, error) {
	if len(payload) < 27 {
		return nil, errors.New("arp packet too small")
	}

	return &arpPacket{
		HTYPE:              uint16(payload[0])<<8 | uint16(payload[1]),
		PTYPE:              uint16(payload[2])<<8 | uint16(payload[3]),
		HLEN:               payload[4],
		PLEN:               payload[5],
		Operation:          uint16(payload[6])<<8 | uint16(payload[7]),
		SenderHardwareAddr: payload[8:14],
		SenderProtoAddr:    payload[14:18],
		TargetHardwareAddr: payload[18:24],
		TargetProtoAddr:    payload[24:28],
	}, nil
}
