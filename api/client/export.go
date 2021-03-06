package client

import (
	"errors"
	"os"

	Cli "github.com/sara-nl/docker-1.9.1/cli"
	flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"
)

// CmdExport exports a filesystem as a tar archive.
//
// The tar archive is streamed to STDOUT by default or written to a file.
//
// Usage: docker export [OPTIONS] CONTAINER
func (cli *DockerCli) CmdExport(args ...string) error {
	cmd := Cli.Subcmd("export", []string{"CONTAINER"}, Cli.DockerCommands["export"].Description, true)
	outfile := cmd.String([]string{"o", "-output"}, "", "Write to a file, instead of STDOUT")
	cmd.Require(flag.Exact, 1)

	cmd.ParseFlags(args, true)

	var (
		output = cli.out
		err    error
	)
	if *outfile != "" {
		output, err = os.Create(*outfile)
		if err != nil {
			return err
		}
	} else if cli.isTerminalOut {
		return errors.New("Cowardly refusing to save to a terminal. Use the -o flag or redirect.")
	}

	image := cmd.Arg(0)
	sopts := &streamOpts{
		rawTerminal: true,
		out:         output,
	}
	if _, err := cli.stream("GET", "/containers/"+image+"/export", sopts); err != nil {
		return err
	}

	return nil
}
