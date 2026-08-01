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

// TestProbeRecovery verifies the dead→alive→dead transition that issue #837
// describes: a node that fails is excluded, and once it recovers (probe flips
// healthy) it resumes carrying traffic. Both relays stay up; only the cmd
// probe flag files in the container flip, so no restart is needed.
//
// Phase 1: node-a marked dead, node-b healthy → every request goes to node-b.
// Phase 2: recover node-a, kill node-b → every request must still succeed,
// which is only possible if node-a's probe recovered it (otherwise both nodes
// would be excluded and the request would fail).
func (s *ProbeSuite) TestProbeRecovery() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/probe/recovery.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// The node probes fire immediately at startup, so let both settle healthy
	// before we start flipping flag files.
	time.Sleep(1 * time.Second)

	// Phase 1: knock node-a down, leave node-b up.
	_, _, err = gostC.Exec(s.ctx, []string{"touch", "/tmp/a_down"})
	s.Require().NoError(err)
	time.Sleep(2500 * time.Millisecond) // > probe interval

	for range 10 {
		code, body := s.proxyRequest(gostC, "8080")
		s.Require().Equal(0, code, "phase1: dead node-a must be excluded, all traffic via node-b")
		s.Require().Contains(body, "hello-gost")
	}

	// Phase 2: recover node-a, knock node-b down. If node-a's probe did not
	// revive it, the request would fail (no healthy candidate).
	_, _, err = gostC.Exec(s.ctx, []string{"rm", "-f", "/tmp/a_down", "/tmp/b_down"})
	s.Require().NoError(err)
	_, _, err = gostC.Exec(s.ctx, []string{"touch", "/tmp/b_down"})
	s.Require().NoError(err)
	time.Sleep(2500 * time.Millisecond)

	for range 10 {
		code, body := s.proxyRequest(gostC, "8080")
		s.Require().Equal(0, code, "phase2: recovered node-a must resume carrying traffic")
		s.Require().Contains(body, "hello-gost")
	}
}

func TestProbeSuite(t *testing.T) {
	suite.Run(t, new(ProbeSuite))
}
