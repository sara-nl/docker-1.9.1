package daemon

import (
	"github.com/sara-nl/docker-1.9.1/image"
	"github.com/sara-nl/docker-1.9.1/runconfig"
)

// createContainerPlatformSpecificSettings performs platform specific container create functionality
func createContainerPlatformSpecificSettings(container *Container, config *runconfig.Config, hostConfig *runconfig.HostConfig, img *image.Image) error {
	return nil
}
