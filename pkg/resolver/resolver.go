package resolver

import (
	"context"
	"errors"
	"net"
)

var (
	ErrInvalid = errors.New("resolver invalid")
)

type Resolver interface {
	// Resolve returns a slice of the host's IPv4 and IPv6 addresses.
	// The network should be 'ip', 'ip4' or 'ip6', default network is 'ip'.
	Resolve(ctx context.Context, network, host string) ([]net.IP, error)
}
