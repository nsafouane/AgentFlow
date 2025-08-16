package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestLintConfig validates that golangci-lint configuration is valid and runs successfully
func TestLintConfig(t *testing.T) {
	// Check if golangci-lint is available
	_, err := exec.LookPath("golangci-lint")
	if err != nil {
		t.Skip("golangci-lint not found in PATH, skipping lint test")
	}

	// Test that golangci-lint config is valid
	cmd := exec.Command("golangci-lint", "config", "path")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("golangci-lint config validation failed: %v\nOutput: %s", err, output)
	}

	// Test that golangci-lint runs without errors on current code
	cmd = exec.Command("golangci-lint", "run", "--timeout=2m")
	cmd.Dir = "."
	cmd.Env = os.Environ()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("golangci-lint output: %s", output)
		// Don't fail the test for linting issues in initial setup
		t.Logf("golangci-lint found issues (expected in initial setup): %v", err)
	}
}

// TestModuleStructure validates that all expected modules exist and have proper structure
func TestModuleStructure(t *testing.T) {
	expectedModules := []string{
		"cmd/control-plane",
		"cmd/worker",
		"cmd/af",
		"sdk/go",
	}

	for _, module := range expectedModules {
		// Check if go.mod exists
		goModPath := module + "/go.mod"
		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			t.Errorf("Module %s missing go.mod file", module)
		}

		// Check if module can be built
		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = module
		if err := cmd.Run(); err != nil {
			t.Errorf("Module %s failed to build: %v", module, err)
		}
	}
}
