package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type DNSSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *DNSSuite) SetupSuite() {
	s.ctx = context.Background()
}

// dnsQuery sends a DNS query via the Python client and retries on failure.
// expected can be an IP string, "empty" (expect zero answer records), or "" (no check).
func (s *DNSSuite) dnsQuery(gostC testcontainers.Container, mode, host, port, qname, qtype, expected string) {
	args := []string{"python3", "/scripts/dns_query.py", mode, host, port, qname, qtype}
	if expected != "" {
		args = append(args, expected)
	}

	for i := range 5 {
		code, out, err := gostC.Exec(s.ctx, args)
		if err != nil {
			s.T().Logf("query attempt %d exec error: %v", i+1, err)
			time.Sleep(time.Second)
			continue
		}
		body, err := io.ReadAll(out)
		if err != nil {
			s.T().Logf("query attempt %d read error: %v", i+1, err)
			time.Sleep(time.Second)
			continue
		}
		output := string(body)
		s.T().Logf("query attempt %d: %s", i+1, strings.TrimSpace(output))

		if code == 0 {
			return
		}
		time.Sleep(time.Second)
	}

	s.T().Fatalf("DNS query %s %s %s %s failed after 5 retries", mode, qname, qtype, host)
}

// dnsQueryOnce sends a single DNS query with no retries. Returns exit code and output.
func (s *DNSSuite) dnsQueryOnce(gostC testcontainers.Container, mode, host, port, qname, qtype string) (int, string) {
	args := []string{"python3", "/scripts/dns_query.py", mode, host, port, qname, qtype}
	code, out, err := gostC.Exec(s.ctx, args)
	if err != nil {
		return 1, fmt.Sprintf("exec error: %v", err)
	}
	body, _ := io.ReadAll(out)
	return code, string(body)
}

// dnsQueryWithDelay calls dnsQuery with a small wait before the first attempt.
func (s *DNSSuite) dnsQueryWithDelay(gostC testcontainers.Container, mode, host, port, qname, qtype, expected string) {
	time.Sleep(500 * time.Millisecond)
	s.dnsQuery(gostC, mode, host, port, qname, qtype, expected)
}

// sendRaw sends raw data via base64+nc and returns the response.
func (s *DNSSuite) sendRaw(gostC testcontainers.Container, host, port, data string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	cmd := []string{"sh", "-c",
		fmt.Sprintf("echo %s | base64 -d | nc -w 3 -u %s %s", encoded, host, port)}
	_, out, _ := gostC.Exec(s.ctx, cmd)
	b, _ := io.ReadAll(out)
	return string(b)
}

// startDNSResponder starts the UDP DNS responder and returns the container.
func (s *DNSSuite) startDNSResponder() testcontainers.Container {
	dnsC, err := RunDNSResponderContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	return dnsC
}

// startGostWithQueryScript starts a gost container with dns_query.py mounted.
func (s *DNSSuite) startGostWithQueryScript(yamlPath, exposedPort string) testcontainers.Container {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		yamlPath,
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_query.py", ContainerFilePath: "/scripts/dns_query.py", FileMode: 0644},
		},
		exposedPort)
	s.Require().NoError(err)
	return gostC
}

// ---------------------------------------------------------------------------
// Upstream resolution
// ---------------------------------------------------------------------------

func (s *DNSSuite) TestDNSUpstream() {
	dnsC := s.startDNSResponder()
	defer dnsC.Terminate(s.ctx)

	gostC := s.startGostWithQueryScript("testdata/dns/server_upstream.yaml", "1053/udp")
	defer gostC.Terminate(s.ctx)

	s.T().Run("a-record", func(t *testing.T) {
		s.dnsQueryWithDelay(gostC, "udp", "127.0.0.1", "1053", "test.example.com", "A", "10.0.0.1")
	})
	s.T().Run("aaaa-record", func(t *testing.T) {
		s.dnsQuery(gostC, "udp", "127.0.0.1", "1053", "example.com", "AAAA", "::1")
	})
	s.T().Run("second-a-record", func(t *testing.T) {
		s.dnsQuery(gostC, "udp", "127.0.0.1", "1053", "test2.example.com", "A", "10.0.0.2")
	})
}

// ---------------------------------------------------------------------------
// TCP mode
// ---------------------------------------------------------------------------

func (s *DNSSuite) TestDNSTCP() {
	s.T().Log("start TCP DNS responder container...")
	dnsC, err := RunTCPDNSResponderContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	defer dnsC.Terminate(s.ctx)

	gostC := s.startGostWithQueryScript("testdata/dns/server_tcp.yaml", "1053/tcp")
	defer gostC.Terminate(s.ctx)

	s.dnsQueryWithDelay(gostC, "tcp", "127.0.0.1", "1053", "test.example.com", "A", "10.0.0.1")
	s.dnsQuery(gostC, "tcp", "127.0.0.1", "1053", "test2.example.com", "A", "10.0.0.2")
}

// ---------------------------------------------------------------------------
// Bypass rules
// ---------------------------------------------------------------------------

func (s *DNSSuite) TestDNSBypass() {
	dnsC := s.startDNSResponder()
	defer dnsC.Terminate(s.ctx)

	gostC := s.startGostWithQueryScript("testdata/dns/server_bypass.yaml", "1053/udp")
	defer gostC.Terminate(s.ctx)

	s.T().Run("blocked-domain-empty-answer", func(t *testing.T) {
		s.dnsQueryWithDelay(gostC, "udp", "127.0.0.1", "1053",
			"test.example.com", "A", "empty")
	})
	s.T().Run("non-blocked-domain", func(t *testing.T) {
		s.dnsQuery(gostC, "udp", "127.0.0.1", "1053",
			"test2.example.com", "A", "10.0.0.2")
	})
}

// ---------------------------------------------------------------------------
// Host mapper
// ---------------------------------------------------------------------------

// TestDNSHostMapper verifies DNS resolution via the host mapper before
// reaching the upstream exchanger.
//
// Config: server_hosts.yaml maps mapped.example.com → 10.0.0.100 with no
// working upstream. The handler checks the host mapper before the exchange
// path, so mapped names resolve without needing any upstream DNS.
func (s *DNSSuite) TestDNSHostMapper() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/dns/server_hosts.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_query.py", ContainerFilePath: "/scripts/dns_query.py", FileMode: 0644},
		},
		"1053/udp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	s.T().Run("mapped-a-record", func(t *testing.T) {
		// mapped.example.com → host mapper returns 10.0.0.100
		// without contacting any upstream.
		s.dnsQueryWithDelay(gostC, "udp", "127.0.0.1", "1053",
			"mapped.example.com", "A", "10.0.0.100")
	})

	s.T().Run("unmapped-domain", func(t *testing.T) {
		// test.example.com is not in the hosts mapping. The handler
		// will fall through to the exchanger path, which fails because
		// no upstream is configured. Query should return empty/NXDOMAIN.
		code, output := s.dnsQueryOnce(gostC, "udp", "127.0.0.1", "1053",
			"test.example.com", "A")
		s.T().Logf("unmapped query: code=%d, %s", code, strings.TrimSpace(output))
		// Non-zero exit indicates no valid response — expected since
		// there's no working upstream.
		s.Assert().NotEqual(0, code,
			"unmapped domain should fail with no upstream")
	})
}

// ---------------------------------------------------------------------------
// Exchange failure (unreachable upstream)
// ---------------------------------------------------------------------------

// TestDNSExchangeFailure verifies graceful handling when the upstream DNS
// exchanger is unreachable.
//
// Config: server_exchange_failure.yaml points to udp://127.0.0.1:1 which
// is unreachable. The handler must not crash or leak goroutines when the
// upstream exchange fails.
func (s *DNSSuite) TestDNSExchangeFailure() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/dns/server_exchange_failure.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_query.py", ContainerFilePath: "/scripts/dns_query.py", FileMode: 0644},
		},
		"1053/udp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	s.T().Run("unreachable-upstream", func(t *testing.T) {
		// Send query once — the exchange will fail (timeout / ICMP
		// unreachable). No response is expected.
		code, output := s.dnsQueryOnce(gostC, "udp", "127.0.0.1", "1053",
			"test.example.com", "A")
		s.T().Logf("exchange failure result: code=%d, %s",
			code, strings.TrimSpace(output))
		// Non-zero exit is expected since the exchange fails.
		s.Assert().NotEqual(0, code,
			"exchange failure should return non-zero exit")
	})

	s.T().Run("container-alive-after-failure", func(t *testing.T) {
		// Verify the gost container is still running after the
		// failed exchange — proves no crash or hang.
		aliveCode, _, aliveErr := gostC.Exec(s.ctx, []string{"true"})
		s.Require().NoError(aliveErr,
			"container exec should succeed after exchange failure")
		s.Require().Equal(0, aliveCode,
			"gost container should be alive after exchange failure")
	})
}

// ---------------------------------------------------------------------------
// Rate limiter
// ---------------------------------------------------------------------------

// TestDNSRateLimiter verifies that rate limiting configuration is accepted
// and the handler processes queries through the rate limiter path without
// crashing. Uses a generous global limit (1000/s) so queries pass.
func (s *DNSSuite) TestDNSRateLimiter() {
	dnsC := s.startDNSResponder()
	defer dnsC.Terminate(s.ctx)

	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/dns/server_rlimiter.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_query.py", ContainerFilePath: "/scripts/dns_query.py", FileMode: 0644},
		},
		"1053/udp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	s.T().Run("query-with-limiter", func(t *testing.T) {
		s.dnsQueryWithDelay(gostC, "udp", "127.0.0.1", "1053",
			"test.example.com", "A", "10.0.0.1")
	})
	s.T().Run("second-query-with-limiter", func(t *testing.T) {
		s.dnsQuery(gostC, "udp", "127.0.0.1", "1053",
			"test2.example.com", "A", "10.0.0.2")
	})
}

// ---------------------------------------------------------------------------
// Invalid query
// ---------------------------------------------------------------------------

// TestDNSInvalidQuery verifies the DNS listener gracefully handles malformed
// DNS messages. Sends garbage bytes and checks the gost container is
// unaffected (no crash, no hang).
func (s *DNSSuite) TestDNSInvalidQuery() {
	gostC := s.startGostWithQueryScript("testdata/dns/server_upstream.yaml", "1053/udp")
	defer gostC.Terminate(s.ctx)

	s.T().Run("send-garbage-bytes", func(t *testing.T) {
		// Send 5 bytes of garbage via UDP. The miekg/dns server
		// will fail to parse them and discard the packet.
		s.sendRaw(gostC, "127.0.0.1", "1053", "garbage!")
	})

	s.T().Run("container-alive-after-garbage", func(t *testing.T) {
		// Verify the gost container is still alive after receiving
		// invalid data.
		aliveCode, _, aliveErr := gostC.Exec(s.ctx, []string{"true"})
		s.Require().NoError(aliveErr,
			"container exec should succeed after invalid query")
		s.Require().Equal(0, aliveCode,
			"gost container should be alive after invalid query")
	})
}

// ---------------------------------------------------------------------------
// DNS over TLS
// ---------------------------------------------------------------------------

// TestDNSTLS verifies the DNS listener in TLS mode (DNS over TLS).
// Starts the UDP DNS responder (unencrypted), a gost DNS server with
// listener mode: tls, and queries through the TLS endpoint.
func (s *DNSSuite) TestDNSTLS() {
	dnsC := s.startDNSResponder()
	defer dnsC.Terminate(s.ctx)

	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/dns/server_tls.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_tls_query.py", ContainerFilePath: "/scripts/dns_tls_query.py", FileMode: 0644},
		},
		"1053/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Query via TLS
	args := []string{"python3", "/scripts/dns_tls_query.py",
		"127.0.0.1", "1053", "test.example.com", "A", "10.0.0.1"}

	for i := range 5 {
		code, out, err := gostC.Exec(s.ctx, args)
		if err != nil {
			s.T().Logf("tls query attempt %d exec error: %v", i+1, err)
			time.Sleep(time.Second)
			continue
		}
		body, err := io.ReadAll(out)
		if err != nil {
			s.T().Logf("tls query attempt %d read error: %v", i+1, err)
			time.Sleep(time.Second)
			continue
		}
		output := string(body)
		s.T().Logf("tls query attempt %d: %s", i+1, strings.TrimSpace(output))

		if code == 0 {
			return
		}
		time.Sleep(time.Second)
	}

	s.T().Fatal("TLS query failed after 5 retries")
}

func TestDNSSuite(t *testing.T) {
	suite.Run(t, new(DNSSuite))
}
