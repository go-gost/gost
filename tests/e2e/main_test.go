package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	// Compile the gost binary
	cmd := exec.Command("go", "build", "-o", "/tmp/gost-test-bin", "../../cmd/gost")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to compile gost: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove("/tmp/gost-test-bin")

	os.Exit(code)
}
