// +build linux

package libcontainer

import (
	"fmt"
	"net"
    "github.com/docker/libcontainer/netlink"
	"github.com/docker/libcontainer/utils"
)

// Strategy that creates a veth pair adding one to the container namespace and assigning a static ip to it. Default
// routes via the veth pair are added on the host and container, thus routing data in and out the container without
// relying on NAT.
type Routed struct {
}
//	create(*network, int) error
//	initialize(*network) error
func (v *Routed) create(n *network, nspid int) error {
	fmt.Printf("Create: %T\n", n.SecondaryAddresses)
	tmpName, err := utils.GenerateRandomName("veth", 7)
	if err != nil {
		return err
	}
	n.TempVethPeerName = tmpName
	defer func() {
		if err != nil {
			netlink.NetworkLinkDel(n.HostInterfaceName)
			netlink.NetworkLinkDel(n.TempVethPeerName)
		}
	}()

	if err := netlink.NetworkCreateVethPair(n.HostInterfaceName, n.TempVethPeerName, n.TxQueueLen); err != nil {
		return err
	}
	
	host, err := net.InterfaceByName(n.HostInterfaceName)
	if err != nil {
		return err
	}

	if err := netlink.NetworkSetMTU(host, n.Mtu); err != nil {
		return err
	}

	if err := netlink.NetworkLinkUp(host); err != nil {
		return err
	}
	
	child, err := net.InterfaceByName(n.TempVethPeerName)
	if err != nil {
		return err
	}
	
	AddRoute(n.Address, "", "", n.HostInterfaceName)
	
	return 	netlink.NetworkSetNsPid(child, nspid)

}

func (v *Routed) initialize(config *network) error {
	fmt.Printf("Initialize: %T\n", config.SecondaryAddresses)
	var vethChild = config.TempVethPeerName
	var defaultDevice = config.Name
	if vethChild == "" {
		return fmt.Errorf("veth peer is not specified")
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

func AddDefaultRoute(ifaceName string) error {
	return AddRoute("0.0.0.0/0", "", "", ifaceName)
}

func AddRoute(dest string, src string, gw string, ifName string) error {
	if _, err := net.InterfaceByName(ifName); err != nil {
		return err
	}
	return netlink.AddRoute(dest, src, gw, ifName)
}

func InterfaceUp(name string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	return netlink.NetworkLinkUp(iface)
}

func InterfaceDown(name string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	return netlink.NetworkLinkDown(iface)
}

func ChangeInterfaceName(old, newName string) error {
	iface, err := net.InterfaceByName(old)
	if err != nil {
		return err
	}
	return netlink.NetworkChangeName(iface, newName)
}

func CreateVethPair(name1, name2 string, txQueueLen int) error {
	return netlink.NetworkCreateVethPair(name1, name2, txQueueLen)
}

func SetInterfaceInNamespacePid(name string, nsPid int) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	return netlink.NetworkSetNsPid(iface, nsPid)
}

func SetInterfaceMac(name string, macaddr string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	return netlink.NetworkSetMacAddress(iface, macaddr)
}

func SetInterfaceIp(name string, rawIp string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	ip, ipNet, err := net.ParseCIDR(rawIp)
	if err != nil {
		return err
	}
	return netlink.NetworkLinkAddIp(iface, ip, ipNet)
}

func SetMtu(name string, mtu int) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	return netlink.NetworkSetMTU(iface, mtu)
}
