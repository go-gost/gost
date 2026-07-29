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

type ChainGroupSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *ChainGroupSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *ChainGroupSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// proxyRequest sends a request through the gost proxy, using the given
// target hostname (not IP) so that Host() matchers at the chain-group level
// can match against it.
func (s *ChainGroupSuite) proxyRequestHost(gostC testcontainers.Container, proxyPort, targetHost string) (int, string) {
	cmd := []string{
		"curl", "-s",
		"-x", fmt.Sprintf("http://127.0.0.1:%s", proxyPort),
		fmt.Sprintf("http://%s:5678", targetHost),
	}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return code, string(body)
}

// proxyRequestIP sends a request via IP (like existing tests do).
func (s *ChainGroupSuite) proxyRequestIP(gostC testcontainers.Container, port string) (int, string) {
	return s.proxyRequestHost(gostC, port, s.echoIP)
}

// TestMatcherRoutesByHost verifies that a chainGroup with per-entry Host() matchers
// routes traffic to the chain whose matcher matches the target hostname. The
// non-matching chain has a dead relay, so if the matcher fails to filter, requests
// would fail.
func (s *ChainGroupSuite) TestMatcherRoutesByHost() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/chaingroup/matcher.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Requests with hostname "tcp-echo" (the echo container's network alias)
	// must match Host("tcp-echo") → chain-target → live relay → echo server.
	for range 10 {
		code, body := s.proxyRequestHost(gostC, "8080", "tcp-echo")
		s.Require().Equal(0, code, "Host('tcp-echo') must match the target chain")
		s.Require().Contains(body, "hello-gost")
	}
}

// TestProbeMarksDeadChain verifies that a TCP probe detects a dead chain entry
// and marks it before real traffic, so the FailFilter excludes it. With the dead
// entry pre-marked, every request succeeds.
func (s *ChainGroupSuite) TestProbeMarksDeadChain() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/chaingroup/probe.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// The probe fires immediately at startup; by the time we send requests
	// the dead chain entry is already marked. All requests converge to the
	// live chain.
	failures := 0
	for range 10 {
		code, body := s.proxyRequestIP(gostC, "8080")
		if code != 0 || !strings.Contains(body, "hello-gost") {
			failures++
		}
	}

	// At most one failure is acceptable (the first request might race with
	// the probe's first Mark). Every request after must succeed.
	s.Require().LessOrEqual(failures, 0,
		"probe must pre-mark the dead chain entry; all requests must succeed")
}

func TestChainGroupSuite(t *testing.T) {
	suite.Run(t, new(ChainGroupSuite))
}
