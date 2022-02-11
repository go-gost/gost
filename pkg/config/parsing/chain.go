package parsing

import (
	"github.com/go-gost/gost/pkg/chain"
	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func ParseChain(cfg *config.ChainConfig) (chain.Chainer, error) {
	if cfg == nil {
		return nil, nil
	}

	chainLogger := logger.Default().WithFields(map[string]interface{}{
		"kind":  "chain",
		"chain": cfg.Name,
	})

	c := &chain.Chain{}
	selector := parseSelector(cfg.Selector)
	for _, hop := range cfg.Hops {
		group := &chain.NodeGroup{}
		for _, v := range hop.Nodes {
			nodeLogger := chainLogger.WithFields(map[string]interface{}{
				"kind":      "node",
				"connector": v.Connector.Type,
				"dialer":    v.Dialer.Type,
				"hop":       hop.Name,
				"node":      v.Name,
			})
			connectorLogger := nodeLogger.WithFields(map[string]interface{}{
				"kind": "connector",
			})

			tlsCfg := v.Connector.TLS
			if tlsCfg == nil {
				tlsCfg = &config.TLSConfig{}
			}
			tlsConfig, err := tls_util.LoadClientConfig(
				tlsCfg.CertFile, tlsCfg.KeyFile, tlsCfg.CAFile,
				tlsCfg.Secure, tlsCfg.ServerName)
			if err != nil {
				chainLogger.Error(err)
				return nil, err
			}

			cr := registry.GetConnector(v.Connector.Type)(
				connector.AuthOption(parseAuth(v.Connector.Auth)),
				connector.TLSConfigOption(tlsConfig),
				connector.LoggerOption(connectorLogger),
			)

			if v.Connector.Metadata == nil {
				v.Connector.Metadata = make(map[string]interface{})
			}
			if err := cr.Init(metadata.MapMetadata(v.Connector.Metadata)); err != nil {
				connectorLogger.Error("init: ", err)
				return nil, err
			}

			dialerLogger := nodeLogger.WithFields(map[string]interface{}{
				"kind": "dialer",
			})

			tlsCfg = v.Dialer.TLS
			if tlsCfg == nil {
				tlsCfg = &config.TLSConfig{}
			}
			tlsConfig, err = tls_util.LoadClientConfig(
				tlsCfg.CertFile, tlsCfg.KeyFile, tlsCfg.CAFile,
				tlsCfg.Secure, tlsCfg.ServerName)
			if err != nil {
				chainLogger.Error(err)
				return nil, err
			}

			d := registry.GetDialer(v.Dialer.Type)(
				dialer.AuthOption(parseAuth(v.Dialer.Auth)),
				dialer.TLSConfigOption(tlsConfig),
				dialer.LoggerOption(dialerLogger),
			)

			if v.Dialer.Metadata == nil {
				v.Dialer.Metadata = make(map[string]interface{})
			}
			if err := d.Init(metadata.MapMetadata(v.Dialer.Metadata)); err != nil {
				dialerLogger.Error("init: ", err)
				return nil, err
			}

			tr := (&chain.Transport{}).
				WithConnector(cr).
				WithDialer(d).
				WithAddr(v.Addr)

			if v.Bypass == "" {
				v.Bypass = hop.Bypass
			}
			if v.Resolver == "" {
				v.Resolver = hop.Resolver
			}
			if v.Hosts == "" {
				v.Hosts = hop.Hosts
			}

			node := &chain.Node{
				Name:      v.Name,
				Addr:      v.Addr,
				Transport: tr,
				Bypass:    registry.Bypass().Get(v.Bypass),
				Resolver:  registry.Resolver().Get(v.Resolver),
				Hosts:     registry.Hosts().Get(v.Hosts),
				Marker:    &chain.FailMarker{},
			}
			group.AddNode(node)
		}

		sel := selector
		if s := parseSelector(hop.Selector); s != nil {
			sel = s
		}
		group.WithSelector(sel)
		c.AddNodeGroup(group)
	}

	return c, nil
}
