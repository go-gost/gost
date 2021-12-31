package hosts

import (
	"net"
)

// Host is a static mapping from hostname to IP.
type Host struct {
	IP       net.IP
	Hostname string
	Aliases  []string
}

// NewHost creates a Host.
func NewHost(ip net.IP, hostname string, aliases ...string) Host {
	return Host{
		IP:       ip,
		Hostname: hostname,
		Aliases:  aliases,
	}
}

// Hosts is a static table lookup for hostnames.
// For each host a single line should be present with the following information:
// IP_address canonical_hostname [aliases...]
// Fields of the entry are separated by any number of blanks and/or tab characters.
// Text from a "#" character until the end of the line is a comment, and is ignored.
type Hosts struct {
	hosts []Host
}

// AddHost adds host(s) to the host table.
func (h *Hosts) AddHost(host ...Host) {
	h.hosts = append(h.hosts, host...)
}

// Lookup searches the IP address corresponds to the given host from the host table.
func (h *Hosts) Lookup(host string) (ip net.IP) {
	if h == nil || host == "" {
		return
	}

	for _, h := range h.hosts {
		if h.Hostname == host {
			ip = h.IP
			break
		}
		for _, alias := range h.Aliases {
			if alias == host {
				ip = h.IP
				break
			}
		}
	}
	return
}
