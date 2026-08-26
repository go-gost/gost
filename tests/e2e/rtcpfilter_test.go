package e2e

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// RTCPFilterSuite reproduces go-gost/gost#898: multiple TCP services sharing
// ONE tunnel ID, routed at the client (rtcp) side by forwarder.nodes[].filter.host.
//
// The server ingress maps two hostnames to the same tunnel endpoint. The client
// rtcp service has two forwarder nodes, both filtered by host, with NO fallback
// node. When the hostname is correctly propagated through the tunnel, a request
// for example.local is routed to the example node (tcp-echo:5678). When the
// hostname is lost (the regression), neither node matches and the connection
// fails with "node not available".
type RTCPFilterSuite struct {
	suite.Suite
	ctx     context.Context
	echoC   testcontainers.Container
	serverC testcontainers.Container
	clientC testcontainers.Container
}

func (s *RTCPFilterSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/rtcpfilter/server.yaml", []string{"gost-server"}, []string{"8420/tcp"})
	s.Require().NoError(err)
	s.serverC = serverC

	clientC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/rtcpfilter/client.yaml", []string{"gost-client"}, []string{"8423/tcp"})
	s.Require().NoError(err)
	s.clientC = clientC
}

func (s *RTCPFilterSuite) TearDownSuite() {
	for _, c := range []testcontainers.Container{s.clientC, s.serverC, s.echoC} {
		if c != nil {
			c.Terminate(s.ctx)
		}
	}
}

// request sends an HTTP GET to the server entrypoint with the given Host
// header, retrying until the reverse tunnel is bound. Returns the response body.
func (s *RTCPFilterSuite) request(host string) string {
	cmd := []string{
		"sh", "-c",
		fmt.Sprintf("for i in $(seq 1 30); do body=$(curl -s -H 'Host: %s' http://gost-server:8420/); echo \"$body\" | grep -q hello-gost && { echo \"$body\"; exit 0; }; sleep 1; done; echo \"$body\"", host),
	}
	_, out, err := s.echoC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestMappedHostnameFiltered verifies that a Host matching a filter.host node is
// routed through the tunnel to that node's backend. This is the gost#898 failure:
// with the regression, the hostname is dropped and no node matches.
func (s *RTCPFilterSuite) TestMappedHostnameFiltered() {
	body := s.request("example.local")
	if !strings.Contains(body, "hello-gost") {
		DumpLogs(s.T(), s.ctx, "rtcp client logs", s.clientC)
		DumpLogs(s.T(), s.ctx, "tunnel server logs", s.serverC)
	}
	s.Require().Contains(body, "hello-gost")
}

func TestRTCPFilterSuite(t *testing.T) {
	suite.Run(t, new(RTCPFilterSuite))
}
