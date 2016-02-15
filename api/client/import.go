package client

import (
	"fmt"
	"io"
	"net/url"
	"os"

	Cli "github.com/sara-nl/docker-1.9.1/cli"
	"github.com/sara-nl/docker-1.9.1/opts"
	flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"
	"github.com/sara-nl/docker-1.9.1/pkg/parsers"
	"github.com/sara-nl/docker-1.9.1/pkg/urlutil"
	"github.com/sara-nl/docker-1.9.1/registry"
)

// CmdImport creates an empty filesystem image, imports the contents of the tarball into the image, and optionally tags the image.
//
// The URL argument is the address of a tarball (.tar, .tar.gz, .tgz, .bzip, .tar.xz, .txz) file or a path to local file relative to docker client. If the URL is '-', then the tar file is read from STDIN.
//
// Usage: docker import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]
func (cli *DockerCli) CmdImport(args ...string) error {
	cmd := Cli.Subcmd("import", []string{"file|URL|- [REPOSITORY[:TAG]]"}, Cli.DockerCommands["import"].Description, true)
	flChanges := opts.NewListOpts(nil)
	cmd.Var(&flChanges, []string{"c", "-change"}, "Apply Dockerfile instruction to the created image")
	message := cmd.String([]string{"m", "-message"}, "", "Set commit message for imported image")
	cmd.Require(flag.Min, 1)

	cmd.ParseFlags(args, true)

	var (
		v          = url.Values{}
		src        = cmd.Arg(0)
		repository = cmd.Arg(1)
	)

	v.Set("fromSrc", src)
	v.Set("repo", repository)
	v.Set("message", *message)
	for _, change := range flChanges.GetAll() {
		v.Add("changes", change)
	}
	if cmd.NArg() == 3 {
		fmt.Fprintf(cli.err, "[DEPRECATED] The format 'file|URL|- [REPOSITORY [TAG]]' has been deprecated. Please use file|URL|- [REPOSITORY[:TAG]]\n")
		v.Set("tag", cmd.Arg(2))
	}

	if repository != "" {
		//Check if the given image name can be resolved
		repo, _ := parsers.ParseRepositoryTag(repository)
		if err := registry.ValidateRepositoryName(repo); err != nil {
			return err
		}
	}

	var in io.Reader

	if src == "-" {
		in = cli.in
	} else if !urlutil.IsURL(src) {
		v.Set("fromSrc", "-")
		file, err := os.Open(src)
		if err != nil {
			return err
		}
		defer file.Close()
		in = file

	}

	sopts := &streamOpts{
		rawTerminal: true,
		in:          in,
		out:         cli.out,
	}

	_, err := cli.stream("POST", "/images/create?"+v.Encode(), sopts)
	return err
}
