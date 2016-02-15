// +build !windows

package daemon

import (
	"os"
	"os/signal"
	"syscall"

	psignal "github.com/sara-nl/docker-1.9.1/pkg/signal"
)

func setupDumpStackTrap() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	go func() {
		for range c {
			psignal.DumpStacks()
		}
	}()
}
