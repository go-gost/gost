package main

import (
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/components/connector"
	"github.com/go-gost/gost/pkg/components/dialer"
	"github.com/go-gost/gost/pkg/components/handler"
	"github.com/go-gost/gost/pkg/components/listener"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/service"
)

func buildService(cfg *config.Config) (services []*service.Service) {
	if cfg == nil || len(cfg.Services) == 0 {
		return
	}

	chains := buildChain(cfg)

	for _, svc := range cfg.Services {
		s := &service.Service{}

		ln := registry.GetListener(svc.Listener.Type)(listener.AddrOption(svc.Addr))
		ln.Init(listener.Metadata(svc.Listener.Metadata))
		s.WithListener(ln)

		var chain *chain.Chain
		for _, ch := range chains {
			if svc.Chain == ch.Name {
				chain = ch
				break
			}
		}
		h := registry.GetHandler(svc.Handler.Type)(handler.ChainOption(chain))
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

				cr := registry.GetConnector(v.Connector.Type)()
				cr.Init(connector.Metadata(v.Connector.Metadata))
				tr.WithConnector(cr)

				d := registry.GetDialer(v.Dialer.Type)()
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
