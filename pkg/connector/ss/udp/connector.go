package ss

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/common/util/ss"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("ssu", NewConnector)
}

type ssuConnector struct {
	md     metadata
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &ssuConnector{
		logger: options.Logger,
	}
}

func (c *ssuConnector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *ssuConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"remote":  conn.RemoteAddr().String(),
		"local":   conn.LocalAddr().String(),
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect %s/%s", address, network)

	switch network {
	case "udp", "udp4", "udp6":
	default:
		err := fmt.Errorf("network %s is unsupported", network)
		c.logger.Error(err)
		return nil, err
	}

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	taddr, _ := net.ResolveUDPAddr(network, address)
	if taddr == nil {
		taddr = &net.UDPAddr{}
	}

	pc, ok := conn.(net.PacketConn)
	if ok {
		if c.md.cipher != nil {
			pc = c.md.cipher.PacketConn(pc)
		}

		// standard UDP relay
		return ss.UDPClientConn(pc, conn.RemoteAddr(), taddr, c.md.bufferSize), nil
	}

	if c.md.cipher != nil {
		conn = ss.ShadowConn(c.md.cipher.StreamConn(conn), nil)
	}

	// UDP over TCP
	return socks.UDPTunClientConn(conn, taddr), nil
}
