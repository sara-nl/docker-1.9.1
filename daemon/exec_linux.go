// +build linux

package daemon

import (
	"strings"

	"github.com/sara-nl/docker-1.9.1/daemon/execdriver/lxc"
)

// checkExecSupport returns an error if the exec driver does not support exec,
// or nil if it is supported.
func checkExecSupport(drivername string) error {
	if strings.HasPrefix(drivername, lxc.DriverName) {
		return lxc.ErrExec
	}
	return nil
}
