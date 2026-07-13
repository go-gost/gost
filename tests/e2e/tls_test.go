package e2e

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// TLSSuite covers TLS listener behavior, in particular the rejectUnknownSNI
// option which drops TLS handshakes with an unknown or empty SNI before any
// certificate is sent.
type TLSSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *TLSSuite) SetupSuite() {
	s.ctx = context.Background()
}

// startGost starts a gost container with the given TLS listener config. The
// listener always binds :8443; checks run openssl inside the container.
func (s *TLSSuite) startGost(yamlPath string) testcontainers.Container {
	s.T().Helper()
	rendered, err := RenderConfig(yamlPath, ConfigData{})
	s.Require().NoError(err)
	s.T().Cleanup(func() { os.Remove(rendered) })

	c, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, rendered, "8443/tcp")
	s.Require().NoError(err)
	return c
}

// sniHandshake runs openssl s_client against the TLS listener and returns its
// exit code: 0 means the handshake completed (a certificate was exchanged),
// non-zero means the handshake was rejected. An empty sni sends no SNI
// extension.
func (s *TLSSuite) sniHandshake(c testcontainers.Container, sni string) int {
	s.T().Helper()
	args := []string{"openssl", "s_client", "-brief", "-connect", "127.0.0.1:8443"}
	if sni == "" {
		args = append(args, "-noservername")
	} else {
		args = append(args, "-servername", sni)
	}
	code, out, err := c.Exec(s.ctx, args)
	if err != nil {
		s.T().Logf("openssl exec error: %v", err)
		return -1
	}
	if b, e := io.ReadAll(out); e == nil {
		s.T().Logf("openssl output (sni=%q):\n%s", sni, string(b))
	}
	return code
}

// TestRejectWithAllowList covers rejectUnknownSNI with a populated serverNames
// list: the allowed SNI completes the handshake, while a wrong or missing SNI
// is rejected.
func (s *TLSSuite) TestRejectWithAllowList() {
	c := s.startGost("testdata/tls/reject_sni.yaml")
	defer c.Terminate(s.ctx)

	s.Require().Zero(s.sniHandshake(c, "example.com"),
		"handshake with allowed SNI should succeed")

	s.Require().NotZero(s.sniHandshake(c, "wrong.com"),
		"handshake with disallowed SNI should fail")

	s.Require().NotZero(s.sniHandshake(c, ""),
		"handshake without SNI should fail")
}

// TestRejectEmptyOnly covers rejectUnknownSNI with an empty serverNames list:
// only a missing SNI is rejected, any named SNI is allowed.
func (s *TLSSuite) TestRejectEmptyOnly() {
	c := s.startGost("testdata/tls/reject_sni_empty.yaml")
	defer c.Terminate(s.ctx)

	s.Require().NotZero(s.sniHandshake(c, ""),
		"handshake without SNI should fail")

	s.Require().Zero(s.sniHandshake(c, "anything.test"),
		"named SNI should be allowed when allow list is empty")
}

func TestTLSSuite(t *testing.T) {
	suite.Run(t, new(TLSSuite))
}
