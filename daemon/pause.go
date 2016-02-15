package daemon

import (
	derr "github.com/sara-nl/docker-1.9.1/errors"
)

// ContainerPause pauses a container
func (daemon *Daemon) ContainerPause(name string) error {
	container, err := daemon.Get(name)
	if err != nil {
		return err
	}

	if err := container.pause(); err != nil {
		return derr.ErrorCodePauseError.WithArgs(name, err)
	}

	return nil
}
