package experimental

import (
	"errors"
	"github.com/docker/docker/engine"
	"net"
	"sync"
)

// Network interface represents the networking stack of a container
type networkInterface struct {
	IP           net.IP
	IPv6         net.IP
	PortMappings []net.Addr // there are mappings to the host interfaces
}

type ifaces struct {
	c map[string]*networkInterface
	sync.Mutex
}

func (i *ifaces) Set(key string, n *networkInterface) {
	i.Lock()
	i.c[key] = n
	i.Unlock()
}

func (i *ifaces) Get(key string) *networkInterface {
	i.Lock()
	res := i.c[key]
	i.Unlock()
	return res
}

var (
	currentInterfaces = ifaces{c: make(map[string]*networkInterface)}
)

func InitDriver(job *engine.Job) engine.Status {
	job.Logf("Initializing Experimental Network Driver")

	for name, f := range map[string]engine.Handler{
		"allocate_interface": Allocate,
		"release_interface":  Release,
		"allocate_port":      AllocatePort,
		"link":               LinkContainers,
	} {
		if err := job.Eng.Register(name, f); err != nil {
			return job.Error(err)
		}
	}
	return engine.StatusOK
}

// Generate a IEEE802 compliant MAC address from the given IP address.
//
// The generator is guaranteed to be consistent: the same IP will always yield the same
// MAC address. This is to avoid ARP cache issues.
// See bridge/driver
func generateMacAddr(ip net.IP) net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x02
	hw[1] = 0x42
	copy(hw[2:], ip.To4())
	return hw
}

// Allocate a network interface
func Allocate(job *engine.Job) engine.Status {
	var (
		ip          net.IP
		ipNet       *net.IPNet
		mac         net.HardwareAddr
		err         error
		id          = job.Args[0]
		requestedIP = job.Getenv("RequestedIP")
	)

	if requestedIP != "" {
		ip, ipNet, err = net.ParseCIDR(requestedIP)
	} else {
		job.Error(errors.New("No ip address requeted. use --ip-address to specify a static api address."))
	}
	if err != nil {
		return job.Error(err)
	}

	// If no explicit mac address was given, generate a random one.
	if mac, err = net.ParseMAC(job.Getenv("RequestedMac")); err != nil {
		mac = generateMacAddr(ip)
	}

	out := engine.Env{}
	out.Set("IP", ip.String())
	out.Set("Mask", ipNet.Mask.String())
	out.Set("MacAddress", mac.String())

	size, _ := ipNet.Mask.Size()
	out.SetInt("IPPrefixLen", size)

	currentInterfaces.Set(id, &networkInterface{
		IP: ip,
	})

	out.WriteTo(job.Stdout)

	return engine.StatusOK
}

// Release an interface for a  selected ip
func Release(job *engine.Job) engine.Status {
	var (
		id                 = job.Args[0]
		containerInterface = currentInterfaces.Get(id)
	)

	if containerInterface == nil {
		return job.Errorf("No network information to release for %s", id)
	}

	return engine.StatusOK
}

func LinkContainers(job *engine.Job) engine.Status {
	// Containers not linked
	return engine.StatusOK
}

func AllocatePort(job *engine.Job) engine.Status {
	// Don't allocate ports
	return engine.StatusOK
}
