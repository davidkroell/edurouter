package main

import (
	"github.com/spf13/cobra"
	"net"
)

func pingCommand() *cobra.Command {
	var numPings uint16
	cmd := &cobra.Command{
		Use:   "ping host [-n <num pings>]",
		Short: "ping a host",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return ErrTooFewArguments
			}

			ip := net.ParseIP(args[0])
			listener.IcmpPing(ip, numPings)
			return nil
		},
	}

	cmd.Flags().Uint16VarP(&numPings, "number", "n", 4, "number of pings")
	return cmd
}
