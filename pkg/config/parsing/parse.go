package parsing

import (
	"net"
	"net/url"

	"github.com/go-gost/gost/v3/pkg/admission"
	"github.com/go-gost/gost/v3/pkg/auth"
	"github.com/go-gost/gost/v3/pkg/bypass"
	"github.com/go-gost/gost/v3/pkg/chain"
	"github.com/go-gost/gost/v3/pkg/config"
	hostspkg "github.com/go-gost/gost/v3/pkg/hosts"
	"github.com/go-gost/gost/v3/pkg/logger"
	"github.com/go-gost/gost/v3/pkg/registry"
	"github.com/go-gost/gost/v3/pkg/resolver"
	resolver_impl "github.com/go-gost/gost/v3/pkg/resolver/impl"
)

func ParseAuther(cfg *config.AutherConfig) auth.Authenticator {
	if cfg == nil {
		return nil
	}

	m := make(map[string]string)

	for _, user := range cfg.Auths {
		if user.Username == "" {
			continue
		}
		m[user.Username] = user.Password
	}

	if len(m) == 0 {
		return nil
	}
	return auth.NewAuthenticator(m)
}

func ParseAutherFromAuth(au *config.AuthConfig) auth.Authenticator {
	if au == nil || au.Username == "" {
		return nil
	}
	return auth.NewAuthenticator(map[string]string{
		au.Username: au.Password,
	})
}

func parseAuth(cfg *config.AuthConfig) *url.Userinfo {
	if cfg == nil || cfg.Username == "" {
		return nil
	}

	if cfg.Password == "" {
		return url.User(cfg.Username)
	}
	return url.UserPassword(cfg.Username, cfg.Password)
}

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

func ParseAdmission(cfg *config.AdmissionConfig) admission.Admission {
	if cfg == nil {
		return nil
	}
	return admission.NewAdmissionPatterns(
		cfg.Reverse,
		cfg.Matchers,
		admission.LoggerOption(logger.Default().WithFields(map[string]any{
			"kind":      "admission",
			"admission": cfg.Name,
		})),
	)
}

func ParseBypass(cfg *config.BypassConfig) bypass.Bypass {
	if cfg == nil {
		return nil
	}
	return bypass.NewBypassPatterns(
		cfg.Reverse,
		cfg.Matchers,
		bypass.LoggerOption(logger.Default().WithFields(map[string]any{
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
			Addr:     server.Addr,
			Chain:    registry.ChainRegistry().Get(server.Chain),
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
			logger.Default().WithFields(map[string]any{
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
	hosts.Logger = logger.Default().WithFields(map[string]any{
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
