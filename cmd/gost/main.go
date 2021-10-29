package main

import (
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
)

var (
	log = logger.NewLogger()
)

func main() {
	cfg := &config.Config{}
	if err := cfg.Load(); err != nil {
		log.Fatal(err)
	}
	log = logFromConfig(cfg.Log)

	services := buildService(cfg)
	for _, svc := range services {
		go svc.Run()
	}

	select {}
}
