package edurouter_test

import (
	"github.com/davidkroell/edurouter"
	"github.com/mdlayher/ethernet"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLinkLayerStrategy_GetHandler(t *testing.T) {
	arpHandler := edurouter.NewARPv4LinkLayerHandler(nil)
	strategy := edurouter.NewLinkLayerStrategy(map[ethernet.EtherType]edurouter.LinkLayerHandler{
		ethernet.EtherTypeARP: arpHandler,
	})

	tests := map[string]struct {
		etherType   ethernet.EtherType
		wantErr     error
		wantHandler edurouter.LinkLayerHandler
	}{
		"ARP": {
			etherType:   ethernet.EtherTypeARP,
			wantErr:     nil,
			wantHandler: arpHandler,
		},
		"IPv6": {
			etherType:   ethernet.EtherTypeIPv6,
			wantErr:     edurouter.ErrNoLinkLayerHandler,
			wantHandler: nil,
		},
	}

	for name, v := range tests {
		t.Run(name, func(t *testing.T) {
			handler, err := strategy.GetHandler(v.etherType)

			assert.EqualValues(t, err, v.wantErr)
			assert.EqualValues(t, handler, v.wantHandler)
		})
	}
}

func TestLinkLayerStrategy_GetSupportedEtherTypes(t *testing.T) {
	strategy := edurouter.NewLinkLayerStrategy(map[ethernet.EtherType]edurouter.LinkLayerHandler{
		ethernet.EtherTypeARP:  nil, // value not checked/used
		ethernet.EtherTypeIPv4: nil, // value not checked/used
	})

	actual := strategy.GetSupportedEtherTypes()

	assert.Contains(t, actual, ethernet.EtherTypeIPv4)
	assert.Contains(t, actual, ethernet.EtherTypeARP)
}
