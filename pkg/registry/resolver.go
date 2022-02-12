package registry

import (
	"context"
	"net"
	"sync"

	"github.com/go-gost/gost/pkg/resolver"
)

var (
	resolverReg = &resolverRegistry{}
)

func Resolver() *resolverRegistry {
	return resolverReg
}

type resolverRegistry struct {
	m sync.Map
}

func (r *resolverRegistry) Register(name string, resolver resolver.Resolver) error {
	if name == "" || resolver == nil {
		return nil
	}
	if _, loaded := r.m.LoadOrStore(name, resolver); loaded {
		return ErrDup
	}

	return nil
}

func (r *resolverRegistry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *resolverRegistry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *resolverRegistry) Get(name string) resolver.Resolver {
	if name == "" {
		return nil
	}
	return &resolverWrapper{name: name}
}

func (r *resolverRegistry) get(name string) resolver.Resolver {
	if v, ok := r.m.Load(name); ok {
		return v.(resolver.Resolver)
	}
	return nil
}

type resolverWrapper struct {
	name string
}

func (w *resolverWrapper) Resolve(ctx context.Context, network, host string) ([]net.IP, error) {
	r := Resolver().get(w.name)
	if r == nil {
		return nil, resolver.ErrInvalid
	}
	return r.Resolve(ctx, network, host)
}
