package daemon

import (
	derr "github.com/sara-nl/docker-1.9.1/errors"
)

// ContainerStop looks for the given container and terminates it,
// waiting the given number of seconds before forcefully killing the
// container. If a negative number of seconds is given, ContainerStop
// will wait for a graceful termination. An error is returned if the
// container is not found, is already stopped, or if there is a
// problem stopping the container.
func (daemon *Daemon) ContainerStop(name string, seconds int) error {
	container, err := daemon.Get(name)
	if err != nil {
		return err
	}
	if !container.IsRunning() {
		return derr.ErrorCodeStopped
	}
	if err := container.Stop(seconds); err != nil {
		return derr.ErrorCodeCantStop.WithArgs(name, err)
	}
	return nil
}
