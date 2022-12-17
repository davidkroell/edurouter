package ifconfigv4

import (
	"github.com/mdlayher/ethernet"
	"log"
)

type LinkLayerResultPdu interface {
	SenderHardwareAddr() []byte
	TargetHardwareAddr() []byte
	MarshalBinary() ([]byte, error)
}

type LinkLayerHandler interface {
	Handle(*ethernet.Frame, *InterfaceConfig) (*ethernet.Frame, error)
}

type arpv4LinkLayerHandler struct{}

type ipv4LinkLayerHandler struct {
	nextHandler *InternetLayerHandler
}

func (llh *arpv4LinkLayerHandler) Handle(f *ethernet.Frame, ifconfig *InterfaceConfig) (*ethernet.Frame, error) {
	var packet arpv4Pdu

	// ARP logic
	err := (&packet).UnmarshalBinary(f.Payload)
	if err != nil {
		return nil, err
	}

	if !packet.isEthernetAndIPv4() {
		return nil, ErrUnsupportedArpProtocol
	}

	if packet.isArpResponse() {
		ifconfig.arpTable.Store(packet.senderProtoAddr, packet.senderHardwareAddr)
		return nil, HandledPdu
	}

	if packet.isArpRequestForConfig(ifconfig) {
		arpResponse := packet.buildArpResponseWithConfig(ifconfig)

		arpBinary, err := arpResponse.MarshalBinary()
		if err != nil {
			return nil, ErrDropPdu
		}

		return &ethernet.Frame{
			Destination: f.Source,
			Source:      ifconfig.HardwareAddr,
			EtherType:   ethernet.EtherTypeARP,
			Payload:     arpBinary,
		}, nil
	}
	return nil, ErrDropPdu
}

func (llh *ipv4LinkLayerHandler) Handle(f *ethernet.Frame, ifconfig *InterfaceConfig) (*ethernet.Frame, error) {
	var ipv4Packet IPv4Pdu

	err := (&ipv4Packet).UnmarshalBinary(f.Payload)
	if err != nil {
		return nil, err
	}

	ifconfig.arpTable.Store(ipv4Packet.SrcIP, f.Source)

	result, outIfconfig, err := llh.nextHandler.Handle(&ipv4Packet, ifconfig)
	if err != nil {
		return nil, err
	}

	framePayload, err := result.MarshalBinary()
	if err != nil {
		log.Printf("failed to marshal frame to binary: %v", err)
	}

	targetHardwareAddr, err := outIfconfig.arpTable.Resolve(result.DstIPAddr())

	return &ethernet.Frame{
		Destination: targetHardwareAddr,
		Source:      outIfconfig.HardwareAddr,
		EtherType:   f.EtherType,
		Payload:     framePayload,
	}, nil
}
