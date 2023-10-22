package main

import (
	"errors"
	"github.com/spf13/cobra"
)

var ErrTooFewArguments = errors.New("edurouter: too few arguments")

func rootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	rootCmd.AddCommand(pingCommand())
	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(interfaceCommands())
	rootCmd.AddCommand(routeCommands())

	return rootCmd
}
