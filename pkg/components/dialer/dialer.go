package dialer

import (
	"context"
	"net"
)

// Transporter is responsible for dialing to the proxy server.
type Dialer interface {
	Init(Metadata) error
	Dial(ctx context.Context, addr string, opts ...DialOption) (net.Conn, error)
}

type Handshaker interface {
	Handshake(ctx context.Context, conn net.Conn) (net.Conn, error)
}

type Multiplexer interface {
	IsMultiplex() bool
}
