package registry

import (
	"github.com/go-gost/gost/pkg/bypass"
)

type bypassRegistry struct {
	registry
}

func (r *bypassRegistry) Register(name string, v bypass.Bypass) error {
	return r.registry.Register(name, v)
}

func (r *bypassRegistry) Get(name string) bypass.Bypass {
	if name != "" {
		return &bypassWrapper{name: name, r: r}
	}
	return nil
}

func (r *bypassRegistry) get(name string) bypass.Bypass {
	if v := r.registry.Get(name); v != nil {
		return v.(bypass.Bypass)
	}
	return nil
}

type bypassWrapper struct {
	name string
	r    *bypassRegistry
}

func (w *bypassWrapper) Contains(addr string) bool {
	bp := w.r.get(w.name)
	if bp == nil {
		return false
	}
	return bp.Contains(addr)
}
