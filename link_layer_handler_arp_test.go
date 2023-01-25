package edurouter_test

import (
	"context"
	"github.com/davidkroell/edurouter"
	"github.com/davidkroell/edurouter/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/mdlayher/ethernet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestARPv4LinkLayerHandler_HandleARPRequests(t *testing.T) {
	publishCh := make(chan *ethernet.Frame)
	handler := edurouter.NewARPv4LinkLayerHandler(publishCh)
	ctx, cancel := context.WithCancel(context.Background())
	handler.RunHandler(ctx)
	defer cancel()

	config, err := edurouter.NewInterfaceConfig("veth0", &net.IPNet{
		IP:   []byte{192, 168, 100, 1},
		Mask: net.CIDRMask(24, 32),
	})
	require.NoError(t, err)
	hwa := net.HardwareAddr([]byte{1, 1, 1, 2, 2, 2})
	config.HardwareAddr = &hwa

	tests := map[string]struct {
		inputArp      edurouter.ARPv4Pdu
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

			handler.SupplierC() <- edurouter.FrameFromInterface{
				Frame:       &inFrame,
				InInterface: config,
			}

			if v.wantArpResult == nil {
				return
			}
			outFrame := <-publishCh

			assert.EqualValues(t, hwa, outFrame.Source)

			var actualArpResponse edurouter.ARPv4Pdu
			err = (&actualArpResponse).UnmarshalBinary(outFrame.Payload)

			assert.EqualValues(t, *v.wantArpResult, actualArpResponse)
		})
	}
}

func TestARPv4LinkLayerHandler_HandleARPResponse(t *testing.T) {
	ch := make(chan *ethernet.Frame)
	handler := edurouter.NewARPv4LinkLayerHandler(ch)
	ctx, cancel := context.WithCancel(context.Background())
	handler.RunHandler(ctx)
	defer cancel()

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

	handler.SupplierC() <- edurouter.FrameFromInterface{
		Frame:       &inFrame,
		InInterface: config,
	}

	select {
	case <-ch:
		t.Fail()
	case <-time.After(time.Second):
		// no answer expected
	}
}
