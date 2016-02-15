// +build windows

package windows

import (
	"fmt"

	"github.com/sara-nl/docker-1.9.1/daemon/execdriver"
)

// Stats implements the exec driver Driver interface.
func (d *Driver) Stats(id string) (*execdriver.ResourceStats, error) {
	return nil, fmt.Errorf("Windows: Stats not implemented")
}
