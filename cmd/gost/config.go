package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/go-gost/core/auth"
	"github.com/go-gost/core/logger"
	"github.com/go-gost/core/service"
	"github.com/go-gost/x/api"
	xauth "github.com/go-gost/x/auth"
	"github.com/go-gost/x/config"
	"github.com/go-gost/x/config/cmd"
	admission_parser "github.com/go-gost/x/config/parsing/admission"
	auth_parser "github.com/go-gost/x/config/parsing/auth"
	bypass_parser "github.com/go-gost/x/config/parsing/bypass"
	chain_parser "github.com/go-gost/x/config/parsing/chain"
	hop_parser "github.com/go-gost/x/config/parsing/hop"
	hosts_parser "github.com/go-gost/x/config/parsing/hosts"
	ingress_parser "github.com/go-gost/x/config/parsing/ingress"
	limiter_parser "github.com/go-gost/x/config/parsing/limiter"
	logger_parser "github.com/go-gost/x/config/parsing/logger"
	observer_parser "github.com/go-gost/x/config/parsing/observer"
	recorder_parser "github.com/go-gost/x/config/parsing/recorder"
	resolver_parser "github.com/go-gost/x/config/parsing/resolver"
	router_parser "github.com/go-gost/x/config/parsing/router"
	sd_parser "github.com/go-gost/x/config/parsing/sd"
	service_parser "github.com/go-gost/x/config/parsing/service"
	xmd "github.com/go-gost/x/metadata"
	mdutil "github.com/go-gost/x/metadata/util"
	metrics "github.com/go-gost/x/metrics/service"
	"github.com/go-gost/x/registry"
)

func parseConfig() (*config.Config, error) {
	cfg := &config.Config{}
	if cfgFile != "" {
		cfgFile = strings.TrimSpace(cfgFile)
		if strings.HasPrefix(cfgFile, "{") && strings.HasSuffix(cfgFile, "}") {
			if err := json.Unmarshal([]byte(cfgFile), cfg); err != nil {
				return nil, err
			}
		} else {
			if err := cfg.ReadFile(cfgFile); err != nil {
				return nil, err
			}
		}
	}

	cmdCfg, err := cmd.BuildConfigFromCmd(services, nodes)
	if err != nil {
		return nil, err
	}
	cfg = mergeConfig(cfg, cmdCfg)

	if len(cfg.Services) == 0 && apiAddr == "" && cfg.API == nil {
		if err := cfg.Load(); err != nil {
			return nil, err
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
	if v := os.Getenv("GOST_METRICS"); v != "" {
		cfg.Metrics = &config.MetricsConfig{
			Addr: v,
		}
	}
	if v := os.Getenv("GOST_PROFILING"); v != "" {
		cfg.Profiling = &config.ProfilingConfig{
			Addr: v,
		}
	}

	if debug || trace {
		if cfg.Log == nil {
			cfg.Log = &config.LogConfig{}
		}

		cfg.Log.Level = string(logger.DebugLevel)
		if trace {
			cfg.Log.Level = string(logger.TraceLevel)
		}
	}

	if apiAddr != "" {
		cfg.API = &config.APIConfig{
			Addr: apiAddr,
		}
		if url, _ := cmd.Norm(apiAddr); url != nil {
			cfg.API.Addr = url.Host
			if url.User != nil {
				username := url.User.Username()
				password, _ := url.User.Password()
				cfg.API.Auth = &config.AuthConfig{
					Username: username,
					Password: password,
				}
			}
			m := map[string]any{}
			for k, v := range url.Query() {
				if len(v) > 0 {
					m[k] = v[0]
				}
			}
			md := xmd.NewMetadata(m)
			cfg.API.PathPrefix = mdutil.GetString(md, "pathPrefix")
			cfg.API.AccessLog = mdutil.GetBool(md, "accesslog")
		}
	}
	if metricsAddr != "" {
		cfg.Metrics = &config.MetricsConfig{
			Addr: metricsAddr,
		}
		if url, _ := cmd.Norm(metricsAddr); url != nil {
			cfg.Metrics.Addr = url.Host
			if url.User != nil {
				username := url.User.Username()
				password, _ := url.User.Password()
				cfg.Metrics.Auth = &config.AuthConfig{
					Username: username,
					Password: password,
				}
			}
			m := map[string]any{}
			for k, v := range url.Query() {
				if len(v) > 0 {
					m[k] = v[0]
				}
			}
			md := xmd.NewMetadata(m)
			cfg.Metrics.Path = mdutil.GetString(md, "path")
		}
	}

	return cfg, nil
}

func mergeConfig(cfg1, cfg2 *config.Config) *config.Config {
	if cfg1 == nil {
		return cfg2
	}
	if cfg2 == nil {
		return cfg1
	}

	cfg := &config.Config{
		Services:   append(cfg1.Services, cfg2.Services...),
		Chains:     append(cfg1.Chains, cfg2.Chains...),
		Hops:       append(cfg1.Hops, cfg2.Hops...),
		Authers:    append(cfg1.Authers, cfg2.Authers...),
		Admissions: append(cfg1.Admissions, cfg2.Admissions...),
		Bypasses:   append(cfg1.Bypasses, cfg2.Bypasses...),
		Resolvers:  append(cfg1.Resolvers, cfg2.Resolvers...),
		Hosts:      append(cfg1.Hosts, cfg2.Hosts...),
		Ingresses:  append(cfg1.Ingresses, cfg2.Ingresses...),
		SDs:        append(cfg1.SDs, cfg2.SDs...),
		Recorders:  append(cfg1.Recorders, cfg2.Recorders...),
		Limiters:   append(cfg1.Limiters, cfg2.Limiters...),
		CLimiters:  append(cfg1.CLimiters, cfg2.CLimiters...),
		RLimiters:  append(cfg1.RLimiters, cfg2.RLimiters...),
		Loggers:    append(cfg1.Loggers, cfg2.Loggers...),
		Routers:    append(cfg1.Routers, cfg2.Routers...),
		Observers:  append(cfg1.Observers, cfg2.Observers...),
		TLS:        cfg1.TLS,
		Log:        cfg1.Log,
		API:        cfg1.API,
		Metrics:    cfg1.Metrics,
		Profiling:  cfg1.Profiling,
	}
	if cfg2.TLS != nil {
		cfg.TLS = cfg2.TLS
	}
	if cfg2.Log != nil {
		cfg.Log = cfg2.Log
	}
	if cfg2.API != nil {
		cfg.API = cfg2.API
	}
	if cfg2.Metrics != nil {
		cfg.Metrics = cfg2.Metrics
	}
	if cfg2.Profiling != nil {
		cfg.Profiling = cfg2.Profiling
	}

	return cfg
}

func register(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	for name := range registry.LoggerRegistry().GetAll() {
		registry.LoggerRegistry().Unregister(name)
	}
	for _, loggerCfg := range cfg.Loggers {
		if err := registry.LoggerRegistry().Register(loggerCfg.Name, logger_parser.ParseLogger(loggerCfg)); err != nil {
			return err
		}
	}

	for name := range registry.AutherRegistry().GetAll() {
		registry.AutherRegistry().Unregister(name)
	}
	for _, autherCfg := range cfg.Authers {
		if err := registry.AutherRegistry().Register(autherCfg.Name, auth_parser.ParseAuther(autherCfg)); err != nil {
			return err
		}
	}

	for name := range registry.AdmissionRegistry().GetAll() {
		registry.AdmissionRegistry().Unregister(name)
	}
	for _, admissionCfg := range cfg.Admissions {
		if err := registry.AdmissionRegistry().Register(admissionCfg.Name, admission_parser.ParseAdmission(admissionCfg)); err != nil {
			return err
		}
	}

	for name := range registry.BypassRegistry().GetAll() {
		registry.BypassRegistry().Unregister(name)
	}
	for _, bypassCfg := range cfg.Bypasses {
		if err := registry.BypassRegistry().Register(bypassCfg.Name, bypass_parser.ParseBypass(bypassCfg)); err != nil {
			return err
		}
	}

	for name := range registry.ResolverRegistry().GetAll() {
		registry.ResolverRegistry().Unregister(name)
	}
	for _, resolverCfg := range cfg.Resolvers {
		r, err := resolver_parser.ParseResolver(resolverCfg)
		if err != nil {
			return err
		}
		if err := registry.ResolverRegistry().Register(resolverCfg.Name, r); err != nil {
			return err
		}
	}

	for name := range registry.HostsRegistry().GetAll() {
		registry.HostsRegistry().Unregister(name)
	}
	for _, hostsCfg := range cfg.Hosts {
		if err := registry.HostsRegistry().Register(hostsCfg.Name, hosts_parser.ParseHostMapper(hostsCfg)); err != nil {
			return err
		}
	}

	for name := range registry.IngressRegistry().GetAll() {
		registry.IngressRegistry().Unregister(name)
	}
	for _, ingressCfg := range cfg.Ingresses {
		if err := registry.IngressRegistry().Register(ingressCfg.Name, ingress_parser.ParseIngress(ingressCfg)); err != nil {
			return err
		}
	}

	for name := range registry.RouterRegistry().GetAll() {
		registry.RouterRegistry().Unregister(name)
	}
	for _, routerCfg := range cfg.Routers {
		if err := registry.RouterRegistry().Register(routerCfg.Name, router_parser.ParseRouter(routerCfg)); err != nil {
			return err
		}
	}

	for name := range registry.SDRegistry().GetAll() {
		registry.SDRegistry().Unregister(name)
	}
	for _, sdCfg := range cfg.SDs {
		if err := registry.SDRegistry().Register(sdCfg.Name, sd_parser.ParseSD(sdCfg)); err != nil {
			return err
		}
	}

	for name := range registry.ObserverRegistry().GetAll() {
		registry.ObserverRegistry().Unregister(name)
	}
	for _, observerCfg := range cfg.Observers {
		if err := registry.ObserverRegistry().Register(observerCfg.Name, observer_parser.ParseObserver(observerCfg)); err != nil {
			return err
		}
	}

	for name := range registry.RecorderRegistry().GetAll() {
		registry.RecorderRegistry().Unregister(name)
	}
	for _, recorderCfg := range cfg.Recorders {
		if err := registry.RecorderRegistry().Register(recorderCfg.Name, recorder_parser.ParseRecorder(recorderCfg)); err != nil {
			return err
		}
	}

	for name := range registry.TrafficLimiterRegistry().GetAll() {
		registry.TrafficLimiterRegistry().Unregister(name)
	}
	for _, limiterCfg := range cfg.Limiters {
		if err := registry.TrafficLimiterRegistry().Register(limiterCfg.Name, limiter_parser.ParseTrafficLimiter(limiterCfg)); err != nil {
			return err
		}
	}

	for name := range registry.ConnLimiterRegistry().GetAll() {
		registry.ConnLimiterRegistry().Unregister(name)
	}
	for _, limiterCfg := range cfg.CLimiters {
		if err := registry.ConnLimiterRegistry().Register(limiterCfg.Name, limiter_parser.ParseConnLimiter(limiterCfg)); err != nil {
			return err
		}
	}

	for name := range registry.RateLimiterRegistry().GetAll() {
		registry.RateLimiterRegistry().Unregister(name)
	}
	for _, limiterCfg := range cfg.RLimiters {
		if err := registry.RateLimiterRegistry().Register(limiterCfg.Name, limiter_parser.ParseRateLimiter(limiterCfg)); err != nil {
			return err
		}
	}

	for name := range registry.HopRegistry().GetAll() {
		registry.HopRegistry().Unregister(name)
	}
	for _, hopCfg := range cfg.Hops {
		hop, err := hop_parser.ParseHop(hopCfg, logger.Default())
		if err != nil {
			return err
		}
		if err := registry.HopRegistry().Register(hopCfg.Name, hop); err != nil {
			return err
		}
	}

	for name := range registry.ChainRegistry().GetAll() {
		registry.ChainRegistry().Unregister(name)
	}
	for _, chainCfg := range cfg.Chains {
		c, err := chain_parser.ParseChain(chainCfg, logger.Default())
		if err != nil {
			return err
		}
		if err := registry.ChainRegistry().Register(chainCfg.Name, c); err != nil {
			return err
		}
	}

	for name := range registry.ServiceRegistry().GetAll() {
		registry.ServiceRegistry().Unregister(name)
	}
	for _, svcCfg := range cfg.Services {
		svc, err := service_parser.ParseService(svcCfg)
		if err != nil {
			return err
		}
		if svc != nil {
			if err := registry.ServiceRegistry().Register(svcCfg.Name, svc); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildAPIService(cfg *config.APIConfig) (service.Service, error) {
	var authers []auth.Authenticator
	if auther := auth_parser.ParseAutherFromAuth(cfg.Auth); auther != nil {
		authers = append(authers, auther)
	}
	if cfg.Auther != "" {
		authers = append(authers, registry.AutherRegistry().Get(cfg.Auther))
	}

	var auther auth.Authenticator
	if len(authers) > 0 {
		auther = xauth.AuthenticatorGroup(authers...)
	}

	network := "tcp"
	addr := cfg.Addr
	if strings.HasPrefix(addr, "unix://") {
		network = "unix"
		addr = strings.TrimPrefix(addr, "unix://")
	}
	return api.NewService(
		network, addr,
		api.PathPrefixOption(cfg.PathPrefix),
		api.AccessLogOption(cfg.AccessLog),
		api.AutherOption(auther),
	)
}

func buildMetricsService(cfg *config.MetricsConfig) (service.Service, error) {
	auther := auth_parser.ParseAutherFromAuth(cfg.Auth)
	if cfg.Auther != "" {
		auther = registry.AutherRegistry().Get(cfg.Auther)
	}

	network := "tcp"
	addr := cfg.Addr
	if strings.HasPrefix(addr, "unix://") {
		network = "unix"
		addr = strings.TrimPrefix(addr, "unix://")
	}
	return metrics.NewService(
		network, addr,
		metrics.PathOption(cfg.Path),
		metrics.AutherOption(auther),
	)
}
