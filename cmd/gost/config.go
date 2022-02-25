package main

import (
	"io"
	"os"

	"github.com/go-gost/gost/pkg/api"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/config/parsing"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/metrics"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/service"
)

func buildService(cfg *config.Config) (services []service.Service) {
	if cfg == nil {
		return
	}

	for _, autherCfg := range cfg.Authers {
		if auther := parsing.ParseAuther(autherCfg); auther != nil {
			if err := registry.Auther().Register(autherCfg.Name, auther); err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, admissionCfg := range cfg.Admissions {
		if adm := parsing.ParseAdmission(admissionCfg); adm != nil {
			if err := registry.Admission().Register(admissionCfg.Name, adm); err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, bypassCfg := range cfg.Bypasses {
		if bp := parsing.ParseBypass(bypassCfg); bp != nil {
			if err := registry.Bypass().Register(bypassCfg.Name, bp); err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, resolverCfg := range cfg.Resolvers {
		r, err := parsing.ParseResolver(resolverCfg)
		if err != nil {
			log.Fatal(err)
		}
		if r != nil {
			if err := registry.Resolver().Register(resolverCfg.Name, r); err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, hostsCfg := range cfg.Hosts {
		if h := parsing.ParseHosts(hostsCfg); h != nil {
			if err := registry.Hosts().Register(hostsCfg.Name, h); err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, chainCfg := range cfg.Chains {
		c, err := parsing.ParseChain(chainCfg)
		if err != nil {
			log.Fatal(err)
		}
		if c != nil {
			if err := registry.Chain().Register(chainCfg.Name, c); err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, svcCfg := range cfg.Services {
		svc, err := parsing.ParseService(svcCfg)
		if err != nil {
			log.Fatal(err)
		}
		if svc != nil {
			if err := registry.Service().Register(svcCfg.Name, svc); err != nil {
				log.Fatal(err)
			}
		}
		services = append(services, svc)
	}

	return
}

func logFromConfig(cfg *config.LogConfig) logger.Logger {
	if cfg == nil {
		cfg = &config.LogConfig{}
	}
	opts := []logger.LoggerOption{
		logger.FormatLoggerOption(logger.LogFormat(cfg.Format)),
		logger.LevelLoggerOption(logger.LogLevel(cfg.Level)),
	}

	var out io.Writer = os.Stderr
	switch cfg.Output {
	case "none", "null":
		return logger.Nop()
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

func buildAPIService(cfg *config.APIConfig) (service.Service, error) {
	auther := parsing.ParseAutherFromAuth(cfg.Auth)
	if cfg.Auther != "" {
		auther = registry.Auther().Get(cfg.Auther)
	}
	return api.NewService(
		cfg.Addr,
		api.PathPrefixOption(cfg.PathPrefix),
		api.AccessLogOption(cfg.AccessLog),
		api.AutherOption(auther),
	)
}

func buildMetricsService(cfg *config.MetricsConfig) (service.Service, error) {
	return metrics.NewService(
		cfg.Addr,
		metrics.PathOption(cfg.Path),
	)
}
