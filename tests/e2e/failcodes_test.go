package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type FailCodesSuite struct {
	suite.Suite
	ctx     context.Context
	echoC   testcontainers.Container
	echoIP  string
	statusC testcontainers.Container
}

func (s *FailCodesSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP

	statusC, err := RunStatusBackendContainer(s.ctx, SharedNetworkName, 429)
	s.Require().NoError(err)
	s.statusC = statusC
}

func (s *FailCodesSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
	if s.statusC != nil {
		s.statusC.Terminate(s.ctx)
	}
}

func (s *FailCodesSuite) sendRawHTTP(gostC testcontainers.Container, host, port string) string {
	req := fmt.Sprintf(
		"GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n",
		s.echoIP,
	)
	encoded := base64.StdEncoding.EncodeToString([]byte(req))
	cmd := []string{"sh", "-c",
		fmt.Sprintf("echo %s | base64 -d | nc -w 5 %s %s", encoded, host, port)}
	_, out, _ := gostC.Exec(s.ctx, cmd)
	b, _ := io.ReadAll(out)
	return string(b)
}

// TestFailCodesConvergence: reverse proxy (tcp handler + sniffing) with FIFO.
// node-429 is first. After the first 429 response, failCodes marks it (Count=1).
// FailFilter (maxFails=1) excludes it. All subsequent requests go to node-good.
func (s *FailCodesSuite) TestFailCodesConvergence() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/failcodes/failcodes.yaml",
		nil,
		"8080/tcp",
	)
	s.Require().NoError(err)
	defer func() {
		DumpLogs(s.T(), s.ctx, "gost-failcodes", gostC)
		gostC.Terminate(s.ctx)
	}()

	time.Sleep(500 * time.Millisecond)

	const totalRequests = 20
	var successCount, failCount int

	for range totalRequests {
		body := s.sendRawHTTP(gostC, "127.0.0.1", "8080")

		if strings.Contains(body, "hello-gost") {
			successCount++
			s.T().Logf("  ✓ 200 (node-good)")
		} else if strings.Contains(body, "status-429") {
			failCount++
			s.T().Logf("  ✗ 429 (node-429)")
		} else {
			s.T().Logf("  ? %s", strings.TrimSpace(body)[:80])
		}
	}

	s.T().Logf("Results: %d successes, %d failures out of %d requests",
		successCount, failCount, totalRequests)

	// After the first 429 marks node-429, FailFilter excludes it for 10s.
	// At most 1-2 requests may fail before the marker is set.
	s.Require().LessOrEqual(failCount, 2,
		"failCodes: node-429 must be excluded after first 429 response. "+
			"Got %d failures out of %d requests", failCount, totalRequests)
}

func TestFailCodesSuite(t *testing.T) {
	suite.Run(t, new(FailCodesSuite))
}

func RunStatusBackendContainer(ctx context.Context, networkName string, statusCode int) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    ".",
			Dockerfile: "Dockerfile",
			Repo:       "gost-e2e",
			Tag:        "latest",
			KeepImage:  true,
			BuildOptionsModifier: func(opts *client.ImageBuildOptions) {
				opts.NetworkMode = "host"
			},
		},
		Networks: []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"status-backend"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/http_status_backend.py", ContainerFilePath: "/scripts/http_status_backend.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5680/tcp"},
		Cmd: []string{
			"python3", "/scripts/http_status_backend.py",
			fmt.Sprintf("%d", statusCode), "5680",
		},
		WaitingFor: wait.ForExposedPort(),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}
