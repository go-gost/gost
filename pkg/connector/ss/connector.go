package ss

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/common/util/ss"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("ss", NewConnector)
}

type ssConnector struct {
	md     metadata
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &ssConnector{
		logger: options.Logger,
	}
}

func (c *ssConnector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *ssConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"remote":  conn.RemoteAddr().String(),
		"local":   conn.LocalAddr().String(),
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect: %s/%s", address, network)

	switch network {
	case "tcp", "tcp4", "tcp6":
	case "udp", "udp4", "udp6":
		if c.md.enableUDP {
			return c.connectUDP(ctx, conn, network, address)
		} else {
			err := errors.New("UDP relay is disabled")
			c.logger.Error(err)
			return nil, err
		}
	default:
		err := fmt.Errorf("network %s unsupported", network)
		c.logger.Error(err)
		return nil, err
	}

	addr := gosocks5.Addr{}
	if err := addr.ParseFrom(address); err != nil {
		c.logger.Error(err)
		return nil, err
	}
	rawaddr := bufpool.Get(512)
	defer bufpool.Put(rawaddr)

	n, err := addr.Encode(rawaddr)
	if err != nil {
		c.logger.Error("encoding addr: ", err)
		return nil, err
	}

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	if c.md.cipher != nil {
		conn = c.md.cipher.StreamConn(conn)
	}

	var sc net.Conn
	if c.md.noDelay {
		sc = ss.ShadowConn(conn, nil)
		// write the addr at once.
		if _, err := sc.Write(rawaddr[:n]); err != nil {
			return nil, err
		}
	} else {
		// cache the header
		sc = ss.ShadowConn(conn, rawaddr[:n])
	}

	return sc, nil
}

func (c *ssConnector) connectUDP(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error) {
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

		return ss.UDPClientConn(pc, conn.RemoteAddr(), taddr, c.md.udpBufferSize), nil
	}

	if c.md.cipher != nil {
		conn = ss.ShadowConn(c.md.cipher.StreamConn(conn), nil)
	}
	return socks.UDPTunClientConn(conn, taddr), nil
}
