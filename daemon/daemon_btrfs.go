// +build !exclude_graphdriver_btrfs,linux

package daemon

import (
	// register the btrfs graphdriver
	_ "github.com/sara-nl/docker-1.9.1/daemon/graphdriver/btrfs"
)
