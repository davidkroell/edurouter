package edurouter

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestParseInterfaceConfig(t *testing.T) {
	tests := map[string]struct {
		configString string
		wantConfig   *InterfaceConfig
		wantErr      error
	}{
		"ValidConfig": {
			configString: "eth0:192.168.0.1/24",
			wantConfig: &InterfaceConfig{
				InterfaceName: "eth0",
				Addr: &net.IPNet{
					IP:   net.IP{192, 168, 0, 1},
					Mask: net.CIDRMask(24, 32),
				},
			},
			wantErr: nil,
		},
		"Empty": {
			configString: "",
			wantConfig:   nil,
			wantErr:      ErrInvalidInterfaceConfigString,
		},
		"MissingInterface": {
			configString: "192.168.0.1/24",
			wantConfig:   nil,
			wantErr:      ErrInvalidInterfaceConfigString,
		},
		"MissingIP": {
			configString: "eth1:",
			wantConfig:   nil,
			wantErr: &net.ParseError{
				Type: "CIDR address",
				Text: "",
			},
		},
		"WrongIP": {
			configString: "eth2:321.123.321.123/12",
			wantConfig:   nil,
			wantErr: &net.ParseError{
				Type: "CIDR address",
				Text: "321.123.321.123/12",
			},
		},
		"WrongMask": {
			configString: "eth2:192.168.0.1/45",
			wantConfig:   nil,
			wantErr: &net.ParseError{
				Type: "CIDR address",
				Text: "192.168.0.1/45",
			},
		},
	}

	for name, v := range tests {
		t.Run(name, func(t *testing.T) {
			actualConfig, err := ParseInterfaceConfig(v.configString)
			assert.EqualValues(t, v.wantConfig, actualConfig)
			assert.EqualValues(t, v.wantErr, err)
		})
	}
}
