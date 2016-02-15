package main

import (
	"path/filepath"

	"github.com/sara-nl/docker-1.9.1/cli"
	"github.com/sara-nl/docker-1.9.1/cliconfig"
	flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"
)

var clientFlags = &cli.ClientFlags{FlagSet: new(flag.FlagSet), Common: commonFlags}

func init() {
	client := clientFlags.FlagSet
	client.StringVar(&clientFlags.ConfigDir, []string{"-config"}, cliconfig.ConfigDir(), "Location of client config files")

	clientFlags.PostParse = func() {
		clientFlags.Common.PostParse()

		if clientFlags.ConfigDir != "" {
			cliconfig.SetConfigDir(clientFlags.ConfigDir)
		}

		if clientFlags.Common.TrustKey == "" {
			clientFlags.Common.TrustKey = filepath.Join(cliconfig.ConfigDir(), defaultTrustKeyFile)
		}
	}
}
