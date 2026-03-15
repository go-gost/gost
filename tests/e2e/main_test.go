package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/testcontainers/testcontainers-go/network"
)

var SharedNetworkName string

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Compile the gost binary
	cmd := exec.Command("go", "build", "-o", "/tmp/gost-test-bin", "../../cmd/gost")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to compile gost: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		os.Remove("/tmp/gost-test-bin")
	}()

	// Create a shared Docker network
	net, err := network.New(ctx)
	if err != nil {
		fmt.Printf("Failed to create network: %v\n", err)
		os.Exit(1)
	}
	SharedNetworkName = net.Name

	// Run tests
	code := m.Run()

	// Cleanup
	net.Remove(ctx)

	os.Exit(code)
}
