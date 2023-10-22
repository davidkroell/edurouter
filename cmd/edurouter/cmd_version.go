package main

import (
	"fmt"
	"github.com/davidkroell/edurouter"
	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), edurouter.Version())
		},
	}

	return cmd
}
