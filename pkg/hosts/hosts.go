package hosts

import (
	"net"
	"sync"

	"github.com/go-gost/gost/pkg/logger"
)

// HostMapper is a mapping from hostname to IP.
type HostMapper interface {
	Lookup(network, host string) ([]net.IP, bool)
}

type hostMapping struct {
	IPs      []net.IP
	Hostname string
}

// Hosts is a static table lookup for hostnames.
// For each host a single line should be present with the following information:
// IP_address canonical_hostname [aliases...]
// Fields of the entry are separated by any number of blanks and/or tab characters.
// Text from a "#" character until the end of the line is a comment, and is ignored.
type Hosts struct {
	mappings sync.Map
	Logger   logger.Logger
}

func NewHosts() *Hosts {
	return &Hosts{}
}

// Map maps ip to hostname or aliases.
func (h *Hosts) Map(ip net.IP, hostname string, aliases ...string) {
	if hostname == "" {
		return
	}

	v, _ := h.mappings.Load(hostname)
	m, _ := v.(*hostMapping)
	if m == nil {
		m = &hostMapping{
			IPs:      []net.IP{ip},
			Hostname: hostname,
		}
	} else {
		m.IPs = append(m.IPs, ip)
	}
	h.mappings.Store(hostname, m)

	for _, alias := range aliases {
		// indirect mapping from alias to hostname
		if alias != "" {
			h.mappings.Store(alias, &hostMapping{
				Hostname: hostname,
			})
		}
	}
}

// Lookup searches the IP address corresponds to the given network and host from the host table.
// The network should be 'ip', 'ip4' or 'ip6', default network is 'ip'.
func (h *Hosts) Lookup(network, host string) (ips []net.IP, ok bool) {
	if h == nil || host == "" {
		return
	}

	v, ok := h.mappings.Load(host)
	if !ok {
		return
	}
	m, _ := v.(*hostMapping)
	if m == nil {
		return
	}

	// hostname alias
	if host != m.Hostname {
		v, _ = h.mappings.Load(m.Hostname)
		m, _ = v.(*hostMapping)
		if m == nil {
			return
		}
	}

	switch network {
	case "ip4":
		for _, ip := range m.IPs {
			if ip = ip.To4(); ip != nil {
				ips = append(ips, ip)
			}
		}
	case "ip6":
		for _, ip := range m.IPs {
			if ip.To4() == nil {
				ips = append(ips, ip)
			}
		}
	default:
		ips = m.IPs
	}

	if len(ips) > 0 {
		h.Logger.Debugf("host mapper: %s -> %s", host, ips)
	}

	return
}
