package registry

import (
	"github.com/go-gost/gost/pkg/auth"
)

type autherRegistry struct {
	registry
}

func (r *autherRegistry) Register(name string, v auth.Authenticator) error {
	return r.registry.Register(name, v)
}

func (r *autherRegistry) Get(name string) auth.Authenticator {
	if name != "" {
		return &autherWrapper{name: name, r: r}
	}
	return nil
}

func (r *autherRegistry) get(name string) auth.Authenticator {
	if v := r.registry.Get(name); v != nil {
		return v.(auth.Authenticator)
	}
	return nil
}

type autherWrapper struct {
	name string
	r    *autherRegistry
}

func (w *autherWrapper) Authenticate(user, password string) bool {
	v := w.r.get(w.name)
	if v == nil {
		return true
	}
	return v.Authenticate(user, password)
}
