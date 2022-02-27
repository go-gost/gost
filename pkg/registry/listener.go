package registry

import (
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
)

type NewListener func(opts ...listener.Option) listener.Listener

type listenerRegistry struct {
	registry
}

func (r *listenerRegistry) Register(name string, v NewListener) error {
	if err := r.registry.Register(name, v); err != nil {
		logger.Default().Fatal(err)
	}
	return nil
}

func (r *listenerRegistry) Get(name string) NewListener {
	if v := r.registry.Get(name); v != nil {
		return v.(NewListener)
	}
	return nil
}
