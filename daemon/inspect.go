package daemon

import (
	"fmt"
	"time"

	"github.com/sara-nl/docker-1.9.1/api/types"
	"github.com/sara-nl/docker-1.9.1/api/types/versions/v1p20"
	"github.com/sara-nl/docker-1.9.1/daemon/network"
)

// ContainerInspect returns low-level information about a
// container. Returns an error if the container cannot be found, or if
// there is an error getting the data.
func (daemon *Daemon) ContainerInspect(name string, size bool) (*types.ContainerJSON, error) {
	container, err := daemon.Get(name)
	if err != nil {
		return nil, err
	}

	container.Lock()
	defer container.Unlock()

	base, err := daemon.getInspectData(container, size)
	if err != nil {
		return nil, err
	}

	mountPoints := addMountPoints(container)
	networkSettings := &types.NetworkSettings{
		NetworkSettingsBase: types.NetworkSettingsBase{
			Bridge:                 container.NetworkSettings.Bridge,
			SandboxID:              container.NetworkSettings.SandboxID,
			HairpinMode:            container.NetworkSettings.HairpinMode,
			LinkLocalIPv6Address:   container.NetworkSettings.LinkLocalIPv6Address,
			LinkLocalIPv6PrefixLen: container.NetworkSettings.LinkLocalIPv6PrefixLen,
			Ports:                  container.NetworkSettings.Ports,
			SandboxKey:             container.NetworkSettings.SandboxKey,
			SecondaryIPAddresses:   container.NetworkSettings.SecondaryIPAddresses,
			SecondaryIPv6Addresses: container.NetworkSettings.SecondaryIPv6Addresses,
		},
		DefaultNetworkSettings: daemon.getDefaultNetworkSettings(container.NetworkSettings.Networks),
		Networks:               container.NetworkSettings.Networks,
	}

	return &types.ContainerJSON{base, mountPoints, container.Config, networkSettings}, nil
}

// ContainerInspect120 serializes the master version of a container into a json type.
func (daemon *Daemon) ContainerInspect120(name string) (*v1p20.ContainerJSON, error) {
	container, err := daemon.Get(name)
	if err != nil {
		return nil, err
	}

	container.Lock()
	defer container.Unlock()

	base, err := daemon.getInspectData(container, false)
	if err != nil {
		return nil, err
	}

	mountPoints := addMountPoints(container)
	config := &v1p20.ContainerConfig{
		container.Config,
		container.Config.MacAddress,
		container.Config.NetworkDisabled,
		container.Config.ExposedPorts,
		container.hostConfig.VolumeDriver,
	}
	networkSettings := daemon.getBackwardsCompatibleNetworkSettings(container.NetworkSettings)

	return &v1p20.ContainerJSON{base, mountPoints, config, networkSettings}, nil
}

func (daemon *Daemon) getInspectData(container *Container, size bool) (*types.ContainerJSONBase, error) {
	// make a copy to play with
	hostConfig := *container.hostConfig

	if children, err := daemon.children(container.Name); err == nil {
		for linkAlias, child := range children {
			hostConfig.Links = append(hostConfig.Links, fmt.Sprintf("%s:%s", child.Name, linkAlias))
		}
	}
	// we need this trick to preserve empty log driver, so
	// container will use daemon defaults even if daemon change them
	if hostConfig.LogConfig.Type == "" {
		hostConfig.LogConfig.Type = daemon.defaultLogConfig.Type
	}

	if len(hostConfig.LogConfig.Config) == 0 {
		hostConfig.LogConfig.Config = daemon.defaultLogConfig.Config
	}

	containerState := &types.ContainerState{
		Status:     container.State.StateString(),
		Running:    container.State.Running,
		Paused:     container.State.Paused,
		Restarting: container.State.Restarting,
		OOMKilled:  container.State.OOMKilled,
		Dead:       container.State.Dead,
		Pid:        container.State.Pid,
		ExitCode:   container.State.ExitCode,
		Error:      container.State.Error,
		StartedAt:  container.State.StartedAt.Format(time.RFC3339Nano),
		FinishedAt: container.State.FinishedAt.Format(time.RFC3339Nano),
	}

	contJSONBase := &types.ContainerJSONBase{
		ID:           container.ID,
		Created:      container.Created.Format(time.RFC3339Nano),
		Path:         container.Path,
		Args:         container.Args,
		State:        containerState,
		Image:        container.ImageID,
		LogPath:      container.LogPath,
		Name:         container.Name,
		RestartCount: container.RestartCount,
		Driver:       container.Driver,
		ExecDriver:   container.ExecDriver,
		MountLabel:   container.MountLabel,
		ProcessLabel: container.ProcessLabel,
		ExecIDs:      container.getExecIDs(),
		HostConfig:   &hostConfig,
	}

	var (
		sizeRw     int64
		sizeRootFs int64
	)
	if size {
		sizeRw, sizeRootFs = container.getSize()
		contJSONBase.SizeRw = &sizeRw
		contJSONBase.SizeRootFs = &sizeRootFs
	}

	// Now set any platform-specific fields
	contJSONBase = setPlatformSpecificContainerFields(container, contJSONBase)

	contJSONBase.GraphDriver.Name = container.Driver
	graphDriverData, err := daemon.driver.GetMetadata(container.ID)
	if err != nil {
		return nil, err
	}
	contJSONBase.GraphDriver.Data = graphDriverData

	return contJSONBase, nil
}

// ContainerExecInspect returns low-level information about the exec
// command. An error is returned if the exec cannot be found.
func (daemon *Daemon) ContainerExecInspect(id string) (*ExecConfig, error) {
	eConfig, err := daemon.getExecConfig(id)
	if err != nil {
		return nil, err
	}
	return eConfig, nil
}

// VolumeInspect looks up a volume by name. An error is returned if
// the volume cannot be found.
func (daemon *Daemon) VolumeInspect(name string) (*types.Volume, error) {
	v, err := daemon.volumes.Get(name)
	if err != nil {
		return nil, err
	}
	return volumeToAPIType(v), nil
}

func (daemon *Daemon) getBackwardsCompatibleNetworkSettings(settings *network.Settings) *v1p20.NetworkSettings {
	result := &v1p20.NetworkSettings{
		NetworkSettingsBase: types.NetworkSettingsBase{
			Bridge:                 settings.Bridge,
			SandboxID:              settings.SandboxID,
			HairpinMode:            settings.HairpinMode,
			LinkLocalIPv6Address:   settings.LinkLocalIPv6Address,
			LinkLocalIPv6PrefixLen: settings.LinkLocalIPv6PrefixLen,
			Ports:                  settings.Ports,
			SandboxKey:             settings.SandboxKey,
			SecondaryIPAddresses:   settings.SecondaryIPAddresses,
			SecondaryIPv6Addresses: settings.SecondaryIPv6Addresses,
		},
		DefaultNetworkSettings: daemon.getDefaultNetworkSettings(settings.Networks),
	}

	return result
}

// getDefaultNetworkSettings creates the deprecated structure that holds the information
// about the bridge network for a container.
func (daemon *Daemon) getDefaultNetworkSettings(networks map[string]*network.EndpointSettings) types.DefaultNetworkSettings {
	var settings types.DefaultNetworkSettings

	if defaultNetwork, ok := networks["bridge"]; ok {
		settings.EndpointID = defaultNetwork.EndpointID
		settings.Gateway = defaultNetwork.Gateway
		settings.GlobalIPv6Address = defaultNetwork.GlobalIPv6Address
		settings.GlobalIPv6PrefixLen = defaultNetwork.GlobalIPv6PrefixLen
		settings.IPAddress = defaultNetwork.IPAddress
		settings.IPPrefixLen = defaultNetwork.IPPrefixLen
		settings.IPv6Gateway = defaultNetwork.IPv6Gateway
		settings.MacAddress = defaultNetwork.MacAddress
	}
	return settings
}
