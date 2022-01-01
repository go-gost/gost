package hosts

import (
	"net"
)

// HostMapper is a mapping from hostname to IP.
type HostMapper interface {
	Lookup(host string) net.IP
}

type host struct {
	IP       net.IP
	Hostname string
	Aliases  []string
}

// Hosts is a static table lookup for hostnames.
// For each host a single line should be present with the following information:
// IP_address canonical_hostname [aliases...]
// Fields of the entry are separated by any number of blanks and/or tab characters.
// Text from a "#" character until the end of the line is a comment, and is ignored.
type Hosts struct {
	mappings []host
}

func NewHosts() *Hosts {
	return &Hosts{}
}

// Map maps ip to hostname or aliases.
func (h *Hosts) Map(ip net.IP, hostname string, aliases ...string) {
	h.mappings = append(h.mappings, host{
		IP:       ip,
		Hostname: hostname,
		Aliases:  aliases,
	})
}

// Lookup searches the IP address corresponds to the given host from the host table.
func (h *Hosts) Lookup(host string) (ip net.IP) {
	if h == nil || host == "" {
		return
	}

	for _, h := range h.mappings {
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
