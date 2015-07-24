package libnetwork

import (
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/drivers/bridge"
	"github.com/docker/libnetwork/drivers/host"
	"github.com/docker/libnetwork/drivers/null"
	"github.com/docker/libnetwork/drivers/remote"
	"github.com/docker/libnetwork/drivers/routed"
)

type driverTable map[string]driverapi.Driver

func initDrivers(dc driverapi.DriverCallback) error {
	for _, fn := range [](func(driverapi.DriverCallback) error){
		bridge.Init,
		host.Init,
		null.Init,
		remote.Init,
		routed.Init,
	} {
		if err := fn(dc); err != nil {
			return err
		}
	}
	return nil
}
