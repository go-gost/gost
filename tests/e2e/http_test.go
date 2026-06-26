package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type HTTPSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *HTTPSuite) SetupSuite() {
	s.ctx = context.Background()

	s.T().Logf("start tcp echo container...")
	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

// sendRaw sends a raw HTTP request via netcat and returns the response.
// Uses base64 to avoid shell quoting issues with CRLF bytes.
func (s *HTTPSuite) sendRaw(gostC testcontainers.Container, host, port, data string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	cmd := []string{"sh", "-c",
		fmt.Sprintf("echo %s | base64 -d | nc -w 5 %s %s", encoded, host, port)}
	_, out, _ := gostC.Exec(s.ctx, cmd)
	b, _ := io.ReadAll(out)
	return string(b)
}

func (s *HTTPSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// TestHTTPProxy verifies basic HTTP forward proxy (no auth, no metadata).
// Covers: handler type http, listener type tcp, basic GET via proxy.
func (s *HTTPSuite) TestHTTPProxy() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// curl -x uses HTTP forward proxy (GET with absolute URL)
	cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "http-proxy logs", gostC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestHTTPProxyAuth verifies proxy authentication and the authBasicRealm
// metadata parameter.
//
// Covers:
//   - auther: basic auth on HTTP proxy
//   - authBasicRealm: custom realm string in 407 Proxy-Authenticate header
func (s *HTTPSuite) TestHTTPProxyAuth() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server_auth.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	s.T().Run("no-auth-407", func(t *testing.T) {
		cmd := []string{"curl", "-v", "-s", "-D", "-", "-o", "/dev/null",
			"-x", "http://127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		_, out, _ := gostC.Exec(s.ctx, cmd)
		body, _ := io.ReadAll(out)
		output := string(body)
		s.Assert().Contains(output, "407",
			"no auth should return 407")
		s.Assert().Contains(output, "gost-e2e-realm",
			"authBasicRealm should appear in 407 Proxy-Authenticate")
	})

	s.T().Run("wrong-auth-407", func(t *testing.T) {
		cmd := []string{"curl", "-v", "-s", "-o", "/dev/null", "-w", "%{http_code}",
			"-x", "http://wrong:pass@127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		_, out, _ := gostC.Exec(s.ctx, cmd)
		body, _ := io.ReadAll(out)
		s.Assert().Contains(string(body), "407",
			"wrong password should return 407")
	})

	s.T().Run("with-auth-success", func(t *testing.T) {
		cmd := []string{"curl", "-v", "-s", "-x", "http://user:pass@127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		code, out, err := gostC.Exec(s.ctx, cmd)
		s.Require().NoError(err)

		body, err := io.ReadAll(out)
		s.Require().NoError(err)
		if code != 0 || !strings.Contains(string(body), "hello-gost") {
			DumpLogs(s.T(), s.ctx, "http-proxy-auth logs", gostC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(string(body), "hello-gost")
	})
}

// TestHTTPProxyMetadata verifies:
//   - probeResist: code:404 — unauthorised clients see 404 instead of 407
//   - keepalive: config parsing (applied at Init time)
//   - compression: config parsing (applied at Init time)
func (s *HTTPSuite) TestHTTPProxyMetadata() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server_metadata.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	s.T().Run("probe-resist-404", func(t *testing.T) {
		cmd := []string{"curl", "-v", "-s", "-o", "/dev/null", "-w", "%{http_code}",
			"-x", "http://127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		_, out, _ := gostC.Exec(s.ctx, cmd)
		body, _ := io.ReadAll(out)
		output := string(body)
		s.Assert().NotContains(output, "407",
			"probeResist should hide 407 status")
		s.Assert().Contains(output, "404",
			"probeResist should return configured decoy code")
	})

	s.T().Run("with-auth-success", func(t *testing.T) {
		cmd := []string{"curl", "-v", "-s", "-x", "http://user:pass@127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		code, out, err := gostC.Exec(s.ctx, cmd)
		s.Require().NoError(err)

		body, err := io.ReadAll(out)
		s.Require().NoError(err)
		if code != 0 || !strings.Contains(string(body), "hello-gost") {
			DumpLogs(s.T(), s.ctx, "http-metadata logs", gostC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(string(body), "hello-gost")
	})
}

// TestHTTPProxyHeaders verifies:
//   - header: custom response headers (X-Proxy-Info, X-Custom) on
//     proxy-originated error responses. The header metadata is set on
//     the skeleton response used for error paths after authentication.
//   - proxyAgent: custom Proxy-Agent header in proxy responses.
//
// Uses a bypass matcher (0.0.0.0/0) that blocks all traffic after auth
// succeeds → requests hit 403 Forbidden path which carries custom headers.
func (s *HTTPSuite) TestHTTPProxyHeaders() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server_headers.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Auth succeeds, but target is bypassed → 403 Forbidden
	// The 403 response is built from the skeleton resp which
	// carries h.md.header (custom headers) and h.md.proxyAgent.
	s.T().Log("bypass: expect 403 with custom headers")
	cmd := []string{"curl", "-v", "-s", "-D", "-", "-o", "/dev/null",
		"-x", "http://user:pass@127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	_, out, _ := gostC.Exec(s.ctx, cmd)
	headers, _ := io.ReadAll(out)
	output := string(headers)

	// Bypass → 403 Forbidden
	s.Assert().Contains(output, "403",
		"bypass should return 403")
	// header: custom X-Proxy-Info header on error response
	s.Assert().Contains(output, "X-Proxy-Info: gost-e2e",
		"custom header should appear in 403 response")
	// header: custom X-Custom header on error response
	s.Assert().Contains(output, "X-Custom: test-value",
		"custom header should appear in 403 response")
	// proxyAgent: custom Proxy-Agent header on error response
	s.Assert().Contains(output, "Proxy-Agent: gost-e2e/1.0",
		"proxyAgent should appear in 403 response")
}

// TestHTTPConnect verifies HTTP CONNECT tunnel:
//   - sniffing-enabled: sniffing:true + sniffing.timeout:2s
//   - no-sniffing: plain CONNECT (default, no metadata)
//   - bypass-403: CONNECT blocked by (0.0.0.0/0) bypass
//
// Uses curl --proxytunnel (-p) to force CONNECT method instead of GET.
func (s *HTTPSuite) TestHTTPConnect() {
	s.T().Run("sniffing-enabled", func(t *testing.T) {
		gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
			"testdata/http/server_connect.yaml", "8080/tcp")
		s.Require().NoError(err)
		defer gostC.Terminate(s.ctx)

		cmd := []string{"curl", "-v", "-s", "-p", "-x", "http://127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		code, out, err := gostC.Exec(s.ctx, cmd)
		s.Require().NoError(err)

		body, err := io.ReadAll(out)
		s.Require().NoError(err)
		if code != 0 || !strings.Contains(string(body), "hello-gost") {
			DumpLogs(s.T(), s.ctx, "http-connect sniffing logs", gostC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(string(body), "hello-gost")
	})

	s.T().Run("no-sniffing", func(t *testing.T) {
		gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
			"testdata/http/server.yaml", "8080/tcp")
		s.Require().NoError(err)
		defer gostC.Terminate(s.ctx)

		cmd := []string{"curl", "-v", "-s", "-p", "-x", "http://127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		code, out, err := gostC.Exec(s.ctx, cmd)
		s.Require().NoError(err)

		body, err := io.ReadAll(out)
		s.Require().NoError(err)
		if code != 0 || !strings.Contains(string(body), "hello-gost") {
			DumpLogs(s.T(), s.ctx, "http-connect no-sniffing logs", gostC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(string(body), "hello-gost")
	})

	s.T().Run("bypass-403", func(t *testing.T) {
		gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
			"testdata/http/server_connect_bypass.yaml", "8080/tcp")
		s.Require().NoError(err)
		defer gostC.Terminate(s.ctx)

		cmd := []string{"curl", "-v", "-s", "-D", "-", "-o", "/dev/null",
			"-p", "-x", "http://user:pass@127.0.0.1:8080",
			fmt.Sprintf("http://%s:5678", s.echoIP)}
		_, out, _ := gostC.Exec(s.ctx, cmd)
		body, _ := io.ReadAll(out)
		output := string(body)
		s.Assert().Contains(output, "403",
			"CONNECT bypass should return 403")
	})
}

// TestHTTPProxyDirectRequest verifies that a non-proxy-form HTTP request
// (raw GET / sent to the proxy port) is rejected with 400 Bad Request.
// A raw request with empty Host header has no valid scheme to infer,
// so the handler returns 400.
func (s *HTTPSuite) TestHTTPProxyDirectRequest() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	resp := s.sendRaw(gostC, "127.0.0.1", "8080",
		"GET / HTTP/1.0\r\nHost: bad!\r\n\r\n")
	s.Assert().Contains(resp, "400",
		"non-proxy GET should return 400")
}

// TestHTTPProxyPOST verifies that POST method forwarding works through
// the HTTP forward proxy. The echo server only handles GET, so the proxy
// should forward the POST request and return whatever the upstream sends
// back (501 Not Implemented from the echo server — not a proxy error).
// 501 is the real upstream response, proving the proxy forwarded it.
func (s *HTTPSuite) TestHTTPProxyPOST() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"-X", "POST", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678/", s.echoIP)}
	_, out, _ := gostC.Exec(s.ctx, cmd)
	body, _ := io.ReadAll(out)
	output := string(body)
	// The echo server's BaseHTTPRequestHandler defaults to 501 for POST.
	// If we get 501 through the proxy, it means the method was forwarded.
	s.Assert().Contains(output, "501",
		"POST through proxy should be forwarded, echo server returns 501")
}

// TestHTTPConnectUnreachable verifies that CONNECT to an unreachable target
// returns 503 Service Unavailable.
func (s *HTTPSuite) TestHTTPConnectUnreachable() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server_connect.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Send a raw CONNECT request to a port where nothing is listening.
	resp := s.sendRaw(gostC, "127.0.0.1", "8080",
		fmt.Sprintf("CONNECT %s:1 HTTP/1.0\r\nHost: %s:1\r\n\r\n", s.echoIP, s.echoIP))
	s.Assert().Contains(resp, "503",
		"CONNECT to unreachable target should return 503")
}

// TestHTTPConnector verifies HTTP connector type: a gost client uses an HTTP
// connector to forward through an upstream HTTP proxy (no auth), reaching the
// target via the proxy chain. This covers the server+client two-container
// pattern where the client connects to the server via connector type: http.
func (s *HTTPSuite) TestHTTPConnector() {
	// Start upstream HTTP proxy server (no auth).
	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/http/server.yaml", []string{"http-server"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	// Render client config that chains through the upstream server.
	rendered, err := RenderConfig("testdata/http/client_connector.yaml",
		ConfigData{ServerAddr: "http-server:8080"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	// Start client with the rendered config.
	clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		rendered, "8080/tcp")
	s.Require().NoError(err)
	defer clientC.Terminate(s.ctx)

	// Request through client proxy → HTTP connector → upstream HTTP server → target.
	cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := clientC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "http-connector client logs", clientC)
		DumpLogs(s.T(), s.ctx, "http-connector server logs", serverC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestHTTPConnectorAuth verifies HTTP connector with authentication:
// no-auth → 407, correct auth → success.
func (s *HTTPSuite) TestHTTPConnectorAuth() {
	// Start upstream HTTP proxy with auth and network alias.
	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/http/server_auth.yaml", []string{"http-server"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	s.T().Run("no-auth-407", func(t *testing.T) {
		// Send raw request directly to server container (no auth) → 407.
		resp := s.sendRaw(serverC, "127.0.0.1", "8080",
			"GET http://127.0.0.1:5678/ HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n")
		s.Assert().Contains(resp, "407",
			"no auth via upstream should return 407")
	})

	s.T().Run("correct-auth-success", func(t *testing.T) {
		rendered, err := RenderConfig("testdata/http/client_connector_auth.yaml",
			ConfigData{ServerAddr: "http-server:8080"})
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
			DumpLogs(s.T(), s.ctx, "http-connector-auth client logs", clientC)
			DumpLogs(s.T(), s.ctx, "http-connector-auth server logs", serverC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(string(body), "hello-gost")
	})
}

// TestHTTPProbeResistHost verifies probeResist host type.
// When auth fails, the handler pipes the raw request to
// the decoy host (tcp-echo:5678). The TCP echo server echoes
// back the request bytes rather than returning a 407.
func (s *HTTPSuite) TestHTTPProbeResistHost() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server_proberesist_host.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Without auth, probeResist host pipes the raw request to tcp-echo:5678.
	// The HTTP echo server returns 200 OK with "hello-gost", proving the
	// request reached the decoy host rather than getting a 407 response.
	resp := s.sendRaw(gostC, "127.0.0.1", "8080",
		"GET http://127.0.0.1:5678/ HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n")
	s.Assert().NotContains(resp, "407",
		"probeResist host should hide 407 response")
	s.Assert().Contains(resp, "hello-gost",
		"probeResist host should pipe request to decoy host")
}

// TestHTTPProbeResistKnock verifies probeResist knock mechanism.
// Clients that connect with a recognized knock Host header
// get a normal 407 auth challenge instead of the decoy response.
//
// Uses server_proberesist_knock.yaml:
//   - probeResist: code:404 (decoy 404 for unknown hosts)
//   - knock: secret.example.com (known hosts get normal 407)
func (s *HTTPSuite) TestHTTPProbeResistKnock() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server_proberesist_knock.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	s.T().Run("unknown-host-decoy", func(t *testing.T) {
		// URL hostname doesn't match knock list → probeResist fires → 404
		resp := s.sendRaw(gostC, "127.0.0.1", "8080",
			"GET http://127.0.0.1:5678/ HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n")
		s.Assert().Contains(resp, "404",
			"unknown host should get decoy 404")
		s.Assert().NotContains(resp, "407",
			"unknown host should NOT get 407")
	})

	s.T().Run("knock-host-407", func(t *testing.T) {
		// URL hostname matches knock list → normal 407 auth challenge.
		// Note: knock checks req.URL.Hostname(), not the Host header.
		resp := s.sendRaw(gostC, "127.0.0.1", "8080",
			"GET http://secret.example.com:5678/ HTTP/1.0\r\nHost: secret.example.com\r\n\r\n")
		s.Assert().Contains(resp, "407",
			"knock host should get normal 407")
	})
}

// TestHTTPConnectorTLS verifies HTTP connector using TLS dialer
// to upstream HTTPS proxy (listener type: tls).
// The TLS listener auto-generates a self-signed cert;
// the TLS dialer defaults to InsecureSkipVerify=true, so
// handshake succeeds without explicit cert configuration.
func (s *HTTPSuite) TestHTTPConnectorTLS() {
	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/http/server_tls.yaml", []string{"tls-server"}, []string{"8443/tcp"})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	rendered, err := RenderConfig("testdata/http/client_connector_tls.yaml",
		ConfigData{ServerAddr: "tls-server:8443"})
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
		DumpLogs(s.T(), s.ctx, "http-connector-tls client logs", clientC)
		DumpLogs(s.T(), s.ctx, "http-connector-tls server logs", serverC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestHTTPProbeResistWeb verifies probeResist web type.
// On auth failure, the handler fetches the decoy URL
// (tcp-echo:5678) and returns its response body.
func (s *HTTPSuite) TestHTTPProbeResistWeb() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http/server_proberesist_web.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Without auth, probeResist web fetches http://tcp-echo:5678
	// and returns the echo server's response ("hello-gost").
	resp := s.sendRaw(gostC, "127.0.0.1", "8080",
		"GET http://127.0.0.1:5678/ HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n")
	s.Assert().NotContains(resp, "407",
		"probeResist web should hide 407 response")
	s.Assert().Contains(resp, "hello-gost",
		"probeResist web should return decoy response body")
}

// TestHTTPProbeResistFile verifies probeResist file type.
// On auth failure, the handler reads a local file and
// returns its contents as the HTTP response body.
func (s *HTTPSuite) TestHTTPProbeResistFile() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/http/server_proberesist_file.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "testdata/http/decoy.html", ContainerFilePath: "/tmp/decoy.html", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Without auth, probeResist file reads /tmp/decoy.html and
	// returns "decoy-response" rather than a 407.
	resp := s.sendRaw(gostC, "127.0.0.1", "8080",
		"GET http://127.0.0.1:5678/ HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n")
	s.Assert().NotContains(resp, "407",
		"probeResist file should hide 407 response")
	s.Assert().Contains(resp, "decoy-response",
		"probeResist file should return decoy file content")
}

// TestHTTPIdleTimeout verifies idleTimeout on CONNECT tunnels.
// After the configured idle timeout, the pipe between client
// and target should close.
func (s *HTTPSuite) TestHTTPIdleTimeout() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/http/server_idle_timeout.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/http_idle_timeout.py", ContainerFilePath: "/scripts/http_idle_timeout.py", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// The Python script:
	// 1. Opens a CONNECT tunnel to tcp-echo:5678
	// 2. Sends and receives data (confirm tunnel is alive)
	// 3. Waits > idleTimeout (3s + 2s buffer)
	// 4. Sends more data — expects connection to be closed
	code, out, err := gostC.Exec(s.ctx, []string{
		"python3", "/scripts/http_idle_timeout.py",
		"127.0.0.1", "8080", "3",
	})
	output, _ := io.ReadAll(out)
	s.T().Logf("idle timeout output:\n%s", string(output))
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "http-idle-timeout logs", gostC)
	}
	s.Require().Equal(0, code, "idle timeout test script should exit 0")
}

// TestHTTPUDPRelay verifies UDP relay over HTTP. Uses
// X-Gost-Protocol: udp in the CONNECT request to establish
// a SOCKS5 UDP tunnel through the HTTP handler.
func (s *HTTPSuite) TestHTTPUDPRelay() {
	// Start UDP echo container on the shared network.
	udpC, err := RunUDPEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	defer udpC.Terminate(s.ctx)

	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/http/server_udp.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/http_udp_relay.py", ContainerFilePath: "/scripts/http_udp_relay.py", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// The Python script:
	// 1. Connects to gost HTTP proxy
	// 2. Sends CONNECT with X-Gost-Protocol: udp
	// 3. After 200 OK, sends a SOCKS5 UDP frame targeting udp-echo:5679
	// 4. Reads back the echoed frame
	code, out, err := gostC.Exec(s.ctx, []string{
		"python3", "/scripts/http_udp_relay.py",
		"127.0.0.1", "8080",
	})
	output, _ := io.ReadAll(out)
	s.T().Logf("udp relay output:\n%s", string(output))
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "http-udp logs", gostC)
	}
	s.Require().Equal(0, code, "udp relay test script should exit 0")
}

func TestHTTPSuite(t *testing.T) {
	suite.Run(t, new(HTTPSuite))
}
