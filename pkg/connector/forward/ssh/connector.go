package ssh

import (
	"context"
	"errors"
	"net"

	"github.com/go-gost/gost/pkg/connector"
	ssh_util "github.com/go-gost/gost/pkg/internal/util/ssh"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("sshd", NewConnector)
}

type forwardConnector struct {
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &forwardConnector{
		logger: options.Logger,
	}
}

func (c *forwardConnector) Init(md md.Metadata) (err error) {
	return nil
}

func (c *forwardConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"remote":  conn.RemoteAddr().String(),
		"local":   conn.LocalAddr().String(),
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect %s/%s", address, network)

	cc, ok := conn.(*ssh_util.ClientConn)
	if !ok {
		return nil, errors.New("ssh: invalid connection")
	}

	conn, err := cc.Client().Dial(network, address)
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}

	return conn, nil
}

// Bind implements connector.Binder.
func (c *forwardConnector) Bind(ctx context.Context, conn net.Conn, network, address string, opts ...connector.BindOption) (net.Listener, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"network": network,
		"address": address,
	})
	c.logger.Infof("bind on %s/%s", address, network)

	cc, ok := conn.(*ssh_util.ClientConn)
	if !ok {
		return nil, errors.New("ssh: invalid connection")
	}

	if host, port, _ := net.SplitHostPort(address); host == "" {
		address = net.JoinHostPort("0.0.0.0", port)
	}

	return cc.Client().Listen(network, address)
}
