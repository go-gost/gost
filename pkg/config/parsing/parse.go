package parsing

import (
	"net"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/config"
	hostspkg "github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/resolver"
	resolver_impl "github.com/go-gost/gost/pkg/resolver/impl"
)

func parseSelector(cfg *config.SelectorConfig) chain.Selector {
	if cfg == nil {
		return nil
	}

	var strategy chain.Strategy
	switch cfg.Strategy {
	case "round", "rr":
		strategy = chain.RoundRobinStrategy()
	case "random", "rand":
		strategy = chain.RandomStrategy()
	case "fifo", "ha":
		strategy = chain.FIFOStrategy()
	default:
		strategy = chain.RoundRobinStrategy()
	}

	return chain.NewSelector(
		strategy,
		chain.InvalidFilter(),
		chain.FailFilter(cfg.MaxFails, cfg.FailTimeout),
	)
}

func ParseBypass(cfg *config.BypassConfig) bypass.Bypass {
	if cfg == nil {
		return nil
	}
	return bypass.NewBypassPatterns(
		cfg.Reverse,
		cfg.Matchers,
		bypass.LoggerBypassOption(logger.Default().WithFields(map[string]interface{}{
			"kind":   "bypass",
			"bypass": cfg.Name,
		})),
	)
}

func ParseResolver(cfg *config.ResolverConfig) (resolver.Resolver, error) {
	if cfg == nil {
		return nil, nil
	}
	var nameservers []resolver_impl.NameServer
	for _, server := range cfg.Nameservers {
		nameservers = append(nameservers, resolver_impl.NameServer{
			Addr: server.Addr,
			// Chain:    chains[server.Chain],
			TTL:      server.TTL,
			Timeout:  server.Timeout,
			ClientIP: net.ParseIP(server.ClientIP),
			Prefer:   server.Prefer,
			Hostname: server.Hostname,
		})
	}

	return resolver_impl.NewResolver(
		nameservers,
		resolver_impl.LoggerResolverOption(
			logger.Default().WithFields(map[string]interface{}{
				"kind":     "resolver",
				"resolver": cfg.Name,
			}),
		),
	)
}

func ParseHosts(cfg *config.HostsConfig) hostspkg.HostMapper {
	if cfg == nil || len(cfg.Mappings) == 0 {
		return nil
	}
	hosts := hostspkg.NewHosts()
	hosts.Logger = logger.Default().WithFields(map[string]interface{}{
		"kind":  "hosts",
		"hosts": cfg.Name,
	})

	for _, host := range cfg.Mappings {
		if host.IP == "" || host.Hostname == "" {
			continue
		}

		ip := net.ParseIP(host.IP)
		if ip == nil {
			continue
		}
		hosts.Map(ip, host.Hostname, host.Aliases...)
	}
	return hosts
}
