package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
)

var (
	log = logger.Default()

	cfgFile      string
	outputFormat string
	services     stringList
	nodes        stringList
	debug        bool
)

func init() {
	var printVersion bool

	flag.Var(&services, "L", "service list")
	flag.Var(&nodes, "F", "chain node list")
	flag.StringVar(&cfgFile, "C", "", "configure file")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.BoolVar(&debug, "D", false, "debug mode")
	flag.StringVar(&outputFormat, "O", "", "output format, one of yaml|json format")
	flag.Parse()

	if printVersion {
		fmt.Fprintf(os.Stdout, "gost %s (%s %s/%s)\n",
			version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}
}

func main() {
	cfg := &config.Config{}
	var err error
	if len(services) > 0 {
		cfg, err = buildConfigFromCmd(services, nodes)
		if debug && cfg != nil {
			if cfg.Log == nil {
				cfg.Log = &config.LogConfig{}
			}
			cfg.Log.Level = string(logger.DebugLevel)
		}
	} else {
		if cfgFile != "" {
			err = cfg.ReadFile(cfgFile)
		} else {
			err = cfg.Load()
		}
	}
	if err != nil {
		log.Fatal(err)
	}

	log = logFromConfig(cfg.Log)

	if outputFormat != "" {
		if err := cfg.Write(os.Stdout, outputFormat); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	if cfg.Profiling != nil && cfg.Profiling.Enabled {
		go func() {
			addr := cfg.Profiling.Addr
			if addr == "" {
				addr = ":6060"
			}
			log.Info("profiling serve on ", addr)
			log.Fatal(http.ListenAndServe(addr, nil))
		}()
	}

	buildDefaultTLSConfig(cfg.TLS)

	services := buildService(cfg)
	for _, svc := range services {
		go svc.Run()
	}

	select {}
}
