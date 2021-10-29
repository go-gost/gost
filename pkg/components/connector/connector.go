package connector

import (
	"context"
	"net"
)

// Connector is responsible for connecting to the destination address.
type Connector interface {
	Init(Metadata) error
	Connect(ctx context.Context, conn net.Conn, network, address string, opts ...ConnectOption) (net.Conn, error)
}
