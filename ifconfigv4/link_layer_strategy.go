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
	case ethernet.EtherTypeIPv4:
		return l.ipv4Handler, nil
	default:
		return nil, ErrNoLinkLayerHandler
	}
}

func (l *linkLayerStrategy) GetSupportedEtherTypes() []ethernet.EtherType {
	return []ethernet.EtherType{ethernet.EtherTypeARP, ethernet.EtherTypeIPv4}
}
