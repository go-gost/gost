package e2e

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type SelectorSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *SelectorSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *SelectorSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// proxyRequest sends one request through the gost proxy and returns the curl
// exit code and response body.
func (s *SelectorSuite) proxyRequest(gostC testcontainers.Container, port string) (int, string) {
	cmd := []string{
		"curl", "-s",
		"-x", fmt.Sprintf("http://127.0.0.1:%s", port),
		fmt.Sprintf("http://%s:5678", s.echoIP),
	}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return code, string(body)
}

// runFailover starts a gost container from cfg and verifies that, despite a
// dead node in the hop, requests converge to the live node after the selector
// marks the dead node and skips it. At most the initial marking attempt may
// fail; every request after must reach the echo server.
func (s *SelectorSuite) runFailover(cfg, port string) {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, cfg, port+"/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	const n = 12
	var failures int
	lastOK := false
	for range n {
		code, body := s.proxyRequest(gostC, port)
		ok := code == 0 && strings.Contains(body, "hello-gost")
		if !ok {
			failures++
		}
		lastOK = ok
	}

	s.Require().LessOrEqual(failures, 1,
		"selector did not filter the dead node; too many requests failed")
	s.Require().True(lastOK, "requests did not converge to the live node")
}

// TestRoundRobinFailover exercises the round-robin strategy combined with the
// fail filter: the dead node is tried once, marked, and then skipped.
func (s *SelectorSuite) TestRoundRobinFailover() {
	s.runFailover("testdata/selector/roundrobin.yaml", "8080")
}

// TestFIFOFailover exercises the fifo (sticky) strategy: it always picks the
// first node until it fails, then falls through to the secondary node.
func (s *SelectorSuite) TestFIFOFailover() {
	s.runFailover("testdata/selector/fifo.yaml", "8080")
}

// TestBackupFailover exercises the backup filter: with the primary node dead,
// the selector falls back to the backup node.
func (s *SelectorSuite) TestBackupFailover() {
	s.runFailover("testdata/selector/backup.yaml", "8080")
}

// TestParallelSelector exercises the parallel strategy: it dials all nodes
// concurrently and uses the first that connects, so a dead node never blocks
// the request.
func (s *SelectorSuite) TestParallelSelector() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/selector/parallel.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	code, body := s.proxyRequest(gostC, "8080")
	s.Require().Equal(0, code)
	s.Require().Contains(body, "hello-gost")
}

func TestSelectorSuite(t *testing.T) {
	suite.Run(t, new(SelectorSuite))
}
