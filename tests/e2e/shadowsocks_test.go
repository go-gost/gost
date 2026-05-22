package e2e

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type ShadowsocksSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
	udpC   testcontainers.Container
}

func (s *ShadowsocksSuite) SetupSuite() {
	s.ctx = context.Background()

	s.T().Logf("start tcp echo container...")
	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP

	s.T().Logf("start udp echo container...")
	udpC, err := RunUDPEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.udpC = udpC
}

func (s *ShadowsocksSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
	if s.udpC != nil {
		s.udpC.Terminate(s.ctx)
	}
}

func (s *ShadowsocksSuite) TestShadowsocksTCP() {
	s.runTCPCase("aes256gcm", "testdata/shadowsocks/tcp_server_aes256gcm.yaml", "testdata/shadowsocks/tcp_client_aes256gcm.yaml")
	s.runTCPCase("chacha20", "testdata/shadowsocks/tcp_server_chacha20.yaml", "testdata/shadowsocks/tcp_client_chacha20.yaml")
}

func (s *ShadowsocksSuite) TestShadowsocks2022TCP() {
	s.runTCPCase("2022-aes128", "testdata/shadowsocks/tcp_server_2022_aes128.yaml", "testdata/shadowsocks/tcp_client_2022_aes128.yaml")
	s.runTCPCase("2022-aes256", "testdata/shadowsocks/tcp_server_2022_aes256.yaml", "testdata/shadowsocks/tcp_client_2022_aes256.yaml")
}

func (s *ShadowsocksSuite) TestShadowsocks2022TCPMultiPSK() {
	s.runTCPCase("2022-aes128-multipsk", "testdata/shadowsocks/tcp_server_2022_aes128_multipsk.yaml", "testdata/shadowsocks/tcp_client_2022_aes128_multipsk.yaml")
	s.runTCPCase("2022-aes256-multipsk", "testdata/shadowsocks/tcp_server_2022_aes256_multipsk.yaml", "testdata/shadowsocks/tcp_client_2022_aes256_multipsk.yaml")
}

func (s *ShadowsocksSuite) TestShadowsocksUDP() {
	s.runUDPCase("aes256gcm", "testdata/shadowsocks/udp_server_aes256gcm.yaml", "testdata/shadowsocks/udp_client_aes256gcm.yaml")
}

func (s *ShadowsocksSuite) TestShadowsocks2022UDP() {
	s.runUDPCase("2022-aes128", "testdata/shadowsocks/udp_server_2022_aes128.yaml", "testdata/shadowsocks/udp_client_2022_aes128.yaml")
	s.runUDPCase("2022-aes256", "testdata/shadowsocks/udp_server_2022_aes256.yaml", "testdata/shadowsocks/udp_client_2022_aes256.yaml")
}

func (s *ShadowsocksSuite) runUDPCase(name, serverConfig, clientConfig string) {
	s.T().Run(name, func(t *testing.T) {
		serverAlias := name + "-ssu-server"
		serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName, serverConfig, []string{serverAlias}, []string{"8389/udp"})
		s.Require().NoError(err)
		defer serverC.Terminate(s.ctx)

		rendered, err := RenderConfig(clientConfig, ConfigData{ServerAddr: serverAlias + ":8389"})
		s.Require().NoError(err)
		defer os.Remove(rendered)

		clientC, err := RunGostContainerWithPorts(
			s.ctx,
			SharedNetworkName,
			rendered,
			"9000/udp",
		)
		s.Require().NoError(err)
		defer clientC.Terminate(s.ctx)

		host, err := clientC.Host(s.ctx)
		s.Require().NoError(err)
		port, err := clientC.MappedPort(s.ctx, "9000/udp")
		s.Require().NoError(err)

		conn, err := net.DialTimeout("udp", net.JoinHostPort(host, port.Port()), 5*time.Second)
		s.Require().NoError(err)
		defer conn.Close()

		payload := []byte("hello-gost-udp")
		buf := make([]byte, 2048)
		var n int
		for i := range 5 {
			_, err = conn.Write(payload)
			s.Require().NoError(err)
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err = conn.Read(buf)
			if err == nil {
				break
			}
			s.T().Logf("udp read attempt %d failed: %v, retrying...", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			DumpLogs(s.T(), s.ctx, name+" udp client logs", clientC)
			DumpLogs(s.T(), s.ctx, name+" udp server logs", serverC)
		}
		s.Require().NoError(err)
		s.Require().Contains(string(buf[:n]), "hello-gost")
	})
}

func TestShadowsocksSuite(t *testing.T) {
	suite.Run(t, new(ShadowsocksSuite))
}

func (s *ShadowsocksSuite) runTCPCase(name, serverConfig, clientConfig string) {
	s.T().Run(name, func(t *testing.T) {
		serverAlias := name + "-ss-server"
		serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName, serverConfig, []string{serverAlias}, []string{"8388/tcp"})
		s.Require().NoError(err)
		defer serverC.Terminate(s.ctx)

		rendered, err := RenderConfig(clientConfig, ConfigData{ServerAddr: serverAlias + ":8388"})
		s.Require().NoError(err)
		defer os.Remove(rendered)

		clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, rendered, "8080/tcp")
		s.Require().NoError(err)
		defer clientC.Terminate(s.ctx)

		cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080", fmt.Sprintf("http://%s:5678", s.echoIP)}
		code, out, err := clientC.Exec(s.ctx, cmd)
		s.Require().NoError(err)

		body, err := io.ReadAll(out)
		s.Require().NoError(err)
		if code != 0 || !strings.Contains(string(body), "hello-gost") {
			DumpLogs(s.T(), s.ctx, name+" client logs", clientC)
			DumpLogs(s.T(), s.ctx, name+" server logs", serverC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(string(body), "hello-gost")
	})
}
