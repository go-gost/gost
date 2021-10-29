package ss

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/components/connector"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("ss", NewConnector)
}

type Connector struct {
	md     metadata
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &Connector{
		logger: options.Logger,
	}
}

func (c *Connector) Init(md connector.Metadata) (err error) {
	c.md, err = c.parseMetadata(md)
	if err != nil {
		return
	}

	return nil
}

func (c *Connector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {

	return conn, nil
}

func (c *Connector) parseMetadata(md connector.Metadata) (m metadata, err error) {
	if md == nil {
		md = connector.Metadata{}
	}

	m.method = md[method]
	m.password = md[password]

	return
}
