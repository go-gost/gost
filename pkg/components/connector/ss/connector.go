package ss

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/components/connector"
	md "github.com/go-gost/gost/pkg/components/metadata"
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

func (c *Connector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *Connector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {

	return conn, nil
}

func (c *Connector) parseMetadata(md md.Metadata) (err error) {
	c.md.method = md.GetString(method)
	c.md.password = md.GetString(password)

	return
}
