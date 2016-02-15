// Package v1p20 provides specific API types for the API version 1, patch 20.
package v1p20

import (
	"github.com/sara-nl/docker-1.9.1/api/types"
	"github.com/sara-nl/docker-1.9.1/pkg/nat"
	"github.com/sara-nl/docker-1.9.1/runconfig"
)

// ContainerJSON is a backcompatibility struct for the API 1.20
type ContainerJSON struct {
	*types.ContainerJSONBase
	Mounts          []types.MountPoint
	Config          *ContainerConfig
	NetworkSettings *NetworkSettings
}

// ContainerConfig is a backcompatibility struct used in ContainerJSON for the API 1.20
type ContainerConfig struct {
	*runconfig.Config

	MacAddress      string
	NetworkDisabled bool
	ExposedPorts    map[nat.Port]struct{}

	// backward compatibility, they now live in HostConfig
	VolumeDriver string
}

// StatsJSON is a backcompatibility struct used in Stats for API prior to 1.21
type StatsJSON struct {
	types.Stats
	Network types.NetworkStats `json:"network,omitempty"`
}

// NetworkSettings is a backward compatible struct for APIs prior to 1.21
type NetworkSettings struct {
	types.NetworkSettingsBase
	types.DefaultNetworkSettings
}
