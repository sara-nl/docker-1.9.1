package daemon

import (
	"github.com/sara-nl/docker-1.9.1/api/types"
	derr "github.com/sara-nl/docker-1.9.1/errors"
)

// ContainerTop is not supported on Windows and returns an error.
func (daemon *Daemon) ContainerTop(name string, psArgs string) (*types.ContainerProcessList, error) {
	return nil, derr.ErrorCodeNoTop
}
