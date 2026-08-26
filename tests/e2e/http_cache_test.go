package e2e

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type HTTPCacheSuite struct {
	suite.Suite
	ctx     context.Context
	beC     testcontainers.Container
	staleC  testcontainers.Container
}

func (s *HTTPCacheSuite) SetupSuite() {
	s.ctx = context.Background()

	s.T().Log("start http cache backend container...")
	beC, err := RunCacheBackendContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.beC = beC

	s.T().Log("start http cache serve-stale backend container...")
	staleC, err := RunServeStaleBackendContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.staleC = staleC
}

func (s *HTTPCacheSuite) TearDownSuite() {
	if s.beC != nil {
		s.beC.Terminate(s.ctx)
	}
	if s.staleC != nil {
		s.staleC.Terminate(s.ctx)
	}
}

// TestHTTPCacheHit verifies that the HTTP response cache returns the cached
// response on repeat requests. The test backend returns a monotonically
// increasing counter. With cache enabled, every request returns the same
// counter value (the cached response from the first request).
func (s *HTTPCacheSuite) TestHTTPCacheHit() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http_cache/server_cache.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8080/"}

	// First request — cache miss, backend returns "cache-test-N"
	code, out, err := ExecOutput(s.ctx, gostC, cmd)
	s.Require().NoError(err)
	body1, _ := io.ReadAll(out)
	if code != 0 || !strings.Contains(string(body1), "cache-test-") {
		DumpLogs(s.T(), s.ctx, "cache-proxy logs (first req)", gostC)
	}
	s.Require().Equal(0, code, "first request should succeed")
	// The backend counter is shared across subtests (a fresh counter backend is
	// not started per test), so the first request here is not necessarily #1.
	// Caching is proven by body2==body1 and body3==body1 below.
	s.Require().Contains(string(body1), "cache-test-",
		"first request should get a backend response")

	// Second request — cache hit, same response
	code, out, err = ExecOutput(s.ctx, gostC, cmd)
	s.Require().NoError(err)
	body2, _ := io.ReadAll(out)
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "cache-proxy logs (second req)", gostC)
	}
	s.Require().Equal(0, code, "second request should succeed")
	s.Require().Equal(string(body1), string(body2),
		"cached response should equal first response")

	// Third request — cache hit, same
	code, out, err = ExecOutput(s.ctx, gostC, cmd)
	s.Require().NoError(err)
	body3, _ := io.ReadAll(out)
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "cache-proxy logs (third req)", gostC)
	}
	s.Require().Equal(0, code, "third request should succeed")
	s.Require().Equal(string(body1), string(body3),
		"cached response should be consistent")
}

// TestHTTPCacheDisabled verifies that without a named cache, every request
// reaches the backend and gets a fresh response (monotonically increasing
// counter). This is the control test: it proves the counting backend works
// and that the cache test actually validates caching, not incidental
// idempotency of the backend.
func (s *HTTPCacheSuite) TestHTTPCacheDisabled() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http_cache/server_nocache.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8080/"}

	code, out, err := ExecOutput(s.ctx, gostC, cmd)
	s.Require().NoError(err)
	body1, _ := io.ReadAll(out)
	if code != 0 || !strings.Contains(string(body1), "cache-test-") {
		DumpLogs(s.T(), s.ctx, "no-cache proxy logs (first req)", gostC)
	}
	s.Require().Equal(0, code)

	code, out, err = ExecOutput(s.ctx, gostC, cmd)
	s.Require().NoError(err)
	body2, _ := io.ReadAll(out)
	if code != 0 {
		DumpLogs(s.T(), s.ctx, "no-cache proxy logs (second req)", gostC)
	}
	s.Require().Equal(0, code)

	// Without a named cache, each request hits the backend directly.
	// The backend increments a counter per request, so consecutive
	// requests return different bodies.
	s.Require().NotEqual(string(body1), string(body2),
		"without cache, consecutive requests should return different responses")
}

// TestHTTPCacheServeStale verifies that a stale (expired) cached response is
// served when the upstream fetch fails and cache.serveStale is enabled.
//
// The serve-stale backend serves one HTTP request, then shuts down its HTTP
// server and switches to accepting TCP connections and immediately closing
// them. This makes the upstream dial succeed but the HTTP response read fail,
// which triggers the serve-stale path.
//
// Steps:
//  1. First request: cache miss, backend responds → cache stores the response
//     with TTL 3s. The backend then switches to connection-dropping mode.
//  2. Wait 4s for the cache entry to expire.
//  3. Second request: cache hit (stale), dial succeeds (TCP acceptor accepts),
//     but no HTTP response arrives → http.ReadResponse fails → serveStale
//     returns the cached "cache-test-1".
func (s *HTTPCacheSuite) TestHTTPCacheServeStale() {
	gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
		"testdata/http_cache/server_cache_servestale.yaml", "8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8080/"}

	// First request — cache miss, backend returns "cache-test-1"
	code, out, err := ExecOutput(s.ctx, gostC, cmd)
	s.Require().NoError(err)
	body1, _ := io.ReadAll(out)
	if code != 0 || !strings.Contains(string(body1), "cache-test-") {
		DumpLogs(s.T(), s.ctx, "serve-stale logs (first req)", gostC)
	}
	s.Require().Equal(0, code, "first request should succeed")
	s.Require().Contains(string(body1), "cache-test-1",
		"first request should get backend response #1")

	// At this point the backend has shut down its HTTP server and switched to
	// connection-dropping mode. Wait for cache TTL (3s) to expire.
	time.Sleep(4 * time.Second)

	// Second request — stale cache hit, upstream connects but yields no
	// response → serve-stale returns the expired cached response.
	code, out, err = ExecOutput(s.ctx, gostC, cmd)
	s.Require().NoError(err)
	body2, _ := io.ReadAll(out)
	if code != 0 || !strings.Contains(string(body2), "cache-test-") {
		DumpLogs(s.T(), s.ctx, "serve-stale logs (second req)", gostC)
	}
	s.Require().Equal(0, code, "serve-stale should return the cached response")
	s.Require().Equal(string(body1), string(body2),
		"serve-stale should return the expired cached response")
}

func TestHTTPCacheSuite(t *testing.T) {
	suite.Run(t, new(HTTPCacheSuite))
}

// RunCacheBackendContainer starts a container running the cache test HTTP server
// that returns a monotonically increasing counter in each response body.
func RunCacheBackendContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
	req := cacheBackendContainerRequest(ctx, networkName)
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func cacheBackendContainerRequest(_ context.Context, networkName string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
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
			networkName: {"cache-backend"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/http_cache_backend.py", ContainerFilePath: "/scripts/http_cache_backend.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5677/tcp"},
		Cmd:          []string{"python3", "/scripts/http_cache_backend.py"},
		WaitingFor:   wait.ForExposedPort(),
	}
}

// RunServeStaleBackendContainer starts a container running the serve-stale test
// HTTP server. It serves one request with a counter body, then replaces itself
// with a raw TCP socket that accepts connections and immediately closes them.
// This makes upstream dials succeed but HTTP response reads fail, which
// triggers the serve-stale path in the sniffer.
func RunServeStaleBackendContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
	req := serveStaleBackendContainerRequest(ctx, networkName)
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func serveStaleBackendContainerRequest(_ context.Context, networkName string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
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
			networkName: {"cache-servestale"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/http_cache_servestale_backend.py", ContainerFilePath: "/scripts/http_cache_servestale_backend.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5676/tcp"},
		Cmd:          []string{"python3", "/scripts/http_cache_servestale_backend.py"},
		WaitingFor:   wait.ForExposedPort(),
	}
}
