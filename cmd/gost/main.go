package main

import (
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
)

func main() {
	log := logger.NewLogger("main")
	log.EnableJSONOutput(true)

	cfg := &config.Config{}
	if err := cfg.Load(); err != nil {
		log.Fatal(err)
	}
	services := buildService(cfg)
	for _, svc := range services {
		go svc.Run()
	}

	select {}
}
