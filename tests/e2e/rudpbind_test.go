package e2e

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// RUDPBindSuite reproduces go-gost/gost#911: concurrent UDP source sockets
// sharing ONE reverse UDP tunnel over a relay server.
//
// Topology (all three gost processes plus a UDP echo server):
//
//	UDP source sockets (N) -> client udp service -> relay server
//	  -> reverse-bound UDP port -> host rudp listener (ONE shared stream)
//	  -> udp-echo
//
// A datagram frame is written to that shared stream as three separate Write
// calls (RSV/FRAG, SOCKS5 address, payload). Before the fix, with N endpoints
// writing concurrently, frames interleaved between those calls: replies came
// back corrupted, and a torn frame left half a header in the stream, which the
// far end read as "unexpected EOF" and answered by tearing down and rebinding
// the whole reverse tunnel every second.
//
// The host is the side that binds the tunnel, so its log is where a reset
// shows up as "unexpected EOF, retrying in 1s" followed by a rebind.
type RUDPBindSuite struct {
	suite.Suite
	ctx     context.Context
	udpC    testcontainers.Container
	serverC testcontainers.Container
	hostC   testcontainers.Container
	clientC testcontainers.Container
}

func (s *RUDPBindSuite) SetupSuite() {
	s.ctx = context.Background()

	udpC, err := RunUDPEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.udpC = udpC

	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/rudpbind/server.yaml", []string{"gost-server"}, []string{"8430/tcp"})
	s.Require().NoError(err)
	s.serverC = serverC

	// The host's rudp listener opens no local port, so it exposes a dummy TCP
	// port (service-1) purely for the readiness wait.
	hostC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/rudpbind/host.yaml", []string{"gost-host"}, []string{"8431/tcp"})
	s.Require().NoError(err)
	s.hostC = hostC

	clientC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/rudpbind/client.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/udp_rudp_concurrent.py", ContainerFilePath: "/scripts/udp_rudp_concurrent.py", FileMode: 0644},
		})
	s.Require().NoError(err)
	s.clientC = clientC

	// The reverse tunnel binds remotely on first traffic, not at startup.
	up := assert.Eventually(s.T(), func() bool {
		out := s.sendConcurrent(1, 1)
		if !strings.Contains(out, "PASS") {
			s.T().Logf("waiting for tunnel, sender said:\n%s", out)
		}
		return out != "" && strings.Contains(out, "PASS")
	}, 30*time.Second, time.Second, "reverse UDP tunnel never came up")
	if !up {
		DumpLogs(s.T(), s.ctx, "host logs", s.hostC)
		DumpLogs(s.T(), s.ctx, "server logs", s.serverC)
		DumpLogs(s.T(), s.ctx, "client logs", s.clientC)
		s.FailNow("reverse UDP tunnel never came up")
	}
}

func (s *RUDPBindSuite) TearDownSuite() {
	for _, c := range []testcontainers.Container{s.clientC, s.hostC, s.serverC, s.udpC} {
		if c != nil {
			c.Terminate(s.ctx)
		}
	}
}

// sendConcurrent runs the concurrent sender inside the client container and
// returns its combined stdout+stderr.
func (s *RUDPBindSuite) sendConcurrent(endpoints, seconds int) string {
	cmd := []string{
		"python3", "/scripts/udp_rudp_concurrent.py",
		// The client service listens in this same container, so loopback is the
		// correct target — no network alias needed for the sender.
		"127.0.0.1", "8432",
		strconv.Itoa(endpoints), strconv.Itoa(seconds),
	}
	_, out, err := s.clientC.Exec(s.ctx, cmd)
	if err != nil {
		return ""
	}
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := out.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}

// TestConcurrentEndpointsShareTunnel is the regression assertion: several UDP
// source sockets sharing one reverse tunnel must each get their own datagrams
// back unchanged, and the tunnel must not be rebuilt underneath them.
func (s *RUDPBindSuite) TestConcurrentEndpointsShareTunnel() {
	out := s.sendConcurrent(4, 8)
	s.T().Logf("concurrent send:\n%s", out)

	if strings.Contains(out, "FAIL") {
		DumpLogs(s.T(), s.ctx, "host logs", s.hostC)
		DumpLogs(s.T(), s.ctx, "server logs", s.serverC)
	}
	s.Require().Contains(out, "PASS", "concurrent endpoints over one reverse tunnel failed:\n%s", out)

	// Belt and braces: the host log is where the reset appears directly.
	hostLogs, err := s.hostC.Logs(s.ctx)
	if err == nil {
		body, readErr := io.ReadAll(hostLogs)
		hostLogs.Close()
		if readErr == nil {
			s.Require().NotContains(string(body), "unexpected EOF",
				"reverse tunnel was torn down while endpoints were sending")
		}
	}
}

func TestRUDPBindSuite(t *testing.T) {
	suite.Run(t, new(RUDPBindSuite))
}