package e2e

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// AuthSuite verifies proxy authentication on the HTTP handler, using both
// inline single-user auth and a named auther with multiple users. Requests
// carry proxy credentials via curl's -x http://user:pass@host form; a rejected
// request gets a 407 and never reaches the echo server.
type AuthSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *AuthSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *AuthSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

// proxyRequest sends one request through the gost proxy using the given
// userinfo ("user:pass", or "" for no credentials) and returns the response
// body. An authenticated request returns "hello-gost"; a rejected one returns
// an empty body.
func (s *AuthSuite) proxyRequest(gostC testcontainers.Container, userinfo string) string {
	proxy := "http://127.0.0.1:8080"
	if userinfo != "" {
		proxy = fmt.Sprintf("http://%s@127.0.0.1:8080", userinfo)
	}
	cmd := []string{
		"curl", "-s",
		"-x", proxy,
		fmt.Sprintf("http://%s:5678", s.echoIP),
	}
	_, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

func (s *AuthSuite) runGost(cfg string) testcontainers.Container {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, cfg, "8080/tcp")
	s.Require().NoError(err)
	return gostC
}

// TestInlineAuthValid verifies that correct inline credentials are accepted.
func (s *AuthSuite) TestInlineAuthValid() {
	gostC := s.runGost("testdata/auth/inline.yaml")
	defer gostC.Terminate(s.ctx)

	s.Require().Contains(s.proxyRequest(gostC, "user:pass"), "hello-gost")
}

// TestInlineAuthWrongPassword verifies that a wrong password is rejected.
func (s *AuthSuite) TestInlineAuthWrongPassword() {
	gostC := s.runGost("testdata/auth/inline.yaml")
	defer gostC.Terminate(s.ctx)

	s.Require().NotContains(s.proxyRequest(gostC, "user:wrong"), "hello-gost")
}

// TestInlineAuthMissing verifies that a request without credentials is rejected.
func (s *AuthSuite) TestInlineAuthMissing() {
	gostC := s.runGost("testdata/auth/inline.yaml")
	defer gostC.Terminate(s.ctx)

	s.Require().NotContains(s.proxyRequest(gostC, ""), "hello-gost")
}

// TestNamedAutherFirstUser verifies that the first user in a named auther is
// accepted.
func (s *AuthSuite) TestNamedAutherFirstUser() {
	gostC := s.runGost("testdata/auth/auther.yaml")
	defer gostC.Terminate(s.ctx)

	s.Require().Contains(s.proxyRequest(gostC, "alice:secret"), "hello-gost")
}

// TestNamedAutherSecondUser verifies that any user in a named auther, not just
// the first, is accepted.
func (s *AuthSuite) TestNamedAutherSecondUser() {
	gostC := s.runGost("testdata/auth/auther.yaml")
	defer gostC.Terminate(s.ctx)

	s.Require().Contains(s.proxyRequest(gostC, "bob:hunter2"), "hello-gost")
}

// TestNamedAutherInvalid verifies that credentials not in the named auther are
// rejected.
func (s *AuthSuite) TestNamedAutherInvalid() {
	gostC := s.runGost("testdata/auth/auther.yaml")
	defer gostC.Terminate(s.ctx)

	s.Require().NotContains(s.proxyRequest(gostC, "alice:wrong"), "hello-gost")
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthSuite))
}
