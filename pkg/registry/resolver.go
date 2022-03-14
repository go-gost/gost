package registry

import (
	"context"
	"net"

	"github.com/go-gost/gost/v3/pkg/resolver"
)

type resolverRegistry struct {
	registry
}

func (r *resolverRegistry) Register(name string, v resolver.Resolver) error {
	return r.registry.Register(name, v)
}

func (r *resolverRegistry) Get(name string) resolver.Resolver {
	if name != "" {
		return &resolverWrapper{name: name, r: r}
	}
	return nil
}

func (r *resolverRegistry) get(name string) resolver.Resolver {
	if v := r.registry.Get(name); v != nil {
		return v.(resolver.Resolver)
	}
	return nil
}

type resolverWrapper struct {
	name string
	r    *resolverRegistry
}

func (w *resolverWrapper) Resolve(ctx context.Context, network, host string) ([]net.IP, error) {
	r := w.r.get(w.name)
	if r == nil {
		return nil, resolver.ErrInvalid
	}
	return r.Resolve(ctx, network, host)
}
