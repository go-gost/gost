package e2e

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// IngressSuite verifies hostname→endpoint routing at the reverse-proxy tunnel
// entrypoint. A public gost runs a tunnel handler with an ingress mapping
// example.local→tunnel-UUID and an HTTP entrypoint on :8420. An internal client
// binds a reverse tunnel (tunnel.id=UUID) that forwards to the echo server.
//
// A request to the entrypoint with Host:example.local is routed via the ingress
// table to the tunnel and reaches the echo server ("hello-gost"); a request
// with an unmapped Host matches no ingress rule and is rejected with no route.
type IngressSuite struct {
	suite.Suite
	ctx     context.Context
	echoC   testcontainers.Container
	serverC testcontainers.Container
	clientC testcontainers.Container
}

func (s *IngressSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/ingress/server.yaml", []string{"gost-server"}, []string{"8420/tcp"})
	s.Require().NoError(err)
	s.serverC = serverC

	clientC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/ingress/client.yaml", []string{"gost-client"}, []string{"8423/tcp"})
	s.Require().NoError(err)
	s.clientC = clientC
}

func (s *IngressSuite) TearDownSuite() {
	for _, c := range []testcontainers.Container{s.clientC, s.serverC, s.echoC} {
		if c != nil {
			c.Terminate(s.ctx)
		}
	}
}

// request sends an HTTP GET to the server entrypoint with the given Host
// header. When waitTunnel is true it retries until the reverse tunnel is bound
// (the mapped-host happy path); otherwise it makes a single attempt (the
// unmapped-host case, which should never reach the echo server). Returns the
// response body.
func (s *IngressSuite) request(host string, waitTunnel bool) string {
	loop := "for i in $(seq 1 30); do "
	if !waitTunnel {
		loop = "for i in 1; do "
	}
	cmd := []string{
		"sh", "-c",
		fmt.Sprintf("%sbody=$(curl -s -H 'Host: %s' http://gost-server:8420/); echo \"$body\" | grep -q hello-gost && { echo \"$body\"; exit 0; }; sleep 1; done; echo \"$body\"", loop, host),
	}
	_, out, err := s.echoC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestMappedHostname verifies that a Host matching an ingress rule is routed
// through the tunnel to the echo server.
func (s *IngressSuite) TestMappedHostname() {
	s.Require().Contains(s.request("example.local", true), "hello-gost")
}

// TestUnmappedHostname verifies that a Host with no ingress rule is rejected
// (no route to host) and never reaches the echo server.
func (s *IngressSuite) TestUnmappedHostname() {
	s.Require().NotContains(s.request("nomapped.local", false), "hello-gost")
}

func TestIngressSuite(t *testing.T) {
	suite.Run(t, new(IngressSuite))
}
