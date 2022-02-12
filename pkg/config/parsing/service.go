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

func ParseService(cfg *config.ServiceConfig) (service.Servicer, error) {
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
	serviceLogger := logger.Default().WithFields(map[string]interface{}{
		"kind":     "service",
		"service":  cfg.Name,
		"listener": cfg.Listener.Type,
		"handler":  cfg.Handler.Type,
	})

	listenerLogger := serviceLogger.WithFields(map[string]interface{}{
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
		auther = registry.Auther().Get(cfg.Listener.Auther)
	}

	ln := registry.GetListener(cfg.Listener.Type)(
		listener.AddrOption(cfg.Addr),
		listener.ChainOption(registry.Chain().Get(cfg.Listener.Chain)),
		listener.AutherOption(auther),
		listener.AuthOption(parseAuth(cfg.Listener.Auth)),
		listener.TLSConfigOption(tlsConfig),
		listener.LoggerOption(listenerLogger),
	)

	if cfg.Listener.Metadata == nil {
		cfg.Listener.Metadata = make(map[string]interface{})
	}
	if err := ln.Init(metadata.MapMetadata(cfg.Listener.Metadata)); err != nil {
		listenerLogger.Error("init: ", err)
		return nil, err
	}

	handlerLogger := serviceLogger.WithFields(map[string]interface{}{
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
		auther = registry.Auther().Get(cfg.Handler.Auther)
	}
	h := registry.GetHandler(cfg.Handler.Type)(
		handler.AutherOption(auther),
		handler.AuthOption(parseAuth(cfg.Handler.Auth)),
		handler.RetriesOption(cfg.Handler.Retries),
		handler.ChainOption(registry.Chain().Get(cfg.Handler.Chain)),
		handler.BypassOption(registry.Bypass().Get(cfg.Bypass)),
		handler.ResolverOption(registry.Resolver().Get(cfg.Resolver)),
		handler.HostsOption(registry.Hosts().Get(cfg.Hosts)),
		handler.TLSConfigOption(tlsConfig),
		handler.LoggerOption(handlerLogger),
	)

	if forwarder, ok := h.(handler.Forwarder); ok {
		forwarder.Forward(parseForwarder(cfg.Forwarder))
	}

	if cfg.Handler.Metadata == nil {
		cfg.Handler.Metadata = make(map[string]interface{})
	}
	if err := h.Init(metadata.MapMetadata(cfg.Handler.Metadata)); err != nil {
		handlerLogger.Error("init: ", err)
		return nil, err
	}

	s := (&service.Service{}).
		WithListener(ln).
		WithHandler(h).
		WithLogger(serviceLogger)

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
