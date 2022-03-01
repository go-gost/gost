package parsing

import (
	"strings"

	"github.com/go-gost/gost/pkg/chain"
	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/service"
)

func ParseService(cfg *config.ServiceConfig) (service.Service, error) {
	if cfg.Listener == nil {
		cfg.Listener = &config.ListenerConfig{
			Type: "tcp",
		}
	}
	if cfg.Handler == nil {
		cfg.Handler = &config.HandlerConfig{
			Type: "auto",
		}
	}
	serviceLogger := logger.Default().WithFields(map[string]any{
		"kind":     "service",
		"service":  cfg.Name,
		"listener": cfg.Listener.Type,
		"handler":  cfg.Handler.Type,
	})

	listenerLogger := serviceLogger.WithFields(map[string]any{
		"kind": "listener",
	})

	tlsCfg := cfg.Listener.TLS
	if tlsCfg == nil {
		tlsCfg = &config.TLSConfig{}
	}
	tlsConfig, err := tls_util.LoadServerConfig(
		tlsCfg.CertFile, tlsCfg.KeyFile, tlsCfg.CAFile)
	if err != nil {
		listenerLogger.Error(err)
		return nil, err
	}

	auther := ParseAutherFromAuth(cfg.Listener.Auth)
	if cfg.Listener.Auther != "" {
		auther = registry.AutherRegistry().Get(cfg.Listener.Auther)
	}

	ln := registry.ListenerRegistry().Get(cfg.Listener.Type)(
		listener.AddrOption(cfg.Addr),
		listener.ChainOption(registry.ChainRegistry().Get(cfg.Listener.Chain)),
		listener.AutherOption(auther),
		listener.AuthOption(parseAuth(cfg.Listener.Auth)),
		listener.TLSConfigOption(tlsConfig),
		listener.LoggerOption(listenerLogger),
	)

	if cfg.Listener.Metadata == nil {
		cfg.Listener.Metadata = make(map[string]any)
	}
	if err := ln.Init(metadata.MapMetadata(cfg.Listener.Metadata)); err != nil {
		listenerLogger.Error("init: ", err)
		return nil, err
	}

	handlerLogger := serviceLogger.WithFields(map[string]any{
		"kind": "handler",
	})

	tlsCfg = cfg.Handler.TLS
	if tlsCfg == nil {
		tlsCfg = &config.TLSConfig{}
	}
	tlsConfig, err = tls_util.LoadServerConfig(
		tlsCfg.CertFile, tlsCfg.KeyFile, tlsCfg.CAFile)
	if err != nil {
		handlerLogger.Error(err)
		return nil, err
	}

	auther = ParseAutherFromAuth(cfg.Handler.Auth)
	if cfg.Handler.Auther != "" {
		auther = registry.AutherRegistry().Get(cfg.Handler.Auther)
	}

	router := (&chain.Router{}).
		WithRetries(cfg.Handler.Retries).
		// WithTimeout(timeout time.Duration).
		WithInterface(cfg.Interface).
		WithChain(registry.ChainRegistry().Get(cfg.Handler.Chain)).
		WithResolver(registry.ResolverRegistry().Get(cfg.Resolver)).
		WithHosts(registry.HostsRegistry().Get(cfg.Hosts)).
		WithLogger(handlerLogger)

	h := registry.HandlerRegistry().Get(cfg.Handler.Type)(
		handler.RouterOption(router),
		handler.AutherOption(auther),
		handler.AuthOption(parseAuth(cfg.Handler.Auth)),
		handler.BypassOption(registry.BypassRegistry().Get(cfg.Bypass)),
		handler.TLSConfigOption(tlsConfig),
		handler.LoggerOption(handlerLogger),
	)

	if forwarder, ok := h.(handler.Forwarder); ok {
		forwarder.Forward(parseForwarder(cfg.Forwarder))
	}

	if cfg.Handler.Metadata == nil {
		cfg.Handler.Metadata = make(map[string]any)
	}
	if err := h.Init(metadata.MapMetadata(cfg.Handler.Metadata)); err != nil {
		handlerLogger.Error("init: ", err)
		return nil, err
	}

	s := service.NewService(ln, h,
		service.AdmissionOption(registry.AdmissionRegistry().Get(cfg.Admission)),
		service.LoggerOption(serviceLogger),
	)

	serviceLogger.Infof("listening on %s/%s", s.Addr().String(), s.Addr().Network())
	return s, nil
}

func parseForwarder(cfg *config.ForwarderConfig) *chain.NodeGroup {
	if cfg == nil || len(cfg.Targets) == 0 {
		return nil
	}

	group := &chain.NodeGroup{}
	for _, target := range cfg.Targets {
		if v := strings.TrimSpace(target); v != "" {
			group.AddNode(&chain.Node{
				Name:   target,
				Addr:   target,
				Marker: &chain.FailMarker{},
			})
		}
	}
	return group.WithSelector(parseSelector(cfg.Selector))
}
