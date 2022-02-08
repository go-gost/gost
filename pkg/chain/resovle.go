package chain

import (
	"context"
	"fmt"
	"net"

	"github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/resolver"
)

func resolve(ctx context.Context, network, addr string, r resolver.Resolver, hosts hosts.HostMapper, log logger.Logger) (string, error) {
	if addr == "" {
		return addr, nil
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	if host == "" {
		return addr, nil
	}

	if hosts != nil {
		if ips, _ := hosts.Lookup(network, host); len(ips) > 0 {
			log.Debugf("hit host mapper: %s -> %s", host, ips)
			return net.JoinHostPort(ips[0].String(), port), nil
		}
	}

	if r != nil {
		ips, err := r.Resolve(ctx, network, host)
		if err != nil {
			if err == resolver.ErrInvalid {
				return addr, nil
			}
			log.Error(err)
		}
		if len(ips) == 0 {
			return "", fmt.Errorf("resolver: domain %s does not exists", host)
		}
		return net.JoinHostPort(ips[0].String(), port), nil
	}
	return addr, nil
}
