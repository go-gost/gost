package e2e

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ProxyProtoSuite verifies go-gost/gost#677: a forward/rtcp handler with
// metadata.proxyProtocol set prepends a HAProxy PROXY protocol header on the
// outbound connection before forwarding. The backend (proxy-capture) reads the
// first bytes and reflects the header line back so the test can assert it.
//
// This exercises the exact send path the issue asks for (forward/remote's
// proxyproto.WrapClientConn). A single gost instance is used so the header's
// source address is the real connecting client — in a full reverse tunnel the
// source would be the tunnel peer (relay CONNECT propagates only the
// destination); that limitation is out of scope here.
type ProxyProtoSuite struct {
	suite.Suite
	ctx      context.Context
	backendC testcontainers.Container
	gostV1C  testcontainers.Container
	gostV2C  testcontainers.Container
}

func (s *ProxyProtoSuite) SetupSuite() {
	s.ctx = context.Background()

	backendC, err := s.runBackend()
	s.Require().NoError(err)
	s.backendC = backendC

	gostV1C, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/proxyproto/v1.yaml", []string{"gost-v1"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	s.gostV1C = gostV1C

	gostV2C, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/proxyproto/v2.yaml", []string{"gost-v2"}, []string{"8080/tcp"})
	s.Require().NoError(err)
	s.gostV2C = gostV2C
}

func (s *ProxyProtoSuite) TearDownSuite() {
	for _, c := range []testcontainers.Container{s.gostV1C, s.gostV2C, s.backendC} {
		if c != nil {
			c.Terminate(s.ctx)
		}
	}
}

// runBackend starts the raw-TCP PROXY-header-capturing backend, aliased
// "proxy-capture" on the shared network, listening on 5678. It is also used as
// the client host (it has python3) to connect through gost.
func (s *ProxyProtoSuite) runBackend() (testcontainers.Container, error) {
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
		Networks: []string{SharedNetworkName},
		NetworkAliases: map[string][]string{
			SharedNetworkName: {"proxy-capture"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/proxy_capture.py", ContainerFilePath: "/scripts/proxy_capture.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5678/tcp"},
		Cmd:          []string{"python3", "/scripts/proxy_capture.py"},
		WaitingFor:   wait.ForExposedPort(),
	}
	return testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// request connects from the backend container to gost on :8080, sends a payload,
// and returns the reflected response (which carries the PROXY header line).
func (s *ProxyProtoSuite) request(gostHost string) string {
	cmd := []string{
		"sh", "-c",
		fmt.Sprintf("python3 -c \"import socket,sys; s=socket.socket(); s.settimeout(5); s.connect(('%s',8080)); s.sendall(b'hello-gost'); sys.stdout.write(s.recv(4096).decode())\"", gostHost),
	}
	_, out, err := s.backendC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	return string(body)
}

// TestV1HeaderSent asserts gost prepends a text PROXY protocol v1 header.
func (s *ProxyProtoSuite) TestV1HeaderSent() {
	body := s.request("gost-v1")
	if !strings.Contains(body, "PROXY-RECEIVED") || !strings.Contains(body, "PROXY TCP") {
		DumpLogs(s.T(), s.ctx, "gost-v1 logs", s.gostV1C)
	}
	s.Require().Contains(body, "PROXY-RECEIVED")
	s.Require().Contains(body, "PROXY TCP")
}

// TestV2HeaderSent asserts gost prepends a binary PROXY protocol v2 header.
func (s *ProxyProtoSuite) TestV2HeaderSent() {
	body := s.request("gost-v2")
	if !strings.Contains(body, "PROXY-V2-RECEIVED") {
		DumpLogs(s.T(), s.ctx, "gost-v2 logs", s.gostV2C)
	}
	s.Require().Contains(body, "PROXY-V2-RECEIVED")
}

func TestProxyProtoSuite(t *testing.T) {
	suite.Run(t, new(ProxyProtoSuite))
}
