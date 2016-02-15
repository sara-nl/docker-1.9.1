// +build exclude_graphdriver_aufs,linux freebsd

package daemon

import (
	"github.com/sara-nl/docker-1.9.1/daemon/graphdriver"
)

func migrateIfAufs(driver graphdriver.Driver, root string) error {
	return nil
}
