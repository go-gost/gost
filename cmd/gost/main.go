package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
)

var (
	log = logger.Default()

	cfgFile       string
	outputCfgFile string
	services      stringList
	nodes         stringList
	debug         bool
)

func init() {
	var printVersion bool

	flag.Var(&services, "L", "service list")
	flag.Var(&nodes, "F", "chain node list")
	flag.StringVar(&cfgFile, "C", "", "configure file")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.BoolVar(&debug, "D", false, "debug mode")
	flag.StringVar(&outputCfgFile, "O", "", "write config to FILE")
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

	normConfig(cfg)

	log = logFromConfig(cfg.Log)

	if outputCfgFile != "" {
		if err := cfg.WriteFile(outputCfgFile); err != nil {
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
			log.Info("profiling serve on: ", addr)
			log.Fatal(http.ListenAndServe(addr, nil))
		}()
	}

	tlsCfg := cfg.TLS
	if tlsCfg == nil {
		tlsCfg = &config.TLSConfig{
			Cert: "cert.pem",
			Key:  "key.pem",
			CA:   "ca.crt",
		}
	}
	tlsConfig, err := tls_util.LoadTLSConfig(tlsCfg.Cert, tlsCfg.Key, tlsCfg.CA)
	if err != nil {
		// generate random self-signed certificate.
		cert, err := tls_util.GenCertificate()
		if err != nil {
			log.Fatal(err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		log.Warn("load TLS certificate files failed, use random generated certificate")
	} else {
		log.Debug("load TLS certificate files OK")
	}
	tls_util.DefaultConfig = tlsConfig

	services := buildService(cfg)
	for _, svc := range services {
		go svc.Run()
	}

	select {}
}
