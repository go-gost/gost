package registry

import (
	"github.com/go-gost/gost/v3/pkg/chain"
)

type chainRegistry struct {
	registry
}

func (r *chainRegistry) Register(name string, v chain.Chainer) error {
	return r.registry.Register(name, v)
}

func (r *chainRegistry) Get(name string) chain.Chainer {
	if name != "" {
		return &chainWrapper{name: name, r: r}
	}
	return nil
}

func (r *chainRegistry) get(name string) chain.Chainer {
	if v := r.registry.Get(name); v != nil {
		return v.(chain.Chainer)
	}
	return nil
}

type chainWrapper struct {
	name string
	r    *chainRegistry
}

func (w *chainWrapper) Route(network, address string) *chain.Route {
	v := w.r.get(w.name)
	if v == nil {
		return nil
	}
	return v.Route(network, address)
}
