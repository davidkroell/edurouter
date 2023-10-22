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
)

var (
	listener            *edurouter.LinkLayerListener
	availableInterfaces []prompt.Suggest
	motd                = `#####################################################################
###                  __                       __                  ###
###        ___  ____/ /_  ___________  __  __/ /____  _____       ###
###       / _ \/ __  / / / / ___/ __ \/ / / / __/ _ \/ ___/       ###
###      /  __/ /_/ / /_/ / /  / /_/ / /_/ / /_/  __/ /           ###
###      \___/\__, _/\__, _/_/   \____/\__, _/\__/\___/_/         ###
#####################################################################`
)

func initSuggestions() {
	ifaces, err := net.Interfaces()
	if err == nil {
		availableInterfaces = []prompt.Suggest{}
		for _, i := range ifaces {
			if i.Name == "lo" {
				// skip loopback
				continue
			}

			availableInterfaces = append(availableInterfaces, prompt.Suggest{Text: i.Name})
		}
	}
}

func executor(in string) {
	cmd := rootCommand()
	cmd.SetArgs(strings.Split(in, " "))
	cmd.Execute()
}

func completer(doc prompt.Document) []prompt.Suggest {
	var s []prompt.Suggest

	text := doc.TextBeforeCursor()
	splitted := strings.Split(strings.TrimSpace(text), " ")

	for i := 0; i < len(splitted); i++ {
		if splitted[i] == "" {
			splitted = append(splitted[:i], splitted[i+1:]...)
		}
	}

	var argToComplete string

	if len(splitted) > 1 && strings.HasPrefix(splitted[len(splitted)-1], "-") {
		argToComplete = splitted[len(splitted)-1]
	} else if len(splitted) > 2 && strings.HasPrefix(splitted[len(splitted)-2], "-") && doc.GetWordBeforeCursor() != "" {
		argToComplete = splitted[len(splitted)-2]
	}

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
		switch argToComplete {
		case "-i", "--interface":
			s = []prompt.Suggest{}

			for _, i := range listener.Interfaces() {
				s = append(s, prompt.Suggest{Text: i.InterfaceName})
			}

		case "-a":
			s = []prompt.Suggest{}

		default:
			s = []prompt.Suggest{
				{Text: "list", Description: "list all routes"},
				{Text: "add", Description: "add a route"},
			}

			if strings.HasPrefix(text, "route add") {
				s = []prompt.Suggest{
					{Text: "-i"},
					{Text: "--interface"},
					{Text: "-a"},
					{Text: "--address"},
					{Text: "--next-hop"},
				}
			}
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
		switch argToComplete {
		case "-i", "--interface":
			s = availableInterfaces
		case "-a":
			s = []prompt.Suggest{}

		default:
			s = []prompt.Suggest{
				{Text: "list", Description: "list all interfaces"},
				{Text: "add", Description: "add an interface"},
			}

			if strings.HasPrefix(text, "if add") {
				s = []prompt.Suggest{
					{Text: "-i"},
					{Text: "--interface"},
					{Text: "-a"},
				}
			}
		}
	}

	return prompt.FilterHasPrefix(s, doc.GetWordBeforeCursor(), true)
}

func exitChecker(in string, breakline bool) bool {
	return in == "exit" && breakline
}

// ExecutePrompt adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecutePrompt() {
	initSuggestions()
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
	fmt.Println(motd)

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
		prompt.OptionSetExitCheckerOnInput(exitChecker),
	)
	p.Run()
}
