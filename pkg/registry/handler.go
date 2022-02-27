package registry

import (
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
)

type NewHandler func(opts ...handler.Option) handler.Handler

type handlerRegistry struct {
	registry
}

func (r *handlerRegistry) Register(name string, v NewHandler) error {
	if err := r.registry.Register(name, v); err != nil {
		logger.Default().Fatal(err)
	}
	return nil
}

func (r *handlerRegistry) Get(name string) NewHandler {
	if v := r.registry.Get(name); v != nil {
		return v.(NewHandler)
	}
	return nil
}
