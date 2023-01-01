package edurouter

import "github.com/mdlayher/ethernet"

type ARPv4LinkLayerHandler struct{}

func NewARPv4LinkLayerHandler() *ARPv4LinkLayerHandler {
	return &ARPv4LinkLayerHandler{}
}
func (llh *ARPv4LinkLayerHandler) Handle(f *ethernet.Frame, ifconfig *InterfaceConfig) (*ethernet.Frame, error) {
	var packet ARPv4Pdu

	// ARP logic
	err := (&packet).UnmarshalBinary(f.Payload)
	if err != nil {
		return nil, err
	}

	if !packet.IsEthernetAndIPv4() {
		return nil, ErrUnsupportedArpProtocol
	}

	if packet.IsArpResponse() {
		ifconfig.ArpTable.Store(packet.SrcProtoAddr, packet.SrcHardwareAddr)
		return nil, HandledPdu
	}

	if packet.IsArpRequestForConfig(ifconfig) {
		arpResponse := packet.BuildARPResponseWithConfig(ifconfig)

		arpBinary, err := arpResponse.MarshalBinary()
		if err != nil {
			return nil, ErrDropPdu
		}

		return &ethernet.Frame{
			Destination: f.Source,
			Source:      *ifconfig.HardwareAddr,
			EtherType:   ethernet.EtherTypeARP,
			Payload:     arpBinary,
		}, nil
	}
	return nil, ErrDropPdu
}
