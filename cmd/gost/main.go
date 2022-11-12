package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/go-gost/core/logger"
	"github.com/go-gost/x/config"
	"github.com/go-gost/x/config/parsing"
	xlogger "github.com/go-gost/x/logger"
	xmetrics "github.com/go-gost/x/metrics"
	"github.com/go-gost/x/registry"
)

var (
	log logger.Logger

	cfgFile      string
	outputFormat string
	services     stringList
	nodes        stringList
	debug        bool
	apiAddr      string
	metricsAddr  string
)

func init() {
	var printVersion bool

	flag.Var(&services, "L", "service list")
	flag.Var(&nodes, "F", "chain node list")
	flag.StringVar(&cfgFile, "C", "", "configuration file")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.StringVar(&outputFormat, "O", "", "output format, one of yaml|json format")
	flag.BoolVar(&debug, "D", false, "debug mode")
	flag.StringVar(&apiAddr, "api", "", "api service address")
	flag.StringVar(&metricsAddr, "metrics", "", "metrics service address")
	flag.Parse()

	if printVersion {
		fmt.Fprintf(os.Stdout, "gost %s (%s %s/%s)\n",
			version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	log = xlogger.NewLogger()
	logger.SetDefault(log)
}

func main() {
	cfg := &config.Config{}
	var err error
	if apiAddr != "" {
		cfg, err = buildConfigFromCmd(services, nodes)
		if err != nil {
			log.Fatal(err)
		}
		if debug && cfg != nil {
			if cfg.Log == nil {
				cfg.Log = &config.LogConfig{}
			}
			cfg.Log.Level = string(logger.DebugLevel)
		}
		if apiAddr != "" {
			cfg.API = &config.APIConfig{
				Addr: apiAddr,
			}
		}
		if metricsAddr != "" {
			cfg.Metrics = &config.MetricsConfig{
				Addr: metricsAddr,
			}
		}
	} else {
		if cfgFile != "" {
			err = cfg.ReadFile(cfgFile)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err = cfg.Load()
			if err != nil {
				if len(services) > 0 {
					log.Info("Cannot load config from default path. ", err)
				} else {
					log.Fatal(err)
				}
			}
		}
		if len(services) > 0 || len(nodes) > 0 {
			var cmdCfg = &config.Config{}
			cmdCfg, err = buildConfigFromCmd(services, nodes)
			if err != nil {
				log.Fatal(err)
			}
			if cmdCfg.Chains != nil && len(cmdCfg.Chains) > 0 {
				if cfg.Chains == nil || len(cfg.Chains) == 0 {
					cfg.Chains = cmdCfg.Chains
				} else {
					for i := 0; i < len(cfg.Chains); i++ {
						var hops = append([]*config.HopConfig{}, cmdCfg.Chains[0].Hops...)
						cfg.Chains[i].Hops = append(hops, cfg.Chains[i].Hops...)
					}
				}
			}
			if cmdCfg.Services != nil && len(cmdCfg.Services) > 0 {
				if (cfg.Chains != nil && len(cfg.Chains) > 0) {
					for i := 0; i < len(cmdCfg.Services); i++ {
						var handler = cmdCfg.Services[i].Handler
						if (handler != nil && (handler.Chain == "" && handler.ChainGroup == nil)) {
							handler.Chain = cfg.Chains[0].Name
						}
					}
				}
				if cfg.Services == nil {
					cfg.Services = []*config.ServiceConfig{}
				}
				cfg.Services = append(append([]*config.ServiceConfig{}, cmdCfg.Services...), cfg.Services...)
			}
		}
	}

	log = logFromConfig(cfg.Log)

	logger.SetDefault(log)

	if outputFormat != "" {
		if err := cfg.Write(os.Stdout, outputFormat); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	if cfg.Profiling != nil {
		go func() {
			addr := cfg.Profiling.Addr
			if addr == "" {
				addr = ":6060"
			}
			log.Info("profiling server on ", addr)
			log.Fatal(http.ListenAndServe(addr, nil))
		}()
	}

	if cfg.API != nil {
		s, err := buildAPIService(cfg.API)
		if err != nil {
			log.Fatal(err)
		}
		defer s.Close()

		go func() {
			log.Info("api service on ", s.Addr())
			log.Fatal(s.Serve())
		}()
	}

	if cfg.Metrics != nil {
		xmetrics.Init(xmetrics.NewMetrics())
		if cfg.Metrics.Addr != "" {
			s, err := buildMetricsService(cfg.Metrics)
			if err != nil {
				log.Fatal(err)
			}
			go func() {
				defer s.Close()
				log.Info("metrics service on ", s.Addr())
				log.Fatal(s.Serve())
			}()
		}
	}

	parsing.BuildDefaultTLSConfig(cfg.TLS)

	services := buildService(cfg)
	for _, svc := range services {
		svc := svc
		go func() {
			svc.Serve()
			// svc.Close()
		}()
	}

	config.SetGlobal(cfg)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	for sig := range sigs {
		switch sig {
		case syscall.SIGHUP:
			return
		default:
			for name, srv := range registry.ServiceRegistry().GetAll() {
				srv.Close()
				log.Debugf("service %s shutdown", name)
			}
			return
		}
	}
}
