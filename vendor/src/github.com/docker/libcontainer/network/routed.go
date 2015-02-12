// +build linux

package network

import (
	"fmt"
)

// Strategy that creates a veth pair adding one to the container namespace and assigning a static ip to it. Default
// routes via the veth pair are added on the host and container, thus routing data in and out the container without
// relying on NAT.
type Routed struct {
}

func (v *Routed) Create(n *Network, nspid int, networkState *NetworkState) error {
	var (
		prefix     = n.VethPrefix
		txQueueLen = n.TxQueueLen
	)

	if prefix == "" {
		return fmt.Errorf("veth prefix is not specified")
	}
	name1, name2, err := createVethPair(prefix, txQueueLen)
	if err != nil {
		return err
	}

	if err := SetMtu(name1, n.Mtu); err != nil {
		return err
	}
	if err := InterfaceUp(name1); err != nil {
		return err
	}
	if err := SetInterfaceInNamespacePid(name2, nspid); err != nil {
		return err
	}
	networkState.VethHost = name1
	networkState.VethChild = name2

	AddRoute(n.Address, "", "", networkState.VethHost)
	return nil
}

func (v *Routed) Initialize(config *Network, networkState *NetworkState) error {

	var vethChild = networkState.VethChild
	if vethChild == "" {
		return fmt.Errorf("vethChild is not specified")
	}
	if err := InterfaceDown(vethChild); err != nil {
		return fmt.Errorf("interface down %s %s", vethChild, err)
	}
	if err := ChangeInterfaceName(vethChild, defaultDevice); err != nil {
		return fmt.Errorf("change %s to %s %s", vethChild, defaultDevice, err)
	}
	if config.MacAddress != "" {
		if err := SetInterfaceMac(defaultDevice, config.MacAddress); err != nil {
			return fmt.Errorf("set %s mac %s", defaultDevice, err)
		}
	}
	if err := SetInterfaceIp(defaultDevice, config.Address); err != nil {
		return fmt.Errorf("set %s ip %s", defaultDevice, err)
	}

	if err := SetMtu(defaultDevice, config.Mtu); err != nil {
		return fmt.Errorf("set %s mtu to %d %s", defaultDevice, config.Mtu, err)
	}
	if err := InterfaceUp(defaultDevice); err != nil {
		return fmt.Errorf("%s up %s", defaultDevice, err)
	}

	if err := AddDefaultRoute(defaultDevice); err != nil {
		return fmt.Errorf("can't add route using device %s. %s", defaultDevice, err)
	}

	return nil
}
