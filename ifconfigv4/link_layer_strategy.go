package ifconfigv4

import "github.com/mdlayher/ethernet"

type linkLayerStrategy struct {
	arpHandler  *arpv4LinkLayerHandler
	ipv4Handler *ipv4LinkLayerHandler
}

func (l *linkLayerStrategy) GetHandler(etherType ethernet.EtherType) (LinkLayerHandler, error) {
	switch etherType {
	case ethernet.EtherTypeARP:
		return l.arpHandler, nil
		//case ipv4EtherType:
		//	return l.ipv4Handler, nil
	default:
		return nil, NoLinkLayerHandlerError
	}
}

func (l *linkLayerStrategy) GetSupportedEtherTypes() []ethernet.EtherType {
	return []ethernet.EtherType{ethernet.EtherTypeARP, ethernet.EtherTypeIPv4}
}
