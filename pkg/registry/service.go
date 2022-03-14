package registry

import (
	"github.com/go-gost/gost/v3/pkg/service"
)

type serviceRegistry struct {
	registry
}

func (r *serviceRegistry) Register(name string, v service.Service) error {
	return r.registry.Register(name, v)
}

func (r *serviceRegistry) Get(name string) service.Service {
	if v := r.registry.Get(name); v != nil {
		return v.(service.Service)
	}
	return nil
}
