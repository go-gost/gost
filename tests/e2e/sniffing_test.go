package e2e

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// SniffingSuite covers protocol sniffing behavior.
type SniffingSuite struct {
	suite.Suite
	ctx      context.Context
	httpsEchoC testcontainers.Container
	httpsIP   string
}

func (s *SniffingSuite) SetupSuite() {
	s.ctx = context.Background()

	c, err := RunHTTPSEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.httpsEchoC = c

	ip, err := c.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.httpsIP = ip
}

func (s *SniffingSuite) TearDownSuite() {
	if s.httpsEchoC != nil {
		s.httpsEchoC.Terminate(s.ctx)
	}
}

// curlThroughSOCKS5 runs curl through the GOST SOCKS5 proxy on the given
// container address and port, connecting to the HTTPS echo server by IP.
// Connecting by IP makes curl omit the TLS SNI extension.
func (s *SniffingSuite) curlThroughSOCKS5(c testcontainers.Container, proxyPort string) string {
	s.T().Helper()
	// curl -k: don't verify the self-signed cert on the echo server
	// curl -x socks5://...: route through GOST SOCKS5 proxy
	// Using the container IP as the host makes curl skip SNI entirely.
	cmd := []string{"curl", "-k", "-s",
		"-x", "socks5://127.0.0.1:" + proxyPort,
		"https://" + s.httpsIP + ":8443",
	}
	code, out, err := c.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	s.Require().Zero(code, "curl should exit 0")
	b, _ := io.ReadAll(out)
	return string(b)
}

// TestSOCKS5NoSNI verifies that a SOCKS5 proxy with sniffing enabled can
// successfully forward TLS connections that lack an SNI extension — such as
// when a client connects to an HTTPS server by IP address.
func (s *SniffingSuite) TestSOCKS5NoSNI() {
	gostC, err := RunGostContainer(s.ctx, SharedNetworkName,
		"testdata/sniffing/no_sni.yaml",
	)
	s.Require().NoError(err)
	defer func() {
		DumpLogs(s.T(), s.ctx, "gost-no-sni", gostC)
		gostC.Terminate(s.ctx)
	}()

	body := s.curlThroughSOCKS5(gostC, "8080")
	s.Require().Contains(body, "hello-gost",
		"HTTPS response should pass through when SNI is empty")
}

// TestSOCKS5NoSNI_Callback verifies the same behavior a second time, serving
// as a sanity check that state doesn't leak between connections.
func (s *SniffingSuite) TestSOCKS5NoSNI_Callback() {
	s.TestSOCKS5NoSNI()
}

// TestSOCKS5NoSNI_LogsOnlyDebug verifies the proxy logs a debug message
// rather than an error when SNI is missing.
func (s *SniffingSuite) TestSOCKS5NoSNI_LogsOnlyDebug() {
	gostC, err := RunGostContainer(s.ctx, SharedNetworkName,
		"testdata/sniffing/no_sni.yaml",
	)
	s.Require().NoError(err)
	defer func() {
		// Do NOT dump logs before the grep — DumpLogs consumes the reader
		gostC.Terminate(s.ctx)
	}()

	body := s.curlThroughSOCKS5(gostC, "8080")
	s.Require().Contains(body, "hello-gost")

	// Verify the debug log message exists and no error-level SNI message.
	logs, err := gostC.Logs(s.ctx)
	s.Require().NoError(err)
	defer logs.Close()
	logBody, _ := io.ReadAll(logs)
	logStr := string(logBody)

	s.Require().Contains(logStr, "no sni in clienthello",
		"should log a debug message when SNI is empty")
	s.Require().NotContains(logStr, "tls: sni is empty",
		"must not error on empty SNI")
	// The log level for "no sni" should be debug, not error.
	s.Require().False(strings.Contains(logStr, `"level":"error"`),
		"should not log any error-level message")
}

func TestSniffingSuite(t *testing.T) {
	suite.Run(t, new(SniffingSuite))
}
