package edurouter

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestRouteTable_AddRoute(t *testing.T) {
	t.Parallel()

	t.Run("DestinationIsNoValidNetwork", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 0, 100})

		ri1 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 20, 5}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err := rt.AddRoute(ri1)
		assert.EqualError(t, err, ErrNotANetworkAddress.Error())

		assert.Empty(t, rt.GetRoutes())
	})

	t.Run("AddSingleRoute", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 0, 100})

		ri1 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 20, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err := rt.AddRoute(ri1)
		assert.NoError(t, err)

		actualRoutes := rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri1}, actualRoutes)
	})

	t.Run("AddLinkLocalAndStaticRoutesEnsureSorting", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 0, 100})

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 50, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err := rt.AddRoute(ri1)
		assert.NoError(t, err)

		ri2 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 40, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri2)
		assert.NoError(t, err)

		actualRoutes := rt.GetRoutes()
		assert.EqualValues(t, []RouteInfo{ri2, ri1}, actualRoutes)
	})

	t.Run("AddTwoStaticRoutesEnsureSorting", func(t *testing.T) {
		rt := NewRouteTable()

		nextHop := net.IP([]byte{192, 168, 0, 100})

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(16, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err := rt.AddRoute(ri1)
		assert.NoError(t, err)

		ri2 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri2)
		assert.NoError(t, err)

		ri3 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 0, 0}),
				Mask: net.CIDRMask(12, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri3)
		assert.NoError(t, err)

		ri4 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 80, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
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

		nextHop := net.IP([]byte{192, 168, 0, 100})

		ri1 := RouteInfo{
			RouteType: LinkLocalRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 20, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err := rt.AddRoute(ri1)
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

		nextHop := net.IP([]byte{192, 168, 0, 100})

		ri1 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(16, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err := rt.AddRoute(ri1)
		assert.NoError(t, err)

		ri2 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{192, 168, 0, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri2)
		assert.NoError(t, err)

		ri3 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 0, 0}),
				Mask: net.CIDRMask(12, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
		}
		err = rt.AddRoute(ri3)
		assert.NoError(t, err)

		ri4 := RouteInfo{
			RouteType: StaticRouteType,
			DstNet: net.IPNet{
				IP:   net.IP([]byte{10, 0, 80, 0}),
				Mask: net.CIDRMask(24, 32),
			},
			OutInterface: nil,
			NextHop:      &nextHop,
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
