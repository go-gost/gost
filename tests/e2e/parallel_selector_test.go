package e2e

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type ParallelSelectorSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
}

func (s *ParallelSelectorSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *ParallelSelectorSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
}

func (s *ParallelSelectorSuite) TestParallelSelector() {
	gostC, err := RunGostContainer(s.ctx, SharedNetworkName, "testdata/parallel_selector/server.yaml")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Test the proxy by running curl inside the gost container
	cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080", fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	s.Require().Equal(0, code)

	s.Require().Contains(string(body), "hello-gost")
}

func TestParallelSelectorSuite(t *testing.T) {
	suite.Run(t, new(ParallelSelectorSuite))
}
