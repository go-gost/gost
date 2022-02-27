package registry

import (
	"github.com/go-gost/gost/pkg/admission"
)

type admissionRegistry struct {
	registry
}

func (r *admissionRegistry) Register(name string, v admission.Admission) error {
	return r.registry.Register(name, v)
}

func (r *admissionRegistry) Get(name string) admission.Admission {
	if name != "" {
		return &admissionWrapper{name: name, r: r}
	}
	return nil
}

func (r *admissionRegistry) get(name string) admission.Admission {
	if v := r.registry.Get(name); v != nil {
		return v.(admission.Admission)
	}
	return nil
}

type admissionWrapper struct {
	name string
	r    *admissionRegistry
}

func (w *admissionWrapper) Admit(addr string) bool {
	p := w.r.get(w.name)
	if p == nil {
		return false
	}
	return p.Admit(addr)
}
