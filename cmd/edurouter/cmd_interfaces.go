package main

import (
	"fmt"
	"github.com/davidkroell/edurouter"
	"github.com/spf13/cobra"
	"net"
	"text/tabwriter"
)

func interfaceCommands() *cobra.Command {
	ifaceCmds := &cobra.Command{
		Use:   "if",
		Short: "show  or configure the interfaces",
	}

	var addr string
	var iface string

	addCmd := &cobra.Command{
		Use:   "add --interface iface -a address",
		Short: "add an interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			ip, ipNet, err := net.ParseCIDR(addr)

			if err != nil {
				return err
			}

			ipNet.IP = ip

			config, err := edurouter.NewInterfaceConfig(iface, ipNet)
			if err != nil {
				return err
			}
			listener.AddInterface(config)
			return nil
		},
	}

	addCmd.Flags().StringVarP(&iface, "interface", "i", "", "")
	addCmd.Flags().StringVarP(&addr, "address", "a", "", "")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "list all interfaces",
		Run: func(cmd *cobra.Command, args []string) {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 1, 2, 4, ' ', 0)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "INTERFACE", "HW ADDR", "IP (EMULATED)", "IP (REAL)")
			for _, iface := range listener.Interfaces() {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", iface.InterfaceName, iface.HardwareAddr, iface.Addr, iface.RealIPAddr)
			}
			w.Flush()
		},
	}

	ifaceCmds.AddCommand(addCmd, listCmd)

	return ifaceCmds
}
