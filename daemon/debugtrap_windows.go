package daemon

import (
	"fmt"
	"os"
	"syscall"

	"github.com/Sirupsen/logrus"
	psignal "github.com/sara-nl/docker-1.9.1/pkg/signal"
	"github.com/sara-nl/docker-1.9.1/pkg/system"
)

func setupDumpStackTrap() {
	// Windows does not support signals like *nix systems. So instead of
	// trapping on SIGUSR1 to dump stacks, we wait on a Win32 event to be
	// signalled.
	go func() {
		sa := syscall.SecurityAttributes{
			Length: 0,
		}
		ev := "Global\\docker-daemon-" + fmt.Sprint(os.Getpid())
		if h, _ := system.CreateEvent(&sa, false, false, ev); h != 0 {
			logrus.Debugf("Stackdump - waiting signal at %s", ev)
			for {
				syscall.WaitForSingleObject(h, syscall.INFINITE)
				psignal.DumpStacks()
			}
		}
	}()
}
