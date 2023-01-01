package edurouter_test

import (
	"github.com/davidkroell/edurouter"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInternetLayerStrategyImpl_GetHandler(t *testing.T) {
	icmpHandler := &edurouter.IcmpHandler{}
	strategy := edurouter.NewInternetLayerStrategy(icmpHandler)

	tests := map[string]struct {
		ipProto     edurouter.IPProtocol
		wantErr     error
		wantHandler edurouter.TransportLayerHandler
	}{
		"ICMP": {
			ipProto:     edurouter.IPProtocolICMPv4,
			wantErr:     nil,
			wantHandler: icmpHandler,
		},
		// currently not implemented
		"TCP": {
			ipProto:     edurouter.IPProtocolTCP,
			wantErr:     edurouter.ErrNoInternetLayerHandler,
			wantHandler: nil,
		},
		// currently not implemented
		"UDP": {
			ipProto:     edurouter.IPProtocolUDP,
			wantErr:     edurouter.ErrNoInternetLayerHandler,
			wantHandler: nil,
		},
	}

	for name, v := range tests {
		t.Run(name, func(t *testing.T) {
			handler, err := strategy.GetHandler(v.ipProto)

			assert.EqualValues(t, err, v.wantErr)
			assert.EqualValues(t, handler, v.wantHandler)
		})
	}

}
