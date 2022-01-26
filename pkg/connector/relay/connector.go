package relay

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/connector"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/relay"
)

func init() {
	registry.RegiserConnector("relay", NewConnector)
}

type relayConnector struct {
	user    *url.Userinfo
	md      metadata
	options connector.Options
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := connector.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &relayConnector{
		user:    options.User,
		options: options,
	}
}

func (c *relayConnector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *relayConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	log := c.options.Logger.WithFields(map[string]interface{}{
		"remote":  conn.RemoteAddr().String(),
		"local":   conn.LocalAddr().String(),
		"network": network,
		"address": address,
	})
	log.Infof("connect %s/%s", address, network)

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	req := relay.Request{
		Version: relay.Version1,
		Flags:   relay.CONNECT,
	}
	if network == "udp" || network == "udp4" || network == "udp6" {
		req.Flags |= relay.FUDP

		// UDP association
		if address == "" {
			baddr, err := c.bind(conn, relay.FUDP|relay.BIND, network, address)
			if err != nil {
				return nil, err
			}
			log.Debugf("associate on %s OK", baddr)

			return socks.UDPTunClientConn(conn, nil), nil
		}
	}

	if c.user != nil {
		pwd, _ := c.user.Password()
		req.Features = append(req.Features, &relay.UserAuthFeature{
			Username: c.user.Username(),
			Password: pwd,
		})
	}

	if address != "" {
		af := &relay.AddrFeature{}
		if err := af.ParseFrom(address); err != nil {
			return nil, err
		}

		// forward mode if port is 0.
		if af.Port > 0 {
			req.Features = append(req.Features, af)
		}
	}

	if c.md.noDelay {
		if _, err := req.WriteTo(conn); err != nil {
			return nil, err
		}
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		cc := &tcpConn{
			Conn: conn,
		}
		if !c.md.noDelay {
			if _, err := req.WriteTo(&cc.wbuf); err != nil {
				return nil, err
			}
		}
		conn = cc
	case "udp", "udp4", "udp6":
		cc := &udpConn{
			Conn: conn,
		}
		if !c.md.noDelay {
			if _, err := req.WriteTo(&cc.wbuf); err != nil {
				return nil, err
			}
		}
		conn = cc
	default:
		err := fmt.Errorf("network %s is unsupported", network)
		log.Error(err)
		return nil, err
	}

	return conn, nil
}
