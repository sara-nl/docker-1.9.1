package daemon

import (
	"github.com/sara-nl/docker-1.9.1/api/types"
	"github.com/opencontainers/runc/libcontainer"
)

// convertStatsToAPITypes converts the libcontainer.Stats to the api specific
// structs. This is done to preserve API compatibility and versioning.
func convertStatsToAPITypes(ls *libcontainer.Stats) *types.StatsJSON {
	// TODO Windows. Refactor accordingly to fill in stats.
	s := &types.StatsJSON{}
	return s
}
