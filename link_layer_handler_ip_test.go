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
)

func TestIPv4LinkLayerHandler_HandleICMPRequest(t *testing.T) {
	routeTable := edurouter.NewRouteTable()

	ctrl := gomock.NewController(t)

	mockInternetLayerStrategy := mocks.NewMockInternetLayerStrategy(ctrl)
	icmpHandler := &edurouter.IcmpHandler{}

	internetLayerHandler := edurouter.NewInternetLayerHandler(mockInternetLayerStrategy, routeTable)

	publishCh := make(chan *ethernet.Frame)
	handler := edurouter.NewIPv4LinkLayerHandler(publishCh, internetLayerHandler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler.RunHandler(ctx)

	routerIP := []byte{192, 168, 100, 1}
	pingSourceIp := []byte{192, 168, 100, 50}

	config, err := edurouter.NewInterfaceConfig("veth0", &net.IPNet{
		IP:   routerIP,
		Mask: net.CIDRMask(24, 32),
	})
	require.NoError(t, err)

	err = routeTable.AddRoute(edurouter.RouteInfo{
		RouteType: edurouter.LinkLocalRouteType,
		DstNet: net.IPNet{
			IP:   []byte{192, 168, 100, 0},
			Mask: net.CIDRMask(24, 32),
		},
		OutInterface: config,
		NextHop:      nil,
	})

	require.NoError(t, err)
	hwaDst := net.HardwareAddr([]byte{1, 1, 1, 2, 2, 2})
	hwaSrc := net.HardwareAddr([]byte{1, 1, 1, 3, 3, 3})
	config.HardwareAddr = &hwaDst
	config.ArpTable = edurouter.NewARPv4Table(config, mocks.NewMockARPWriter(ctrl))
	config.RealIPAddr = &net.IPNet{
		IP:   []byte{192, 168, 0, 254},
		Mask: net.CIDRMask(24, 32),
	}

	icmpSamplePayload := []byte{0xde, 0xad, 0xbe, 0xef}
	icmpRequest := edurouter.ICMPPacket{
		IcmpType: edurouter.IcmpTypeEchoRequest,
		IcmpCode: 0,
		Checksum: 0,
		Id:       1,
		Seq:      2,
		Data:     icmpSamplePayload,
	}

	icmpRequestBinary, err := icmpRequest.MarshalBinary()
	require.NoError(t, err)

	expectedIcmpResponse := edurouter.ICMPPacket{
		IcmpType: edurouter.IcmpTypeEchoReply,
		IcmpCode: 0,
		Checksum: 0,
		Id:       1,
		Seq:      2,
		Data:     icmpSamplePayload,
	}

	icmpResponseBinary, err := expectedIcmpResponse.MarshalBinary()
	require.NoError(t, err)

	inputIP := *edurouter.NewIPv4Pdu(pingSourceIp, routerIP, edurouter.IPProtocolICMPv4, icmpRequestBinary)
	wantIPResult := edurouter.NewIPv4Pdu(routerIP, pingSourceIp, edurouter.IPProtocolICMPv4, icmpResponseBinary)
	wantIPResult.HeaderChecksum = 0x3159

	ipBinary, err := inputIP.MarshalBinary()
	require.NoError(t, err)

	inFrame := ethernet.Frame{
		Destination: hwaDst,
		Source:      hwaSrc,
		EtherType:   ethernet.EtherTypeIPv4,
		Payload:     ipBinary,
	}

	mockInternetLayerStrategy.EXPECT().GetHandler(inputIP.Protocol).Return(icmpHandler, nil)

	handler.SupplierC() <- edurouter.FrameFromInterface{
		Frame:       &inFrame,
		InInterface: config,
	}

	outFrame := <-publishCh

	assert.EqualValues(t, hwaDst, outFrame.Source)

	var actualIPResult edurouter.IPv4Pdu
	err = (&actualIPResult).UnmarshalBinary(outFrame.Payload)

	assert.EqualValues(t, *wantIPResult, actualIPResult)
}
