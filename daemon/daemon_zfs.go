// +build !exclude_graphdriver_zfs,linux !exclude_graphdriver_zfs,freebsd

package daemon

import (
	// register the zfs driver
	_ "github.com/sara-nl/docker-1.9.1/daemon/graphdriver/zfs"
)
