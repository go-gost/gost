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
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type MTLSSuite struct {
	suite.Suite
	ctx     context.Context
	echoC   testcontainers.Container
	echoIP  string
	pluginC testcontainers.Container
	certDir string
}

func (s *MTLSSuite) SetupSuite() {
	s.ctx = context.Background()

	certDir, err := os.MkdirTemp("", "gost-mtls-certs-*")
	s.Require().NoError(err)
	s.certDir = certDir

	s.generateCerts()

	pluginC, err := runAuthPluginContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.pluginC = pluginC

	echoC, err := RunEchoContainer(s.ctx, SharedNetworkName)
	s.Require().NoError(err)
	s.echoC = echoC

	echoIP, err := echoC.ContainerIP(s.ctx)
	s.Require().NoError(err)
	s.echoIP = echoIP
}

func (s *MTLSSuite) TearDownSuite() {
	if s.pluginC != nil {
		s.pluginC.Terminate(s.ctx)
	}
	if s.echoC != nil {
		s.echoC.Terminate(s.ctx)
	}
	os.RemoveAll(s.certDir)
}

// generateCerts creates a self-signed CA, a server cert signed by the CA,
// and a client cert signed by the CA, writing PEM files to s.certDir.
func (s *MTLSSuite) generateCerts() {
	// CA
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

	// Server
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)
	serverTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, caTmpl, &serverKey.PublicKey, caKey)
	s.Require().NoError(err)
	s.writeCertPEM(s.certDir+"/server.pem", "CERTIFICATE", serverDER)
	s.writeKeyPEM(s.certDir+"/server-key.pem", "EC PRIVATE KEY", serverKey)

	// Client
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)
	clientTmpl := &x509.Certificate{
		SerialNumber:   big.NewInt(3),
		Subject:        pkix.Name{CommonName: "test-client"},
		NotBefore:      time.Now(),
		NotAfter:       time.Now().Add(24 * time.Hour),
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		EmailAddresses: []string{"test@example.com"},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTmpl, caTmpl, &clientKey.PublicKey, caKey)
	s.Require().NoError(err)
	s.writeCertPEM(s.certDir+"/client.pem", "CERTIFICATE", clientDER)
	s.writeKeyPEM(s.certDir+"/client-key.pem", "EC PRIVATE KEY", clientKey)
}

// TestMTLSProxy verifies HTTP forward proxy works over an mTLS listener
// with a valid client certificate.
func (s *MTLSSuite) TestMTLSProxy() {
	rendered, err := RenderConfig("testdata/mtls/server.yaml",
		ConfigData{ServerAddr: "auth-plugin"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName, rendered,
		[]testcontainers.ContainerFile{
			{HostFilePath: s.certDir + "/ca.pem", ContainerFilePath: "/certs/ca.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/server.pem", ContainerFilePath: "/certs/server.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/server-key.pem", ContainerFilePath: "/certs/server-key.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/client.pem", ContainerFilePath: "/certs/client.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/client-key.pem", ContainerFilePath: "/certs/client-key.pem", FileMode: 0644},
		},
		"8443/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Verify proxy works with valid client cert
	cmd := []string{"curl", "-v", "-s", "--connect-timeout", "5",
		"--cacert", "/certs/ca.pem",
		"--cert", "/certs/client.pem",
		"--key", "/certs/client-key.pem",
		"-x", "https://127.0.0.1:8443",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := gostC.Exec(s.ctx, cmd)
	body, err2 := io.ReadAll(out)
	if err != nil || err2 != nil || code != 0 || !strings.Contains(string(body), "hello-gost") {
		DumpLogs(s.T(), s.ctx, "mtls gost logs", gostC)
		DumpLogs(s.T(), s.ctx, "mtls auth-plugin logs", s.pluginC)
	}
	s.Require().NoError(err)
	s.Require().NoError(err2)
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")
}

// TestMTLSWithoutClientCert verifies that an mTLS listener rejects
// connections from clients that do not present a certificate.
func (s *MTLSSuite) TestMTLSWithoutClientCert() {
	rendered, err := RenderConfig("testdata/mtls/server.yaml",
		ConfigData{ServerAddr: "auth-plugin"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName, rendered,
		[]testcontainers.ContainerFile{
			{HostFilePath: s.certDir + "/ca.pem", ContainerFilePath: "/certs/ca.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/server.pem", ContainerFilePath: "/certs/server.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/server-key.pem", ContainerFilePath: "/certs/server-key.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/client.pem", ContainerFilePath: "/certs/client.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/client-key.pem", ContainerFilePath: "/certs/client-key.pem", FileMode: 0644},
		},
		"8443/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// curl without client cert should fail TLS handshake
	cmd := []string{"curl", "-v", "-s", "--connect-timeout", "5",
		"--cacert", "/certs/ca.pem",
		"-x", "https://127.0.0.1:8443",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, _, err := gostC.Exec(s.ctx, cmd)
	if err == nil && code == 0 {
		DumpLogs(s.T(), s.ctx, "mtls gost logs", gostC)
	}
	// Expect failure (non-zero exit code from curl)
	s.Require().NotEqual(0, code, "curl without client cert should fail with non-zero exit code")
}

// TestMTLSAuthPluginLogs verifies that the HTTP auth plugin receives
// the mTLS client certificate identity (client_cn, client_san,
// client_cert_fingerprint) when a request passes through the mTLS listener.
func (s *MTLSSuite) TestMTLSAuthPluginLogs() {
	rendered, err := RenderConfig("testdata/mtls/server.yaml",
		ConfigData{ServerAddr: "auth-plugin"})
	s.Require().NoError(err)
	defer os.Remove(rendered)

	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName, rendered,
		[]testcontainers.ContainerFile{
			{HostFilePath: s.certDir + "/ca.pem", ContainerFilePath: "/certs/ca.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/server.pem", ContainerFilePath: "/certs/server.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/server-key.pem", ContainerFilePath: "/certs/server-key.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/client.pem", ContainerFilePath: "/certs/client.pem", FileMode: 0644},
			{HostFilePath: s.certDir + "/client-key.pem", ContainerFilePath: "/certs/client-key.pem", FileMode: 0644},
		},
		"8443/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Send a request with valid client cert through the mTLS proxy.
	cmd := []string{"curl", "-v", "-s", "--connect-timeout", "5",
		"--cacert", "/certs/ca.pem",
		"--cert", "/certs/client.pem",
		"--key", "/certs/client-key.pem",
		"-x", "https://127.0.0.1:8443",
		fmt.Sprintf("http://%s:5678", s.echoIP)}
	code, out, err := gostC.Exec(s.ctx, cmd)
	body, _ := io.ReadAll(out)
	s.Require().NoError(err)
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost")

	// Read auth plugin logs and verify PeerCert fields are present.
	logs, err := s.pluginC.Logs(s.ctx)
	s.Require().NoError(err)
	logData, _ := io.ReadAll(logs)
	logStr := string(logData)
	s.T().Logf("auth plugin logs:\n%s", logStr)

	s.Require().Contains(logStr, "clientCn", "auth plugin should receive client_cn")
	s.Require().Contains(logStr, "test-client", "client_cn should be 'test-client'")
	s.Require().Contains(logStr, "clientSan", "auth plugin should receive client_san")
	s.Require().Contains(logStr, "clientCertFingerprint", "auth plugin should receive client_cert_fingerprint")
}

func TestMTLSSuite(t *testing.T) {
	suite.Run(t, new(MTLSSuite))
}

// --- helpers ---

func runAuthPluginContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
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
			networkName: {"auth-plugin"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/auth_plugin.py", ContainerFilePath: "/scripts/auth_plugin.py", FileMode: 0644},
		},
		ExposedPorts: []string{"9000/tcp"},
		Cmd:          []string{"python3", "/scripts/auth_plugin.py", "9000"},
		WaitingFor:   wait.ForExposedPort(),
	}
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// writeCertPEM writes a DER-encoded certificate to a PEM file.
func (s *MTLSSuite) writeCertPEM(path, blockType string, der []byte) {
	err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0644)
	s.Require().NoError(err)
}

// writeKeyPEM marshals an ECDSA private key and writes it to a PEM file.
func (s *MTLSSuite) writeKeyPEM(path, blockType string, key *ecdsa.PrivateKey) {
	b, err := x509.MarshalECPrivateKey(key)
	s.Require().NoError(err)
	err = os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: b}), 0600)
	s.Require().NoError(err)
}
