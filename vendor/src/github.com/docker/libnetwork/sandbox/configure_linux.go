package sandbox

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func configureInterface(iface netlink.Link, settings *Interface) error {
	ifaceName := iface.Attrs().Name
	ifaceConfigurators := []struct {
		Fn         func(netlink.Link, *Interface) error
		ErrMessage string
	}{
		{setInterfaceName, fmt.Sprintf("error renaming interface %q to %q", ifaceName, settings.DstName)},
		{setInterfaceIP, fmt.Sprintf("error setting interface %q IP to %q", ifaceName, settings.Addresses)},
		{setInterfaceIPv6, fmt.Sprintf("error setting interface %q IPv6 to %q", ifaceName, settings.AddressIPv6)},
		{setDefaultRoute, fmt.Sprintf("error setting default route to %q", ifaceName)},
	}

	for _, config := range ifaceConfigurators {
		if err := config.Fn(iface, settings); err != nil {
			return fmt.Errorf("%s: %v", config.ErrMessage, err)
		}
	}
	return nil
}

func programGateway(path string, gw net.IP) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origns, err := netns.Get()
	if err != nil {
		return err
	}
	defer origns.Close()

	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed get network namespace %q: %v", path, err)
	}
	defer f.Close()

	nsFD := f.Fd()
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		return err
	}
	defer netns.Set(origns)

	gwRoutes, err := netlink.RouteGet(gw)
	if err != nil {
		return fmt.Errorf("route for the gateway could not be found: %v", err)
	}

	return netlink.RouteAdd(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: gwRoutes[0].LinkIndex,
		Gw:        gw,
	})
}

func setInterfaceIP(iface netlink.Link, settings *Interface) error {
	logrus.Debugf("Setting IP address %s", settings.Addresses)
	for i := range settings.Addresses {
		logrus.Debugf("Setting IP %s to %s", settings.Addresses[i], iface)
		ipAddr := &netlink.Addr{IPNet: settings.Addresses[i], Label: ""}
		if err := netlink.AddrAdd(iface, ipAddr); err != nil {
			return err
		}
	}
	return nil;
}

func setInterfaceIPv6(iface netlink.Link, settings *Interface) error {
	if settings.AddressIPv6 == nil {
		return nil
	}
	ipAddr := &netlink.Addr{IPNet: settings.AddressIPv6, Label: ""}
	return netlink.AddrAdd(iface, ipAddr)
}

func setInterfaceName(iface netlink.Link, settings *Interface) error {
	return netlink.LinkSetName(iface, settings.DstName)
}

func setDefaultRoute(iface netlink.Link, settings *Interface) error {
	if strings.HasPrefix(iface.Attrs().Name, "vethr") {
		_, dip, _ := net.ParseCIDR("0.0.0.0/0")
		route := netlink.Route{LinkIndex: iface.Attrs().Index, Dst: dip}
		logrus.Debugf("Adding Default Route %s", route)
		return netlink.RouteAdd(&route)
	}
	return nil
}