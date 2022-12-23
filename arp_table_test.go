package edurouter_test

import (
	"errors"
	"github.com/davidkroell/edurouter"
	"github.com/davidkroell/edurouter/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestARPv4Table_Store(t *testing.T) {
	config, err := edurouter.NewInterfaceConfig("veth0", &net.IPNet{
		IP:   []byte{192, 168, 100, 1},
		Mask: net.CIDRMask(24, 32),
	})
	require.NoError(t, err)

	t.Run("ErrorWrongIPorMAC", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockArpWriter := mocks.NewMockARPWriter(ctrl)

		arpTable := edurouter.NewARPv4Table(config, mockArpWriter)

		err := arpTable.Store([]byte{0}, []byte{0})
		assert.EqualError(t, err, edurouter.ErrNotAnMACHardwareAddress.Error())
		err = arpTable.Store([]byte{0}, []byte{0, 1, 2, 3, 4, 5, 6})
		assert.EqualError(t, err, edurouter.ErrNotAnMACHardwareAddress.Error())

		err = arpTable.Store([]byte{0}, []byte{0, 1, 2, 3, 4, 5})
		assert.EqualError(t, err, edurouter.ErrNotAnIPv4Address.Error())
		err = arpTable.Store([]byte{1, 2, 3, 4, 5}, []byte{0, 1, 2, 3, 4, 5})
		assert.EqualError(t, err, edurouter.ErrNotAnIPv4Address.Error())

		_, err = arpTable.Resolve([]byte{0})
		assert.EqualError(t, err, edurouter.ErrNotAnIPv4Address.Error())

		_, err = arpTable.Resolve([]byte{1, 2, 3, 4, 5})
		assert.EqualError(t, err, edurouter.ErrNotAnIPv4Address.Error())
	})

	t.Run("ErrorInARPWriter", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockArpWriter := mocks.NewMockARPWriter(ctrl)

		arpTable := edurouter.NewARPv4Table(config, mockArpWriter)

		expectedIp := []byte{192, 168, 0, 100}

		testErr := errors.New("test error")

		mockArpWriter.EXPECT().SendArpRequest(expectedIp).DoAndReturn(func(ip net.IP) error {
			// noop
			return testErr
		})

		actualMac, err := arpTable.Resolve(expectedIp)
		assert.EqualError(t, err, testErr.Error())
		assert.Nil(t, actualMac)
	})

	t.Run("ErrorARPTimeout", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockArpWriter := mocks.NewMockARPWriter(ctrl)

		arpTable := edurouter.NewARPv4Table(config, mockArpWriter)

		expectedIp := []byte{192, 168, 0, 100}

		mockArpWriter.EXPECT().SendArpRequest(expectedIp).DoAndReturn(func(ip net.IP) error {
			// noop
			return nil
		}).Times(10)

		actualMac, err := arpTable.Resolve(expectedIp)
		assert.EqualError(t, err, edurouter.ErrARPTimeout.Error())
		assert.Nil(t, actualMac)
	})

	t.Run("OKWithARPRequest", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockArpWriter := mocks.NewMockARPWriter(ctrl)

		arpTable := edurouter.NewARPv4Table(config, mockArpWriter)

		expectedIp := []byte{192, 168, 0, 100}
		expectedMac := []byte{0, 1, 2, 3, 4, 5}

		mockArpWriter.EXPECT().SendArpRequest(expectedIp).DoAndReturn(func(ip net.IP) error {
			err := arpTable.Store(ip, expectedMac)
			require.NoError(t, err)
			return nil
		})

		actualMac, err := arpTable.Resolve(expectedIp)
		assert.NoError(t, err)
		assert.EqualValues(t, expectedMac, actualMac)
	})

	t.Run("OKWithoutARPRequest", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockArpWriter := mocks.NewMockARPWriter(ctrl)

		arpTable := edurouter.NewARPv4Table(config, mockArpWriter)

		expectedIp := []byte{192, 168, 0, 100}
		expectedMac := []byte{0, 1, 2, 3, 4, 5}

		err := arpTable.Store(expectedIp, expectedMac)
		require.NoError(t, err)

		actualMac, err := arpTable.Resolve(expectedIp)
		assert.NoError(t, err)
		assert.EqualValues(t, expectedMac, actualMac)
	})
}
