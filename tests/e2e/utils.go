package e2e

import (
	"context"
	"io"
	"os"
	"testing"
	"text/template"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type ConfigData struct {
	ServerAddr string
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

func RunGostContainer(ctx context.Context, networkName, yamlPath string) (testcontainers.Container, error) {
	return runGostContainer(ctx, networkName, yamlPath, nil, nil)
}

func RunGostContainerWithPorts(ctx context.Context, networkName, yamlPath string, exposedPorts ...string) (testcontainers.Container, error) {
	return runGostContainer(ctx, networkName, yamlPath, nil, exposedPorts)
}

func RunGostContainerWithOptions(ctx context.Context, networkName, yamlPath string, aliases, exposedPorts []string) (testcontainers.Container, error) {
	return runGostContainer(ctx, networkName, yamlPath, aliases, exposedPorts)
}

func runGostContainer(ctx context.Context, networkName, yamlPath string, aliases, exposedPorts []string) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    ".",
			Dockerfile: "Dockerfile",
			Repo:       "gost-e2e",
			Tag:        "latest",
			KeepImage:  true,
		},
		ExposedPorts: exposedPorts,
		// interal check for udp ports will be failed
		WaitingFor: wait.ForExposedPort().SkipInternalCheck(),
		Networks:   []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: aliases,
		},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: GostBinPath, ContainerFilePath: "/bin/gost", FileMode: 0755},
			{HostFilePath: yamlPath, ContainerFilePath: "/config.yaml", FileMode: 0644},
		},
		Cmd: []string{"/bin/gost", "-C", "/config.yaml"},
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}
