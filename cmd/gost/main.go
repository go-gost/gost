package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"strings"
	"runtime"

	"github.com/go-gost/core/logger"
	"github.com/go-gost/core/metrics"
	"github.com/go-gost/x/config"
	"github.com/go-gost/x/config/parsing"
	xlogger "github.com/go-gost/x/logger"
	xmetrics "github.com/go-gost/x/metrics"
)

type WorkerSync struct {
	wg sync.WaitGroup
	mu sync.Mutex
}

var (
	log logger.Logger
)

func init() {
	log = xlogger.NewLogger()
	logger.SetDefault(log)
}

func main() {
	var ws WorkerSync

	ws.wg.Add(1)  // Gost must exit if any of the workers exit

	// Split os.Args using -- and create a worker with each slice
	args := strings.Split(" " + strings.Join(os.Args[1:], "  ") + " ", " -- ")
	if strings.Join(args, "") == "" {
		// Fix to show gost help if the resulting array is empty
		args[0] = " "
	}
	for wid, wargs := range args {
		if wargs != "" {
			go worker(wid, wargs, &ws)
		}
	}
	ws.wg.Wait()
}

func worker(id int, args string, ws *WorkerSync) {
	defer ws.wg.Done()

	var (
		cfgFile      string
		outputFormat string
		services     stringList
		nodes        stringList
		debug        bool
		apiAddr      string
		metricsAddr  string

		err          error

		cfg = &config.Config{}
	)

	init := func () error {
		var printVersion bool

		wf := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		wf.Var(&services, "L", "service list")
		wf.Var(&nodes, "F", "chain node list")
		wf.StringVar(&cfgFile, "C", "", "configure file")
		wf.BoolVar(&printVersion, "V", false, "print version")
		wf.StringVar(&outputFormat, "O", "", "output format, one of yaml|json format")
		wf.BoolVar(&debug, "D", false, "debug mode")
		wf.StringVar(&apiAddr, "api", "", "api service address")
		wf.StringVar(&metricsAddr, "metrics", "", "metrics service address")

		wf.Parse(strings.Fields(args))

		if printVersion {
			fmt.Fprintf(os.Stdout, "gost %s (%s %s/%s)\n",
				version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
			os.Exit(0)
		} else if wf.NFlag() == 0 {
			wf.Usage()
			os.Exit(0)
		}
		return nil
	}

	main := func () error {
		if len(services) > 0 || apiAddr != "" {
			cfg, err = buildConfigFromCmd(id, services, nodes)
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
			} else {
				err = cfg.Load()
			}
			if err != nil {
				log.Fatal(err)
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
					// Each worker uses a different profiling server
					addr = fmt.Sprintf(":606%d", id)
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
			metrics.SetGlobal(xmetrics.NewMetrics())
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

		svcs := buildService(cfg)
		for _, svc := range svcs {
			svc := svc
			go func() {
				svc.Serve()
				svc.Close()
			}()
		}
		config.SetGlobal(cfg)

		return nil
	}

	// Using mutex to avoid duplicated service creation race condition
	ws.mu.Lock()
	if err := init(); err != nil {
		log.Fatal(err)
		return
	}
	if err := main(); err != nil {
		log.Fatal(err)
		return
	}
	ws.mu.Unlock()

	// Allow local functions to be garbage-collected
	init = nil
	main = nil

	select {}
}
