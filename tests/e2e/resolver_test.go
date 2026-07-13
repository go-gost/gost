package e2e

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// ResolverSuite verifies that a configured resolver drives the proxy's
// outbound DNS resolution. A gost HTTP proxy uses a resolver whose only
// nameserver is a test responder: it answers echo.test with the real echo
// server IP and NXDOMAIN for everything else. A request to echo.test is
// resolved by the custom resolver and reaches the echo server; a request to an
// unmapped host fails to resolve.
type ResolverSuite struct {
	suite.Suite
	ctx     context.Context
	echoC   testcontainers.Container
	echoIP  string
	dnsC    testcontainers.Container
	proxyC  testcontainers.Container
}

func (s *ResolverSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP

	dnsC, err := RunResolverResponderContainer(s.ctx, SharedNetworkName, s.echoIP)
	s.Require().NoError(err)
	s.dnsC = dnsC

	proxyC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/resolver/server.yaml", []string{"gost-proxy"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	s.proxyC = proxyC
}

func (s *ResolverSuite) TearDownSuite() {
	for _, c := range []testcontainers.Container{s.proxyC, s.dnsC, s.echoC} {
		if c != nil {
			c.Terminate(s.ctx)
		}
	}
}

// proxyRequest sends an HTTP GET to the given host through the gost proxy. When
// waitResolve is true it retries until the resolver/responder is ready (the
// resolved-host happy path); otherwise it makes a single attempt. Returns the
// response body.
func (s *ResolverSuite) proxyRequest(host string, waitResolve bool) string {
	loop := "for i in $(seq 1 30); do "
	if !waitResolve {
		loop = "for i in 1; do "
	}
	cmd := []string{
		"sh", "-c",
		fmt.Sprintf("%sbody=$(curl -s -x http://gost-proxy:8080 http://%s:5678/); echo \"$body\" | grep -q hello-gost && { echo \"$body\"; exit 0; }; sleep 1; done; echo \"$body\"", loop, host),
	}
	_, out, err := s.echoC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestResolvedHostname verifies that the custom resolver resolves echo.test to
// the echo server IP, so the proxied request reaches the echo server.
func (s *ResolverSuite) TestResolvedHostname() {
	s.Require().Contains(s.proxyRequest("echo.test", true), "hello-gost")
}

// TestUnresolvedHostname verifies that a host absent from the resolver gets
// NXDOMAIN and never reaches the echo server.
func (s *ResolverSuite) TestUnresolvedHostname() {
	s.Require().NotContains(s.proxyRequest("nomapped.test", false), "hello-gost")
}

func TestResolverSuite(t *testing.T) {
	suite.Run(t, new(ResolverSuite))
}
