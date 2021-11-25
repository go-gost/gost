package v4

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gosocks4"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("socks4", NewConnector)
	registry.RegiserConnector("socks4a", NewConnector)
}

type socks4Connector struct {
	md     metadata
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &socks4Connector{
		logger: options.Logger,
	}
}

func (c *socks4Connector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *socks4Connector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
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

	var addr *gosocks4.Addr

	if c.md.disable4a {
		taddr, err := net.ResolveTCPAddr("tcp4", address)
		if err != nil {
			c.logger.Error("resolve: ", err)
			return nil, err
		}
		if len(taddr.IP) == 0 {
			taddr.IP = net.IPv4zero
		}
		addr = &gosocks4.Addr{
			Type: gosocks4.AddrIPv4,
			Host: taddr.IP.String(),
			Port: uint16(taddr.Port),
		}
	} else {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		p, _ := strconv.Atoi(port)
		addr = &gosocks4.Addr{
			Type: gosocks4.AddrDomain,
			Host: host,
			Port: uint16(p),
		}
	}

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	req := gosocks4.NewRequest(gosocks4.CmdConnect, addr, nil)
	if err := req.Write(conn); err != nil {
		c.logger.Error(err)
		return nil, err
	}
	c.logger.Debug(req)

	reply, err := gosocks4.ReadReply(conn)
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}
	c.logger.Debug(reply)

	if reply.Code != gosocks4.Granted {
		err = errors.New("host unreachable")
		c.logger.Error(err)
		return nil, err
	}

	return conn, nil
}
