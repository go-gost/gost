package ssu

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/internal/utils/socks"
	"github.com/go-gost/gost/pkg/internal/utils/ss"
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

	switch network {
	case "udp", "udp4", "udp6":
	default:
		err := fmt.Errorf("network %s unsupported, should be udp, udp4 or udp6", network)
		c.logger.Error(err)
		return nil, err
	}

	c.logger.Info("connect: ", address)

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

		return ss.UDPClientConn(pc, conn.RemoteAddr(), taddr, c.md.bufferSize), nil
	}

	return socks.UDPTunClientConn(conn, taddr), nil
}

func (c *ssuConnector) parseMetadata(md md.Metadata) (err error) {
	c.md.cipher, err = ss.ShadowCipher(
		md.GetString(method),
		md.GetString(password),
		md.GetString(key),
	)
	if err != nil {
		return
	}

	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.bufferSize = md.GetInt(bufferSize)
	if c.md.bufferSize > 0 {
		if c.md.bufferSize < 512 {
			c.md.bufferSize = 512
		}
		if c.md.bufferSize > 65*1024 {
			c.md.bufferSize = 65 * 1024
		}
	} else {
		c.md.bufferSize = 4096
	}

	return
}
