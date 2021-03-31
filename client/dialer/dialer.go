package dialer

import (
	"context"
	"net"
)

// Dialer dials to target server.
type Dialer interface {
	Init(md Metadata) error
	Dial(ctx context.Context, addr string) (net.Conn, error)
}

type Handshaker interface {
	Handshake(ctx context.Context, conn net.Conn) (net.Conn, error)
}

type Multiplexer interface {
	Multiplexed() bool
}
