package main

import (
	stdlog "log"
	"net/http"
	_ "net/http/pprof"

	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
)

var (
	log = logger.NewLogger()
)

func main() {
	stdlog.SetFlags(stdlog.LstdFlags | stdlog.Lshortfile)
	cfg := &config.Config{}
	if err := cfg.Load(); err != nil {
		log.Fatal(err)
	}
	log = logFromConfig(cfg.Log)

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
	services := buildService(cfg)
	for _, svc := range services {
		go svc.Run()
	}

	select {}
}
