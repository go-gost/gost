package registry

import (
	"net"
	"sync"

	"github.com/go-gost/gost/pkg/hosts"
)

var (
	hostsReg = &hostsRegistry{}
)

func Hosts() *hostsRegistry {
	return hostsReg
}

type hostsRegistry struct {
	m sync.Map
}

func (r *hostsRegistry) Register(name string, hosts hosts.HostMapper) error {
	if _, loaded := r.m.LoadOrStore(name, hosts); loaded {
		return ErrDup
	}

	return nil
}

func (r *hostsRegistry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *hostsRegistry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *hostsRegistry) Get(name string) hosts.HostMapper {
	if name == "" {
		return nil
	}
	return &hostsWrapper{name: name}
}

func (r *hostsRegistry) get(name string) hosts.HostMapper {
	if v, ok := r.m.Load(name); ok {
		return v.(hosts.HostMapper)
	}
	return nil
}

type hostsWrapper struct {
	name string
}

func (w *hostsWrapper) Lookup(network, host string) ([]net.IP, bool) {
	v := Hosts().get(w.name)
	if v == nil {
		return nil, false
	}
	return v.Lookup(network, host)
}
