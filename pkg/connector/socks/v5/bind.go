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
)

// Bind implements connector.Binder.
func (c *socks5Connector) Bind(ctx context.Context, conn net.Conn, network, address string, opts ...connector.BindOption) (connector.Accepter, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"network": network,
		"address": address,
	})
	c.logger.Infof("bind: %s/%s", address, network)

	options := connector.BindOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if options.Mux {
			return c.muxBindTCP(ctx, conn, network, address)
		}
		return c.bindTCP(ctx, conn, network, address)
	case "udp", "udp4", "udp6":
		return c.bindUDP(ctx, conn, network, address)
	default:
		err := fmt.Errorf("network %s is unsupported", network)
		c.logger.Error(err)
		return nil, err
	}
}

func (c *socks5Connector) bindTCP(ctx context.Context, conn net.Conn, network, address string) (connector.Accepter, error) {
	laddr, err := c.bind(conn, gosocks5.CmdBind, network, address)
	if err != nil {
		return nil, err
	}

	return &tcpAccepter{
		addr:   laddr,
		conn:   conn,
		logger: c.logger,
		done:   make(chan struct{}),
	}, nil
}

func (c *socks5Connector) muxBindTCP(ctx context.Context, conn net.Conn, network, address string) (connector.Accepter, error) {
	laddr, err := c.bind(conn, socks.CmdMuxBind, network, address)
	if err != nil {
		return nil, err
	}

	session, err := mux.ServerSession(conn)
	if err != nil {
		return nil, err
	}

	return &tcpMuxAccepter{
		addr:    laddr,
		session: session,
		logger:  c.logger,
	}, nil
}

func (c *socks5Connector) bindUDP(ctx context.Context, conn net.Conn, network, address string) (connector.Accepter, error) {
	laddr, err := c.bind(conn, socks.CmdUDPTun, network, address)
	if err != nil {
		return nil, err
	}

	accepter := &udpAccepter{
		addr:           laddr,
		conn:           socks.UDPTunClientPacketConn(conn),
		cqueue:         make(chan net.Conn, c.md.backlog),
		connPool:       udp.NewConnPool(c.md.ttl).WithLogger(c.logger),
		readQueueSize:  c.md.readQueueSize,
		readBufferSize: c.md.readBufferSize,
		closed:         make(chan struct{}),
		logger:         c.logger,
	}
	go accepter.acceptLoop()

	return accepter, nil
}

func (l *socks5Connector) bind(conn net.Conn, cmd uint8, network, address string) (net.Addr, error) {
	laddr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		return nil, err
	}

	addr := gosocks5.Addr{}
	addr.ParseFrom(laddr.String())
	req := gosocks5.NewRequest(cmd, &addr)
	if err := req.Write(conn); err != nil {
		return nil, err
	}
	l.logger.Debug(req)

	// first reply, bind status
	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		return nil, err
	}

	l.logger.Debug(reply)

	if reply.Rep != gosocks5.Succeeded {
		return nil, fmt.Errorf("bind on %s/%s failed", laddr, laddr.Network())
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
	l.logger.Debugf("bind on %s/%s OK", baddr, baddr.Network())

	return laddr, nil
}
