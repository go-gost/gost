package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/testcontainers/testcontainers-go/network"
)

var SharedNetworkName string

func TestMain(m *testing.M) {
	flag.Parse()
	ctx := context.Background()

	shouldCleanup := false

	if GostBinPath == "" {
		GostBinPath = "/tmp/gost-test-bin"
		cmd := exec.Command("go", "build", "-o", GostBinPath, "../../cmd/gost")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to compile gost: %v\n", err)
			os.Exit(1)
		}
		shouldCleanup = true
	}

	net, err := network.New(ctx)
	if err != nil {
		fmt.Printf("Failed to create network: %v\n", err)
		os.Exit(1)
	}
	SharedNetworkName = net.Name

	code := m.Run()

	net.Remove(ctx)
	if shouldCleanup {
		os.Remove(GostBinPath)
	}

	os.Exit(code)
}
