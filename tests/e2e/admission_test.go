package e2e

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// AdmissionSuite verifies service-level admission control, which gates
// connections by the client's source address. It contrasts a loopback client
// (curl run inside the gost container, source 127.0.0.1) against an external
// client (curl run inside the echo container, source = its container IP).
type AdmissionSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *AdmissionSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *AdmissionSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// curlIn runs curl inside the given container, proxying through the gost proxy
// at proxyAddr to the echo server, and returns the response body. An admitted
// request returns "hello-gost"; a denied connection returns an empty body.
func (s *AdmissionSuite) curlIn(c testcontainers.Container, proxyAddr string) string {
	cmd := []string{
		"curl", "-s",
		"-x", fmt.Sprintf("http://%s", proxyAddr),
		fmt.Sprintf("http://%s:5678", s.echoIP),
	}
	_, out, err := c.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestWhitelistAdmitLoopback verifies that a whitelisted source (127.0.0.1,
// a loopback client inside the gost container) is admitted and reaches the
// echo server.
func (s *AdmissionSuite) TestWhitelistAdmitLoopback() {
	gostC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/admission/whitelist.yaml", []string{"gost-proxy"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.curlIn(gostC, "127.0.0.1:8080")
	s.Require().Contains(body, "hello-gost")
}

// TestWhitelistDenyExternal verifies that a non-whitelisted source (an external
// container, not 127.0.0.1) is denied, so the request never reaches the echo
// server.
func (s *AdmissionSuite) TestWhitelistDenyExternal() {
	gostC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/admission/whitelist.yaml", []string{"gost-proxy"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.curlIn(s.echoC, "gost-proxy:8080")
	s.Require().NotContains(body, "hello-gost")
}

// TestBlacklistDenyLoopback verifies that a blacklisted source (127.0.0.1,
// a loopback client inside the gost container) is denied.
func (s *AdmissionSuite) TestBlacklistDenyLoopback() {
	gostC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/admission/blacklist.yaml", []string{"gost-proxy"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.curlIn(gostC, "127.0.0.1:8080")
	s.Require().NotContains(body, "hello-gost")
}

// TestBlacklistAdmitExternal verifies that a source outside the blacklist (an
// external container, not 127.0.0.1) is admitted and reaches the echo server.
func (s *AdmissionSuite) TestBlacklistAdmitExternal() {
	gostC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/admission/blacklist.yaml", []string{"gost-proxy"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	body := s.curlIn(s.echoC, "gost-proxy:8080")
	s.Require().Contains(body, "hello-gost")
}

func TestAdmissionSuite(t *testing.T) {
	suite.Run(t, new(AdmissionSuite))
}
