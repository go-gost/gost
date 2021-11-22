package main

import (
	"net/url"
	"strings"

	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/registry"
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
	if u.User != nil {
		md["users"] = []interface{}{u.User.String()}
	}

	svc.Addr = u.Host

	if h := registry.GetHandler(handler); h == nil {
		handler = "auto"
	}
	if ln := registry.GetListener(listener); ln == nil {
		listener = "tcp"
		if handler == "ssu" {
			listener = "udp"
		}
	}

	if remotes := strings.Trim(u.EscapedPath(), "/"); remotes != "" {
		svc.Forwarder = &config.ForwarderConfig{
			Targets: strings.Split(remotes, ","),
		}
		if handler != "relay" {
			if listener == "tcp" || listener == "udp" ||
				listener == "rtcp" || listener == "rudp" {
				handler = listener
			} else {
				handler = "tcp"
			}
		}
	}

	svc.Handler = &config.HandlerConfig{
		Type:     handler,
		Metadata: md,
	}
	svc.Listener = &config.ListenerConfig{
		Type:     listener,
		Metadata: md,
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
			if u.User != nil {
				md["user"] = []interface{}{u.User.String()}
			}

			node.Addr = u.Host

			if c := registry.GetConnector(connector); c == nil {
				connector = "http"
			}
			if d := registry.GetDialer(dialer); d == nil {
				dialer = "tcp"
				if connector == "ssu" {
					dialer = "udp"
				}
			}

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
