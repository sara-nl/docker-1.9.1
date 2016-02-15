// +build !windows

package windows

import (
	"fmt"

	"github.com/sara-nl/docker-1.9.1/daemon/execdriver"
)

// NewDriver returns a new execdriver.Driver
func NewDriver(root, initPath string) (execdriver.Driver, error) {
	return nil, fmt.Errorf("Windows driver not supported on non-Windows")
}
