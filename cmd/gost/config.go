package main

import (
	"io"
	"net"
	"os"
	"strings"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/handler"
	hostspkg "github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/resolver"
	resolver_impl "github.com/go-gost/gost/pkg/resolver/impl"
	"github.com/go-gost/gost/pkg/service"
)

var (
	chains    = make(map[string]*chain.Chain)
	bypasses  = make(map[string]bypass.Bypass)
	resolvers = make(map[string]resolver.Resolver)
	hosts     = make(map[string]hostspkg.HostMapper)
)

func buildService(cfg *config.Config) (services []*service.Service) {
	if cfg == nil || len(cfg.Services) == 0 {
		return
	}

	for _, bypassCfg := range cfg.Bypasses {
		bypasses[bypassCfg.Name] = bypassFromConfig(bypassCfg)
	}

	for _, resolverCfg := range cfg.Resolvers {
		r, err := resolverFromConfig(resolverCfg)
		if err != nil {
			log.Fatal(err)
		}
		resolvers[resolverCfg.Name] = r
	}
	for _, hostsCfg := range cfg.Hosts {
		hosts[hostsCfg.Name] = hostsFromConfig(hostsCfg)
	}

	for _, chainCfg := range cfg.Chains {
		chains[chainCfg.Name] = chainFromConfig(chainCfg)
	}

	for _, svc := range cfg.Services {
		serviceLogger := log.WithFields(map[string]interface{}{
			"kind":     "service",
			"service":  svc.Name,
			"listener": svc.Listener.Type,
			"handler":  svc.Handler.Type,
			"chain":    svc.Chain,
		})

		listenerLogger := serviceLogger.WithFields(map[string]interface{}{
			"kind": "listener",
		})
		ln := registry.GetListener(svc.Listener.Type)(
			listener.AddrOption(svc.Addr),
			listener.LoggerOption(listenerLogger),
		)

		if chainable, ok := ln.(chain.Chainable); ok {
			chainable.WithChain(chains[svc.Chain])
		}

		if svc.Listener.Metadata == nil {
			svc.Listener.Metadata = make(map[string]interface{})
		}
		if err := ln.Init(metadata.MapMetadata(svc.Listener.Metadata)); err != nil {
			listenerLogger.Fatal("init: ", err)
		}

		handlerLogger := serviceLogger.WithFields(map[string]interface{}{
			"kind": "handler",
		})

		h := registry.GetHandler(svc.Handler.Type)(
			handler.BypassOption(bypasses[svc.Bypass]),
			handler.LoggerOption(handlerLogger),
			handler.RouterOption(&chain.Router{
				Chain:    chains[svc.Chain],
				Resolver: resolvers[svc.Resolver],
				Hosts:    hosts[svc.Hosts],
				Logger:   handlerLogger,
			}),
		)

		if forwarder, ok := h.(handler.Forwarder); ok {
			forwarder.Forward(forwarderFromConfig(svc.Forwarder))
		}

		if svc.Handler.Metadata == nil {
			svc.Handler.Metadata = make(map[string]interface{})
		}
		if err := h.Init(metadata.MapMetadata(svc.Handler.Metadata)); err != nil {
			handlerLogger.Fatal("init: ", err)
		}

		s := (&service.Service{}).
			WithListener(ln).
			WithHandler(h).
			WithLogger(serviceLogger)
		services = append(services, s)

		serviceLogger.Infof("listening on %s/%s", s.Addr().String(), s.Addr().Network())
	}

	return
}

func chainFromConfig(cfg *config.ChainConfig) *chain.Chain {
	if cfg == nil {
		return nil
	}

	chainLogger := log.WithFields(map[string]interface{}{
		"kind":  "chain",
		"chain": cfg.Name,
	})

	c := &chain.Chain{}
	selector := selectorFromConfig(cfg.Selector)
	for _, hop := range cfg.Hops {
		group := &chain.NodeGroup{}
		for _, v := range hop.Nodes {
			connectorLogger := chainLogger.WithFields(map[string]interface{}{
				"kind":      "connector",
				"connector": v.Connector.Type,
				"dialer":    v.Dialer.Type,
				"hop":       hop.Name,
				"node":      v.Name,
			})
			cr := registry.GetConnector(v.Connector.Type)(
				connector.LoggerOption(connectorLogger),
			)

			if v.Connector.Metadata == nil {
				v.Connector.Metadata = make(map[string]interface{})
			}
			if err := cr.Init(metadata.MapMetadata(v.Connector.Metadata)); err != nil {
				connectorLogger.Fatal("init: ", err)
			}

			dialerLogger := chainLogger.WithFields(map[string]interface{}{
				"kind":      "dialer",
				"connector": v.Connector.Type,
				"dialer":    v.Dialer.Type,
				"hop":       hop.Name,
				"node":      v.Name,
			})
			d := registry.GetDialer(v.Dialer.Type)(
				dialer.LoggerOption(dialerLogger),
			)

			if v.Dialer.Metadata == nil {
				v.Dialer.Metadata = make(map[string]interface{})
			}
			if err := d.Init(metadata.MapMetadata(v.Dialer.Metadata)); err != nil {
				dialerLogger.Fatal("init: ", err)
			}

			tr := (&chain.Transport{}).
				WithConnector(cr).
				WithDialer(d).
				WithAddr(v.Addr)

			node := chain.NewNode(v.Name, v.Addr).
				WithTransport(tr).
				WithBypass(bypasses[v.Bypass])
			group.AddNode(node)
		}

		sel := selector
		if s := selectorFromConfig(hop.Selector); s != nil {
			sel = s
		}
		group.WithSelector(sel)
		c.AddNodeGroup(group)
	}

	return c
}

func forwarderFromConfig(cfg *config.ForwarderConfig) *chain.NodeGroup {
	if cfg == nil || len(cfg.Targets) == 0 {
		return nil
	}

	group := &chain.NodeGroup{}
	for _, target := range cfg.Targets {
		if v := strings.TrimSpace(target); v != "" {
			group.AddNode(chain.NewNode(target, target))
		}
	}
	return group.WithSelector(selectorFromConfig(cfg.Selector))
}

func logFromConfig(cfg *config.LogConfig) logger.Logger {
	if cfg == nil {
		cfg = &config.LogConfig{}
	}
	opts := []logger.LoggerOption{
		logger.FormatLoggerOption(logger.LogFormat(cfg.Format)),
		logger.LevelLoggerOption(logger.LogLevel(cfg.Level)),
	}

	var out io.Writer = os.Stderr
	switch cfg.Output {
	case "none":
		return logger.Nop()
	case "stdout", "":
		out = os.Stdout
	case "stderr":
		out = os.Stderr
	default:
		f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Warnf("log", err)
		} else {
			out = f
		}
	}
	opts = append(opts, logger.OutputLoggerOption(out))

	return logger.NewLogger(opts...)
}

func selectorFromConfig(cfg *config.SelectorConfig) chain.Selector {
	if cfg == nil {
		return nil
	}

	var strategy chain.Strategy
	switch cfg.Strategy {
	case "round":
		strategy = chain.RoundRobinStrategy()
	case "random":
		strategy = chain.RandomStrategy()
	case "fifo":
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

func bypassFromConfig(cfg *config.BypassConfig) bypass.Bypass {
	if cfg == nil {
		return nil
	}
	return bypass.NewBypassPatterns(cfg.Reverse, cfg.Matchers...)
}

func resolverFromConfig(cfg *config.ResolverConfig) (resolver.Resolver, error) {
	if cfg == nil {
		return nil, nil
	}
	var nameservers []resolver_impl.NameServer
	for _, server := range cfg.Nameservers {
		nameservers = append(nameservers, resolver_impl.NameServer{
			Addr:     server.Addr,
			Chain:    chains[server.Chain],
			TTL:      server.TTL,
			Timeout:  server.Timeout,
			ClientIP: net.ParseIP(server.ClientIP),
			Prefer:   server.Prefer,
			Hostname: server.Hostname,
		})
	}
	return resolver_impl.NewResolver(nameservers)
}

func hostsFromConfig(cfg *config.HostsConfig) hostspkg.HostMapper {
	if cfg == nil || len(cfg.Entries) == 0 {
		return nil
	}
	hosts := hostspkg.NewHosts()

	for _, host := range cfg.Entries {
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
