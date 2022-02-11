package registry

import (
	"sync"

	"github.com/go-gost/gost/pkg/auth"
)

var (
	autherReg = &autherRegistry{}
)

func Auther() *autherRegistry {
	return autherReg
}

type autherRegistry struct {
	m sync.Map
}

func (r *autherRegistry) Register(name string, auth auth.Authenticator) error {
	if _, loaded := r.m.LoadOrStore(name, auth); loaded {
		return ErrDup
	}

	return nil
}

func (r *autherRegistry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *autherRegistry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *autherRegistry) Get(name string) auth.Authenticator {
	if name == "" {
		return nil
	}
	return &autherWrapper{name: name}
}

func (r *autherRegistry) get(name string) auth.Authenticator {
	if v, ok := r.m.Load(name); ok {
		return v.(auth.Authenticator)
	}
	return nil
}

type autherWrapper struct {
	name string
}

func (w *autherWrapper) Authenticate(user, password string) bool {
	v := autherReg.get(w.name)
	if v == nil {
		return true
	}
	return v.Authenticate(user, password)
}
