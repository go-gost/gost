package chain

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/dialer"
)

type Transport struct {
	route     *Route
	dialer    dialer.Dialer
	connector connector.Connector
}

func (tr *Transport) Copy() *Transport {
	tr2 := &Transport{}
	*tr2 = *tr
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
	return tr.dialer.Dial(ctx, addr, tr.dialOptions()...)
}

func (tr *Transport) dialOptions() []dialer.DialOption {
	var opts []dialer.DialOption
	if tr.route != nil {
		opts = append(opts,
			dialer.DialFuncDialOption(
				func(ctx context.Context, addr string) (net.Conn, error) {
					return tr.route.Dial(ctx, "tcp", addr)
				},
			),
		)
	}
	return opts
}

func (tr *Transport) Handshake(ctx context.Context, conn net.Conn) (net.Conn, error) {
	if hs, ok := tr.dialer.(dialer.Handshaker); ok {
		return hs.Handshake(ctx, conn)
	}
	return conn, nil
}

func (tr *Transport) Connect(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error) {
	return tr.connector.Connect(ctx, conn, network, address)
}

func (tr *Transport) IsMultiplex() bool {
	if mux, ok := tr.dialer.(dialer.Multiplexer); ok {
		return mux.IsMultiplex()
	}
	return false
}

func (tr *Transport) WithRoute(r *Route) *Transport {
	tr.route = r
	return tr
}
