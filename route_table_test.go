package edurouter

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestRouteTable_AddRoute(t *testing.T) {
	t.Run("DestinationIsNoValidNetwork", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 0, 100})

		ri1 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP{192, 168, 20, 5},
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err := rt.AddRoute(ri1)
		assert.EqualError(t, err, ErrNotANetworkAddress.Error())

		assert.Empty(t, rt.GetRoutes())
	})

	t.Run("ErrNextHopNotOnLinkLocalNetwork", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 100, 100})
		outIface, err := NewInterfaceConfig("veth0", &net.IPNet{
			IP:   net.IP{192, 168, 0, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP{192, 168, 20, 0},
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri1)
		assert.EqualError(t, err, ErrNextHopNotOnLinkLocalNetwork.Error())

		assert.Empty(t, rt.GetRoutes())
	})

	t.Run("ErrLinkLocalRouteShouldNotHaveNextHop", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 10, 100})
		outIface, err := NewInterfaceConfig("veth2", &net.IPNet{
			IP:   net.IP{192, 168, 10, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri1)
		require.EqualError(t, err, ErrLinkLocalRouteShouldNotHaveNextHop.Error())
	})

	t.Run("AddSingleRoute", func(t *testing.T) {
		rt := NewRouteTable()

		outIface, err := NewInterfaceConfig("veth0", &net.IPNet{
			IP:   net.IP{192, 168, 0, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 20, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface,
			NextHop:      nil,
		}
		err = rt.AddRoute(ri1)
		assert.NoError(t, err)

		actualRoutes := rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri1}, actualRoutes)
	})

	t.Run("AddLinkLocalAndStaticRoutesEnsureSorting", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop1 := net.IP{192, 168, 0, 100}
		outIface1, err := NewInterfaceConfig("veth0", &net.IPNet{
			IP:   net.IP{192, 168, 0, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 50, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface1,
			NextHop:      &nextHop1,
		}
		err = rt.AddRoute(ri1)
		assert.NoError(t, err)

		outIface2, err := NewInterfaceConfig("veth1", &net.IPNet{
			IP:   net.IP{192, 168, 2, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri2 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 40, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface2,
			NextHop:      nil,
		}
		err = rt.AddRoute(ri2)
		assert.NoError(t, err)

		actualRoutes := rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri2, ri1}, actualRoutes)
	})

	t.Run("AddTwoStaticRoutesEnsureSorting", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop1 := net.IP{192, 168, 10, 100}
		outIface1, err := NewInterfaceConfig("veth0", &net.IPNet{
			IP:   net.IP{192, 168, 10, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(16, 32),
			},
			OutInterface: outIface1,
			NextHop:      &nextHop1,
		}
		err = rt.AddRoute(ri1)
		assert.NoError(t, err)

		nextHop2 := net.IP{192, 168, 11, 100}
		outIface2, err := NewInterfaceConfig("veth1", &net.IPNet{
			IP:   net.IP{192, 168, 11, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri2 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface2,
			NextHop:      &nextHop2,
		}
		err = rt.AddRoute(ri2)
		assert.NoError(t, err)

		nextHop3 := net.IP{192, 168, 12, 100}
		outIface3, err := NewInterfaceConfig("veth2", &net.IPNet{
			IP:   net.IP{192, 168, 12, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri3 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 0, 0}),
				Mask: net.CIDRMask(12, 32),
			},
			OutInterface: outIface3,
			NextHop:      &nextHop3,
		}
		err = rt.AddRoute(ri3)
		assert.NoError(t, err)

		nextHop4 := net.IP{192, 168, 13, 100}
		outIface4, err := NewInterfaceConfig("veth3", &net.IPNet{
			IP:   net.IP{192, 168, 13, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri4 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 80, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface4,
			NextHop:      &nextHop4,
		}
		err = rt.AddRoute(ri4)
		assert.NoError(t, err)

		actualRoutes := rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri4, ri2, ri1, ri3}, actualRoutes)
	})
}

func TestRouteTable_GetRoutes(t *testing.T) {
	t.Run("DoesNotModifyOriginalRoutes", func(t *testing.T) {
		rt := NewRouteTable()

		outIface1, err := NewInterfaceConfig("veth0", &net.IPNet{
			IP:   net.IP{192, 168, 10, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 20, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface1,
			NextHop:      nil,
		}
		err = rt.AddRoute(ri1)
		assert.NoError(t, err)

		actualRoutes := rt.GetRoutes()

		actualRoutes[0].RouteType = StaticRouteType
		assert.NotEqualValues(t, rt.GetRoutes(), actualRoutes)
	})
}

func TestRouteTable_DeleteRouteAtIndex(t *testing.T) {
	t.Run("DeleteOutOfBounds", func(t *testing.T) {
		rt := NewRouteTable()
		rt.DeleteRouteAtIndex(0)
		rt.DeleteRouteAtIndex(1)
	})

	t.Run("DeletePreserversOrder", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop1 := net.IP([]byte{192, 168, 10, 100})
		outIface1, err := NewInterfaceConfig("veth0", &net.IPNet{
			IP:   net.IP{192, 168, 10, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(16, 32),
			},
			OutInterface: outIface1,
			NextHop:      &nextHop1,
		}
		err = rt.AddRoute(ri1)
		assert.NoError(t, err)

		nextHop2 := net.IP([]byte{192, 168, 11, 100})
		outIface2, err := NewInterfaceConfig("veth1", &net.IPNet{
			IP:   net.IP{192, 168, 11, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri2 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface2,
			NextHop:      &nextHop2,
		}
		err = rt.AddRoute(ri2)
		assert.NoError(t, err)

		nextHop3 := net.IP([]byte{192, 168, 12, 100})
		outIface3, err := NewInterfaceConfig("veth1", &net.IPNet{
			IP:   net.IP{192, 168, 12, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri3 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 0, 0}),
				Mask: net.CIDRMask(12, 32),
			},
			OutInterface: outIface3,
			NextHop:      &nextHop3,
		}
		err = rt.AddRoute(ri3)
		assert.NoError(t, err)

		nextHop4 := net.IP([]byte{192, 168, 13, 100})
		outIface4, err := NewInterfaceConfig("veth2", &net.IPNet{
			IP:   net.IP{192, 168, 13, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri4 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 80, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface4,
			NextHop:      &nextHop4,
		}
		err = rt.AddRoute(ri4)
		assert.NoError(t, err)

		rt.DeleteRouteAtIndex(2)

		actualRoutes := rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri4, ri2, ri3}, actualRoutes)

		rt.DeleteRouteAtIndex(0)
		actualRoutes = rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri2, ri3}, actualRoutes)

		rt.DeleteRouteAtIndex(1)
		actualRoutes = rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri2}, actualRoutes)

		rt.DeleteRouteAtIndex(0)
		actualRoutes = rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{}, actualRoutes)
	})
}

func TestRouteTable_RoutePacket(t *testing.T) {
	t.Run("NoRoutesDropPacket", func(t *testing.T) {
		rt := NewRouteTable()

		packet := NewIPv4Pdu([]byte{192, 168, 1, 10}, []byte{192, 168, 2, 20}, IPProtocolICMPv4, []byte{})

		packet, routeInfo, err := rt.RoutePacket(packet)
		assert.EqualError(t, err, ErrDropPdu.Error())
		assert.Nil(t, routeInfo)
		assert.Nil(t, packet)
	})

	t.Run("PacketTTLIsZeroDropPacket", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 10, 100})
		outIface, err := NewInterfaceConfig("veth2", &net.IPNet{
			IP:   net.IP{192, 168, 10, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri1)
		require.NoError(t, err)

		packet := NewIPv4Pdu([]byte{192, 168, 1, 10}, []byte{192, 168, 0, 20}, IPProtocolICMPv4, []byte{})
		packet.TTL = 1

		packet, routeInfo, err := rt.RoutePacket(packet)
		assert.EqualError(t, err, ErrDropPdu.Error())
		assert.Nil(t, routeInfo)
		assert.Nil(t, packet)
	})

	t.Run("OK", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 10, 100})
		outIface, err := NewInterfaceConfig("veth0", &net.IPNet{
			IP:   net.IP{192, 168, 10, 1},
			Mask: net.CIDRMask(24, 32),
		})
		require.NoError(t, err)

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: outIface,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri1)
		require.NoError(t, err)

		packet := NewIPv4Pdu([]byte{192, 168, 1, 10}, []byte{192, 168, 0, 20}, IPProtocolICMPv4, []byte{})

		packet, routeInfo, err := rt.RoutePacket(packet)
		assert.NoError(t, err)
		assert.EqualValues(t, ri1, *routeInfo)
		assert.EqualValues(t, 63, packet.TTL)
	})
}
