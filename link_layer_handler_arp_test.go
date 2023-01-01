package edurouter_test

import (
	"github.com/davidkroell/edurouter"
	"github.com/davidkroell/edurouter/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/mdlayher/ethernet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestARPv4LinkLayerHandler_HandleARPRequests(t *testing.T) {
	handler := edurouter.NewARPv4LinkLayerHandler()

	config, err := edurouter.NewInterfaceConfig("veth0", &net.IPNet{
		IP:   []byte{192, 168, 100, 1},
		Mask: net.CIDRMask(24, 32),
	})
	require.NoError(t, err)
	hwa := net.HardwareAddr([]byte{1, 1, 1, 2, 2, 2})
	config.HardwareAddr = &hwa

	tests := map[string]struct {
		inputArp      edurouter.ARPv4Pdu
		wantErr       error
		wantArpResult *edurouter.ARPv4Pdu
	}{
		"ARPRequestSuccessfulResponse": {
			inputArp: edurouter.ARPv4Pdu{
				HTYPE:           edurouter.HTYPEEthernet,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            edurouter.HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       edurouter.ARPOperationRequest,
				SrcHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				SrcProtoAddr:    []byte{192, 168, 100, 100},
				DstHardwareAddr: edurouter.EmptyHardwareAddr,
				DstProtoAddr:    []byte{192, 168, 100, 1},
			},
			wantErr: nil,
			wantArpResult: &edurouter.ARPv4Pdu{
				HTYPE:           edurouter.HTYPEEthernet,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            edurouter.HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       edurouter.ARPOperationResponse,
				SrcHardwareAddr: hwa,
				SrcProtoAddr:    []byte{192, 168, 100, 1},
				DstHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				DstProtoAddr:    []byte{192, 168, 100, 100},
			},
		},
		"ErrUnsupportedArpProtocol": {
			inputArp: edurouter.ARPv4Pdu{
				HTYPE:           2,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            edurouter.HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       edurouter.ARPOperationRequest,
				SrcHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				SrcProtoAddr:    []byte{192, 168, 100, 100},
				DstHardwareAddr: edurouter.EmptyHardwareAddr,
				DstProtoAddr:    []byte{192, 168, 100, 1},
			},
			wantErr:       edurouter.ErrUnsupportedArpProtocol,
			wantArpResult: nil,
		},
		"ARPRequestNotForInterfaceConfig": {
			inputArp: edurouter.ARPv4Pdu{
				HTYPE:           edurouter.HTYPEEthernet,
				PTYPE:           ethernet.EtherTypeARP,
				HLEN:            edurouter.HardwareAddrLen,
				PLEN:            net.IPv4len,
				Operation:       edurouter.ARPOperationRequest,
				SrcHardwareAddr: []byte{1, 1, 1, 3, 3, 3},
				SrcProtoAddr:    []byte{192, 168, 100, 100},
				DstHardwareAddr: edurouter.EmptyHardwareAddr,
				DstProtoAddr:    []byte{192, 168, 100, 50},
			},
			wantErr:       edurouter.ErrDropPdu,
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

			var actualArpResponse edurouter.ARPv4Pdu
			err = (&actualArpResponse).UnmarshalBinary(outFrame.Payload)

			assert.EqualValues(t, *v.wantArpResult, actualArpResponse)
		})
	}
}

func TestARPv4LinkLayerHandler_HandleARPResponse(t *testing.T) {
	handler := edurouter.NewARPv4LinkLayerHandler()

	ctrl := gomock.NewController(t)

	config, err := edurouter.NewInterfaceConfig("veth0", &net.IPNet{
		IP:   []byte{192, 168, 100, 1},
		Mask: net.CIDRMask(24, 32),
	})
	require.NoError(t, err)
	hwa := net.HardwareAddr([]byte{1, 1, 1, 2, 2, 2})
	config.HardwareAddr = &hwa
	mockArpWriter := mocks.NewMockARPWriter(ctrl)
	config.ArpTable = edurouter.NewARPv4Table(config, mockArpWriter)

	srcProtoAddr := []byte{192, 168, 100, 100}
	srcHardwareAddr := []byte{1, 1, 1, 3, 3, 3}
	inputArp := edurouter.ARPv4Pdu{
		HTYPE:           edurouter.HTYPEEthernet,
		PTYPE:           ethernet.EtherTypeARP,
		HLEN:            edurouter.HardwareAddrLen,
		PLEN:            net.IPv4len,
		Operation:       edurouter.ARPOperationResponse,
		SrcHardwareAddr: srcHardwareAddr,
		SrcProtoAddr:    srcProtoAddr,
		DstHardwareAddr: edurouter.EmptyHardwareAddr,
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
	require.EqualError(t, err, edurouter.HandledPdu.Error())
	require.Nil(t, outFrame)

	actualHardwareAddr, err := config.ArpTable.Resolve([]byte{192, 168, 100, 100})
	assert.NoError(t, err)
	assert.EqualValues(t, srcHardwareAddr, actualHardwareAddr)
}
