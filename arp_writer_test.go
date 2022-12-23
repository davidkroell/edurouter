package edurouter_test

import (
	"github.com/davidkroell/edurouter"
	"github.com/davidkroell/edurouter/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestARPv4Writer(t *testing.T) {
	config, err := edurouter.NewInterfaceConfig("veth0", &net.IPNet{
		IP:   []byte{192, 168, 100, 1},
		Mask: net.CIDRMask(24, 32),
	})

	hwAddr := net.HardwareAddr([]byte{10, 10, 10, 20, 20, 20})

	config.HardwareAddr = &hwAddr
	require.NoError(t, err)

	arpWriter := edurouter.NewARPv4Writer(config)

	t.Run("ErrorNoPacketConn", func(t *testing.T) {
		err := arpWriter.SendArpRequest([]byte{0})
		assert.EqualError(t, err, edurouter.ErrARPPacketConn.Error())
	})

	t.Run("OK", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockPacketConn := mocks.NewMockPacketConn(ctrl)

		arpWriter.Initialize(mockPacketConn)

		ipToResolve := []byte{192, 168, 100, 2}

		mockPacketConn.EXPECT().WriteTo(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			DoAndReturn(func(p []byte, addr net.Addr) (int, error) {
				assert.EqualValues(t, &raw.Addr{HardwareAddr: ethernet.Broadcast}, addr)
				var frame ethernet.Frame

				err := (&frame).UnmarshalBinary(p)
				require.NoError(t, err)

				assert.EqualValues(t, ethernet.Broadcast, frame.Destination)
				assert.EqualValues(t, hwAddr, frame.Source)
				assert.Nil(t, frame.ServiceVLAN)
				assert.Nil(t, frame.VLAN)
				assert.EqualValues(t, ethernet.EtherTypeARP, frame.EtherType)

				var arpReq edurouter.ARPv4Pdu
				err = (&arpReq).UnmarshalBinary(frame.Payload)
				require.NoError(t, err)

				assert.EqualValues(t, edurouter.HTYPEEthernet, arpReq.HTYPE)
				assert.EqualValues(t, ethernet.EtherTypeIPv4, arpReq.PTYPE)
				assert.EqualValues(t, edurouter.HardwareAddrLen, arpReq.HLEN)
				assert.EqualValues(t, net.IPv4len, arpReq.PLEN)
				assert.EqualValues(t, edurouter.ARPOperationRequest, arpReq.Operation)
				assert.EqualValues(t, hwAddr, arpReq.SrcHardwareAddr)
				assert.EqualValues(t, config.Addr.IP, arpReq.SrcProtoAddr)
				assert.EqualValues(t, edurouter.EmptyHardwareAddr, arpReq.DstHardwareAddr)
				assert.EqualValues(t, ipToResolve, arpReq.DstProtoAddr)

				return 0, nil
			})

		err := arpWriter.SendArpRequest(ipToResolve)

		assert.Nil(t, err)
	})
}
