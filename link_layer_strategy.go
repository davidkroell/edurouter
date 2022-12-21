package edurouter

import (
	"github.com/mdlayher/ethernet"
)

type LinkLayerStrategy struct {
	strategies map[ethernet.EtherType]LinkLayerHandler
}

func NewLinkLayerStrategy(strategies map[ethernet.EtherType]LinkLayerHandler) *LinkLayerStrategy {
	return &LinkLayerStrategy{strategies: strategies}
}

func (l *LinkLayerStrategy) GetHandler(etherType ethernet.EtherType) (LinkLayerHandler, error) {
	strategy, ok := l.strategies[etherType]
	if ok {
		return strategy, nil
	}

	return nil, ErrNoLinkLayerHandler
}

func (l *LinkLayerStrategy) GetSupportedEtherTypes() []ethernet.EtherType {
	etherTypes := make([]ethernet.EtherType, len(l.strategies))

	i := 0
	for k := range l.strategies {
		etherTypes[i] = k
		i++
	}
	return etherTypes
}
