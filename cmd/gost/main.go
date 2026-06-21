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
	"sync/atomic"
	"time"

	"github.com/go-gost/core/logger"
	xlogger "github.com/go-gost/x/logger"
	"github.com/judwhite/go-svc"
)

type stringList []string

func (l *stringList) String() string {
	return fmt.Sprintf("%s", *l)
}
func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

var (
	cfgFiles     stringList
	outputFormat string
	services     stringList
	nodes        stringList
	debug        bool
	trace        bool
	apiAddr      string
	metricsAddr  string
	reload       time.Duration
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	args := strings.Join(os.Args[1:], "  ")

	if strings.Contains(args, " -- ") {
		var (
			wg  sync.WaitGroup
			ret atomic.Int32
		)

		wargsList := strings.Split(" "+args+" ", " -- ")

		ctx, cancel := context.WithCancel(context.Background())

		for wid, wargs := range wargsList {
			wg.Go(func() {
				worker(wid, strings.Split(strings.TrimSpace(wargs), "  "), ctx, &ret)
			})
		}

		wg.Wait()
		cancel()

		os.Exit(int(ret.Load()))
	}
}

func worker(id int, args []string, ctx context.Context, ret *atomic.Int32) {
	cmd := exec.CommandContext(ctx, os.Args[0], args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("_GOST_ID=%d", id))

	if err := cmd.Run(); err != nil {
		// Context cancellation is expected when one worker exits early.
		// Only log fatal on other errors.
		if ctx.Err() == nil {
			log.Fatal(err)
		}
		return
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		ret.Store(int32(cmd.ProcessState.ExitCode()))
	}
}

func init() {
	var printVersion bool

	flag.Var(&services, "L", "service list")
	flag.Var(&nodes, "F", "chain node list")
	flag.Var(&cfgFiles, "C", "config file(s), URL(s), or inline JSON")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.StringVar(&outputFormat, "O", "", "output format, one of yaml|json format")
	flag.BoolVar(&debug, "D", false, "debug mode")
	flag.BoolVar(&trace, "DD", false, "trace mode")
	flag.StringVar(&apiAddr, "api", "", "api service address")
	flag.StringVar(&metricsAddr, "metrics", "", "metrics service address")
	flag.DurationVar(&reload, "R", 0, "auto reload period (e.g. 30s, 1m)")
	flag.Parse()

	if printVersion {
		fmt.Fprintf(os.Stdout, "gost %s (%s %s/%s)\n",
			version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}
}

func main() {
	log := xlogger.NewLogger()
	logger.SetDefault(log)

	p := &program{}

	if err := svc.Run(p); err != nil {
		logger.Default().Fatal(err)
	}
}
