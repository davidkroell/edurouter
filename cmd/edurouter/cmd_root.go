package main

import (
	"context"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/davidkroell/edurouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
)

//func RootCommand() *cobra.Command {
//	var interfacesString []string
//
//	rootCmd := &cobra.Command{
//		Use:   "edurouter",
//		Short: "An education router software",
//		Long: `edurouter is a command-line interface (CLI) program for Linux systems
//that implements the functionality of a network router.
//It provides users with a hands-on learning experience,
//allowing them to explore and understand the inner workings of a router.
//The router supports following protocols on the IPv4 network stack:
//
//  * ARP
//  * ICMP
//  * IP routing
//
//It can be configured via the CLI, and supports a wide range of Linux distributions.
//Requires root privileges to run.
//
//Version: ` + edurouter.Version(),
//		// Uncomment the following line if your bare application
//		// has an action associated with it:
//		RunE: func(cmd *cobra.Command, args []string) error {
//			interfaceConfigs := make([]*edurouter.InterfaceConfig, len(interfacesString))
//
//			for i := range interfacesString {
//				config, err := edurouter.ParseInterfaceConfig(interfacesString[i])
//				if err != nil {
//					return err
//				}
//				interfaceConfigs[i] = config
//			}
//
//			log.Logger = log.Output(zerolog.NewConsoleWriter())
//
//			listener := edurouter.NewLinkLayerListener(interfaceConfigs...)
//
//			ctx, cancel := context.WithCancel(context.Background())
//
//			go func(ctx context.Context) {
//				listener.ListenAndServe(ctx)
//				log.Info().Msg("edurouter closed")
//				cancel()
//			}(cmd.Context())
//
//			// inputCommandLoop(listener)
//
//			// TODO use go-prompt to interact with the listener
//			<-ctx.Done()
//
//			return nil
//		},
//	}
//
//	rootCmd.Flags().StringSliceVarP(&interfacesString, "interface", "i", nil, "The interface definition(s) in the following format: '"+edurouter.InterfaceConfigFormatString+"'")
//	err := rootCmd.MarkFlagRequired("interface")
//	if err != nil {
//		panic(err)
//	}
//	return rootCmd
//}

var (
	listener *edurouter.LinkLayerListener
)

func executor(in string) {
	switch in {
	case "version":
		fmt.Println(edurouter.Version())
	}

	if strings.HasPrefix(in, "ping") {
		ip := net.ParseIP(strings.TrimSpace(in[4:])).To4()

		listener.IcmpPing(ip, 4)
	}

	if strings.HasPrefix(in, "route list") {
		table := listener.RouteTable()
		w := tabwriter.NewWriter(os.Stdout, 1, 2, 4, ' ', 0)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "TYPE", "DST NET", "NEXT HOP", "OUT INTERFACE")
		for _, route := range table.GetRoutes() {

			nextHop := "-"
			if route.NextHop != nil {
				nextHop = route.NextHop.String()
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", route.RouteType, route.DstNet.IP, nextHop, route.OutInterface.InterfaceName)
		}

		w.Flush()
	}

	if strings.HasPrefix(in, "if add") {
		config, err := edurouter.ParseInterfaceConfig(in[6:])

		if err != nil {
			fmt.Printf("error adding interface: %v\n", err)
			return
		}

		listener.AddInterface(config)
	}

	if strings.HasPrefix(in, "if list") {
		w := tabwriter.NewWriter(os.Stdout, 1, 2, 4, ' ', 0)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "INTERFACE", "HW ADDR", "IP (EMULATED)", "IP (REAL)")
		for _, iface := range listener.Interfaces() {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", iface.InterfaceName, iface.HardwareAddr, iface.Addr, iface.RealIPAddr)
		}
		w.Flush()
	}
}

func completer(doc prompt.Document) []prompt.Suggest {
	var s []prompt.Suggest

	text := doc.TextBeforeCursor()

	// top-level prompt
	s = []prompt.Suggest{
		{Text: "version", Description: "show version"},
		{Text: "help", Description: "show help"},
		{Text: "exit", Description: "exit edurouter"},
		{Text: "ping", Description: "ping a host"},

		{Text: "route", Description: "show or configure the IP routes"},
		{Text: "if", Description: "show  or configure the interfaces"},
		{Text: "log", Description: "show or configure the log level"},
	}

	// top-level commands
	if strings.HasPrefix(text, "version") ||
		strings.HasPrefix(text, "help") ||
		strings.HasPrefix(text, "exit") ||
		strings.HasPrefix(text, "ping") {
		s = []prompt.Suggest{}
	}

	if strings.HasPrefix(text, "route") {
		s = []prompt.Suggest{
			{Text: "list", Description: "list all routes"},
			{Text: "add", Description: "add a route"},
		}
	}

	if strings.HasPrefix(text, "log") {
		s = []prompt.Suggest{
			{Text: "none", Description: "disable logging"},
			{Text: "debug", Description: "set loglevel to debug"},
			{Text: "info", Description: "set loglevel to info"},
			{Text: "error", Description: "set loglevel to error"},
		}
	}

	if strings.HasPrefix(text, "if") {
		s = []prompt.Suggest{
			{Text: "list", Description: "list all interfaces"},
			{Text: "add", Description: "add an interface"},
		}

		if strings.HasPrefix(text, "if add") {
			ifaces, err := net.Interfaces()
			if err == nil {
				s = []prompt.Suggest{}
				for _, i := range ifaces {
					s = append(s, prompt.Suggest{Text: i.Name})
				}
			}
		}
	}

	return prompt.FilterHasPrefix(s, doc.GetWordBeforeCursor(), true)
}

func exitChecker(in string, breakline bool) bool {
	return in == "exit" && breakline
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		log.Info().Msg("edurouter close requested")
		cancel()
	}()

	log.Logger = log.Output(zerolog.NewConsoleWriter())
	log.Logger = log.Level(zerolog.Disabled)

	l := edurouter.NewLinkLayerListener()

	go func() {
		l.ListenAndServe(ctx)
		log.Info().Msg("edurouter closed")
		cancel()
	}()

	listener = l

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
		prompt.OptionSetExitCheckerOnInput(exitChecker),
	)
	p.Run()
}
