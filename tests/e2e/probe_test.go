package e2e

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type ProbeSuite struct {
	suite.Suite
	ctx   context.Context
	echoC testcontainers.Container
	echoIP string
}

func (s *ProbeSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *ProbeSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

func (s *ProbeSuite) proxyRequest(gostC testcontainers.Container, port string) (int, string) {
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

// TestTCPProbeFailover verifies that the TCP probe detects a dead node and
// marks it before any real traffic, so the FailFilter excludes it. With the
// dead node pre-marked, every request succeeds immediately.
func (s *ProbeSuite) TestTCPProbeFailover() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/probe/tcp.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// The probe fires immediately at startup, so the dead node is already
	// marked by the time we send requests.
	for range 10 {
		code, body := s.proxyRequest(gostC, "8080")
		s.Require().Equal(0, code, "every request must succeed; dead node pre-marked by probe")
		s.Require().Contains(body, "hello-gost")
	}
}

// TestLowestLatencyProbe verifies that the lowestlatency strategy works with
// probed nodes. Both nodes are live; the strategy selects the one with the
// lowest measured latency.
func (s *ProbeSuite) TestLowestLatencyProbe() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/probe/lowestlatency.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Give the initial probe a moment to fire so both nodes show healthy.
	time.Sleep(200 * time.Millisecond)

	for range 10 {
		code, body := s.proxyRequest(gostC, "8080")
		s.Require().Equal(0, code, "all requests must succeed with lowestlatency strategy")
		s.Require().Contains(body, "hello-gost")
	}
}

// TestCmdProbeFailover verifies that the cmd probe detects a dead node via
// shell exit code and marks it before real traffic, so FailFilter excludes it.
func (s *ProbeSuite) TestCmdProbeFailover() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/probe/cmd.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// The probe fires at startup; the dead node is already marked.
	for range 10 {
		code, body := s.proxyRequest(gostC, "8080")
		s.Require().Equal(0, code, "all requests must succeed; dead cmd node pre-marked")
		s.Require().Contains(body, "hello-gost")
	}
}

func TestProbeSuite(t *testing.T) {
	suite.Run(t, new(ProbeSuite))
}
