package e2e

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// UTLSSuite exercises the utls dialer in a forward-proxy chain.
//
// Topology:
//
//	curl -> client gost (http proxy, :8080)
//	         -> chain node (http connector + utls dialer)
//	           -> server gost (http proxy over TLS listener, :8443)
//	             -> tcp-echo
type UTLSSuite struct {
	suite.Suite
	ctx     context.Context
	echoC   testcontainers.Container
	echoIP  string
	certDir string
}

func (s *UTLSSuite) SetupSuite() {
	s.ctx = context.Background()

	certDir, err := os.MkdirTemp("", "gost-utls-certs-*")
	s.Require().NoError(err)
	s.certDir = certDir
	s.generateCerts()

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *UTLSSuite) TearDownSuite() {
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
	os.RemoveAll(s.certDir)
}

// generateCerts creates a self-signed CA and a server cert for the
// "utls-server" hostname, writing PEM files to s.certDir.
func (s *UTLSSuite) generateCerts() {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	s.Require().NoError(err)
	s.writeCertPEM(s.certDir+"/ca.pem", "CERTIFICATE", caDER)
	s.writeKeyPEM(s.certDir+"/ca-key.pem", "EC PRIVATE KEY", caKey)

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)
	serverTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "utls-server"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"utls-server"},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, caTmpl, &serverKey.PublicKey, caKey)
	s.Require().NoError(err)
	s.writeCertPEM(s.certDir+"/server.pem", "CERTIFICATE", serverDER)
	s.writeKeyPEM(s.certDir+"/server-key.pem", "EC PRIVATE KEY", serverKey)
}

// TestUTLSInsecure is the regression test for go-gost/gost#887.
//
// A utls dialer with `secure: false` must still complete the handshake
// (InsecureSkipVerify must be honoured). The previous unsafe.Pointer cast
// read garbage for InsecureSkipVerify, so it came back false and the
// handshake failed with "at least one of ServerName, InsecureSkipVerify or
// InsecureServerNameToVerify must be specified".
func (s *UTLSSuite) TestUTLSInsecure() {
	rendered, err := RenderConfig("testdata/utls/client_insecure.yaml",
		ConfigData{ServerAddr: "utls-server:8443"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	serverC, err := RunGostContainerWithOptions(s.ctx, SharedNetworkName,
		"testdata/utls/server_auto.yaml", []string{"utls-server"}, []string{"8443/tcp"})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	clientC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName, rendered, "8080/tcp")
	s.Require().NoError(err)
	defer clientC.Terminate(s.ctx)

	s.requireProxyWorks(clientC, serverC)
}

// TestUTLSSecureWithCA verifies the converter's RootCAs/ServerName path:
// with `secure: true` and a CA file, the utls dialer must verify the
// server's CA-signed certificate and still reach the echo server.
func (s *UTLSSuite) TestUTLSSecureWithCA() {
	rendered, err := RenderConfig("testdata/utls/client_secure.yaml",
		ConfigData{ServerAddr: "utls-server:8443"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	serverC, err := runGostContainer(s.ctx, SharedNetworkName,
		"testdata/utls/server_ca.yaml",
		[]string{"utls-server"}, []string{"8443/tcp"},
		[]testcontainers.ContainerFile{
			{HostFilePath: s.certDir + "/server.pem", ContainerFilePath: "/certs/server.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/server-key.pem", ContainerFilePath: "/certs/server-key.pem", FileMode: 0644},
		})
	s.Require().NoError(err)
	defer serverC.Terminate(s.ctx)

	clientC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName, rendered,
		[]testcontainers.ContainerFile{
			{HostFilePath: s.certDir + "/ca.pem", ContainerFilePath: "/certs/ca.pem", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer clientC.Terminate(s.ctx)

	s.requireProxyWorks(clientC, serverC)
}

// requireProxyWorks drives curl through the client proxy and asserts the
// echo server's "hello-gost" response is returned.
func (s *UTLSSuite) requireProxyWorks(clientC, serverC testcontainers.Container) {
	cmd := []string{"curl", "-v", "-s", "--connect-timeout", "5",
		"-x", "http://127.0.0.1:8080",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := clientC.Exec(s.ctx, cmd)
	body, err2 := io.ReadAll(out)
	if err != nil || err2 != nil || code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "utls client logs", clientC)
		DumpLogs(s.T(), s.ctx, "utls server logs", serverC)
	}
	s.Require().NoError(err)
	s.Require().NoError(err2)
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

func TestUTLSSuite(t *testing.T) {
	suite.Run(t, new(UTLSSuite))
}

// --- helpers (mirror mtls_test.go) ---

func (s *UTLSSuite) writeCertPEM(path, blockType string, der []byte) {
	err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0644)
	s.Require().NoError(err)
}

func (s *UTLSSuite) writeKeyPEM(path, blockType string, key *ecdsa.PrivateKey) {
	b, err := x509.MarshalECPrivateKey(key)
	s.Require().NoError(err)
	err = os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: b}), 0600)
	s.Require().NoError(err)
}
