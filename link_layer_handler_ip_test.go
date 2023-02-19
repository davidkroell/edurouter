package edurouter_test

import (
	"context"
	"github.com/davidkroell/edurouter"
	"github.com/davidkroell/edurouter/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/mdlayher/ethernet"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestIPv4LinkLayerHandler_HandleICMPRequest(t *testing.T) {
	// TODO rework tests

	ctrl := gomock.NewController(t)

	publishCh := make(chan *edurouter.InternetV4PacketIn)
	handler := edurouter.NewIPv4LinkLayerInputHandler(publishCh)
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

	inputIP := *edurouter.NewIPv4Pdu(pingSourceIp, routerIP, edurouter.IPProtocolICMPv4, icmpRequestBinary)

	ipBinary, err := inputIP.MarshalBinary()
	require.NoError(t, err)

	inFrame := ethernet.Frame{
		Destination: hwaDst,
		Source:      hwaSrc,
		EtherType:   ethernet.EtherTypeIPv4,
		Payload:     ipBinary,
	}

	handler.SupplierC() <- edurouter.FrameIn{
		Frame:     &inFrame,
		Interface: config,
	}

	//outFrame := <-publishCh
	//
	//
	//assert.EqualValues(t, hwaDst, outFrame.Source)
	//
	//var actualIPResult edurouter.IPv4Pdu
	//err = (&actualIPResult).UnmarshalBinary(outFrame.Payload)
	//
	//assert.EqualValues(t, *wantIPResult, actualIPResult)
}
