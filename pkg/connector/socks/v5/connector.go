package v5

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("socks5", NewConnector)
	registry.RegiserConnector("socks", NewConnector)
}

type socks5Connector struct {
	selector gosocks5.Selector
	logger   logger.Logger
	md       metadata
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &socks5Connector{
		logger: options.Logger,
	}
}

func (c *socks5Connector) Init(md md.Metadata) (err error) {
	if err = c.parseMetadata(md); err != nil {
		return
	}

	selector := &clientSelector{
		methods: []uint8{
			gosocks5.MethodNoAuth,
			gosocks5.MethodUserPass,
		},
		logger:    c.logger,
		User:      c.md.User,
		TLSConfig: c.md.tlsConfig,
	}
	if !c.md.noTLS {
		selector.methods = append(selector.methods, socks.MethodTLS)
		if selector.TLSConfig == nil {
			selector.TLSConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
	c.selector = selector

	return
}

// Handshake implements connector.Handshaker.
func (c *socks5Connector) Handshake(ctx context.Context, conn net.Conn) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	cc := gosocks5.ClientConn(conn, c.selector)
	if err := cc.Handleshake(); err != nil {
		c.logger.Error(err)
		return nil, err
	}

	return cc, nil
}

func (c *socks5Connector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect %s/%s", address, network)

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	switch network {
	case "udp", "udp4", "udp6":
		return c.connectUDP(ctx, conn, network, address)
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

	req := gosocks5.NewRequest(gosocks5.CmdConnect, &addr)
	if err := req.Write(conn); err != nil {
		c.logger.Error(err)
		return nil, err
	}
	c.logger.Debug(req)

	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}
	c.logger.Debug(reply)

	if reply.Rep != gosocks5.Succeeded {
		err = errors.New("host unreachable")
		c.logger.Error(err)
		return nil, err
	}

	return conn, nil
}

func (c *socks5Connector) connectUDP(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error) {
	addr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}

	req := gosocks5.NewRequest(socks.CmdUDPTun, nil)
	if err := req.Write(conn); err != nil {
		c.logger.Error(err)
		return nil, err
	}
	c.logger.Debug(req)

	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}
	c.logger.Debug(reply)

	if reply.Rep != gosocks5.Succeeded {
		return nil, errors.New("get socks5 UDP tunnel failure")
	}

	baddr, err := net.ResolveUDPAddr("udp", reply.Addr.String())
	if err != nil {
		return nil, err
	}
	c.logger.Debugf("associate on %s OK", baddr)

	return socks.UDPTunClientConn(conn, addr), nil
}
