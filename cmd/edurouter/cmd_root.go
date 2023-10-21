package main

import (
	"github.com/davidkroell/edurouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"

	"github.com/spf13/cobra"
)

func RootCommand() *cobra.Command {
	var interfacesString []string

	rootCmd := &cobra.Command{
		Use:   "edurouter",
		Short: "An education router software",
		Long: `edurouter is a command-line interface (CLI) program for Linux systems
that implements the functionality of a network router.
It provides users with a hands-on learning experience,
allowing them to explore and understand the inner workings of a router.
The router supports following protocols on the IPv4 network stack:

  * ARP
  * ICMP
  * IP routing

It can be configured via the CLI, and supports a wide range of Linux distributions.
Requires root privileges to run.

Version: ` + edurouter.Version(),
		// Uncomment the following line if your bare application
		// has an action associated with it:
		RunE: func(cmd *cobra.Command, args []string) error {
			interfaceConfigs := make([]*edurouter.InterfaceConfig, len(interfacesString))

			for i := range interfacesString {
				config, err := edurouter.ParseInterfaceConfig(interfacesString[i])
				if err != nil {
					return err
				}
				interfaceConfigs[i] = config
			}

			log.Logger = log.Output(zerolog.NewConsoleWriter())

			listener := edurouter.NewLinkLayerListener(interfaceConfigs...)

			listener.ListenAndServe(cmd.Context())
			return nil
		},
	}

	rootCmd.Flags().StringSliceVarP(&interfacesString, "interface", "i", nil, "The interface definition(s) in the following format: '"+edurouter.InterfaceConfigFormatString+"'")
	err := rootCmd.MarkFlagRequired("interface")
	if err != nil {
		panic(err)
	}
	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCommand().Execute()
	if err != nil {
		os.Exit(1)
	}
}
