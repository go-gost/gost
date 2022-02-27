package v5

import (
	"context"
	"fmt"
	"net"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/util/mux"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/common/util/udp"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
)

// Bind implements connector.Binder.
func (c *socks5Connector) Bind(ctx context.Context, conn net.Conn, network, address string, opts ...connector.BindOption) (net.Listener, error) {
	log := c.options.Logger.WithFields(map[string]any{
		"remote":  conn.RemoteAddr().String(),
		"local":   conn.LocalAddr().String(),
		"network": network,
		"address": address,
	})
	log.Infof("bind on %s/%s", address, network)

	options := connector.BindOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if options.Mux {
			return c.muxBindTCP(ctx, conn, network, address, log)
		}
		return c.bindTCP(ctx, conn, network, address, log)
	case "udp", "udp4", "udp6":
		return c.bindUDP(ctx, conn, network, address, &options, log)
	default:
		err := fmt.Errorf("network %s is unsupported", network)
		log.Error(err)
		return nil, err
	}
}

func (c *socks5Connector) bindTCP(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) (net.Listener, error) {
	laddr, err := c.bind(conn, gosocks5.CmdBind, network, address, log)
	if err != nil {
		return nil, err
	}

	return &tcpListener{
		addr:   laddr,
		conn:   conn,
		logger: log,
	}, nil
}

func (c *socks5Connector) muxBindTCP(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) (net.Listener, error) {
	laddr, err := c.bind(conn, socks.CmdMuxBind, network, address, log)
	if err != nil {
		return nil, err
	}

	session, err := mux.ServerSession(conn)
	if err != nil {
		return nil, err
	}

	return &tcpMuxListener{
		addr:    laddr,
		session: session,
		logger:  log,
	}, nil
}

func (c *socks5Connector) bindUDP(ctx context.Context, conn net.Conn, network, address string, opts *connector.BindOptions, log logger.Logger) (net.Listener, error) {
	laddr, err := c.bind(conn, socks.CmdUDPTun, network, address, log)
	if err != nil {
		return nil, err
	}

	ln := udp.NewListener(
		socks.UDPTunClientPacketConn(conn),
		laddr,
		opts.Backlog,
		opts.UDPDataQueueSize, opts.UDPDataBufferSize,
		opts.UDPConnTTL,
		log)

	return ln, nil
}

func (l *socks5Connector) bind(conn net.Conn, cmd uint8, network, address string, log logger.Logger) (net.Addr, error) {
	addr := gosocks5.Addr{}
	addr.ParseFrom(address)
	req := gosocks5.NewRequest(cmd, &addr)
	if err := req.Write(conn); err != nil {
		return nil, err
	}
	log.Debug(req)

	// first reply, bind status
	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		return nil, err
	}

	log.Debug(reply)

	if reply.Rep != gosocks5.Succeeded {
		return nil, fmt.Errorf("bind on %s/%s failed", address, network)
	}

	var baddr net.Addr
	switch network {
	case "tcp", "tcp4", "tcp6":
		baddr, err = net.ResolveTCPAddr(network, reply.Addr.String())
	case "udp", "udp4", "udp6":
		baddr, err = net.ResolveUDPAddr(network, reply.Addr.String())
	default:
		err = fmt.Errorf("unknown network %s", network)
	}
	if err != nil {
		return nil, err
	}
	log.Debugf("bind on %s/%s OK", baddr, baddr.Network())

	return baddr, nil
}
