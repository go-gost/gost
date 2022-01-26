package chain

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/resolver"
)

type Router struct {
	Retries  int
	Chain    *Chain
	Hosts    hosts.HostMapper
	Resolver resolver.Resolver
	Logger   logger.Logger
}

func (r *Router) Dial(ctx context.Context, network, address string) (conn net.Conn, err error) {
	conn, err = r.dial(ctx, network, address)
	if err != nil {
		return
	}
	if network == "udp" || network == "udp4" || network == "udp6" {
		if _, ok := conn.(net.PacketConn); !ok {
			return &packetConn{conn}, nil
		}
	}
	return
}

func (r *Router) dial(ctx context.Context, network, address string) (conn net.Conn, err error) {
	count := r.Retries + 1
	if count <= 0 {
		count = 1
	}
	r.Logger.Debugf("dial %s/%s", address, network)

	for i := 0; i < count; i++ {
		route := r.Chain.GetRouteFor(network, address)

		if r.Logger.IsLevelEnabled(logger.DebugLevel) {
			buf := bytes.Buffer{}
			for _, node := range route.Path() {
				fmt.Fprintf(&buf, "%s@%s > ", node.Name, node.Addr)
			}
			fmt.Fprintf(&buf, "%s", address)
			r.Logger.Debugf("route(retry=%d) %s", i, buf.String())
		}

		address, err = r.resolve(ctx, address)
		if err != nil {
			r.Logger.Error(err)
			break
		}

		conn, err = route.Dial(ctx, network, address)
		if err == nil {
			break
		}
		r.Logger.Errorf("route(retry=%d) %s", i, err)
	}

	return
}

func (r *Router) resolve(ctx context.Context, addr string) (string, error) {
	if addr == "" {
		return addr, nil
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	if host == "" {
		return addr, nil
	}

	if r.Hosts != nil {
		if ips, _ := r.Hosts.Lookup("ip", host); len(ips) > 0 {
			r.Logger.Debugf("hit host mapper: %s -> %s", host, ips)
			return net.JoinHostPort(ips[0].String(), port), nil
		}
	}

	if r.Resolver != nil {
		ips, err := r.Resolver.Resolve(ctx, host)
		if err != nil {
			r.Logger.Error(err)
		}
		if len(ips) == 0 {
			return "", fmt.Errorf("resolver: domain %s does not exists", host)
		}
		return net.JoinHostPort(ips[0].String(), port), nil
	}
	return addr, nil
}

func (r *Router) Bind(ctx context.Context, network, address string, opts ...connector.BindOption) (ln net.Listener, err error) {
	count := r.Retries + 1
	if count <= 0 {
		count = 1
	}
	r.Logger.Debugf("bind on %s/%s", address, network)

	for i := 0; i < count; i++ {
		route := r.Chain.GetRouteFor(network, address)

		if r.Logger.IsLevelEnabled(logger.DebugLevel) {
			buf := bytes.Buffer{}
			for _, node := range route.Path() {
				fmt.Fprintf(&buf, "%s@%s > ", node.Name, node.Addr)
			}
			fmt.Fprintf(&buf, "%s", address)
			r.Logger.Debugf("route(retry=%d) %s", i, buf.String())
		}

		ln, err = route.Bind(ctx, network, address, opts...)
		if err == nil {
			break
		}
		r.Logger.Errorf("route(retry=%d) %s", i, err)
	}

	return
}

type packetConn struct {
	net.Conn
}

func (c *packetConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, err = c.Read(b)
	addr = c.Conn.RemoteAddr()
	return
}

func (c *packetConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return c.Write(b)
}
