// +build daemon

package main

import (
	"os"

	apiserver "github.com/sara-nl/docker-1.9.1/api/server"
	"github.com/sara-nl/docker-1.9.1/daemon"
)

func setPlatformServerConfig(serverConfig *apiserver.Config, daemonCfg *daemon.Config) *apiserver.Config {
	return serverConfig
}

// currentUserIsOwner checks whether the current user is the owner of the given
// file.
func currentUserIsOwner(f string) bool {
	return false
}

// setDefaultUmask doesn't do anything on windows
func setDefaultUmask() error {
	return nil
}

func getDaemonConfDir() string {
	return os.Getenv("PROGRAMDATA") + `\docker\config`
}

// notifySystem sends a message to the host when the server is ready to be used
func notifySystem() {
}
