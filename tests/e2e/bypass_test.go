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

type BypassSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *BypassSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *BypassSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// proxyRequest sends one request through the gost proxy and returns the
// response body. A bypassed (direct) request that reaches the echo server
// returns "hello-gost"; a request forced through the dead chain node returns
// a 503 and no echo body.
func (s *BypassSuite) proxyRequest(gostC testcontainers.Container, port, target string) string {
	cmd := []string{
		"curl", "-s",
		"-x", fmt.Sprintf("http://127.0.0.1:%s", port),
		target,
	}
	_, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestBlacklistBypassDirect verifies that a destination matching a blacklist
// bypass rule skips the (dead) chain node and connects directly to the echo
// server, so the request succeeds despite the dead node.
func (s *BypassSuite) TestBlacklistBypassDirect() {
	cfg, err := RenderConfig("testdata/bypass/blacklist.yaml", ConfigData{ServerAddr: s.echoIP})
	s.Require().NoError(err)
	defer os.Remove(cfg)

	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, cfg, "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.proxyRequest(gostC, "8080", fmt.Sprintf("http://%s:5678", s.echoIP))
	s.Require().Contains(body, "hello-gost")
}

// TestBlacklistBypassViaChain verifies that a destination NOT matching the
// blacklist bypass is routed through the chain node. With the node dead the
// request never reaches the echo server, confirming the prior success was due
// to the bypass.
func (s *BypassSuite) TestBlacklistBypassViaChain() {
	cfg, err := RenderConfig("testdata/bypass/blacklist.yaml", ConfigData{ServerAddr: s.echoIP})
	s.Require().NoError(err)
	defer os.Remove(cfg)

	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, cfg, "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.proxyRequest(gostC, "8080", "http://10.0.0.1:5678")
	s.Require().NotContains(body, "hello-gost")
}

// TestWhitelistBypassDirect verifies that a destination outside the whitelist
// rule is bypassed (connects directly) and reaches the echo server even though
// the chain node is dead.
func (s *BypassSuite) TestWhitelistBypassDirect() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/bypass/whitelist.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.proxyRequest(gostC, "8080", fmt.Sprintf("http://%s:5678", s.echoIP))
	s.Require().Contains(body, "hello-gost")
}

// TestWhitelistBypassViaChain verifies that a destination matching the
// whitelist rule is forced through the chain node; with the node dead the
// request never reaches the echo server.
func (s *BypassSuite) TestWhitelistBypassViaChain() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, "testdata/bypass/whitelist.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.proxyRequest(gostC, "8080", "http://10.0.0.1:5678")
	s.Require().NotContains(body, "hello-gost")
}

func TestBypassSuite(t *testing.T) {
	suite.Run(t, new(BypassSuite))
}
