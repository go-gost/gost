package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type ForwardSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
	udpC   testcontainers.Container
}

func (s *ForwardSuite) SetupSuite() {
	s.ctx = context.Background()

	s.T().Logf("start tcp echo container...")
	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP

	s.T().Logf("start udp echo container...")
	udpC, err := RunUDPEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.udpC = udpC
}

// sendRaw sends a raw HTTP request via netcat and returns the response.
// Uses base64 to avoid shell quoting issues with CRLF bytes.
func (s *ForwardSuite) sendRaw(gostC testcontainers.Container, host, port, data string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	cmd := []string{"sh", "-c",
		fmt.Sprintf("echo %s | base64 -d | nc -w 5 %s %s", encoded, host, port)}
	_, out, _ := gostC.Exec(s.ctx, cmd)
	b, _ := io.ReadAll(out)
	return string(b)
}

func (s *ForwardSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
	if s.udpC != nil {
		s.udpC.Terminate(s.ctx)
	}
}

// TestTCPForward verifies basic TCP forward handler (handler type: tcp).
// The forward handler pipes raw TCP connections to the configured forwarder
// node (tcp-echo:5678). curl connects directly to the handler port and sends
// an HTTP request, expecting the echo server's "hello-gost" response.
func (s *ForwardSuite) TestTCPForward() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/forward/server.yaml", "8000/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// curl directly to the handler port (not via -x proxy flag).
	// The TCP forward handler pipes our connection to tcp-echo:5678.
	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8000/"}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "tcp-forward logs", gostC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestForwardAlias verifies that the "forward" handler type (alias for "tcp")
// works identically to the "tcp" handler type.
func (s *ForwardSuite) TestForwardAlias() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/forward/server.yaml", "8000/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8000/"}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "forward-alias logs", gostC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestTCPForwardSniffing verifies TCP forward handler with sniffing enabled.
// When sniffing is enabled and the connection starts with HTTP data, the handler
// detects the protocol via sniffing.Sniff and delegates to the HTTP sniffer
// for protocol-aware forwarding. The result is the same as raw forwarding:
// "hello-gost" from the echo server.
func (s *ForwardSuite) TestTCPForwardSniffing() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/forward/server_sniffing.yaml", "8000/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// curl directly — sniffing detects HTTP and handles via sniffer.
	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8000/"}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "tcp-forward-sniffing logs", gostC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestTCPForwardRaw verifies raw TCP forwarding by sending an HTTP request
// via netcat through the forward handler. This tests the handleRawForwarding
// code path (no sniffing), proving that raw bytes are piped through correctly.
func (s *ForwardSuite) TestTCPForwardRaw() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/forward/server.yaml", "8000/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Send raw HTTP request via nc (not curl) to exercise the raw pipe path.
	resp := s.sendRaw(gostC, "127.0.0.1", "8000",
		"GET / HTTP/1.0\r\nHost: tcp-echo\r\n\r\n")
	s.Assert().Contains(resp, "hello-gost",
		"raw request through TCP forward should reach echo server")
}

// TestTCPForwardIdleTimeout verifies that idleTimeout closes the pipe after
// a period of inactivity. The forward handler's xnet.Pipe uses idleTimeout
// as a read deadline on the upstream connection — if no data flows for
// that duration, the pipe closes both directions.
func (s *ForwardSuite) TestTCPForwardIdleTimeout() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/forward/server_idle_timeout.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/tcp_idle_timeout.py", ContainerFilePath: "/scripts/tcp_idle_timeout.py", FileMode: 0644},
		},
		"8000/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// The Python script:
	// 1. Connects to gost TCP forward port
	// 2. Sends HTTP GET → expects "hello-gost" confirm pipe is alive
	// 3. Waits > idleTimeout (3s + 2s buffer)
	// 4. Sends more data — expects connection to be closed
	code, out, err := gostC.Exec(s.ctx, []string{
		"python3", "/scripts/tcp_idle_timeout.py",
		"127.0.0.1", "8000", "3",
	})
	output, _ := io.ReadAll(out)
	s.T().Logf("idle timeout output:\n%s", string(output))
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "tcp-idle-timeout logs", gostC)
	}
	s.Require().Equal(0, code, "idle timeout test script should exit 0")
}

// TestUDPForward verifies basic UDP forwarding (handler: udp, listener: udp).
// The handler uses handleRawDatagram via the stateful UDP listener's
// per-client session conns (which implement net.PacketConn).
// A Python script inside the gost container sends a UDP datagram through
// the forward handler and verifies the echo response from udp-echo:5679.
func (s *ForwardSuite) TestUDPForward() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/forward/server_udp.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/udp_forward_test.py", ContainerFilePath: "/scripts/udp_forward_test.py", FileMode: 0644},
		},
		"9000/udp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	code, out, err := gostC.Exec(s.ctx, []string{
		"python3", "/scripts/udp_forward_test.py",
		"127.0.0.1", "9000",
	})
	output, _ := io.ReadAll(out)
	s.T().Logf("udp forward output:\n%s", string(output))
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "udp-forward logs", gostC)
	}
	s.Require().Equal(0, code, "udp forward test script should exit 0")
}

// TestUDPForwardStateless verifies UDP forwarding with stateless mode.
// Both listener and handler use stateless: true, so each datagram is a
// single request-response cycle with no per-client session tracking.
func (s *ForwardSuite) TestUDPForwardStateless() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/forward/server_udp_stateless.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/udp_forward_test.py", ContainerFilePath: "/scripts/udp_forward_test.py", FileMode: 0644},
		},
		"9000/udp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	code, out, err := gostC.Exec(s.ctx, []string{
		"python3", "/scripts/udp_forward_test.py",
		"127.0.0.1", "9000",
	})
	output, _ := io.ReadAll(out)
	s.T().Logf("udp forward stateless output:\n%s", string(output))
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "udp-forward-stateless logs", gostC)
	}
	s.Require().Equal(0, code, "udp forward stateless test script should exit 0")
}

// TestTCPForwardSniffingBypass verifies that when sniffing is enabled and a
// bypass rule blocks the target, the HTTP sniffer returns 403 Forbidden.
// The forward handler delegates to the HTTP sniffer (via handleSniffedProtocol),
// and the sniffer's resolveHTTPNode checks h.options.Bypass and returns 403
// when the destination is matched.
func (s *ForwardSuite) TestTCPForwardSniffingBypass() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/forward/server_bypass_sniffing.yaml", "8000/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// curl connects directly; sniffing detects HTTP. The sniffer bypass
	// check matches 0.0.0.0/0 and returns 403 Forbidden.
	cmd := []string{"curl", "-v", "-s", "-D", "-", "-o", "/dev/null",
		"http://127.0.0.1:8000/"}
	_, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, _ := io.ReadAll(out)
	output := string(body)

	s.Assert().Contains(output, "403",
		"bypass should return 403 Forbidden from sniffer")
}

// TestTCPForwardMultiNodeProtocol verifies that sniffing + protocol-filtered
// forwarder nodes correctly routes an HTTP request to the node with matching
// protocol (http), rather than the one with protocol: tls.
//
// Config has two nodes:
//   - echo-http (protocol: http) → tcp-echo:5678 (works, returns "hello-gost")
//   - echo-tls  (protocol: tls)  → tcp-echo:1 (closed port, would fail)
//
// With sniffing enabled, an HTTP request is detected as protocol "http",
// Select("http") filters to only the echo-http node. If protocol filtering
// fails and the tls node is selected instead, the connection to port 1
// fails and the test fails.
func (s *ForwardSuite) TestTCPForwardMultiNodeProtocol() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/forward/server_multi_node.yaml", "8000/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// curl sends an HTTP request → sniffed as "http" → Select("http")
	// filters to echo-http (protocol: http) → pipes to tcp-echo:5678
	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8000/"}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "tcp-forward-multi-node logs", gostC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost",
		"HTTP request should route to the http-protocol node via protocol filtering")
}

func TestForwardSuite(t *testing.T) {
	suite.Run(t, new(ForwardSuite))
}
