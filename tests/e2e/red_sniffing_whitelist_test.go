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

// RedSniffingWhitelistSuite reproduces go-gost/gost#899: whitelist bypass +
// sniffing on a transparent (red) proxy.
//
// Inside a single privileged container, iptables redirects all outbound TCP to
// the gost red service, so the handler sees the original destination IP via
// SO_ORIGINAL_DST. The whitelist contains only a domain (the docker network
// alias "https-echo"); a bare destination IP is never in it. On a broken build
// (go-gost/x@fe394a8, x >= v0.13.12) the pre-sniffing bypass check on the
// destination IP rejects every connection before SNI sniffing runs, making the
// domain whitelist dead. The fix skips that check in whitelist mode and lets
// the sniffer match the SNI host. This suite also asserts the security
// property the skip must not weaken: a non-whitelisted SNI is still rejected.
type RedSniffingWhitelistSuite struct {
	suite.Suite
	ctx        context.Context
	httpsEchoC testcontainers.Container
	gostC      testcontainers.Container
}

// redirectRules are installed in the gost container before gost starts:
// loopback traffic (incl. Docker's embedded DNS) and gost's own marked sockets
// are exempt; every other TCP packet is redirected to the red service.
const redirectRules = `
iptables -t nat -A OUTPUT -m addrtype --dst-type LOCAL -j RETURN
iptables -t nat -A OUTPUT -m mark --mark 114514 -j RETURN
iptables -t nat -A OUTPUT -p tcp -j REDIRECT --to-ports 12345
`

func (s *RedSniffingWhitelistSuite) SetupSuite() {
	s.ctx = context.Background()

	echoC, err := RunHTTPSEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.httpsEchoC = echoC

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
		CapAdd:   []string{"NET_ADMIN"},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: GostBinPath, ContainerFilePath: "/bin/gost", FileMode: 0755},
			{HostFilePath: "testdata/red_sniffing/whitelist.yaml", ContainerFilePath: "/config.yaml", FileMode: 0644},
		},
		Cmd:        []string{"sh", "-c", redirectRules + "\nexec /bin/gost -C /config.yaml"},
		WaitingFor: wait.ForLog("listening on"),
	}

	gostC, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err)
	s.gostC = gostC
}

func (s *RedSniffingWhitelistSuite) TearDownSuite() {
	if s.gostC != nil {
		s.gostC.Terminate(s.ctx)
	}
	if s.httpsEchoC != nil {
		s.httpsEchoC.Terminate(s.ctx)
	}
}

// TestWhitelistedDomainViaSNI is the gost#899 golden path: curl to a domain in
// the whitelist should be forwarded because the sniffer matches its SNI. The
// kernel redirects the outbound TCP to the red service, which must not reject
// the bare destination IP before sniffing. Retries tolerate startup timing.
func (s *RedSniffingWhitelistSuite) TestWhitelistedDomainViaSNI() {
	s.T().Helper()
	cmd := []string{"sh", "-c", `
for i in $(seq 1 30); do
  body=$(curl -ks --max-time 5 https://https-echo:8443/ 2>/dev/null)
  echo "$body" | grep -q hello-gost && { echo "$body"; exit 0; }
  sleep 1
done
echo "$body"`}
	_, out, err := s.gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, _ := io.ReadAll(out)

	if !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "gost-red logs", s.gostC)
	}
	s.Require().Contains(string(body), "hello-gost",
		"whitelisted domain must be forwarded via SNI, not rejected on the bare IP")
}

// TestNonWhitelistedHostRejected verifies that skipping the pre-sniffing check
// in whitelist mode does not weaken the whitelist: a connection whose sniffed
// SNI is not allowlisted must still be rejected. curl connects to the same
// echo IP but with a different SNI, so only the SNI can drive the decision.
func (s *RedSniffingWhitelistSuite) TestNonWhitelistedHostRejected() {
	s.T().Helper()
	ip, err := s.httpsEchoC.ContainerIP(s.ctx)
	s.Require().NoError(err)

	cmd := []string{"sh", "-c", fmt.Sprintf(
		"curl -ks --resolve other.test:8443:%s --max-time 5 https://other.test:8443/ 2>&1", ip)}
	code, _, err := s.gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	s.Require().NotZero(code,
		"curl to a non-whitelisted SNI should fail; got exit %d", code)

	logs, err := s.gostC.Logs(s.ctx)
	s.Require().NoError(err)
	defer logs.Close()
	logBody, _ := io.ReadAll(logs)
	s.Require().Contains(string(logBody), "bypass:",
		"gost should log a bypass decision for the non-whitelisted SNI")
}

func TestRedSniffingWhitelistSuite(t *testing.T) {
	suite.Run(t, new(RedSniffingWhitelistSuite))
}