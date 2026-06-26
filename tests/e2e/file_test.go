package e2e

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type FileSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *FileSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *FileSuite) TearDownSuite() {}

// TestFileGetExisting verifies that GET on an existing file returns
// the file content with HTTP 200. Covers the basic file serving path:
// file handler → http.FileServer → file read.
func (s *FileSuite) TestFileGetExisting() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/file/server.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "testdata/file/hello.txt", ContainerFilePath: "/srv/files/hello.txt", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8080/hello.txt"}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)

	if code != 0 || !strings.Contains(string(body), "hello-gost-file") {
		DumpLogs(s.T(), s.ctx, "file-server logs", gostC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "hello-gost-file")
}

// TestFileGetNotFound verifies that GET on a nonexistent file returns
// 404 Not Found.
func (s *FileSuite) TestFileGetNotFound() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/file/server.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "testdata/file/hello.txt", ContainerFilePath: "/srv/files/hello.txt", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"http://127.0.0.1:8080/nonexistent.txt"}
	_, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, _ := io.ReadAll(out)
	s.Assert().Contains(string(body), "404")
}

// TestFileGetIndexHtml verifies that GET / serves index.html when
// present in the served directory. Covers the default index document
// behavior of http.FileServer.
func (s *FileSuite) TestFileGetIndexHtml() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/file/server.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "testdata/file/index.html", ContainerFilePath: "/srv/files/index.html", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "http://127.0.0.1:8080/"}
	code, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)

	body, err := io.ReadAll(out)
	s.Require().NoError(err)
	if code != 0 || !strings.Contains(string(body), "gost file index") {
		DumpLogs(s.T(), s.ctx, "file-server index logs", gostC)
	}
	s.Require().Equal(0, code)
	s.Require().Contains(string(body), "gost file index")
}

// TestFilePutUpload verifies that PUT uploads a file when file.put is
// enabled. Uploads a file and then reads it back via GET to confirm
// the content was persisted.
func (s *FileSuite) TestFilePutUpload() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/file/server_put.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "testdata/file/.empty", ContainerFilePath: "/srv/files/.empty", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	// Create source content inside the container.
	_, _, err = gostC.Exec(s.ctx, []string{
		"sh", "-c", "printf 'uploaded-content' > /tmp/upload_src.txt",
	})
	s.Require().NoError(err)

	// Upload via PUT.
	_, out, err := gostC.Exec(s.ctx, []string{
		"curl", "-v", "-s", "-T", "/tmp/upload_src.txt",
		"http://127.0.0.1:8080/uploaded.txt",
	})
	s.Require().NoError(err)
	putBody, _ := io.ReadAll(out)
	s.T().Logf("PUT response:\n%s", string(putBody))

	// Read back the uploaded file via GET.
	_, out2, err := gostC.Exec(s.ctx, []string{
		"curl", "-v", "-s",
		"http://127.0.0.1:8080/uploaded.txt",
	})
	s.Require().NoError(err)
	body, _ := io.ReadAll(out2)
	s.Assert().Contains(string(body), "uploaded-content")
}

// TestFilePutNoPermission verifies that PUT returns 405 Method Not
// Allowed when file.put is not enabled (default: false).
func (s *FileSuite) TestFilePutNoPermission() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/file/server.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "testdata/file/.empty", ContainerFilePath: "/srv/files/.empty", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	cmd := []string{"curl", "-v", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"-T", "/dev/null", "http://127.0.0.1:8080/test.txt"}
	_, out, err := gostC.Exec(s.ctx, cmd)
	s.Require().NoError(err)
	body, _ := io.ReadAll(out)
	s.Assert().Contains(string(body), "405")
}

// TestFileAuth verifies authentication on the file handler.
// Without credentials the handler returns 401 Unauthorized,
// with valid credentials it serves the file.
func (s *FileSuite) TestFileAuth() {
	gostC, err := RunGostContainerWithFiles(s.ctx, SharedNetworkName,
		"testdata/file/server_auth.yaml",
		[]testcontainers.ContainerFile{
			{HostFilePath: "testdata/file/hello.txt", ContainerFilePath: "/srv/files/hello.txt", FileMode: 0644},
		},
		"8080/tcp")
	s.Require().NoError(err)
	defer gostC.Terminate(s.ctx)

	s.T().Run("no-auth-401", func(t *testing.T) {
		cmd := []string{"curl", "-v", "-s", "-o", "/dev/null", "-w", "%{http_code}",
			"http://127.0.0.1:8080/hello.txt"}
		_, out, _ := gostC.Exec(s.ctx, cmd)
		body, _ := io.ReadAll(out)
		s.Assert().Contains(string(body), "401")
	})

	s.T().Run("with-auth-success", func(t *testing.T) {
		cmd := []string{"curl", "-v", "-s", "-u", "user:pass",
			"http://127.0.0.1:8080/hello.txt"}
		code, out, err := gostC.Exec(s.ctx, cmd)
		s.Require().NoError(err)

		body, err := io.ReadAll(out)
		s.Require().NoError(err)
		if code != 0 || !strings.Contains(string(body), "hello-gost-file") {
			DumpLogs(s.T(), s.ctx, "file-server auth logs", gostC)
		}
		s.Require().Equal(0, code)
		s.Require().Contains(string(body), "hello-gost-file")
	})
}

func TestFileSuite(t *testing.T) {
	suite.Run(t, new(FileSuite))
}
