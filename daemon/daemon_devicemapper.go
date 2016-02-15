// +build !exclude_graphdriver_devicemapper,linux

package daemon

import (
	// register the devmapper graphdriver
	_ "github.com/sara-nl/docker-1.9.1/daemon/graphdriver/devmapper"
)
