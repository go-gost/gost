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

func (r *chainRegistry) Get(name string) chain.Chainer {
	if _, ok := r.m.Load(name); !ok {
		return nil
	}
	return &chainWrapper{name: name}
}

type chainWrapper struct {
	name string
}

func (w *chainWrapper) Route(network, address string) *chain.Route {
	v := Chain().Get(w.name)
	if v == nil {
		return nil
	}
	return v.Route(network, address)
}
