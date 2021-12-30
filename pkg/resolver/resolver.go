package resolver

import (
	"context"
	"net"
)

type Resolver interface {
	// Resolve returns a slice of the host's IPv4 and IPv6 addresses.
	Resolve(ctx context.Context, host string) ([]net.IP, error)
}
