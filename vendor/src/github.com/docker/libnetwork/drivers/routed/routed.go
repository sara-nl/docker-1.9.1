package routed

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/types"
	"github.com/docker/libnetwork/netutils"
	"github.com/docker/libnetwork/sandbox"
	"github.com/vishvananda/netlink"
	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/netlabel"
	"github.com/docker/libnetwork/options"
)

const (
	networkType = "routed"
	ifaceID     = 1
	VethPrefix  = "vethr" 
)

type routedNetwork struct {
	id        types.UUID
	endpoints map[types.UUID]*routedEndpoint
	sync.Mutex
}

type driver struct{
	network *routedNetwork
	mtu     int
}

type routedEndpoint struct {
	id              types.UUID
	iface           *sandbox.Interface
	macAddress      net.HardwareAddr
	hostInterface   string
	ipv4Addresses   []netlink.Addr
}

// Init registers a new instance of routed driver
func Init(dc driverapi.DriverCallback) error {
	logrus.Warnf("Registering Driver %s", networkType)
	return dc.RegisterDriver(networkType, &driver{})
}

func (d *driver) Config(option map[string]interface{}) error {
	m := option[netlabel.GenericData]
	ops := m.(options.Generic)
	mtu := ops["Mtu"].(int)
	logrus.Infof("Mtu %d", mtu)
	d.mtu = mtu
	return nil
}

func (d *driver) CreateNetwork(id types.UUID, option map[string]interface{}) error {
	logrus.Warnf("CreateNetwork: id=%s - option=%s", id, option)
	d.network = &routedNetwork{id: id, endpoints: make(map[types.UUID]*routedEndpoint)}
	return nil
}

func (d *driver) DeleteNetwork(id types.UUID) error {
	logrus.Warnf("DeleteNetwork: id=%s", id)
	d.network = nil
	return nil
}

func (d *driver) CreateEndpoint(nid, eid types.UUID, epInfo driverapi.EndpointInfo, epOptions map[string]interface{}) error {
	logrus.Warnf("CreatedEndpoint:")
	logrus.Debugf("nid=%s", nid)
	logrus.Debugf("eid=%s", eid)
	logrus.Debugf("epOptions= %s", epOptions)
	
	if epInfo == nil {
		return errors.New("invalid endpoint info passed")
	}
	logrus.Debugf("IPV4: %s", epOptions[netlabel.IPv4Addresses])
	if epOptions[netlabel.IPv4Addresses] == nil {
		return errors.New("Empty list of IP addresses passed to the routed(local) driver")
	}
	
	// Generate host veth
	hostIfaceName, err := generateIfaceName(VethPrefix + string(eid)[:4])
	if err != nil {
		return err
	}
	logrus.Debugf("Endpoint %s Host Interface: %s", eid, hostIfaceName)

	// Generate sandbox veth
	sandboxIfaceName, err := generateIfaceName(VethPrefix + string(eid)[:4])
	if err != nil {
		return err
	}
	logrus.Debugf("Endpoint %s Sandbox Interface: %s", eid, sandboxIfaceName)

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: hostIfaceName,
			TxQLen: 0},
		PeerName: sandboxIfaceName}

	if err = netlink.LinkAdd(veth); err != nil {
		return err
	}

	hostIface, err := netlink.LinkByName(hostIfaceName)
	if err != nil {
		logrus.Errorf("Can't find Host Interface: %s", hostIfaceName)	
		return err
	}
	defer func() {
		if err != nil {
			logrus.Infof("Deleting Host veth %s", hostIfaceName)
			netlink.LinkDel(hostIface)
		}
	}()

	logrus.Infof("Sandbox Interface: %s", sandboxIfaceName)
	sandboxIface, err := netlink.LinkByName(sandboxIfaceName)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			logrus.Infof("Deleting Container veth %s", sandboxIfaceName)
			netlink.LinkDel(sandboxIface)
		}
	}()

	network := d.network
	network.Mutex.Lock()
	defer network.Mutex.Unlock()

	endpoint := &routedEndpoint{id: eid}
	network.endpoints[eid] = endpoint

	ipv4Addresses := epOptions[netlabel.IPv4Addresses].([]net.IPNet)

	if d.mtu != 0 {
		err = netlink.LinkSetMTU(hostIface, d.mtu)
		if err != nil {
			return err
		}
		err = netlink.LinkSetMTU(sandboxIface, d.mtu)
		if err != nil {
			return err
		}
	}

	// Down the interface before configuring mac address.
	if err := netlink.LinkSetDown(sandboxIface); err != nil {
		return fmt.Errorf("could not set link down for container interface %s: %v", sandboxIfaceName, err)
	}
	var imac net.HardwareAddr
	if opt, ok := epOptions[netlabel.MacAddress]; ok {
		if mac, ok := opt.(net.HardwareAddr); ok {
			logrus.Debugf("Using Mac Address: %s", mac)
			imac = mac
		}
	}
	// Set the sbox's MAC. If specified, use the one configured by user, otherwise generate one based on IP.
	mac := electMacAddress(imac, ipv4Addresses[0].IP)

	err = netlink.LinkSetHardwareAddr(sandboxIface, mac)
	if err != nil {
		return fmt.Errorf("could not set mac address %s for container interface %s: %v", mac, sandboxIfaceName, err)
	}
	addr6 := &net.IPNet{}
	err = epInfo.AddInterface(1, mac, ipv4Addresses, *addr6)
	if err != nil {
		return err
	}

	// Up the host interface after finishing all netlink configuration
	if err := netlink.LinkSetUp(hostIface); err != nil {
		return fmt.Errorf("could not set link up for host interface %s: %v", hostIfaceName, err)
	}

	addresses := make([]*net.IPNet, len(ipv4Addresses))
	for i, _ := range ipv4Addresses {
		addresses[i] = &ipv4Addresses[i]
	}

	intf := &sandbox.Interface{}
	intf.SrcName = sandboxIfaceName
	intf.DstName = "eth"
	intf.Addresses =  addresses

	logrus.Debugf("Addresses added %s", intf.Addresses)

	endpoint.iface = intf
	endpoint.hostInterface = hostIfaceName
	return nil
}

// ErrIfaceName error is returned when a new name could not be generated.
type ErrIfaceName struct{}

func (ein *ErrIfaceName) Error() string {
	return "failed to find name for new interface"
}

func (d *driver) DeleteEndpoint(nid, eid types.UUID) error {
	network := d.network
	endpoint := network.endpoints[eid]
	delete(network.endpoints, eid)

	// Try removal of link. Discard error: link pair might have
	// already been deleted by sandbox delete.
	link, err := netlink.LinkByName(endpoint.hostInterface)
	if err == nil {
		logrus.Debugf("Deleting host interface %s", endpoint.hostInterface)
		netlink.LinkDel(link)
	} else {
		logrus.Debugf("Can't find host interface: %s ", err)
	}
	return nil
}

func (d *driver) EndpointOperInfo(nid, eid types.UUID) (map[string]interface{}, error) {
	return make(map[string]interface{}, 0), nil
}

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *driver) Join(nid, eid types.UUID, sboxKey string, jinfo driverapi.JoinInfo, options map[string]interface{}) error {

	network := d.network
	endpoint := network.endpoints[eid]
	
	for _, ipv4 := range endpoint.iface.Addresses {
		err := routeAdd(ipv4, "", "", endpoint.hostInterface)
		if err != nil {
			logrus.Errorf("Can't Add Route to %s -> %s : %s", ipv4, endpoint.hostInterface, err)
			return err
		}
	}
	
	logrus.Warnf("Join Network: %s - %s", endpoint.iface.SrcName, endpoint.iface.DstName)
	if endpoint == nil {
		logrus.Errorf("Endpoint not found %s", eid)
		return errors.New("Endpoint not found")
	}
	logrus.Infof("endpoint iface %s", endpoint.iface)
	
	for _, iNames := range jinfo.InterfaceNames() {
		// Make sure to set names on the correct interface ID.
		if iNames.ID() == ifaceID {
			logrus.Warnf("Info: %s - %s - %s", iNames.ID(), endpoint.iface.SrcName, endpoint.iface.DstName)
			err := iNames.SetNames(endpoint.iface.SrcName, endpoint.iface.DstName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Leave method is invoked when a Sandbox detaches from an endpoint.
func (d *driver) Leave(nid, eid types.UUID) error {
	return nil
}

func (d *driver) Type() string {
	return networkType
}

func electMacAddress(mac net.HardwareAddr, ip net.IP) net.HardwareAddr {
	if mac != nil {
		return mac
	}
	logrus.Debugf("Generating MacAddress")
	return generateMacAddr(ip)
}

func generateIfaceName(vethPrefix string) (string, error) {
	vethLen:= 12 - len(vethPrefix)
	for i := 0; i < 3; i++ {
		name, err := netutils.GenerateRandomName(vethPrefix, vethLen)
		if err != nil {
			continue
		}
		if _, err := net.InterfaceByName(name); err != nil {
			if strings.Contains(err.Error(), "no such") {
				return name, nil
			}
			return "", err
		}
	}
	return "", &ErrIfaceName{}
}

// Generate a IEEE802 compliant MAC address from the given IP address.
// The generator is guaranteed to be consistent: the same IP will always yield the same
// MAC address. This is to avoid ARP cache issues.
func generateMacAddr(ip net.IP) net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x02
	hw[1] = 0x42
	copy(hw[2:], ip.To4())
	return hw
}

func routeAdd(dest *net.IPNet, src string, gw string, ifaceName string) error {
	iface, _ := netlink.LinkByName(ifaceName)
	route := netlink.Route{LinkIndex: iface.Attrs().Index, Dst: dest}
	routeBroad := netlink.Route{Dst: dest}
	logrus.Infof("Adding Route in host %s", route)
	err := netlink.RouteAdd(&route)
	if err != nil {
		logrus.Debugf("Can't add route %s : %s", route, err)
		err = netlink.RouteDel(&routeBroad)
		logrus.Debugf("Deleted route %s :%s", routeBroad, err)
		err = netlink.RouteAdd(&route)
		logrus.Debugf("Re-Added Route %s :%s", route, err)
	}
	return err
}
