// +build !windows

package daemon

import (
	"os"
	"path/filepath"

	derr "github.com/sara-nl/docker-1.9.1/errors"
	"github.com/sara-nl/docker-1.9.1/image"
	"github.com/sara-nl/docker-1.9.1/pkg/stringid"
	"github.com/sara-nl/docker-1.9.1/runconfig"
	"github.com/sara-nl/docker-1.9.1/volume"
	"github.com/opencontainers/runc/libcontainer/label"
)

// createContainerPlatformSpecificSettings performs platform specific container create functionality
func createContainerPlatformSpecificSettings(container *Container, config *runconfig.Config, hostConfig *runconfig.HostConfig, img *image.Image) error {
	for spec := range config.Volumes {
		name := stringid.GenerateNonCryptoID()
		destination := filepath.Clean(spec)

		// Skip volumes for which we already have something mounted on that
		// destination because of a --volume-from.
		if container.isDestinationMounted(destination) {
			continue
		}
		path, err := container.GetResourcePath(destination)
		if err != nil {
			return err
		}

		stat, err := os.Stat(path)
		if err == nil && !stat.IsDir() {
			return derr.ErrorCodeMountOverFile.WithArgs(path)
		}

		volumeDriver := hostConfig.VolumeDriver
		if destination != "" && img != nil {
			if _, ok := img.ContainerConfig.Volumes[destination]; ok {
				// check for whether bind is not specified and then set to local
				if _, ok := container.MountPoints[destination]; !ok {
					volumeDriver = volume.DefaultDriverName
				}
			}
		}

		v, err := container.daemon.createVolume(name, volumeDriver, nil)
		if err != nil {
			return err
		}

		if err := label.Relabel(v.Path(), container.MountLabel, true); err != nil {
			return err
		}

		// never attempt to copy existing content in a container FS to a shared volume
		if v.DriverName() == volume.DefaultDriverName {
			if err := container.copyImagePathContent(v, destination); err != nil {
				return err
			}
		}

		container.addMountPointWithVolume(destination, v, true)
	}
	return nil
}
