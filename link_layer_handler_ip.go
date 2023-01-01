package edurouter

import "github.com/mdlayher/ethernet"

type IPv4LinkLayerHandler struct {
	internetLayerHandler *Internetv4LayerHandler
}

func NewIPv4LinkLayerHandler(internetLayerHandler *Internetv4LayerHandler) *IPv4LinkLayerHandler {
	return &IPv4LinkLayerHandler{internetLayerHandler: internetLayerHandler}
}

func (llh *IPv4LinkLayerHandler) Handle(f *ethernet.Frame, ifconfig *InterfaceConfig) (*ethernet.Frame, error) {
	var ipv4Packet IPv4Pdu

	err := (&ipv4Packet).UnmarshalBinary(f.Payload)
	if err != nil {
		return nil, err
	}

	// TODO handle error
	ifconfig.ArpTable.Store(ipv4Packet.SrcIP, f.Source)

	result, routeInfo, err := llh.internetLayerHandler.Handle(&ipv4Packet, ifconfig)
	if err != nil {
		return nil, err
	}

	framePayload, err := result.MarshalBinary()
	if err != nil {
		return nil, err
	}

	outFrame := &ethernet.Frame{
		Source:    *routeInfo.OutInterface.HardwareAddr,
		EtherType: f.EtherType,
		Payload:   framePayload,
	}

	if routeInfo.RouteType == LinkLocalRouteType {
		outFrame.Destination, err = routeInfo.OutInterface.ArpTable.Resolve(result.DstIP)
	} else {
		outFrame.Destination, err = routeInfo.OutInterface.ArpTable.Resolve(*routeInfo.NextHop)
	}

	if err != nil {
		return nil, err
	}

	return outFrame, nil
}
