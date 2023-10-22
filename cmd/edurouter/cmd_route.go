package main

import (
	"fmt"
	"github.com/spf13/cobra"
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

	routeCmds.AddCommand(listCmd)
	return routeCmds
}
