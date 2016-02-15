// +build !exclude_graphdriver_overlay,linux

package daemon

import (
	// register the overlay graphdriver
	_ "github.com/sara-nl/docker-1.9.1/daemon/graphdriver/overlay"
)
