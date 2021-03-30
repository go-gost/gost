package connector

import (
	"context"
	"net"
)

// Connector is responsible for connecting to the destination address.
type Connector interface {
	Connect(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error)
}
