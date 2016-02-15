package cli

import flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"

// ClientFlags represents flags for the docker client.
type ClientFlags struct {
	FlagSet   *flag.FlagSet
	Common    *CommonFlags
	PostParse func()

	ConfigDir string
}
