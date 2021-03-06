package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sara-nl/docker-1.9.1/api/types"
	Cli "github.com/sara-nl/docker-1.9.1/cli"
	flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"
	"github.com/sara-nl/docker-1.9.1/pkg/stringid"
	"github.com/sara-nl/docker-1.9.1/pkg/stringutils"
	"github.com/sara-nl/docker-1.9.1/pkg/units"
)

// CmdHistory shows the history of an image.
//
// Usage: docker history [OPTIONS] IMAGE
func (cli *DockerCli) CmdHistory(args ...string) error {
	cmd := Cli.Subcmd("history", []string{"IMAGE"}, Cli.DockerCommands["history"].Description, true)
	human := cmd.Bool([]string{"H", "-human"}, true, "Print sizes and dates in human readable format")
	quiet := cmd.Bool([]string{"q", "-quiet"}, false, "Only show numeric IDs")
	noTrunc := cmd.Bool([]string{"#notrunc", "-no-trunc"}, false, "Don't truncate output")
	cmd.Require(flag.Exact, 1)

	cmd.ParseFlags(args, true)

	serverResp, err := cli.call("GET", "/images/"+cmd.Arg(0)+"/history", nil, nil)
	if err != nil {
		return err
	}

	defer serverResp.body.Close()

	history := []types.ImageHistory{}
	if err := json.NewDecoder(serverResp.body).Decode(&history); err != nil {
		return err
	}

	w := tabwriter.NewWriter(cli.out, 20, 1, 3, ' ', 0)
	if !*quiet {
		fmt.Fprintln(w, "IMAGE\tCREATED\tCREATED BY\tSIZE\tCOMMENT")
	}

	for _, entry := range history {
		if *noTrunc {
			fmt.Fprintf(w, entry.ID)
		} else {
			fmt.Fprintf(w, stringid.TruncateID(entry.ID))
		}
		if !*quiet {
			if *human {
				fmt.Fprintf(w, "\t%s ago\t", units.HumanDuration(time.Now().UTC().Sub(time.Unix(entry.Created, 0))))
			} else {
				fmt.Fprintf(w, "\t%s\t", time.Unix(entry.Created, 0).Format(time.RFC3339))
			}

			if *noTrunc {
				fmt.Fprintf(w, "%s\t", strings.Replace(entry.CreatedBy, "\t", " ", -1))
			} else {
				fmt.Fprintf(w, "%s\t", stringutils.Truncate(strings.Replace(entry.CreatedBy, "\t", " ", -1), 45))
			}

			if *human {
				fmt.Fprintf(w, "%s\t", units.HumanSize(float64(entry.Size)))
			} else {
				fmt.Fprintf(w, "%d\t", entry.Size)
			}

			fmt.Fprintf(w, "%s", entry.Comment)
		}
		fmt.Fprintf(w, "\n")
	}
	w.Flush()
	return nil
}
