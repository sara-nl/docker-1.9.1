// +build daemon

package main

import (
	systemdDaemon "github.com/coreos/go-systemd/daemon"
	_ "github.com/sara-nl/docker-1.9.1/daemon/execdriver/lxc"
)

// notifySystem sends a message to the host when the server is ready to be used
func notifySystem() {
	// Tell the init daemon we are accepting requests
	go systemdDaemon.SdNotify("READY=1")
}
