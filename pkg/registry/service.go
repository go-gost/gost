package registry

import (
	"sync"

	"github.com/go-gost/gost/pkg/service"
)

var (
	svcReg = &serviceRegistry{}
)

func Service() *serviceRegistry {
	return svcReg
}

type serviceRegistry struct {
	m sync.Map
}

func (r *serviceRegistry) Register(name string, svc *service.Service) error {
	if _, loaded := r.m.LoadOrStore(name, svc); loaded {
		return ErrDup
	}

	return nil
}

func (r *serviceRegistry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *serviceRegistry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *serviceRegistry) Get(name string) *service.Service {
	v, ok := r.m.Load(name)
	if !ok {
		return nil
	}
	return v.(*service.Service)
}
