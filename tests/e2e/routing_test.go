package e2e

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// RoutingSuite verifies node-level routing matchers. A forward proxy's only
// chain node relays to an upstream proxy, but the node carries a matcher
// Host(`tcp-echo`) so it is only eligible for requests whose Host matches. A
// request to tcp-echo:5678 is matched, relayed upstream, and reaches the echo
// server ("hello-gost"); a request to a non-matching host is excluded (no
// eligible node) and never reaches the echo server.
type RoutingSuite struct {
	suite.Suite
	ctx        context.Context
	echoC      testcontainers.Container
	proxyC     testcontainers.Container
	upstreamC  testcontainers.Container
}

func (s *RoutingSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	proxyC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/routing/server.yaml", []string{"gost-proxy"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	s.proxyC = proxyC

	upstreamC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/routing/upstream.yaml", []string{"gost-upstream"}, []string{"8081/tcp"})
	s.Require().NoError(err)
	s.upstreamC = upstreamC
}

func (s *RoutingSuite) TearDownSuite() {
	for _, c := range []testcontainers.Container{s.proxyC, s.upstreamC, s.echoC} {
		if c != nil {
			c.Terminate(s.ctx)
		}
	}
}

// proxyRequest sends an HTTP GET to target through the matcher-gated proxy and
// returns the response body.
func (s *RoutingSuite) proxyRequest(target string) string {
	cmd := []string{
		"curl", "-s",
		"-x", "http://gost-proxy:8080",
		target,
	}
	_, out, err := s.echoC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestMatchedHost verifies that a request whose Host matches the node matcher is
// relayed upstream and reaches the echo server.
func (s *RoutingSuite) TestMatchedHost() {
	s.Require().Contains(s.proxyRequest("http://tcp-echo:5678/"), "hello-gost")
}

// TestUnmatchedHost verifies that a request whose Host does not match the node
// matcher is excluded (no eligible node) and never reaches the echo server.
func (s *RoutingSuite) TestUnmatchedHost() {
	s.Require().NotContains(s.proxyRequest("http://other.local:5678/"), "hello-gost")
}

func TestRoutingSuite(t *testing.T) {
	suite.Run(t, new(RoutingSuite))
}
