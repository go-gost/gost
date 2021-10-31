package main

import (
	"io"
	"os"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/components/connector"
	"github.com/go-gost/gost/pkg/components/dialer"
	"github.com/go-gost/gost/pkg/components/handler"
	"github.com/go-gost/gost/pkg/components/listener"
	"github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/service"
)

func buildService(cfg *config.Config) (services []*service.Service) {
	if cfg == nil || len(cfg.Services) == 0 {
		return
	}

	chains := buildChain(cfg)

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

		var chain *chain.Chain
		for _, ch := range chains {
			if svc.Chain == ch.Name {
				chain = ch
				break
			}
		}

		handlerLogger := log.WithFields(map[string]interface{}{
			"kind":    "handler",
			"type":    svc.Handler.Type,
			"service": svc.Name,
		})
		h := registry.GetHandler(svc.Handler.Type)(
			handler.ChainOption(chain),
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

func buildChain(cfg *config.Config) (chains []*chain.Chain) {
	if cfg == nil || len(cfg.Chains) == 0 {
		return nil
	}

	for _, ch := range cfg.Chains {
		c := &chain.Chain{
			Name: ch.Name,
		}

		selector := selectorFromConfig(ch.LB)
		for _, hop := range ch.Hops {
			group := &chain.NodeGroup{}
			for _, v := range hop.Nodes {
				node := chain.NewNode(v.Name, v.Addr)

				connectorLogger := log.WithFields(map[string]interface{}{
					"kind": "connector",
					"type": v.Connector.Type,
					"hop":  hop.Name,
					"node": node.Name(),
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
					"node": node.Name(),
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

				node.WithTransport(tr)
				group.AddNode(node)
			}

			sel := selector
			if s := selectorFromConfig(hop.LB); s != nil {
				sel = s
			}
			group.WithSelector(sel)
			c.AddNodeGroup(group)
		}

		chains = append(chains, c)
	}

	return
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
