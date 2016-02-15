// +build windows

package windows

import (
	"fmt"

	"github.com/sara-nl/docker-1.9.1/daemon/execdriver"
)

// Pause implements the exec driver Driver interface.
func (d *Driver) Pause(c *execdriver.Command) error {
	return fmt.Errorf("Windows: Containers cannot be paused")
}

// Unpause implements the exec driver Driver interface.
func (d *Driver) Unpause(c *execdriver.Command) error {
	return fmt.Errorf("Windows: Containers cannot be paused")
}
