package registry

import (
	"sync"

	"github.com/go-gost/gost/pkg/bypass"
)

var (
	bypassReg = &bypassRegistry{}
)

func Bypass() *bypassRegistry {
	return bypassReg
}

type bypassRegistry struct {
	m sync.Map
}

func (r *bypassRegistry) Register(name string, bypass bypass.Bypass) error {
	if name == "" || bypass == nil {
		return nil
	}
	if _, loaded := r.m.LoadOrStore(name, bypass); loaded {
		return ErrDup
	}

	return nil
}

func (r *bypassRegistry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *bypassRegistry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *bypassRegistry) Get(name string) bypass.Bypass {
	if name == "" {
		return nil
	}
	return &bypassWrapper{name: name}
}

func (r *bypassRegistry) get(name string) bypass.Bypass {
	if v, ok := r.m.Load(name); ok {
		return v.(bypass.Bypass)
	}
	return nil
}

type bypassWrapper struct {
	name string
}

func (w *bypassWrapper) Contains(addr string) bool {
	bp := bypassReg.get(w.name)
	if bp == nil {
		return false
	}
	return bp.Contains(addr)
}
