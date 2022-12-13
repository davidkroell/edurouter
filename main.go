package main

import (
	"context"
	"fmt"
	"github.com/davidkroell/edurouter/ifconfigv4"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"log"
	"net"
	"net/http"
)

func main() {
	interfaceConfig, err := ifconfigv4.NewInterfaceConfig("wlp0s20f3", []byte{0x0, 0x93, 0x37, 0x79, 0x06, 0x85}, []byte{192, 168, 0, 80}, 24)
	if err != nil {
		log.Println(err)
		return
	}

	ctx := context.Background()

	listener := ifconfigv4.NewListener(interfaceConfig)

	listener.ListenAndServe(ctx)
}

func main1() {
	go func() {
		http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte("ok"))
		})
		http.ListenAndServe(":8080", nil)
	}()

	// Select the interface to use for Ethernet traffic
	ifi, err := net.InterfaceByName("wlp0s20f3")
	if err != nil {
		log.Fatalf("failed to open interface: %v", err)
	}

	// choose IPv4 as EtherType (proto called here)
	c, err := raw.ListenPacket(ifi, 0x0800, nil)
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

		//log.Printf("Recieved frame from %s", addr.String())
		unpackIPv4Packet(f.Payload)
	}
}

func unpackIPv4Packet(payload []byte) {
	version := 4 >> (payload[0] & 0b1111_0000) >> 4 // select first 4 bits and shift right 4 times

	// totalLength is stored in byte 2-3, byte2 is the first part of the unit16, therefore shift
	totalLength := uint16(payload[2])<<8 | uint16(payload[3])

	// detect inner protocol
	innerProto := payload[9]

	ttl := payload[8]

	sourceIp := payload[12:16]
	destIp := payload[16:20]

	if sourceIp[0] != 192 {
		return
	}

	fmt.Println("==== IPv4 Header details")
	fmt.Printf("ver: %d, totalLength: %d, innerProto: %d, ttl: %d\n", version, totalLength, innerProto, ttl)
	fmt.Printf("source: %d.%d.%d.%d ", sourceIp[0], sourceIp[1], sourceIp[2], sourceIp[3])
	fmt.Printf("dest  : %d.%d.%d.%d\n", destIp[0], destIp[1], destIp[2], destIp[3])

	var datagram []byte

	if totalLength > 20 {
		datagram = payload[20:]

		switch innerProto {
		case 1:
			// innerProto 1 = ICMP
			unpackIcmpDatagram(datagram)
		case 6:
			// innerProto 6 = TCP
			unpackTcpDatagram(datagram)
		}
	}
	fmt.Println("==== IPv4 end")
}

func unpackTcpDatagram(datagram []byte) {
	// totalLength is stored in byte 2-3, byte2 is the first part of the unit16, therefore shift
	sourcePort := uint16(datagram[0])<<8 | uint16(datagram[1])
	destPort := uint16(datagram[2])<<8 | uint16(datagram[3])

	seqNumber := uint32(datagram[4])<<24 | uint32(datagram[5])<<16 | uint32(datagram[6])<<8 | uint32(datagram[7])
	ackNum := uint32(datagram[8])<<24 | uint32(datagram[9])<<16 | uint32(datagram[10])<<8 | uint32(datagram[11])

	tcpFlags := datagram[13]

	fmt.Println("======== TCP details")
	fmt.Printf("src: %d, dst: %d\n", sourcePort, destPort)
	fmt.Printf("seqNum: %d, ackNum: %d\n", seqNumber, ackNum)
	fmt.Printf("tcpFlags: %08b\n", tcpFlags)
	fmt.Println("======== TCP end")
}

func unpackIcmpDatagram(datagram []byte) {
	icmpType := datagram[0]

	fmt.Println("======== ICMP details")
	fmt.Printf("ICMP type: %d\n", icmpType)
}
