package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-gost/core/metadata"
	"github.com/go-gost/core/registry"
	"github.com/go-gost/x/config"
)

var (
	ErrInvalidCmd  = errors.New("invalid cmd")
	ErrInvalidNode = errors.New("invalid node")
)

type stringList []string

func (l *stringList) String() string {
	return fmt.Sprintf("%s", *l)
}
func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func buildConfigFromCmd(services, nodes stringList) (*config.Config, error) {
	cfg := &config.Config{}

	if v := os.Getenv("GOST_PROFILING"); v != "" {
		cfg.Profiling = &config.ProfilingConfig{
			Addr: v,
		}
	}
	if v := os.Getenv("GOST_METRICS"); v != "" {
		cfg.Metrics = &config.MetricsConfig{
			Addr: v,
		}
	}

	if v := os.Getenv("GOST_LOGGER_LEVEL"); v != "" {
		cfg.Log = &config.LogConfig{
			Level: v,
		}
	}

	if v := os.Getenv("GOST_API"); v != "" {
		cfg.API = &config.APIConfig{
			Addr: v,
		}
	}

	var chain *config.ChainConfig
	if len(nodes) > 0 {
		chain = &config.ChainConfig{
			Name: "chain-0",
		}
		cfg.Chains = append(cfg.Chains, chain)
	}

	for i, node := range nodes {
		url, err := normCmd(node)
		if err != nil {
			return nil, err
		}

		nodeConfig, err := buildNodeConfig(url)
		if err != nil {
			return nil, err
		}
		nodeConfig.Name = "node-0"

		var nodes []*config.NodeConfig
		for _, host := range strings.Split(nodeConfig.Addr, ",") {
			if host == "" {
				continue
			}
			nodeCfg := &config.NodeConfig{}
			*nodeCfg = *nodeConfig
			nodeCfg.Name = fmt.Sprintf("node-%d", len(nodes))
			nodeCfg.Addr = host
			nodes = append(nodes, nodeCfg)
		}

		md := metadata.MapMetadata(nodeConfig.Connector.Metadata)

		hopConfig := &config.HopConfig{
			Name:     fmt.Sprintf("hop-%d", i),
			Selector: parseSelector(md),
			Nodes:    nodes,
		}

		if v := metadata.GetString(md, "bypass"); v != "" {
			bypassCfg := &config.BypassConfig{
				Name: fmt.Sprintf("bypass-%d", len(cfg.Bypasses)),
			}
			if v[0] == '~' {
				bypassCfg.Reverse = true
				v = v[1:]
			}
			for _, s := range strings.Split(v, ",") {
				if s == "" {
					continue
				}
				bypassCfg.Matchers = append(bypassCfg.Matchers, s)
			}
			hopConfig.Bypass = bypassCfg.Name
			cfg.Bypasses = append(cfg.Bypasses, bypassCfg)
			md.Del("bypass")
		}
		if v := metadata.GetString(md, "resolver"); v != "" {
			resolverCfg := &config.ResolverConfig{
				Name: fmt.Sprintf("resolver-%d", len(cfg.Resolvers)),
			}
			for _, rs := range strings.Split(v, ",") {
				if rs == "" {
					continue
				}
				resolverCfg.Nameservers = append(
					resolverCfg.Nameservers,
					&config.NameserverConfig{
						Addr: rs,
					},
				)
			}
			hopConfig.Resolver = resolverCfg.Name
			cfg.Resolvers = append(cfg.Resolvers, resolverCfg)
			md.Del("resolver")
		}
		if v := metadata.GetString(md, "hosts"); v != "" {
			hostsCfg := &config.HostsConfig{
				Name: fmt.Sprintf("hosts-%d", len(cfg.Hosts)),
			}
			for _, s := range strings.Split(v, ",") {
				ss := strings.SplitN(s, ":", 2)
				if len(ss) != 2 {
					continue
				}
				hostsCfg.Mappings = append(
					hostsCfg.Mappings,
					&config.HostMappingConfig{
						Hostname: ss[0],
						IP:       ss[1],
					},
				)
			}
			hopConfig.Hosts = hostsCfg.Name
			cfg.Hosts = append(cfg.Hosts, hostsCfg)
			md.Del("hosts")
		}

		if v := metadata.GetString(md, "interface"); v != "" {
			hopConfig.Interface = v
			md.Del("interface")
		}
		if v := metadata.GetInt(md, "so_mark"); v > 0 {
			hopConfig.SockOpts = &config.SockOptsConfig{
				Mark: v,
			}
			md.Del("so_mark")
		}

		chain.Hops = append(chain.Hops, hopConfig)
	}

	for i, svc := range services {
		url, err := normCmd(svc)
		if err != nil {
			return nil, err
		}

		service, err := buildServiceConfig(url)
		if err != nil {
			return nil, err
		}
		service.Name = fmt.Sprintf("service-%d", i)
		if chain != nil {
			if service.Listener.Type == "rtcp" || service.Listener.Type == "rudp" {
				service.Listener.Chain = chain.Name
			} else {
				service.Handler.Chain = chain.Name
			}
		}
		cfg.Services = append(cfg.Services, service)

		md := metadata.MapMetadata(service.Handler.Metadata)
		if v := metadata.GetInt(md, "retries"); v > 0 {
			service.Handler.Retries = v
			md.Del("retries")
		}
		if v := metadata.GetString(md, "admission"); v != "" {
			admCfg := &config.AdmissionConfig{
				Name: fmt.Sprintf("admission-%d", len(cfg.Admissions)),
			}
			if v[0] == '~' {
				admCfg.Reverse = true
				v = v[1:]
			}
			for _, s := range strings.Split(v, ",") {
				if s == "" {
					continue
				}
				admCfg.Matchers = append(admCfg.Matchers, s)
			}
			service.Admission = admCfg.Name
			cfg.Admissions = append(cfg.Admissions, admCfg)
			md.Del("admission")
		}
		if v := metadata.GetString(md, "bypass"); v != "" {
			bypassCfg := &config.BypassConfig{
				Name: fmt.Sprintf("bypass-%d", len(cfg.Bypasses)),
			}
			if v[0] == '~' {
				bypassCfg.Reverse = true
				v = v[1:]
			}
			for _, s := range strings.Split(v, ",") {
				if s == "" {
					continue
				}
				bypassCfg.Matchers = append(bypassCfg.Matchers, s)
			}
			service.Bypass = bypassCfg.Name
			cfg.Bypasses = append(cfg.Bypasses, bypassCfg)
			md.Del("bypass")
		}
		if v := metadata.GetString(md, "resolver"); v != "" {
			resolverCfg := &config.ResolverConfig{
				Name: fmt.Sprintf("resolver-%d", len(cfg.Resolvers)),
			}
			for _, rs := range strings.Split(v, ",") {
				if rs == "" {
					continue
				}
				resolverCfg.Nameservers = append(
					resolverCfg.Nameservers,
					&config.NameserverConfig{
						Addr: rs,
					},
				)
			}
			service.Resolver = resolverCfg.Name
			cfg.Resolvers = append(cfg.Resolvers, resolverCfg)
			md.Del("resolver")
		}
		if v := metadata.GetString(md, "hosts"); v != "" {
			hostsCfg := &config.HostsConfig{
				Name: fmt.Sprintf("hosts-%d", len(cfg.Hosts)),
			}
			for _, s := range strings.Split(v, ",") {
				ss := strings.SplitN(s, ":", 2)
				if len(ss) != 2 {
					continue
				}
				hostsCfg.Mappings = append(
					hostsCfg.Mappings,
					&config.HostMappingConfig{
						Hostname: ss[0],
						IP:       ss[1],
					},
				)
			}
			service.Hosts = hostsCfg.Name
			cfg.Hosts = append(cfg.Hosts, hostsCfg)
			md.Del("hosts")
		}
	}

	return cfg, nil
}

func buildServiceConfig(url *url.URL) (*config.ServiceConfig, error) {
	var handler, listener string
	schemes := strings.Split(url.Scheme, "+")
	if len(schemes) == 1 {
		handler = schemes[0]
		listener = schemes[0]
	}
	if len(schemes) == 2 {
		handler = schemes[0]
		listener = schemes[1]
	}

	svc := &config.ServiceConfig{
		Addr: url.Host,
	}

	if h := registry.HandlerRegistry().Get(handler); h == nil {
		handler = "auto"
	}
	if ln := registry.ListenerRegistry().Get(listener); ln == nil {
		listener = "tcp"
		if handler == "ssu" {
			listener = "udp"
		}
	}

	// forward mode
	if remotes := strings.Trim(url.EscapedPath(), "/"); remotes != "" {
		svc.Forwarder = &config.ForwarderConfig{
			Targets: strings.Split(remotes, ","),
		}
		if handler != "relay" {
			if listener == "tcp" || listener == "udp" ||
				listener == "rtcp" || listener == "rudp" ||
				listener == "tun" || listener == "tap" {
				handler = listener
			} else {
				handler = "forward"
			}
		}
	}

	var auth *config.AuthConfig
	if url.User != nil {
		auth = &config.AuthConfig{
			Username: url.User.Username(),
		}
		auth.Password, _ = url.User.Password()
	}

	md := metadata.MapMetadata{}
	for k, v := range url.Query() {
		if len(v) > 0 {
			md[k] = v[0]
		}
	}

	if sa := metadata.GetString(md, "auth"); sa != "" {
		au, err := parseAuthFromCmd(sa)
		if err != nil {
			return nil, err
		}
		auth = au
	}
	md.Del("auth")

	tlsConfig := &config.TLSConfig{
		CertFile: metadata.GetString(md, "certFile"),
		KeyFile:  metadata.GetString(md, "keyFile"),
		CAFile:   metadata.GetString(md, "caFile"),
	}
	md.Del("certFile")
	md.Del("keyFile")
	md.Del("caFile")

	if tlsConfig.CertFile == "" {
		tlsConfig = nil
	}

	if v := metadata.GetString(md, "dns"); v != "" {
		md.Set("dns", strings.Split(v, ","))
	}
	if v := metadata.GetString(md, "interface"); v != "" {
		svc.Interface = v
		md.Del("interface")
	}
	if v := metadata.GetInt(md, "so_mark"); v > 0 {
		svc.SockOpts = &config.SockOptsConfig{
			Mark: v,
		}
		md.Del("so_mark")
	}

	if svc.Forwarder != nil {
		svc.Forwarder.Selector = parseSelector(md)
	}

	svc.Handler = &config.HandlerConfig{
		Type:     handler,
		Auth:     auth,
		Metadata: md,
	}
	svc.Listener = &config.ListenerConfig{
		Type:     listener,
		TLS:      tlsConfig,
		Metadata: md,
	}

	if svc.Handler.Type == "sshd" {
		svc.Handler.Auth = nil
	}
	if svc.Listener.Type == "sshd" {
		svc.Listener.Auth = auth
	}

	return svc, nil
}

func buildNodeConfig(url *url.URL) (*config.NodeConfig, error) {
	var connector, dialer string
	schemes := strings.Split(url.Scheme, "+")
	if len(schemes) == 1 {
		connector = schemes[0]
		dialer = schemes[0]
	}
	if len(schemes) == 2 {
		connector = schemes[0]
		dialer = schemes[1]
	}

	node := &config.NodeConfig{
		Addr: url.Host,
	}

	if c := registry.ConnectorRegistry().Get(connector); c == nil {
		connector = "http"
	}
	if d := registry.DialerRegistry().Get(dialer); d == nil {
		dialer = "tcp"
		if connector == "ssu" {
			dialer = "udp"
		}
	}

	var auth *config.AuthConfig
	if url.User != nil {
		auth = &config.AuthConfig{
			Username: url.User.Username(),
		}
		auth.Password, _ = url.User.Password()
	}

	md := metadata.MapMetadata{}
	for k, v := range url.Query() {
		if len(v) > 0 {
			md[k] = v[0]
		}
	}

	if sauth := metadata.GetString(md, "auth"); sauth != "" && auth == nil {
		au, err := parseAuthFromCmd(sauth)
		if err != nil {
			return nil, err
		}
		auth = au
	}
	md.Del("auth")

	tlsConfig := &config.TLSConfig{
		CertFile:   metadata.GetString(md, "certFile"),
		KeyFile:    metadata.GetString(md, "keyFile"),
		CAFile:     metadata.GetString(md, "caFile"),
		Secure:     metadata.GetBool(md, "secure"),
		ServerName: metadata.GetString(md, "serverName"),
	}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = url.Hostname()
	}
	md.Del("certFile")
	md.Del("keyFile")
	md.Del("caFile")
	md.Del("secure")
	md.Del("serverName")

	if !tlsConfig.Secure && tlsConfig.CertFile == "" && tlsConfig.CAFile == "" {
		tlsConfig = nil
	}

	node.Connector = &config.ConnectorConfig{
		Type:     connector,
		Auth:     auth,
		Metadata: md,
	}
	node.Dialer = &config.DialerConfig{
		Type:     dialer,
		TLS:      tlsConfig,
		Metadata: md,
	}

	if node.Connector.Type == "sshd" {
		node.Connector.Auth = nil
	}
	if node.Dialer.Type == "sshd" {
		node.Dialer.Auth = auth
	}

	return node, nil
}

func normCmd(s string) (*url.URL, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrInvalidCmd
	}

	if s[0] == ':' || !strings.Contains(s, "://") {
		s = "auto://" + s
	}

	url, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "https" {
		url.Scheme = "http+tls"
	}

	return url, nil
}

func parseAuthFromCmd(sa string) (*config.AuthConfig, error) {
	v, err := base64.StdEncoding.DecodeString(sa)
	if err != nil {
		return nil, err
	}
	cs := string(v)
	n := strings.IndexByte(cs, ':')
	if n < 0 {
		return &config.AuthConfig{
			Username: cs,
		}, nil
	}

	return &config.AuthConfig{
		Username: cs[:n],
		Password: cs[n+1:],
	}, nil
}

func parseSelector(md metadata.MapMetadata) *config.SelectorConfig {
	strategy := metadata.GetString(md, "strategy")
	maxFails := metadata.GetInt(md, "maxFails")
	failTimeout := metadata.GetDuration(md, "failTimeout")
	if strategy == "" && maxFails <= 0 && failTimeout <= 0 {
		return nil
	}
	if strategy == "" {
		strategy = "round"
	}
	if maxFails <= 0 {
		maxFails = 1
	}
	if failTimeout <= 0 {
		failTimeout = time.Second
	}

	md.Del("strategy")
	md.Del("maxFails")
	md.Del("failTimeout")

	return &config.SelectorConfig{
		Strategy:    strategy,
		MaxFails:    maxFails,
		FailTimeout: failTimeout,
	}
}
