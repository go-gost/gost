package e2e

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// HTTP2Suite covers the HTTP/2 proxy handler (handler type: http2) together
// with the HTTP/2 listener (listener type: http2). The http2 listener wraps
// the underlying TCP listener with TLS and configures an h2 server, so the
// handler is always reached over HTTP/2 frames.
//
// Because Alpine's curl cannot be relied on to speak HTTP/2 proxy to the
// server directly, the suite uses the canonical GOST chaining pattern: a
// client container exposes a plain HTTP proxy which forwards through an
// http2 connector + http2 dialer to the http2 server. This exercises the
// h2 listener, the h2 handler, and the h2 connector/dialer together.
type HTTP2Suite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *HTTP2Suite) SetupSuite() {
	s.ctx = context.Background()

	s.T().Logf("start tcp echo container...")
	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *HTTP2Suite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// startChain brings up an http2 server (alias "h2-server") and an http client
// container that chains to it through connector/dialer type http2. The client
// exposes port 8080 for curl. The rendered client config is cleaned up by the
// caller-supplied template path resolving {{.ServerAddr}} to h2-server:8443.
func (s *HTTP2Suite) startChain(serverYAML, clientTmpl string) (testcontainers.Container, testcontainers.Container) {
	s.T().Helper()

	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		serverYAML, []string{"h2-server"}, []string{"8443/tcp"})
	s.Require().NoError(err)

	rendered, err := RenderConfig(clientTmpl, ConfigData{ServerAddr: "h2-server:8443"})
	s.Require().NoError(err)
	s.T().Cleanup(func() { os.Remove(rendered) })

	clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		rendered, "8080/tcp")
	s.Require().NoError(err)

	return serverC, clientC
}

// curlEcho runs curl through the client's local http proxy and returns the
// process exit code plus the captured body.
func (s *HTTP2Suite) curlEcho(clientC testcontainers.Container) (int, string) {
	s.T().Helper()
	cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, _ := clientC.Exec(s.ctx, cmd)
	body, _ := io.ReadAll(out)
	return code, string(body)
}

func (s *HTTP2Suite) dump(label string, cs ...testcontainers.Container) {
	for _, c := range cs {
		DumpLogs(s.T(), s.ctx, label, c)
	}
}

// TestHTTP2ForwardProxy verifies the core HTTP/2 proxy path: a plain HTTP
// request through the client proxy is tunneled to the http2 server via an h2
// CONNECT stream and reaches the echo backend.
//
// Covers:
//   - listener type: http2 (TLS + h2 server)
//   - handler type: http2 (CONNECT tunnel + bidirectional pipe)
//   - connector type: http2, dialer type: http2 (h2 client)
func (s *HTTP2Suite) TestHTTP2ForwardProxy() {
	serverC, clientC := s.startChain("testdata/http2/server.yaml", "testdata/http2/client.yaml")
	defer serverC.Terminate(s.ctx)
	defer clientC.Terminate(s.ctx)

	code, body := s.curlEcho(clientC)
	if code != 0 || !strings.Contains(body, "hello-gost") {
		s.dump("http2-forward logs", clientC, serverC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(body, "hello-gost")
}

// TestHTTP2Auth verifies proxy authentication over the h2 tunnel, plus
// authBasicRealm and hash metadata parsing on the http2 handler.
//
//   - with auth:    CONNECT succeeds → echo body returned
//   - without auth: server rejects CONNECT (407) → client cannot reach echo
func (s *HTTP2Suite) TestHTTP2Auth() {
	s.T().Run("with-auth-success", func(t *testing.T) {
		serverC, clientC := s.startChain("testdata/http2/server_auth.yaml", "testdata/http2/client_auth.yaml")
		defer serverC.Terminate(s.ctx)
		defer clientC.Terminate(s.ctx)

		code, body := s.curlEcho(clientC)
		if code != 0 || !strings.Contains(body, "hello-gost") {
			s.dump("http2-auth-success logs", clientC, serverC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(body, "hello-gost")
	})

	s.T().Run("no-auth-fails", func(t *testing.T) {
		// Server requires auth; client sends none → the http2 CONNECT is
		// rejected with 407, so curl cannot obtain the echo body.
		serverC, clientC := s.startChain("testdata/http2/server_auth.yaml", "testdata/http2/client.yaml")
		defer serverC.Terminate(s.ctx)
		defer clientC.Terminate(s.ctx)

		code, body := s.curlEcho(clientC)
		if strings.Contains(body, "hello-gost") {
			s.dump("http2-auth-noauth logs", clientC, serverC)
		}
		s.Require().Zero(code,
			"curl should still receive an HTTP response from the local proxy")
		s.Require().NotContains(body, "hello-gost",
			"without auth the echo backend must be unreachable")
	})
}

// TestHTTP2Bypass verifies that a bypass matcher on the http2 handler blocks
// the CONNECT target after authentication succeeds. The server returns 403, so
// the client cannot reach the echo backend.
func (s *HTTP2Suite) TestHTTP2Bypass() {
	serverC, clientC := s.startChain("testdata/http2/server_bypass.yaml", "testdata/http2/client_auth.yaml")
	defer serverC.Terminate(s.ctx)
	defer clientC.Terminate(s.ctx)

	code, body := s.curlEcho(clientC)
	if strings.Contains(body, "hello-gost") {
		s.dump("http2-bypass logs", clientC, serverC)
	}
	s.Require().Zero(code,
		"curl should still receive an HTTP response from the local proxy")
	s.Require().NotContains(body, "hello-gost",
		"bypass should prevent reaching the echo backend")
}

// TestHTTP2ProbeResist verifies that probeResist, header and authBasicRealm
// metadata parse cleanly on the http2 handler and do not break the success
// path. With correct credentials the request still reaches the echo backend.
func (s *HTTP2Suite) TestHTTP2ProbeResist() {
	serverC, clientC := s.startChain("testdata/http2/server_proberesist.yaml", "testdata/http2/client_auth.yaml")
	defer serverC.Terminate(s.ctx)
	defer clientC.Terminate(s.ctx)

	code, body := s.curlEcho(clientC)
	if code != 0 || !strings.Contains(body, "hello-gost") {
		s.dump("http2-proberesist logs", clientC, serverC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(body, "hello-gost")
}

// TestHTTP2Multiplex verifies HTTP/2 stream multiplexing: many concurrent
// requests through the client proxy are tunneled as parallel h2 streams over a
// single underlying connection to the http2 server. All requests must reach
// the echo backend.
func (s *HTTP2Suite) TestHTTP2Multiplex() {
	serverC, clientC := s.startChain("testdata/http2/server.yaml", "testdata/http2/client.yaml")
	defer serverC.Terminate(s.ctx)
	defer clientC.Terminate(s.ctx)

	const n = 6
	var wg sync.WaitGroup
	wg.Add(n)
	failed := make(chan string, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, body := s.curlEcho(clientC)
			if !strings.Contains(body, "hello-gost") {
				failed <- "missing echo body"
			}
		}()
	}
	wg.Wait()
	close(failed)

	if len(failed) > 0 {
		s.dump("http2-multiplex logs", clientC, serverC)
	}
	s.Require().Empty(failed, "all concurrent h2 requests should reach the echo backend")
}

func TestHTTP2Suite(t *testing.T) {
	suite.Run(t, new(HTTP2Suite))
}
