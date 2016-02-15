// +build !daemon

package main

import "github.com/sara-nl/docker-1.9.1/cli"

const daemonUsage = ""

var daemonCli cli.Handler

// TODO: remove once `-d` is retired
func handleGlobalDaemonFlag() {}

// notifySystem sends a message to the host when the server is ready to be used
func notifySystem() {
}
