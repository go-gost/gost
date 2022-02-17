package registry

import (
	"sync"

	"github.com/go-gost/gost/pkg/admission"
)

var (
	admissionReg = &admissionRegistry{}
)

func Admission() *admissionRegistry {
	return admissionReg
}

type admissionRegistry struct {
	m sync.Map
}

func (r *admissionRegistry) Register(name string, admission admission.Admission) error {
	if name == "" || admission == nil {
		return nil
	}
	if _, loaded := r.m.LoadOrStore(name, admission); loaded {
		return ErrDup
	}

	return nil
}

func (r *admissionRegistry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *admissionRegistry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *admissionRegistry) Get(name string) admission.Admission {
	if name == "" {
		return nil
	}
	return &admissionWrapper{name: name}
}

func (r *admissionRegistry) get(name string) admission.Admission {
	if v, ok := r.m.Load(name); ok {
		return v.(admission.Admission)
	}
	return nil
}

type admissionWrapper struct {
	name string
}

func (w *admissionWrapper) Admit(addr string) bool {
	p := admissionReg.get(w.name)
	if p == nil {
		return false
	}
	return p.Admit(addr)
}
