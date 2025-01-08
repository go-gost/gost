package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-gost/core/logger"
	"github.com/go-gost/core/service"
	"github.com/go-gost/x/config"
	"github.com/go-gost/x/config/parsing"
	logger_parser "github.com/go-gost/x/config/parsing/logger"
	xmetrics "github.com/go-gost/x/metrics"
	"github.com/go-gost/x/registry"
	"github.com/judwhite/go-svc"
)

type program struct {
	srvApi       service.Service
	srvMetric    service.Service
	srvProfiling *http.Server

	cancel context.CancelFunc
}

func (p *program) Init(env svc.Environment) error {
	cfg, err := parseConfig()
	if err != nil {
		return err
	}

	config.Set(cfg)

	return nil
}

func (p *program) Start() error {
	cfg := config.Global()

	if outputFormat != "" {
		if err := cfg.Write(os.Stdout, outputFormat); err != nil {
			return err
		}
		os.Exit(0)
	}

	if cfg.Metrics != nil {
		xmetrics.Init(xmetrics.NewMetrics())
	}

	if err := p.build(cfg); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go p.reload(ctx)

	return nil
}

func (p *program) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}

	for name, srv := range registry.ServiceRegistry().GetAll() {
		srv.Close()
		logger.Default().Debugf("service %s shutdown", name)
	}
	if p.srvApi != nil {
		p.srvApi.Close()
	}
	if p.srvMetric != nil {
		p.srvMetric.Close()
	}
	if p.srvProfiling != nil {
		p.srvProfiling.Close()
	}
	return nil
}

func (p *program) reload(ctx context.Context) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	for {
		select {
		case <-c:
			if err := p.reloadConfig(); err != nil {
				logger.Default().Error(err)
			} else {
				logger.Default().Info("config reloaded")
			}

		case <-ctx.Done():
			return
		}
	}
}

func (p *program) reloadConfig() error {
	cfg, err := parseConfig()
	if err != nil {
		return err
	}

	config.Set(cfg)

	if err := p.build(cfg); err != nil {
		return err
	}

	return nil
}

func (p *program) build(cfg *config.Config) error {
	logCfg := cfg.Log
	if logCfg == nil {
		logCfg = &config.LogConfig{}
	}
	logger.SetDefault(logger_parser.ParseLogger(&config.LoggerConfig{Log: logCfg}))

	tlsCfg, err := parsing.BuildDefaultTLSConfig(cfg.TLS)
	if err != nil {
		return err
	}
	parsing.SetDefaultTLSConfig(tlsCfg)

	if err := register(cfg); err != nil {
		return err
	}

	for _, svc := range registry.ServiceRegistry().GetAll() {
		svc := svc
		go func() {
			svc.Serve()
		}()
	}

	if cfg.API != nil {
		if p.srvApi != nil {
			p.srvApi.Close()
		}

		s, err := buildAPIService(cfg.API)
		if err != nil {
			return err
		}

		p.srvApi = s

		go func() {
			defer s.Close()

			log := logger.Default().WithFields(map[string]any{"kind": "service", "service": "@api"})

			log.Info("listening on ", s.Addr())
			if err := s.Serve(); !errors.Is(err, http.ErrServerClosed) {
				log.Error(err)
			}
		}()
	}

	if cfg.Metrics != nil && cfg.Metrics.Addr != "" {
		if p.srvMetric != nil {
			p.srvMetric.Close()
		}

		s, err := buildMetricsService(cfg.Metrics)
		if err != nil {
			return err
		}

		p.srvMetric = s

		go func() {
			defer s.Close()

			log := logger.Default().WithFields(map[string]any{"kind": "service", "service": "@metrics"})

			log.Info("listening on ", s.Addr())
			if err := s.Serve(); !errors.Is(err, http.ErrServerClosed) {
				log.Error(err)
			}
		}()
	}

	if cfg.Profiling != nil {
		if p.srvProfiling != nil {
			p.srvProfiling.Close()
		}

		addr := cfg.Profiling.Addr
		if addr == "" {
			addr = ":6060"
		}
		s := &http.Server{
			Addr: addr,
		}
		p.srvProfiling = s

		go func() {
			defer s.Close()

			log := logger.Default().WithFields(map[string]any{"kind": "service", "service": "@profiling"})

			log.Info("listening on ", addr)
			if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				log.Error(err)
			}
		}()
	}

	return nil
}
