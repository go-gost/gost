package chain

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/dialer"
)

type Transport struct {
	addr      string
	ifceName  string
	route     *Route
	dialer    dialer.Dialer
	connector connector.Connector
}

func (tr *Transport) Copy() *Transport {
	tr2 := &Transport{}
	*tr2 = *tr
	return tr
}

func (tr *Transport) WithInterface(ifceName string) *Transport {
	tr.ifceName = ifceName
	return tr
}

func (tr *Transport) WithDialer(dialer dialer.Dialer) *Transport {
	tr.dialer = dialer
	return tr
}

func (tr *Transport) WithConnector(connector connector.Connector) *Transport {
	tr.connector = connector
	return tr
}

func (tr *Transport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	netd := &dialer.NetDialer{
		Interface: tr.ifceName,
		Timeout:   30 * time.Second,
	}
	if tr.route.Len() > 0 {
		netd.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return tr.route.Dial(ctx, network, addr)
		}
	}
	opts := []dialer.DialOption{
		dialer.HostDialOption(tr.addr),
		dialer.NetDialerDialOption(netd),
	}
	return tr.dialer.Dial(ctx, addr, opts...)
}

func (tr *Transport) Handshake(ctx context.Context, conn net.Conn) (net.Conn, error) {
	var err error
	if hs, ok := tr.dialer.(dialer.Handshaker); ok {
		conn, err = hs.Handshake(ctx, conn,
			dialer.AddrHandshakeOption(tr.addr))
		if err != nil {
			return nil, err
		}
	}
	if hs, ok := tr.connector.(connector.Handshaker); ok {
		return hs.Handshake(ctx, conn)
	}
	return conn, nil
}

func (tr *Transport) Connect(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error) {
	return tr.connector.Connect(ctx, conn, network, address)
}

func (tr *Transport) Bind(ctx context.Context, conn net.Conn, network, address string, opts ...connector.BindOption) (net.Listener, error) {
	if binder, ok := tr.connector.(connector.Binder); ok {
		return binder.Bind(ctx, conn, network, address, opts...)
	}
	return nil, connector.ErrBindUnsupported
}

func (tr *Transport) Multiplex() bool {
	if mux, ok := tr.dialer.(dialer.Multiplexer); ok {
		return mux.Multiplex()
	}
	return false
}

func (tr *Transport) WithRoute(r *Route) *Transport {
	tr.route = r
	return tr
}

func (tr *Transport) WithAddr(addr string) *Transport {
	tr.addr = addr
	return tr
}
