package registry

import (
	"sync"

	"github.com/go-gost/gost/pkg/chain"
)

var (
	chainReg = &chainRegistry{}
)

func Chain() *chainRegistry {
	return chainReg
}

type chainRegistry struct {
	m sync.Map
}

func (r *chainRegistry) Register(name string, chain chain.Chainer) error {
	if _, loaded := r.m.LoadOrStore(name, chain); loaded {
		return ErrDup
	}

	return nil
}

func (r *chainRegistry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *chainRegistry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *chainRegistry) Get(name string) chain.Chainer {
	if name == "" {
		return nil
	}
	return &chainWrapper{name: name}
}

func (r *chainRegistry) get(name string) chain.Chainer {
	if v, ok := r.m.Load(name); ok {
		return v.(chain.Chainer)
	}
	return nil
}

type chainWrapper struct {
	name string
}

func (w *chainWrapper) Route(network, address string) *chain.Route {
	v := Chain().get(w.name)
	if v == nil {
		return nil
	}
	return v.Route(network, address)
}
