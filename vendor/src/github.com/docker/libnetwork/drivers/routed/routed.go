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
)

const (
	networkType = "routed"
	ifaceID		= 1
)

type routedNetwork struct {
	id        types.UUID
//	config    *NetworkConfiguration
	endpoints map[types.UUID]*routedEndpoint // key: endpoint id
	sync.Mutex
}

type driver struct{
	network *routedNetwork
}

type routedEndpoint struct {
	id              types.UUID
	iface           *sandbox.Interface
	macAddress      net.HardwareAddr
	hostInterface   string
	ipv4Addresses   []netlink.Addr
}

// Init registers a new instance of host driver
func Init(dc driverapi.DriverCallback) error {
	logrus.Warnf("Registering Driver %s", networkType)
	return dc.RegisterDriver(networkType, &driver{})
}

func (d *driver) Config(option map[string]interface{}) error {
	logrus.Warnf("Config: %s", option)
	return nil
}

func (d *driver) CreateNetwork(id types.UUID, option map[string]interface{}) error {
	logrus.Warnf("CreateNetwork: id=%s - option=%s", id, option)	
	d.network = &routedNetwork{id: id, endpoints: make(map[types.UUID]*routedEndpoint)}
	return nil
}

func (d *driver) DeleteNetwork(nid types.UUID) error {
	logrus.Warnf("DeleteNetwork: nid=%s", nid)
	d.network = nil
	return nil
}

func (d *driver) CreateEndpoint(nid, eid types.UUID, epInfo driverapi.EndpointInfo, epOptions map[string]interface{}) error {
	logrus.Warnf("CreatedEndpoint:")
	logrus.Debugf("nid=%s", nid)
	logrus.Debugf("eid=%s", eid)
	logrus.Debugf("epInfo.Interfaces= %s", epInfo.Interfaces())
	logrus.Debugf("epOptions= %s", epOptions)
	
	if epInfo == nil {
		return errors.New("invalid endpoint info passed")
	}
	logrus.Debugf("IPV4: %s", epOptions[netlabel.IPv4Addresses])
	//|| len(epOptions[netlabel.IPv4Addresses]) == 0 
	if epOptions[netlabel.IPv4Addresses] == nil {
		return errors.New("empty list of IP addresses passed to the routed(local) driver")
	}
	
	// Generate host veth
	hostIfaceName, err := generateIfaceName()
	if err != nil {
		return err
	}
	logrus.Debugf("Host Interface: %s", hostIfaceName)
	
	// Generate sandbox veth
	sandboxIfaceName, err := generateIfaceName()
	if err != nil {
		return err
	}
	
	// Generate and add the interface pipe host <-> sandbox
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
	logrus.Debugf("*** Generate sandbox Veth")
	
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

	endpoint := &routedEndpoint{id: eid} // config: epConfig
	network.endpoints[eid] = endpoint

	ipv4Addresses := epOptions[netlabel.IPv4Addresses].([]net.IPNet)
	
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
	
	logrus.Debugf("routed.go Addresses added %s", intf.Addresses)
	
	endpoint.iface = intf
	endpoint.hostInterface = hostIfaceName
	////
	return nil
}
// ErrIfaceName error is returned when a new name could not be generated.
type ErrIfaceName struct{}

func (ein *ErrIfaceName) Error() string {
	return "failed to find name for new interface"
}

func generateIfaceName() (string, error) {
	vethPrefix:= "veth"
	vethLen:=7
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

func (d *driver) DeleteEndpoint(nid, eid types.UUID) error {
	network := d.network
	endpoint := network.endpoints[eid]
	delete(network.endpoints, eid)

	// Try removal of link. Discard error: link pair might have
	// already been deleted by sandbox delete.
	link, err := netlink.LinkByName(endpoint.hostInterface)
	if err == nil {
		netlink.LinkDel(link)
	}
	logrus.Debugf("Deleting Endpoint")
	return nil
}

func (d *driver) EndpointOperInfo(nid, eid types.UUID) (map[string]interface{}, error) {
	return make(map[string]interface{}, 0), nil
}

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *driver) Join(nid, eid types.UUID, sboxKey string, jinfo driverapi.JoinInfo, options map[string]interface{}) error {

	network := d.network
	//	if err != nil {
	//		return err
	//	}
	endpoint := network.endpoints[eid]
	//	if err != nil {
	//		return err
	//	}
	//	addDefaultRoute(endpoint.iface.DstName)
	logrus.Warnf("addreses endpoint.iface.Addresses %s", endpoint.iface.Addresses)
	
	for _, ipv4 := range endpoint.iface.Addresses {
		logrus.Warnf("adding route %s", ipv4)
		addRoute(ipv4, "", "", endpoint.hostInterface)
	}
	
	logrus.Warnf("Join Network: %s - %s", endpoint.iface.SrcName, endpoint.iface.DstName)
	if endpoint == nil {
		logrus.Errorf("Endpoint not found %s", eid)
		return errors.New("endpoint not found")
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

// Generate a IEEE802 compliant MAC address from the given IP address.
//
// The generator is guaranteed to be consistent: the same IP will always yield the same
// MAC address. This is to avoid ARP cache issues.
func generateMacAddr(ip net.IP) net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)

	// The first byte of the MAC address has to comply with these rules:
	// 1. Unicast: Set the least-significant bit to 0.
	// 2. Address is locally administered: Set the second-least-significant bit (U/L) to 1.
	// 3. As "small" as possible: The veth address has to be "smaller" than the bridge address.
	hw[0] = 0x02

	// The first 24 bits of the MAC represent the Organizationally Unique Identifier (OUI).
	// Since this address is locally administered, we can do whatever we want as long as
	// it doesn't conflict with other addresses.
	hw[1] = 0x42

	// Insert the IP address into the last 32 bits of the MAC address.
	// This is a simple way to guarantee the address will be consistent and unique.
	copy(hw[2:], ip.To4())

	return hw
}

func addRoute(dest *net.IPNet, src string, gw string, ifaceName string) error {
	iface, _ := netlink.LinkByName(ifaceName)
	route := netlink.Route{LinkIndex: iface.Attrs().Index, Dst: dest}
	logrus.Debugf("Adding Route %s", route)
	return netlink.RouteAdd(&route)
}
