package e2e

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// HTTPPolicyBypassSuite reproduces the destination-policy bypass where a valid
// Gost-Target header replaces req.Host (evaluated by the bypass) but, before the
// fix, left req.URL.Host (dialed by the transport) as the absolute-URL host.
// The echo server stands in for the protected backend.
type HTTPPolicyBypassSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *HTTPPolicyBypassSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *HTTPPolicyBypassSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// sendRaw sends a raw HTTP request to the proxy inside the gost container.
func (s *HTTPPolicyBypassSuite) sendRaw(gostC testcontainers.Container, data string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	cmd := []string{"sh", "-c",
		fmt.Sprintf("echo %s | base64 -d | nc -w 5 127.0.0.1 8080", encoded)}
	_, out, _ := gostC.Exec(s.ctx, cmd)
	b, _ := io.ReadAll(out)
	return string(b)
}

// encodeTarget encodes a host:port in the GOST v2 Gost-Target format:
// raw-URL-base64( big-endian-CRC32(name) + raw-URL-base64(name) ).
func encodeTarget(name string) string {
	v := []byte(name)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, crc32.ChecksumIEEE(v))
	inner := base64.RawURLEncoding.EncodeToString(v)
	return base64.RawURLEncoding.EncodeToString(append(b, []byte(inner)...))
}

// TestGostTargetPolicyBypass asserts that a Gost-Target header naming an
// allowed authority cannot route a request to a blocked absolute-URL authority.
func (s *HTTPPolicyBypassSuite) TestGostTargetPolicyBypass() {
	cfg, err := RenderConfig("testdata/http/server_policy_bypass.yaml", ConfigData{ServerAddr: s.echoIP})
	s.Require().NoError(err)
	defer os.Remove(cfg)

	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, cfg, "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	target := fmt.Sprintf("http://%s:5678/", s.echoIP)
	marker := "hello-gost"

	// Control: absolute-form request to the blocked address, no target header.
	// The bypass must deny it before it reaches the echo server.
	control := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s:5678\r\nConnection: close\r\n\r\n",
		target, s.echoIP)
	out := s.sendRaw(gostC, control)
	s.Require().Contains(out, "403", "control request to blocked address should be denied")
	s.Require().NotContains(out, marker, "control request must not reach the backend")

	// Exploit: same request plus a valid Gost-Target header naming an allowed
	// authority. Before the fix the transport dials the URL host (echo server)
	// while the policy checks the header host, leaking the backend response.
	exploit := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s:5678\r\nGost-Target: %s\r\nConnection: close\r\n\r\n",
		target, s.echoIP, encodeTarget("allowed.example.com:80"))
	out = s.sendRaw(gostC, exploit)
	s.Require().NotContains(out, marker,
		"Gost-Target header must not bypass the destination policy to reach the blocked backend")

	// Fail-closed control: a malformed target header is ignored, so the request
	// is still evaluated (and denied) against the absolute-URL authority.
	malformed := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s:5678\r\nGost-Target: %s\r\nConnection: close\r\n\r\n",
		target, s.echoIP, "not-a-valid-target")
	out = s.sendRaw(gostC, malformed)
	s.Require().Contains(out, "403", "malformed target header must be ignored and the request denied")
	s.Require().NotContains(out, marker, "malformed target header must not reach the backend")
}

func TestHTTPPolicyBypassSuite(t *testing.T) {
	suite.Run(t, new(HTTPPolicyBypassSuite))
}
