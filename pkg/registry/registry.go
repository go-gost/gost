package registry

import (
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/listener"
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
	listeners[name] = newf
}

func GetListener(name string) NewListener {
	return listeners[name]
}

func RegisterHandler(name string, newf NewHandler) {
	handlers[name] = newf
}

func GetHandler(name string) NewHandler {
	return handlers[name]
}

func RegisterDialer(name string, newf NewDialer) {
	dialers[name] = newf
}

func GetDialer(name string) NewDialer {
	return dialers[name]
}

func RegiserConnector(name string, newf NewConnector) {
	connectors[name] = newf
}

func GetConnector(name string) NewConnector {
	return connectors[name]
}
