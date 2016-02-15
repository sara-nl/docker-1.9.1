// +build !experimental

package daemon

import flag "github.com/sara-nl/docker-1.9.1/pkg/mflag"

func (config *Config) attachExperimentalFlags(cmd *flag.FlagSet, usageFn func(string) string) {
}
