package registry

import (
	"errors"

	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
)

var (
	ErrDup      = errors.New("registry: duplicate instance")
	ErrNotFound = errors.New("registry: instance not found")
)

type NewListener func(opts ...listener.Option) listener.Listener
type NewHandler func(opts ...handler.Option) handler.Handler
type NewDialer func(opts ...dialer.Option) dialer.Dialer
type NewConnector func(opts ...connector.Option) connector.Connector

var (
	listeners  = make(map[string]NewListener)
	handlers   = make(map[string]NewHandler)
	dialers    = make(map[string]NewDialer)
	connectors = make(map[string]NewConnector)
)

func RegisterListener(name string, newf NewListener) {
	if listeners[name] != nil {
		logger.Default().Fatalf("register duplicate listener: %s", name)
	}
	listeners[name] = newf
}

func GetListener(name string) NewListener {
	return listeners[name]
}

func RegisterHandler(name string, newf NewHandler) {
	if handlers[name] != nil {
		logger.Default().Fatalf("register duplicate handler: %s", name)
	}
	handlers[name] = newf
}

func GetHandler(name string) NewHandler {
	return handlers[name]
}

func RegisterDialer(name string, newf NewDialer) {
	if dialers[name] != nil {
		logger.Default().Fatalf("register duplicate dialer: %s", name)
	}
	dialers[name] = newf
}

func GetDialer(name string) NewDialer {
	return dialers[name]
}

func RegiserConnector(name string, newf NewConnector) {
	if connectors[name] != nil {
		logger.Default().Fatalf("register duplicate connector: %s", name)
	}
	connectors[name] = newf
}

func GetConnector(name string) NewConnector {
	return connectors[name]
}
