package client

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/sara-nl/docker-1.9.1/api/types"
	Cli "github.com/sara-nl/docker-1.9.1/cli"
	"github.com/sara-nl/docker-1.9.1/opts"
	flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"
	"github.com/sara-nl/docker-1.9.1/pkg/parsers"
	"github.com/sara-nl/docker-1.9.1/registry"
	"github.com/sara-nl/docker-1.9.1/runconfig"
)

// CmdCommit creates a new image from a container's changes.
//
// Usage: docker commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]
func (cli *DockerCli) CmdCommit(args ...string) error {
	cmd := Cli.Subcmd("commit", []string{"CONTAINER [REPOSITORY[:TAG]]"}, Cli.DockerCommands["commit"].Description, true)
	flPause := cmd.Bool([]string{"p", "-pause"}, true, "Pause container during commit")
	flComment := cmd.String([]string{"m", "-message"}, "", "Commit message")
	flAuthor := cmd.String([]string{"a", "#author", "-author"}, "", "Author (e.g., \"John Hannibal Smith <hannibal@a-team.com>\")")
	flChanges := opts.NewListOpts(nil)
	cmd.Var(&flChanges, []string{"c", "-change"}, "Apply Dockerfile instruction to the created image")
	// FIXME: --run is deprecated, it will be replaced with inline Dockerfile commands.
	flConfig := cmd.String([]string{"#run", "#-run"}, "", "This option is deprecated and will be removed in a future version in favor of inline Dockerfile-compatible commands")
	cmd.Require(flag.Max, 2)
	cmd.Require(flag.Min, 1)

	cmd.ParseFlags(args, true)

	var (
		name            = cmd.Arg(0)
		repository, tag = parsers.ParseRepositoryTag(cmd.Arg(1))
	)

	//Check if the given image name can be resolved
	if repository != "" {
		if err := registry.ValidateRepositoryName(repository); err != nil {
			return err
		}
	}

	v := url.Values{}
	v.Set("container", name)
	v.Set("repo", repository)
	v.Set("tag", tag)
	v.Set("comment", *flComment)
	v.Set("author", *flAuthor)
	for _, change := range flChanges.GetAll() {
		v.Add("changes", change)
	}

	if *flPause != true {
		v.Set("pause", "0")
	}

	var (
		config   *runconfig.Config
		response types.ContainerCommitResponse
	)

	if *flConfig != "" {
		config = &runconfig.Config{}
		if err := json.Unmarshal([]byte(*flConfig), config); err != nil {
			return err
		}
	}
	serverResp, err := cli.call("POST", "/commit?"+v.Encode(), config, nil)
	if err != nil {
		return err
	}

	defer serverResp.body.Close()

	if err := json.NewDecoder(serverResp.body).Decode(&response); err != nil {
		return err
	}

	fmt.Fprintln(cli.out, response.ID)
	return nil
}
