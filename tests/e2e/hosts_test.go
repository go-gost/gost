package e2e

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// HostsSuite verifies the static hosts mapping (HostMapper), which overrides
// DNS for matched hostnames. A mapped hostname resolves to the configured IP
// and reaches the echo server; an unmapped hostname fails to resolve.
type HostsSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
	gostC  testcontainers.Container
}

func (s *HostsSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP

	cfg, err := RenderConfig("testdata/hosts/hosts.yaml", ConfigData{ServerAddr: s.echoIP})
	s.Require().NoError(err)
	defer os.Remove(cfg)

	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, cfg, "8080/tcp")
	s.Require().NoError(err)
	s.gostC = gostC
}

func (s *HostsSuite) TearDownSuite() {
	if s.gostC != nil {
		s.gostC.Terminate(s.ctx)
	}
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// proxyRequest sends a request to the given hostname through the gost proxy and
// returns the response body. A resolvable host that reaches the echo server
// returns "hello-gost"; an unresolvable host returns an empty body.
func (s *HostsSuite) proxyRequest(host string) string {
	cmd := []string{
		"curl", "-s",
		"-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", host),
	}
	_, out, err := s.gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestMappedHostname verifies that a hostname in the hosts mapping resolves to
// the configured IP and reaches the echo server.
func (s *HostsSuite) TestMappedHostname() {
	s.Require().Contains(s.proxyRequest("echo.internal"), "hello-gost")
}

// TestUnmappedHostname verifies that a hostname absent from the mapping fails to
// resolve, so the request never reaches the echo server.
func (s *HostsSuite) TestUnmappedHostname() {
	s.Require().NotContains(s.proxyRequest("nomap.internal"), "hello-gost")
}

func TestHostsSuite(t *testing.T) {
	suite.Run(t, new(HostsSuite))
}
