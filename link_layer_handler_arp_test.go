package edurouter

import (
	"github.com/davidkroell/edurouter/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/mdlayher/ethernet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestARPv4LinkLayerHandler_HandleARPRequests(t *testing.T) {
	handler := NewARPv4LinkLayerHandler()

	config, err := NewInterfaceConfig("veth0", &net.IPNet{
		IP:   []byte{192, 168, 100, 1},
		Mask: net.CIDRMask(24, 32),
	})
	require.NoError(t, err)
	hwa := net.HardwareAddr([]byte{1, 1, 1, 2, 2, 2})
	config.HardwareAddr = &hwa

	tests := map[string]struct {
		inputArp      ARPv4Pdu
		wantErr       error
		wantArpResult *ARPv4Pdu
	}{
		"ARPRequestSuccessfulResponse": {
			inputArp: ARPv4Pdu{
				HTYPE:           HTYPEEthernet,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       ARPOperationRequest,
				SrcHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				SrcProtoAddr:    []byte{192, 168, 100, 100},
				DstHardwareAddr: EmptyHardwareAddr,
				DstProtoAddr:    []byte{192, 168, 100, 1},
			},
			wantErr: nil,
			wantArpResult: &ARPv4Pdu{
				HTYPE:           HTYPEEthernet,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       ARPOperationResponse,
				SrcHardwareAddr: hwa,
				SrcProtoAddr:    []byte{192, 168, 100, 1},
				DstHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				DstProtoAddr:    []byte{192, 168, 100, 100},
			},
		},
		"ErrUnsupportedArpProtocol": {
			inputArp: ARPv4Pdu{
				HTYPE:           2,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       ARPOperationRequest,
				SrcHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				SrcProtoAddr:    []byte{192, 168, 100, 100},
				DstHardwareAddr: EmptyHardwareAddr,
				DstProtoAddr:    []byte{192, 168, 100, 1},
			},
			wantErr:       ErrUnsupportedArpProtocol,
			wantArpResult: nil,
		},
		"ARPRequestNotForInterfaceConfig": {
			inputArp: ARPv4Pdu{
				HTYPE:           HTYPEEthernet,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       ARPOperationRequest,
				SrcHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				SrcProtoAddr:    []byte{192, 168, 100, 100},
				DstHardwareAddr: EmptyHardwareAddr,
				DstProtoAddr:    []byte{192, 168, 100, 50},
			},
			wantErr:       ErrDropPdu,
			wantArpResult: nil,
		},
	}

	for name, v := range tests {
		t.Run(name, func(t *testing.T) {

			arpBinary, err := v.inputArp.MarshalBinary()
			require.NoError(t, err)

			inFrame := ethernet.Frame{
				Destination: nil,
				Source:      nil,
				EtherType:   ethernet.EtherTypeARP,
				Payload:     arpBinary,
			}

			outFrame, err := handler.Handle(&inFrame, config)
			require.EqualValues(t, v.wantErr, err)

			if outFrame == nil {
				return
			}

			assert.EqualValues(t, hwa, outFrame.Source)

			var actualArpResponse ARPv4Pdu
			err = (&actualArpResponse).UnmarshalBinary(outFrame.Payload)

			assert.EqualValues(t, *v.wantArpResult, actualArpResponse)
		})
	}
}

func TestARPv4LinkLayerHandler_HandleARPResponse(t *testing.T) {
	handler := NewARPv4LinkLayerHandler()

	ctrl := gomock.NewController(t)

	config, err := NewInterfaceConfig("veth0", &net.IPNet{
		IP:   []byte{192, 168, 100, 1},
		Mask: net.CIDRMask(24, 32),
	})
	require.NoError(t, err)
	hwa := net.HardwareAddr([]byte{1, 1, 1, 2, 2, 2})
	config.HardwareAddr = &hwa
	mockArpWriter := mocks.NewMockARPWriter(ctrl)
	config.arpTable = NewARPv4Table(config, mockArpWriter)

	srcProtoAddr := []byte{192, 168, 100, 100}
	srcHardwareAddr := []byte{1, 1, 1, 3, 3, 3}
	inputArp := ARPv4Pdu{
		HTYPE:           HTYPEEthernet,
		PTYPE:           ethernet.EtherTypeARP,
		HLEN:            HardwareAddrLen,
		PLEN:            net.IPv4len,
		Operation:       ARPOperationResponse,
		SrcHardwareAddr: srcHardwareAddr,
		SrcProtoAddr:    srcProtoAddr,
		DstHardwareAddr: EmptyHardwareAddr,
		DstProtoAddr:    []byte{192, 168, 100, 1},
	}

	arpBinary, err := inputArp.MarshalBinary()
	require.NoError(t, err)

	inFrame := ethernet.Frame{
		Destination: nil,
		Source:      nil,
		EtherType:   ethernet.EtherTypeARP,
		Payload:     arpBinary,
	}

	outFrame, err := handler.Handle(&inFrame, config)
	require.EqualError(t, err, HandledPdu.Error())
	require.Nil(t, outFrame)

	actualHardwareAddr, err := config.arpTable.Resolve([]byte{192, 168, 100, 100})
	assert.NoError(t, err)
	assert.EqualValues(t, srcHardwareAddr, actualHardwareAddr)
}
