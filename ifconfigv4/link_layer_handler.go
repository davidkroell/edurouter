package ifconfigv4

import (
	"github.com/mdlayher/ethernet"
)

type LinkLayerResultPdu interface {
	SenderHardwareAddr() []byte
	TargetHardwareAddr() []byte
	MarshalBinary() ([]byte, error)
}

type LinkLayerHandler interface {
	Handle(ethernet.Frame) (LinkLayerResultPdu, error)
}

type arpv4LinkLayerHandler struct {
	ifconfig  *InterfaceConfig
	arp4Table *arp4Table
}

type ipv4LinkLayerHandler struct {
	ifconfig    *InterfaceConfig
	arp4Table   *arp4Table
	nextHandler *NetworkLayerHandler
}

func (llh *arpv4LinkLayerHandler) Handle(f ethernet.Frame) (LinkLayerResultPdu, error) {
	var packet arpv4Pdu

	// ARP logic
	err := (&packet).UnmarshalBinary(f.Payload)
	if err != nil {
		return nil, err
	}

	if !packet.isEthernetAndIPv4() {
		return nil, UnsupportedArpProtocolError
	}

	if packet.isArpResponse() {
		llh.arp4Table.Store(packet.senderProtoAddr, packet.senderHardwareAddr)
		return nil, HandledPdu
	}

	if packet.isArpRequestForConfig(llh.ifconfig) {
		return packet.buildArpResponseWithConfig(llh.ifconfig), nil
	}
	return nil, DropPduError
}

type linkLayerWrappedResultPdu struct {
	senderHardwareAddr []byte
	targetHardwareAddr []byte
	ipv4Packet         NetworkLayerResultPdu
}

func (n linkLayerWrappedResultPdu) SenderHardwareAddr() []byte {
	return n.senderHardwareAddr
}

func (n linkLayerWrappedResultPdu) TargetHardwareAddr() []byte {
	return n.targetHardwareAddr
}

func (n linkLayerWrappedResultPdu) MarshalBinary() ([]byte, error) {
	return n.ipv4Packet.MarshalBinary()
}

func (llh *ipv4LinkLayerHandler) Handle(f ethernet.Frame) (LinkLayerResultPdu, error) {
	var ipv4Packet Ipv4Pdu

	err := (&ipv4Packet).UnmarshalBinary(f.Payload)
	if err != nil {
		return nil, err
	}

	llh.arp4Table.Store(ipv4Packet.sourceIp, f.Source)

	result, err := llh.nextHandler.Handle(&ipv4Packet)
	if err != nil {
		return nil, err
	}

	targetHardwareAddr, err := llh.arp4Table.Resolve(result.TargetIPAddr())

	return linkLayerWrappedResultPdu{
		llh.ifconfig.HardwareAddr,
		targetHardwareAddr,
		result,
	}, nil
}
