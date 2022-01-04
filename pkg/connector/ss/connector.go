package ss

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/ss"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

func init() {
	registry.RegiserConnector("ss", NewConnector)
}

type ssConnector struct {
	cipher  core.Cipher
	md      metadata
	logger  logger.Logger
	options connector.Options
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := connector.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &ssConnector{
		options: options,
		logger:  options.Logger,
	}
}

func (c *ssConnector) Init(md md.Metadata) (err error) {
	if err = c.parseMetadata(md); err != nil {
		return
	}

	if c.options.User != nil {
		method := c.options.User.Username()
		password, _ := c.options.User.Password()
		c.cipher, err = ss.ShadowCipher(method, password, c.md.key)
	}

	return
}

func (c *ssConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"remote":  conn.RemoteAddr().String(),
		"local":   conn.LocalAddr().String(),
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect %s/%s", address, network)

	switch network {
	case "tcp", "tcp4", "tcp6":
		if _, ok := conn.(net.PacketConn); ok {
			err := fmt.Errorf("tcp over udp is unsupported")
			c.logger.Error(err)
			return nil, err
		}
	default:
		err := fmt.Errorf("network %s is unsupported", network)
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

	if c.cipher != nil {
		conn = c.cipher.StreamConn(conn)
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
