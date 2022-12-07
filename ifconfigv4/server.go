package ifconfigv4

import (
	"errors"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"log"
	"net"
)

func ListenAndServe(ifconfig *InterfaceConfig) {
	// Select the interface to use for Ethernet traffic
	ifi, err := net.InterfaceByName(ifconfig.InterfaceName)
	if err != nil {
		log.Fatalf("failed to open interface: %v", err)
	}

	// choose ARP as EtherType (proto called here)
	c, err := raw.ListenPacket(ifi, arpEtherType, nil)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer c.Close()

	// Accept frames up to interface's MTU in size
	b := make([]byte, ifi.MTU)
	var f ethernet.Frame

	// Keep reading frames
	for {
		n, _, err := c.ReadFrom(b)
		if err != nil {
			log.Printf("failed to receive message: %v\n", err)
		}

		// Unpack Ethernet frame into Go representation.
		if err := (&f).UnmarshalBinary(b[:n]); err != nil {
			log.Printf("failed to unmarshal ethernet frame: %v\n", err)
		}

		var packet arpPacket

		// ARP logic
		err = (&packet).UnmarshalBinary(f.Payload)
		if err != nil {
			log.Printf("failed to unmarshall ethernet frame: %v\n", err)
			continue
		}

		err = handleArpPacket(c, &packet, ifconfig)
		if err != nil {
			log.Printf("failed to handle ethernet frame: %v\n", err)
		}
		log.Printf("handled packet successfully")
	}
}

func handleArpPacket(c *raw.Conn, packet *arpPacket, ifconfig *InterfaceConfig) error {
	if !packet.isEthernetAndIPv4() {
		return errors.New("unsupported ARP version. requires ethernet+IPv4")
	}

	if packet.isArpRequestForConfig(ifconfig) {
		response := packet.buildArpResponseWithConfig(ifconfig)

		etherResponse := &ethernet.Frame{
			Destination: response.TargetHardwareAddr,
			Source:      response.SenderHardwareAddr,
			EtherType:   0x0806,
			Payload:     response.MarshallBinary(),
		}

		frameBinary, err := etherResponse.MarshalBinary()
		if err != nil {
			return err
		}

		addr := &raw.Addr{
			HardwareAddr: response.TargetHardwareAddr,
		}

		_, err = c.WriteTo(frameBinary, addr)
		if err != nil {
			return err
		}
	}

	return nil
}
