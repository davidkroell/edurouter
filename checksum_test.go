package edurouter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOnesComplementChecksum(t *testing.T) {
	tests := map[string]struct {
		inputBytes []byte
		want       uint16
	}{
		"zero":                    {inputBytes: []byte{0, 0}, want: 0xffff},
		"one":                     {inputBytes: []byte{0, 1}, want: 0xfffe},
		"10bytes random":          {inputBytes: []byte{42, 69, 42, 69, 42, 69, 42, 69, 42, 69}, want: 0x2ca6},
		"10bytes newpaltz sample": {inputBytes: []byte{0x23, 0xfb, 0x34, 0xc0, 0xa0, 0x90, 0xbc, 0xaf, 0xfc, 0x05}, want: 0x4dfe},
	}

	for name, v := range tests {
		t.Run(name, func(t *testing.T) {
			actualChecksum := onesComplementChecksum(v.inputBytes)
			assert.EqualValues(t, v.want, actualChecksum)
		})
	}
}
