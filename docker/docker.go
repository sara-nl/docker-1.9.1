package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/sara-nl/docker-1.9.1/api/client"
	"github.com/sara-nl/docker-1.9.1/autogen/dockerversion"
	"github.com/sara-nl/docker-1.9.1/cli"
	flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"
	"github.com/sara-nl/docker-1.9.1/pkg/reexec"
	"github.com/sara-nl/docker-1.9.1/pkg/term"
	"github.com/sara-nl/docker-1.9.1/utils"
)

func main() {
	if reexec.Init() {
		return
	}

	// Set terminal emulation based on platform as required.
	stdin, stdout, stderr := term.StdStreams()

	logrus.SetOutput(stderr)

	flag.Merge(flag.CommandLine, clientFlags.FlagSet, commonFlags.FlagSet)

	flag.Usage = func() {
		fmt.Fprint(os.Stdout, "Usage: docker [OPTIONS] COMMAND [arg...]\n"+daemonUsage+"       docker [ --help | -v | --version ]\n\n")
		fmt.Fprint(os.Stdout, "A self-sufficient runtime for containers.\n\nOptions:\n")

		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()

		help := "\nCommands:\n"

		for _, cmd := range dockerCommands {
			help += fmt.Sprintf("    %-10.10s%s\n", cmd.Name, cmd.Description)
		}

		help += "\nRun 'docker COMMAND --help' for more information on a command."
		fmt.Fprintf(os.Stdout, "%s\n", help)
	}

	flag.Parse()

	if *flVersion {
		showVersion()
		return
	}

	if *flHelp {
		// if global flag --help is present, regardless of what other options and commands there are,
		// just print the usage.
		flag.Usage()
		return
	}

	// TODO: remove once `-d` is retired
	handleGlobalDaemonFlag()
	clientCli := client.NewDockerCli(stdin, stdout, stderr, clientFlags)

	c := cli.New(clientCli, daemonCli)
	if err := c.Run(flag.Args()...); err != nil {
		if sterr, ok := err.(cli.StatusError); ok {
			if sterr.Status != "" {
				fmt.Fprintln(os.Stderr, sterr.Status)
				os.Exit(1)
			}
			os.Exit(sterr.StatusCode)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func showVersion() {
	if utils.ExperimentalBuild() {
		fmt.Printf("Docker version %s, build %s, experimental\n", dockerversion.VERSION, dockerversion.GITCOMMIT)
	} else {
		fmt.Printf("Docker version %s, build %s\n", dockerversion.VERSION, dockerversion.GITCOMMIT)
	}
}
