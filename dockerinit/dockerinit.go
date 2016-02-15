package main

import (
	_ "github.com/sara-nl/docker-1.9.1/daemon/execdriver/lxc"
	_ "github.com/sara-nl/docker-1.9.1/daemon/execdriver/native"
	"github.com/sara-nl/docker-1.9.1/pkg/reexec"
)

func main() {
	// Running in init mode
	reexec.Init()
}
