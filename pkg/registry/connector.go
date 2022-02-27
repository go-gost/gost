package registry

import (
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
)

type NewConnector func(opts ...connector.Option) connector.Connector

type connectorRegistry struct {
	registry
}

func (r *connectorRegistry) Register(name string, v NewConnector) error {
	if err := r.registry.Register(name, v); err != nil {
		logger.Default().Fatal(err)
	}
	return nil
}

func (r *connectorRegistry) Get(name string) NewConnector {
	if v := r.registry.Get(name); v != nil {
		return v.(NewConnector)
	}
	return nil
}
