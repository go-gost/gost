package transporter

import (
	"context"
	"net"
)

// Transporter is responsible for handshaking with server.
type Transporter interface {
	Dial(ctx context.Context, addr string) (net.Conn, error)
	Handshake(ctx context.Context, conn net.Conn) (net.Conn, error)
	// Indicate that the Transporter supports multiplex
	Multiplex() bool
}
