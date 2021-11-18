package main

import (
	"net/url"
	"strings"

	"github.com/go-gost/gost/pkg/config"
)

// normConfig normalizes the config.
func normConfig(cfg *config.Config) {
	for _, svc := range cfg.Services {
		normService(svc)
	}
	for _, chain := range cfg.Chains {
		normChain(chain)
	}
}

func normService(svc *config.ServiceConfig) {
	if svc.URL == "" {
		return
	}

	u, _ := url.Parse(svc.URL)

	var handler, listener string
	schemes := strings.Split(u.Scheme, "+")
	if len(schemes) == 1 {
		handler = schemes[0]
		listener = schemes[0]
	}
	if len(schemes) == 2 {
		handler = schemes[0]
		listener = schemes[1]
	}

	md := make(map[string]interface{})
	for k, v := range u.Query() {
		if len(v) > 0 {
			md[k] = v[0]
		}
	}

	svc.Addr = u.Host
	svc.Handler = &config.HandlerConfig{
		Type:     handler,
		Metadata: md,
	}
	svc.Listener = &config.ListenerConfig{
		Type:     listener,
		Metadata: md,
	}

	if remotes := strings.Trim(u.EscapedPath(), "/"); remotes != "" {
		svc.Forwarder = &config.ForwarderConfig{
			Targets: strings.Split(remotes, ","),
		}
	}
}

func normChain(chain *config.ChainConfig) {
	for _, hop := range chain.Hops {
		for _, node := range hop.Nodes {
			if node.URL == "" {
				continue
			}

			u, _ := url.Parse(node.URL)

			var connector, dialer string
			schemes := strings.Split(u.Scheme, "+")
			if len(schemes) == 1 {
				connector = schemes[0]
				dialer = schemes[0]
			}
			if len(schemes) == 2 {
				connector = schemes[0]
				dialer = schemes[1]
			}

			md := make(map[string]interface{})
			for k, v := range u.Query() {
				if len(v) > 0 {
					md[k] = v[0]
				}
			}

			node.Addr = u.Host
			node.Connector = &config.ConnectorConfig{
				Type:     connector,
				Metadata: md,
			}
			node.Dialer = &config.DialerConfig{
				Type:     dialer,
				Metadata: md,
			}
		}
	}
}
