// +build !experimental

package daemon

import (
	"os"

	"github.com/sara-nl/docker-1.9.1/pkg/idtools"
	"github.com/sara-nl/docker-1.9.1/pkg/system"
	"github.com/sara-nl/docker-1.9.1/runconfig"
)

func setupRemappedRoot(config *Config) ([]idtools.IDMap, []idtools.IDMap, error) {
	return nil, nil, nil
}

func setupDaemonRoot(config *Config, rootDir string, rootUID, rootGID int) error {
	config.Root = rootDir
	// Create the root directory if it doesn't exists
	if err := system.MkdirAll(config.Root, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (daemon *Daemon) verifyExperimentalContainerSettings(hostConfig *runconfig.HostConfig, config *runconfig.Config) ([]string, error) {
	return nil, nil
}
