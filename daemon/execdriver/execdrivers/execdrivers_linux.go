// +build linux

package execdrivers

import (
	"fmt"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/sara-nl/docker-1.9.1/daemon/execdriver"
	"github.com/sara-nl/docker-1.9.1/daemon/execdriver/lxc"
	"github.com/sara-nl/docker-1.9.1/daemon/execdriver/native"
	"github.com/sara-nl/docker-1.9.1/pkg/sysinfo"
)

// NewDriver returns a new execdriver.Driver from the given name configured with the provided options.
func NewDriver(name string, options []string, root, libPath, initPath string, sysInfo *sysinfo.SysInfo) (execdriver.Driver, error) {
	switch name {
	case "lxc":
		// we want to give the lxc driver the full docker root because it needs
		// to access and write config and template files in /var/lib/docker/containers/*
		// to be backwards compatible
		logrus.Warn("LXC built-in support is deprecated.")
		return lxc.NewDriver(root, libPath, initPath, sysInfo.AppArmor)
	case "native":
		return native.NewDriver(path.Join(root, "execdriver", "native"), initPath, options)
	}
	return nil, fmt.Errorf("unknown exec driver %s", name)
}
