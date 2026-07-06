package e2e

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type PHTSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *PHTSuite) SetupSuite() {
	s.ctx = context.Background()

	s.T().Logf("start tcp echo container...")
	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *PHTSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// TestPHTTunnel verifies a basic PHT tunnel: a client HTTP proxy chains
// through a PHT dialer to reach a PHT server, which forwards the request
// to the echo server.
func (s *PHTSuite) TestPHTTunnel() {
	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/pht/server.yaml",
		[]string{"pht-server"}, []string{"8443/tcp"})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	rendered, err := RenderConfig("testdata/pht/client_connector.yaml",
		ConfigData{ServerAddr: "pht-server:8443"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		rendered, "8080/tcp")
	s.Require().NoError(err)
	defer clientC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := clientC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "pht client logs", clientC)
		DumpLogs(s.T(), s.ctx, "pht server logs", serverC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestPHTSTunnel verifies PHT over TLS. The PHTs listener auto-generates a
// self-signed cert; the PHTs dialer defaults to InsecureSkipVerify so the
// handshake succeeds without explicit cert configuration.
func (s *PHTSuite) TestPHTSTunnel() {
	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/pht/server_phts.yaml",
		[]string{"phts-server"}, []string{"8443/tcp"})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	rendered, err := RenderConfig("testdata/pht/client_connector_phts.yaml",
		ConfigData{ServerAddr: "phts-server:8443"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		rendered, "8080/tcp")
	s.Require().NoError(err)
	defer clientC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := clientC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "phts client logs", clientC)
		DumpLogs(s.T(), s.ctx, "phts server logs", serverC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestPHTHeartbeat verifies that the PHT tunnel stays alive across idle
// periods. Sends a request, waits >readTimeout (default 10s), and sends
// another request to confirm the heartbeat kept the connection open.
func (s *PHTSuite) TestPHTHeartbeat() {
	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/pht/server.yaml",
		[]string{"pht-server"}, []string{"8443/tcp"})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	rendered, err := RenderConfig("testdata/pht/client_connector.yaml",
		ConfigData{ServerAddr: "pht-server:8443"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		rendered, "8080/tcp")
	s.Require().NoError(err)
	defer clientC.Terminate(s.ctx)

	// First request: establish the tunnel.
	cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := clientC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, _ := io.ReadAll(out)
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")

	// Wait longer than readTimeout to trigger heartbeat.
	// The PHT server's default read timeout is 10s. We wait 12s.
	cmd = []string{"sh", "-c",
		fmt.Sprintf("sleep 12 && curl -s -x http://127.0.0.1:8080 http://%s:5678", s.echoIP)}
	code, out, err = clientC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, _ = io.ReadAll(out)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "pht heartbeat client logs", clientC)
		DumpLogs(s.T(), s.ctx, "pht heartbeat server logs", serverC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

func TestPHTSuite(t *testing.T) {
	suite.Run(t, new(PHTSuite))
}
