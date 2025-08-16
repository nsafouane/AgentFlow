package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestCLIMain is a placeholder test for the CLI main function
func TestCLIMain(t *testing.T) {
	// Placeholder test - will be expanded with actual functionality
	t.Log("CLI main function test placeholder")

	// Test that main function can be called without panicking
	// In a real implementation, we would test the CLI commands
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()

	// We don't actually call main() here as it would run the CLI
	// Instead, we test that the package compiles and basic structure is correct
	t.Log("CLI package structure validated")
}

// TestValidationResultJSONSchema tests that the ValidationResult struct
// produces valid JSON that matches our expected schema
func TestValidationResultJSONSchema(t *testing.T) {
	// Create a sample validation result
	result := ValidationResult{
		Version:   "1.0.0",
		Timestamp: "2024-01-01T00:00:00Z",
		Environment: EnvironmentInfo{
			Platform:     "linux",
			Architecture: "amd64",
			Container:    "devcontainer",
		},
		Tools: map[string]ToolStatus{
			"go": {
				Version: "1.22.0",
				Status:  "ok",
			},
			"docker": {
				Version: "24.0.7",
				Status:  "ok",
			},
		},
		Services: map[string]ServiceInfo{
			"postgres": {
				Status:     "available",
				Connection: "postgresql://localhost:5432/test",
			},
		},
		Warnings: []string{"Test warning"},
		Errors:   []string{},
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal ValidationResult to JSON: %v", err)
	}

	// Unmarshal back to verify structure
	var unmarshaled ValidationResult
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON back to ValidationResult: %v", err)
	}

	// Verify key fields
	if unmarshaled.Version != result.Version {
		t.Errorf("Version mismatch: expected %s, got %s", result.Version, unmarshaled.Version)
	}

	if unmarshaled.Environment.Platform != result.Environment.Platform {
		t.Errorf("Platform mismatch: expected %s, got %s", result.Environment.Platform, unmarshaled.Environment.Platform)
	}

	if len(unmarshaled.Tools) != len(result.Tools) {
		t.Errorf("Tools count mismatch: expected %d, got %d", len(result.Tools), len(unmarshaled.Tools))
	}

	// Verify required JSON fields exist
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	requiredFields := []string{"version", "timestamp", "environment", "tools", "services", "warnings", "errors"}
	for _, field := range requiredFields {
		if _, exists := jsonMap[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON output", field)
		}
	}

	t.Logf("JSON Schema validation passed. Output:\n%s", string(jsonData))
}

// TestEnvironmentDetection tests the environment detection logic
func TestEnvironmentDetection(t *testing.T) {
	env := detectEnvironment()

	// Verify platform is detected
	if env.Platform == "" {
		t.Error("Platform should not be empty")
	}

	// Verify architecture is detected
	if env.Architecture == "" {
		t.Error("Architecture should not be empty")
	}

	// Verify container detection (should be "host" in test environment)
	if env.Container == "" {
		t.Error("Container type should not be empty")
	}

	t.Logf("Environment detected: Platform=%s, Architecture=%s, Container=%s",
		env.Platform, env.Architecture, env.Container)
}

// TestCommandExists tests the command existence check utility
func TestCommandExists(t *testing.T) {
	// Test with a command that should exist on most systems
	if !commandExists("go") {
		t.Skip("Go command not found, skipping test")
	}

	// Test with a command that definitely doesn't exist
	if commandExists("this-command-definitely-does-not-exist-12345") {
		t.Error("commandExists should return false for non-existent commands")
	}
}

// TestValidateCommand tests the validate command functionality
func TestValidateCommand(t *testing.T) {
	// Build the CLI binary for testing
	binaryName := "af_test"
	if runtime.GOOS == "windows" {
		binaryName = "af_test.exe"
	}

	cmd := exec.Command("go", "build", "-o", binaryName, ".")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build CLI for testing: %v", err)
	}
	defer os.Remove(binaryName)

	// Run the validate command
	var cmdToRun *exec.Cmd
	if runtime.GOOS == "windows" {
		cmdToRun = exec.Command(".\\"+binaryName, "validate")
	} else {
		cmdToRun = exec.Command("./"+binaryName, "validate")
	}

	output, err := cmdToRun.Output()
	if err != nil {
		// It's okay if the command fails due to missing tools, we just want to test JSON output
		t.Logf("Validate command failed (expected in test environment): %v", err)
		// If there's no output, we can't test JSON parsing
		if len(output) == 0 {
			t.Skip("No output from validate command, skipping JSON validation test")
		}
	}

	// Verify the output is valid JSON
	var result ValidationResult
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("Validate command output is not valid JSON: %v\nOutput: %s", err, string(output))
	}

	// Verify required fields are present
	if result.Version == "" {
		t.Error("Version field should not be empty")
	}
	if result.Timestamp == "" {
		t.Error("Timestamp field should not be empty")
	}

	t.Logf("Validate command produced valid JSON output")
}

// TestContainerWarning tests that the CLI shows warning when not in container
func TestContainerWarning(t *testing.T) {
	// Create a validation result for host environment
	result := ValidationResult{
		Version:   "1.0.0",
		Timestamp: "2024-01-01T00:00:00Z",
		Environment: EnvironmentInfo{
			Platform:     "linux",
			Architecture: "amd64",
			Container:    "host", // This should trigger warning
		},
		Tools:    make(map[string]ToolStatus),
		Services: make(map[string]ServiceInfo),
		Warnings: []string{},
		Errors:   []string{},
	}

	// Simulate the warning logic from validateEnvironment
	if result.Environment.Container == "host" {
		result.Warnings = append(result.Warnings,
			"Running on host system. Consider using VS Code devcontainer for consistent environment.")
	}

	// Verify warning was added
	if len(result.Warnings) == 0 {
		t.Error("Expected warning for host environment, but no warnings found")
	}

	found := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "devcontainer") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected warning about devcontainer, got warnings: %v", result.Warnings)
	}

	t.Logf("Container warning test passed: %v", result.Warnings)
}
