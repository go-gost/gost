package main

import (
	"io"
	"os"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/components/connector"
	"github.com/go-gost/gost/pkg/components/dialer"
	"github.com/go-gost/gost/pkg/components/handler"
	"github.com/go-gost/gost/pkg/components/listener"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/service"
)

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

func buildService(cfg *config.Config) (services []*service.Service) {
	if cfg == nil || len(cfg.Services) == 0 {
		return
	}

	chains := buildChain(cfg)

	for _, svc := range cfg.Services {
		s := &service.Service{}

		ln := registry.GetListener(svc.Listener.Type)(
			listener.AddrOption(svc.Addr),
			listener.LoggerOption(
				log.WithFields(map[string]interface{}{
					"kind": "listener",
					"type": svc.Listener.Type,
				}),
			),
		)
		ln.Init(listener.Metadata(svc.Listener.Metadata))
		s.WithListener(ln)

		var chain *chain.Chain
		for _, ch := range chains {
			if svc.Chain == ch.Name {
				chain = ch
				break
			}
		}
		h := registry.GetHandler(svc.Handler.Type)(
			handler.ChainOption(chain),
			handler.LoggerOption(
				log.WithFields(map[string]interface{}{
					"kind": "handler",
					"type": svc.Handler.Type,
				}),
			),
		)
		h.Init(handler.Metadata(svc.Handler.Metadata))
		s.WithHandler(h)

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
		for _, hop := range ch.Hops {
			group := &chain.NodeGroup{}
			for _, v := range hop.Nodes {
				node := chain.NewNode(v.Name, v.Addr)

				tr := &chain.Transport{}

				cr := registry.GetConnector(v.Connector.Type)(
					connector.LoggerOption(
						log.WithFields(map[string]interface{}{
							"kind": "connector",
							"type": v.Connector.Type,
						}),
					),
				)
				cr.Init(connector.Metadata(v.Connector.Metadata))
				tr.WithConnector(cr)

				d := registry.GetDialer(v.Dialer.Type)(
					dialer.LoggerOption(
						log.WithFields(map[string]interface{}{
							"kind": "dialer",
							"type": v.Dialer.Type,
						}),
					),
				)
				d.Init(dialer.Metadata(v.Dialer.Metadata))
				tr.WithDialer(d)

				node.WithTransport(tr)

				group.AddNode(node)
			}
			c.AddNodeGroup(group)
		}

		chains = append(chains, c)
	}

	return
}
