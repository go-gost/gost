package registry

import (
	"errors"
	"sync"

	"github.com/go-gost/gost/pkg/admission"
	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/resolver"
	"github.com/go-gost/gost/pkg/service"
)

var (
	ErrDup = errors.New("registry: duplicate object")
)

var (
	listenerReg  Registry[NewListener]  = &listenerRegistry{}
	handlerReg   Registry[NewHandler]   = &handlerRegistry{}
	dialerReg    Registry[NewDialer]    = &dialerRegistry{}
	connectorReg Registry[NewConnector] = &connectorRegistry{}

	serviceReg   Registry[service.Service]     = &serviceRegistry{}
	chainReg     Registry[chain.Chainer]       = &chainRegistry{}
	autherReg    Registry[auth.Authenticator]  = &autherRegistry{}
	admissionReg Registry[admission.Admission] = &admissionRegistry{}
	bypassReg    Registry[bypass.Bypass]       = &bypassRegistry{}
	resolverReg  Registry[resolver.Resolver]   = &resolverRegistry{}
	hostsReg     Registry[hosts.HostMapper]    = &hostsRegistry{}
)

type Registry[T any] interface {
	Register(name string, v T) error
	Unregister(name string)
	IsRegistered(name string) bool
	Get(name string) T
}

type registry struct {
	m sync.Map
}

func (r *registry) Register(name string, v any) error {
	if name == "" || v == nil {
		return nil
	}
	if _, loaded := r.m.LoadOrStore(name, v); loaded {
		return ErrDup
	}

	return nil
}

func (r *registry) Unregister(name string) {
	r.m.Delete(name)
}

func (r *registry) IsRegistered(name string) bool {
	_, ok := r.m.Load(name)
	return ok
}

func (r *registry) Get(name string) any {
	if name == "" {
		return nil
	}
	v, _ := r.m.Load(name)
	return v
}

func ListenerRegistry() Registry[NewListener] {
	return listenerReg
}

func HandlerRegistry() Registry[NewHandler] {
	return handlerReg
}

func DialerRegistry() Registry[NewDialer] {
	return dialerReg
}

func ConnectorRegistry() Registry[NewConnector] {
	return connectorReg
}

func ServiceRegistry() Registry[service.Service] {
	return serviceReg
}

func ChainRegistry() Registry[chain.Chainer] {
	return chainReg
}

func AutherRegistry() Registry[auth.Authenticator] {
	return autherReg
}

func AdmissionRegistry() Registry[admission.Admission] {
	return admissionReg
}

func BypassRegistry() Registry[bypass.Bypass] {
	return bypassReg
}

func ResolverRegistry() Registry[resolver.Resolver] {
	return resolverReg
}

func HostsRegistry() Registry[hosts.HostMapper] {
	return hostsReg
}
