package main

import (
	"fmt"
	"github.com/davidkroell/edurouter"
	"github.com/spf13/cobra"
	"net"
	"text/tabwriter"
)

func routeCommands() *cobra.Command {
	routeCmds := &cobra.Command{
		Use:   "route",
		Short: "show or configure the IP routes",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "list all routes",
		Run: func(cmd *cobra.Command, args []string) {
			table := listener.RouteTable()
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 1, 2, 4, ' ', 0)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "TYPE", "DST NET", "NEXT HOP", "OUT INTERFACE")
			for _, route := range table.GetRoutes() {

				nextHop := "-"
				if route.NextHop != nil {
					nextHop = route.NextHop.String()
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", route.RouteType, route.DstNet.IP, nextHop, route.OutInterface.InterfaceName)
			}

			w.Flush()
		},
	}

	var addr string
	var iface string
	var nextHop string

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "add a route",
		RunE: func(cmd *cobra.Command, args []string) error {
			ip, ipNet, err := net.ParseCIDR(addr)

			if err != nil {
				return err
			}

			ipNet.IP = ip

			nextHopIP := net.ParseIP(nextHop).To4()

			var outIface *edurouter.InterfaceConfig

			for _, i := range listener.Interfaces() {
				if i.InterfaceName == iface {
					outIface = i
				}
			}

			listener.RouteTable().AddRoute(edurouter.RouteInfo{
				RouteType:    edurouter.StaticRouteType,
				DstNet:       *ipNet,
				NextHop:      &nextHopIP,
				OutInterface: outIface,
			})

			return nil
		},
	}

	addCmd.Flags().StringVarP(&iface, "interface", "i", "", "interface")
	addCmd.Flags().StringVarP(&addr, "address", "a", "", "")
	addCmd.Flags().StringVar(&nextHop, "next-hop", "", "")

	routeCmds.AddCommand(listCmd, addCmd)
	return routeCmds
}
