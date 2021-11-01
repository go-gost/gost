package main

import (
	"io"
	"os"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/service"
)

var (
	chains   = make(map[string]*chain.Chain)
	bypasses = make(map[string]bypass.Bypass)
)

func buildService(cfg *config.Config) (services []*service.Service) {
	if cfg == nil || len(cfg.Services) == 0 {
		return
	}

	for _, bypassCfg := range cfg.Bypasses {
		bypasses[bypassCfg.Name] = bypassFromConfig(&bypassCfg)
	}

	for _, chainCfg := range cfg.Chains {
		chains[chainCfg.Name] = chainFromConfig(&chainCfg)
	}

	for _, svc := range cfg.Services {
		listenerLogger := log.WithFields(map[string]interface{}{
			"kind":    "listener",
			"type":    svc.Listener.Type,
			"service": svc.Name,
		})
		ln := registry.GetListener(svc.Listener.Type)(
			listener.AddrOption(svc.Addr),
			listener.LoggerOption(listenerLogger),
		)
		if err := ln.Init(metadata.MapMetadata(svc.Listener.Metadata)); err != nil {
			listenerLogger.Fatal("init:", err)
		}

		handlerLogger := log.WithFields(map[string]interface{}{
			"kind":    "handler",
			"type":    svc.Handler.Type,
			"service": svc.Name,
		})

		h := registry.GetHandler(svc.Handler.Type)(
			handler.ChainOption(chains[svc.Chain]),
			handler.BypassOption(bypasses[svc.Bypass]),
			handler.LoggerOption(handlerLogger),
		)
		if err := h.Init(metadata.MapMetadata(svc.Handler.Metadata)); err != nil {
			handlerLogger.Fatal("init:", err)
		}

		s := (&service.Service{}).
			WithListener(ln).
			WithHandler(h)
		services = append(services, s)
	}

	return
}

func chainFromConfig(cfg *config.ChainConfig) *chain.Chain {
	if cfg == nil {
		return nil
	}

	c := &chain.Chain{}

	selector := selectorFromConfig(cfg.LB)
	for _, hop := range cfg.Hops {
		group := &chain.NodeGroup{}
		for _, v := range hop.Nodes {

			connectorLogger := log.WithFields(map[string]interface{}{
				"kind": "connector",
				"type": v.Connector.Type,
				"hop":  hop.Name,
				"node": v.Name,
			})
			cr := registry.GetConnector(v.Connector.Type)(
				connector.LoggerOption(connectorLogger),
			)
			if err := cr.Init(metadata.MapMetadata(v.Connector.Metadata)); err != nil {
				connectorLogger.Fatal("init:", err)
			}

			dialerLogger := log.WithFields(map[string]interface{}{
				"kind": "dialer",
				"type": v.Dialer.Type,
				"hop":  hop.Name,
				"node": v.Name,
			})
			d := registry.GetDialer(v.Dialer.Type)(
				dialer.LoggerOption(dialerLogger),
			)
			if err := d.Init(metadata.MapMetadata(v.Dialer.Metadata)); err != nil {
				dialerLogger.Fatal("init:", err)
			}

			tr := (&chain.Transport{}).
				WithConnector(cr).
				WithDialer(d)

			node := chain.NewNode(v.Name, v.Addr).
				WithTransport(tr).
				WithBypass(bypasses[v.Bypass])
			group.AddNode(node)
		}

		sel := selector
		if s := selectorFromConfig(hop.LB); s != nil {
			sel = s
		}
		group.WithSelector(sel)
		c.AddNodeGroup(group)
	}

	return c
}

func logFromConfig(cfg *config.LogConfig) logger.Logger {
	opts := []logger.LoggerOption{
		logger.FormatLoggerOption(logger.LogFormat(cfg.Format)),
		logger.LevelLoggerOption(logger.LogLevel(cfg.Level)),
	}

	var out io.Writer = os.Stderr
	switch cfg.Output {
	case "stdout":
		out = os.Stdout
	case "stderr", "":
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

func selectorFromConfig(cfg *config.LoadbalancingConfig) chain.Selector {
	if cfg == nil {
		return nil
	}

	var strategy chain.Strategy
	switch cfg.Strategy {
	case "round":
		strategy = &chain.RoundRobinStrategy{}
	case "random":
		strategy = &chain.RandomStrategy{}
	case "fifio":
		strategy = &chain.FIFOStrategy{}
	default:
		strategy = &chain.RoundRobinStrategy{}
	}

	return chain.NewSelector(
		strategy,
		&chain.InvalidFilter{},
		&chain.FailFilter{
			MaxFails:    cfg.MaxFails,
			FailTimeout: cfg.FailTimeout,
		},
	)
}

func bypassFromConfig(cfg *config.BypassConfig) bypass.Bypass {
	if cfg == nil {
		return nil
	}

	return bypass.NewBypassPatterns(cfg.Reverse, cfg.Matchers...)
}
