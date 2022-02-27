package registry

import (
	"net"

	"github.com/go-gost/gost/pkg/hosts"
)

type hostsRegistry struct {
	registry
}

func (r *hostsRegistry) Register(name string, v hosts.HostMapper) error {
	return r.registry.Register(name, v)
}

func (r *hostsRegistry) Get(name string) hosts.HostMapper {
	if name != "" {
		return &hostsWrapper{name: name, r: r}
	}
	return nil
}

func (r *hostsRegistry) get(name string) hosts.HostMapper {
	if v := r.registry.Get(name); v != nil {
		return v.(hosts.HostMapper)
	}
	return nil
}

type hostsWrapper struct {
	name string
	r    *hostsRegistry
}

func (w *hostsWrapper) Lookup(network, host string) ([]net.IP, bool) {
	v := w.r.get(w.name)
	if v == nil {
		return nil, false
	}
	return v.Lookup(network, host)
}
