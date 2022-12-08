package ifconfigv4

import "github.com/mdlayher/ethernet"

type LinkLayerResultPdu interface {
	SenderHardwareAddr() []byte
	TargetHardwareAddr() []byte
	MarshalBinary() ([]byte, error)
}

type LinkLayerHandler interface {
	Handle(ethernet.Frame) (LinkLayerResultPdu, error)
}

type arpv4LinkLayerHandler struct {
	ifconfig *InterfaceConfig
}

type ipv4LinkLayerHandler struct {
	ifconfig *InterfaceConfig
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

	if packet.isArpRequestForConfig(llh.ifconfig) {
		return packet.buildArpResponseWithConfig(llh.ifconfig), nil
	}
	return nil, DropPduError
}
