package connector

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/components/metadata"
)

// Connector is responsible for connecting to the destination address.
type Connector interface {
	Init(metadata.Metadata) error
	Connect(ctx context.Context, conn net.Conn, network, address string, opts ...ConnectOption) (net.Conn, error)
}
