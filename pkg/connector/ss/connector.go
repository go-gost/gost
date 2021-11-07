package ss

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/internal/bufpool"
	"github.com/go-gost/gost/pkg/internal/utils/ss"
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
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
		"target": address,
	})
	c.logger.Infof("connect: ", address)

	socksAddr, err := gosocks5.NewAddr(address)
	if err != nil {
		c.logger.Error("parse addr: ", err)
		return nil, err
	}
	rawaddr := bufpool.Get(512)
	defer bufpool.Put(rawaddr)

	n, err := socksAddr.Encode(rawaddr)
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

func (c *ssConnector) parseMetadata(md md.Metadata) (err error) {
	c.md.cipher, err = ss.ShadowCipher(
		md.GetString(method),
		md.GetString(password),
		md.GetString(key),
	)
	if err != nil {
		return
	}

	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.noDelay = md.GetBool(noDelay)

	return
}
