package dialer

import (
	"context"
	"net"

	"github.com/go-gost/gost/v3/pkg/metadata"
)

// Transporter is responsible for dialing to the proxy server.
type Dialer interface {
	Init(metadata.Metadata) error
	Dial(ctx context.Context, addr string, opts ...DialOption) (net.Conn, error)
}

type Handshaker interface {
	Handshake(ctx context.Context, conn net.Conn, opts ...HandshakeOption) (net.Conn, error)
}

type Multiplexer interface {
	Multiplex() bool
}
