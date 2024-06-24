package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/go-gost/core/logger"
	xlogger "github.com/go-gost/x/logger"
	"github.com/judwhite/go-svc"
)

var (
	cfgFile      string
	outputFormat string
	services     stringList
	nodes        stringList
	debug        bool
	apiAddr      string
	metricsAddr  string
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	args := strings.Join(os.Args[1:], "  ")

	if strings.Contains(args, " -- ") {
		var (
			wg  sync.WaitGroup
			ret int
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for wid, wargs := range strings.Split(" "+args+" ", " -- ") {
			wg.Add(1)
			go func(wid int, wargs string) {
				defer wg.Done()
				defer cancel()
				worker(wid, strings.Split(wargs, "  "), &ctx, &ret)
			}(wid, strings.TrimSpace(wargs))
		}

		wg.Wait()

		os.Exit(ret)
	}
}

func worker(id int, args []string, ctx *context.Context, ret *int) {
	cmd := exec.CommandContext(*ctx, os.Args[0], args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("_GOST_ID=%d", id))

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	if cmd.ProcessState.Exited() {
		*ret = cmd.ProcessState.ExitCode()
	}
}

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

	logger.SetDefault(xlogger.NewLogger())
}

func main() {
	p := &program{}
	if err := svc.Run(p); err != nil {
		log.Fatal(err)
	}
}
