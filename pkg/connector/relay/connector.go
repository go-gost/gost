package relay

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/relay"
)

func init() {
	registry.RegiserConnector("relay", NewConnector)
}

type relayConnector struct {
	logger logger.Logger
	md     metadata
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &relayConnector{
		logger: options.Logger,
	}
}

func (c *relayConnector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *relayConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"remote":  conn.RemoteAddr().String(),
		"local":   conn.LocalAddr().String(),
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect: %s/%s", address, network)

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	var udpMode bool
	if network == "udp" || network == "udp4" || network == "udp6" {
		udpMode = true
	}

	req := relay.Request{
		Version: relay.Version1,
	}
	if udpMode {
		req.Flags |= relay.FUDP
	}

	if c.md.user != nil {
		pwd, _ := c.md.user.Password()
		req.Features = append(req.Features, &relay.UserAuthFeature{
			Username: c.md.user.Username(),
			Password: pwd,
		})
	}

	if address != "" {
		af := &relay.AddrFeature{}
		if err := af.ParseFrom(address); err != nil {
			return nil, err
		}

		req.Features = append(req.Features, af)
	}

	return conn, nil
}
