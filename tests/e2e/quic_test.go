package e2e

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// QUICSuite covers the QUIC listener/dialer pair (listener type: quic, dialer
// type: quic) with the cipherKey obfuscation enabled. It uses the canonical
// GOST chaining pattern: a client container exposes a plain HTTP proxy that
// tunnels through a quic chain to a quic server (HTTP over QUIC).
type QUICSuite struct {
	suite.Suite
	ctx    context.Context
	echoC  testcontainers.Container
	echoIP string
	bigC   testcontainers.Container
}

func (s *QUICSuite) SetupSuite() {
	s.ctx = context.Background()

	s.T().Logf("start tcp echo container...")
	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP

	s.T().Logf("start big body container...")
	bigC, err := RunBigBodyContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.bigC = bigC
}

func (s *QUICSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
	if s.bigC != nil {
		s.bigC.Terminate(s.ctx)
	}
}

// startChain brings up a quic server (alias "quic-server", cipherKey enabled,
// http handler) and a client container that chains to it through a quic
// dialer. The client exposes port 8080 for curl.
func (s *QUICSuite) startChain() (testcontainers.Container, testcontainers.Container) {
	s.T().Helper()

	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/quic/server.yaml", []string{"quic-server"}, []string{"18843/udp"})
	s.Require().NoError(err)

	rendered, err := RenderConfig("testdata/quic/client.yaml", ConfigData{ServerAddr: "quic-server:18843"})
	s.Require().NoError(err)
	s.T().Cleanup(func() { os.Remove(rendered) })

	clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, rendered, "8080/tcp")
	s.Require().NoError(err)

	return serverC, clientC
}

// curlEcho runs curl through the client's local http proxy and returns the
// exit code plus the captured body.
func (s *QUICSuite) curlEcho(clientC testcontainers.Container) (int, string) {
	s.T().Helper()
	cmd := []string{"curl", "-s", "--max-time", "20", "-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678/", s.echoIP)}
	code, out, _ := clientC.Exec(s.ctx, cmd)
	body, _ := io.ReadAll(out)
	return code, string(body)
}

// sendInvalidDatagram sends a single malformed (non-encrypted) UDP datagram to
// the quic server's listener, reproducing the issue's repro packet.
func (s *QUICSuite) sendInvalidDatagram(clientC testcontainers.Container) {
	s.T().Helper()
	cmd := []string{"python3", "-c",
		"import socket; s=socket.socket(socket.AF_INET, socket.SOCK_DGRAM); s.sendto(b\"x\", (\"quic-server\", 18843))"}
	code, out, err := clientC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	if code != 0 {
		b, _ := io.ReadAll(out)
		s.T().Fatalf("send invalid datagram failed (%d): %s", code, string(b))
	}
}

// TestQUICForwardCipherKeySurvivesInvalidDatagram verifies that a malformed
// UDP datagram does not close the shared QUIC transport (go-gost/x#125).
//
// With the bug, cipherConn.ReadFrom returned the decrypt error straight to
// quic-go, which treats any packet-socket error as fatal and closes the shared
// transport: a single invalid datagram kills the listener and correct clients
// subsequently time out. With the fix the invalid datagram is discarded and
// the listener keeps serving.
//
// Sequence: baseline forward works -> malformed datagram -> forward still works.
func (s *QUICSuite) TestQUICForwardCipherKeySurvivesInvalidDatagram() {
	serverC, clientC := s.startChain()
	defer serverC.Terminate(s.ctx)
	defer clientC.Terminate(s.ctx)

	// Baseline: a valid QUIC forward must work before we poke the listener.
	code, body := s.curlEcho(clientC)
	if code != 0 || !strings.Contains(body, "hello-gost") {
		s.dump("quic-baseline logs", clientC, serverC)
	}
	s.Require().Equal(0, code, "baseline QUIC forward should work")
	s.Require().Contains(body, "hello-gost")

	// Send one malformed UDP datagram to the quic listener's cipher socket.
	s.sendInvalidDatagram(clientC)

	// The listener must still be alive: a fresh QUIC forward must succeed.
	code, body = s.curlEcho(clientC)
	if code != 0 || !strings.Contains(body, "hello-gost") {
		s.dump("quic-after-invalid logs", clientC, serverC)
	}
	s.Require().Equal(0, code,
		"QUIC forward should still work after an invalid datagram (issue #125)")
	s.Require().Contains(body, "hello-gost",
		"listener must not close on an invalid datagram (issue #125)")
}

// abortDownload starts a download of the big body through the client's http
// proxy and aborts it while the transfer is still running. The client reads
// very slowly, so the data piles up in the connection that carries it.
//
// The abort makes the copy towards the client fail, and the client closes the
// quic conn. The data the server already sent stays unread in the stream and
// keeps occupying the connection level receive window, unless the read side
// is cancelled.
func (s *QUICSuite) abortDownload(clientC testcontainers.Container) {
	s.T().Helper()

	cmd := []string{"curl", "-s", "--max-time", "2", "--limit-rate", "1k",
		"-x", "http://127.0.0.1:8080", "http://big-body:5679/big"}
	code, out, _ := clientC.Exec(s.ctx, cmd)
	body, _ := io.ReadAll(out)
	s.T().Logf("aborted download: exit=%d, %d bytes read", code, len(body))
	// The download is expected to be cut short.
	s.Require().NotEqual(0, code, "the download should have been aborted")
}

// TestQUICForwardSurvivesAbortedDownloads verifies that an aborted transfer
// does not exhaust the quic connection's flow control window.
//
// Data the peer already sent stays unread in the stream and keeps occupying
// the connection level receive window. When the local app aborts a download,
// that data is never consumed, so the credit is only returned when the read
// side is cancelled.
//
// Without the cancellation the window drains. Once it is exhausted the peer
// cannot send data on any stream of the session: new requests still reach
// the server, the server still acknowledges, there is no idle timeout and no
// error in the log, and the client waits forever for a response. Only a new
// session clears it.
//
// Sequence: abort several downloads -> a normal request must still work.
func (s *QUICSuite) TestQUICForwardSurvivesAbortedDownloads() {
	serverC, clientC := s.startChain()
	defer serverC.Terminate(s.ctx)
	defer clientC.Terminate(s.ctx)

	// Baseline: the chain must work before anything is aborted.
	code, body := s.curlEcho(clientC)
	if code != 0 || !strings.Contains(body, "hello-gost") {
		s.dump("quic-baseline logs", clientC, serverC)
	}
	s.Require().Equal(0, code, "baseline QUIC forward should work")
	s.Require().Contains(body, "hello-gost")

	// Abort enough downloads to drain the connection window. The initial
	// stream window is 512 KB and the initial connection window is 768 KB,
	// so every aborted download has to leave a large part of a stream
	// window unread.
	for i := 0; i < 3; i++ {
		s.abortDownload(clientC)
	}

	// Give the peer the chance to notice the cancellation and to return
	// the credit.
	time.Sleep(time.Second)

	// The tunnel must still forward. Without the credit being returned the
	// connection window is exhausted and this request never gets an answer.
	code, body = s.curlEcho(clientC)
	if code != 0 || !strings.Contains(body, "hello-gost") {
		s.dump("quic-after-aborts logs", clientC, serverC)
	}
	s.Require().Equal(0, code,
		"QUIC forward should still work after aborted downloads")
	s.Require().Contains(body, "hello-gost",
		"the tunnel must not stall after aborted downloads")
}

func (s *QUICSuite) dump(label string, cs ...testcontainers.Container) {
	for _, c := range cs {
		DumpLogs(s.T(), s.ctx, fmt.Sprintf("%s: %s", label, c.GetContainerID()), c)
	}
}

func TestQUICSuite(t *testing.T) {
	suite.Run(t, new(QUICSuite))
}
