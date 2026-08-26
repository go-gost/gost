package e2e

import (
	"context"
	"flag"
	"io"
	"os"
	"testing"
	"text/template"

	"github.com/moby/moby/client"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

var GostBinPath string

func init() {
	flag.StringVar(&GostBinPath, "gost-bin", "", "Path to a pre-built gost binary (skips compilation)")
}

type ConfigData struct {
	ServerAddr string
}

// ExecOutput runs cmd in c and returns the demultiplexed output reader.
// testcontainers' Exec returns the raw Docker stream, which multiplexes stdout
// and stderr with 8-byte framing headers (type byte + 6 zero + length) that
// interleave with the real data. Demultiplexing produces a clean combined
// stdout+stderr stream that tests can assert against.
func ExecOutput(ctx context.Context, c testcontainers.Container, cmd []string) (int, io.Reader, error) {
	return c.Exec(ctx, cmd, tcexec.Multiplexed())
}

func DumpLogs(t *testing.T, ctx context.Context, label string, c testcontainers.Container) {
	logs, err := c.Logs(ctx)
	if err != nil {
		return
	}
	defer logs.Close()

	body, err := io.ReadAll(logs)
	if err != nil {
		return
	}

	t.Logf("%s:\n%s", label, string(body))
}

func RenderConfig(tmplPath string, data ConfigData) (string, error) {
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", err
	}

	f, err := os.CreateTemp("", "gost-e2e-config-*.yaml")
	if err != nil {
		return "", err
	}

	if err := tmpl.Execute(f, data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}

	return f.Name(), nil
}

func RunEchoContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
	req := echoContainerRequest(ctx, networkName)
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func echoContainerRequest(_ context.Context, networkName string) testcontainers.ContainerRequest {
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
			networkName: {"tcp-echo"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/tcp_echo.py", ContainerFilePath: "/scripts/tcp_echo.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5678/tcp"},
		Cmd:          []string{"python3", "/scripts/tcp_echo.py"},
		WaitingFor:   wait.ForExposedPort(),
	}
}

func RunUDPEchoContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
	req := udpEchoContainerRequest(ctx, networkName)
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func udpEchoContainerRequest(_ context.Context, networkName string) testcontainers.ContainerRequest {
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
			networkName: {"udp-echo"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/udp_echo.py", ContainerFilePath: "/scripts/udp_echo.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5679/udp"},
		Cmd:          []string{"python3", "/scripts/udp_echo.py"},
		WaitingFor:   wait.ForExposedPort().SkipInternalCheck(),
	}
}

// RunDNSResponderContainer starts a UDP-based DNS responder for e2e DNS tests.
// The container is registered with the network alias "dns-server".
func RunDNSResponderContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
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
			networkName: {"dns-server"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_responder.py", ContainerFilePath: "/scripts/dns_server.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5353/udp"},
		Cmd:          []string{"python3", "/scripts/dns_server.py"},
		WaitingFor:   wait.ForExposedPort().SkipInternalCheck(),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// RunTCPDNSResponderContainer starts a TCP-based DNS responder for e2e DNS tests.
// The container is registered with the network alias "dns-server".
func RunTCPDNSResponderContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
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
			networkName: {"dns-server"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_responder_tcp.py", ContainerFilePath: "/scripts/dns_server.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5353/tcp"},
		Cmd:          []string{"python3", "/scripts/dns_server.py"},
		WaitingFor:   wait.ForExposedPort(),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// RunResolverResponderContainer starts a UDP DNS responder that answers A
// queries for "echo.test" with targetIP (an echo server's address) and returns
// NXDOMAIN for all other names. It is used to prove the resolver module drives
// outbound dialing: a gost proxy pointed at this responder resolves echo.test
// to a reachable address. The container is aliased "dns-responder".
func RunResolverResponderContainer(ctx context.Context, networkName, targetIP string) (testcontainers.Container, error) {
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
			networkName: {"dns-responder"},
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: "scripts/dns_resolver_responder.py", ContainerFilePath: "/scripts/dns_server.py", FileMode: 0644},
		},
		ExposedPorts: []string{"5353/udp"},
		Cmd:          []string{"python3", "/scripts/dns_server.py", targetIP, "5353"},
		WaitingFor:   wait.ForExposedPort().SkipInternalCheck(),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// RunHTTPSEchoContainer starts an HTTPS echo server (self-signed cert) that
// responds with "hello-gost" on port 8443. The container is registered with
// the network alias "https-echo".
func RunHTTPSEchoContainer(ctx context.Context, networkName string) (testcontainers.Container, error) {
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
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
				networkName: {"https-echo"},
			},
			Files: []testcontainers.ContainerFile{
				{HostFilePath: "scripts/https_echo.py", ContainerFilePath: "/scripts/https_echo.py", FileMode: 0644},
			},
			ExposedPorts: []string{"8443/tcp"},
			Cmd:          []string{"python3", "/scripts/https_echo.py"},
			WaitingFor:   wait.ForExposedPort(),
		},
		Started: true,
	})
}

func RunGostContainer(ctx context.Context, networkName, yamlPath string) (testcontainers.Container, error) {
	return runGostContainer(ctx, networkName, yamlPath, nil, nil, nil)
}

func RunGostContainerWithPorts(ctx context.Context, networkName, yamlPath string, exposedPorts ...string) (testcontainers.Container, error) {
	return runGostContainer(ctx, networkName, yamlPath, nil, exposedPorts, nil)
}

func RunGostContainerWithOptions(ctx context.Context, networkName, yamlPath string, aliases, exposedPorts []string) (testcontainers.Container, error) {
	return runGostContainer(ctx, networkName, yamlPath, aliases, exposedPorts, nil)
}

// RunGostContainerWithFiles starts a gost container with extra files mounted.
func RunGostContainerWithFiles(ctx context.Context, networkName, yamlPath string, extraFiles []testcontainers.ContainerFile, exposedPorts ...string) (testcontainers.Container, error) {
	return runGostContainer(ctx, networkName, yamlPath, nil, exposedPorts, extraFiles)
}

func runGostContainer(ctx context.Context, networkName, yamlPath string, aliases, exposedPorts []string, extraFiles []testcontainers.ContainerFile) (testcontainers.Container, error) {
	files := []testcontainers.ContainerFile{
		{HostFilePath: GostBinPath, ContainerFilePath: "/bin/gost", FileMode: 0755},
		{HostFilePath: yamlPath, ContainerFilePath: "/config.yaml", FileMode: 0644},
	}
	files = append(files, extraFiles...)

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
		ExposedPorts: exposedPorts,
		Networks:     []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: aliases,
		},
		Files: files,
		Cmd:   []string{"/bin/gost", "-C", "/config.yaml"},
	}

	// Wait for the gost process to be ready. With exposed ports we wait for the
	// port to accept connections; without them (e.g. a socks5 proxy consumed
	// from inside the container) we wait for a startup log line instead.
	if len(exposedPorts) > 0 {
		// internal check for udp ports will be failed
		req.WaitingFor = wait.ForExposedPort().SkipInternalCheck()
	} else {
		req.WaitingFor = wait.ForLog("listening on")
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}
