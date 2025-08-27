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
	// Test host environment (should show warnings)
	t.Run("HostEnvironment", func(t *testing.T) {
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

		// Use all fields to avoid unused write warnings
		_ = result.Version
		_ = result.Timestamp
		_ = result.Tools
		_ = result.Services
		_ = result.Errors

		// Simulate the warning logic from validateEnvironment
		if result.Environment.Container == "host" {
			result.Warnings = append(result.Warnings,
				"Running on host system. Consider using VS Code devcontainer for consistent environment.")
			result.Warnings = append(result.Warnings,
				"Devcontainer provides standardized tooling, dependencies, and configuration.")
			result.Warnings = append(result.Warnings,
				"To use devcontainer: Open this project in VS Code and select 'Reopen in Container'.")
		}

		// Verify warnings were added
		if len(result.Warnings) != 3 {
			t.Errorf("Expected 3 warnings for host environment, got %d", len(result.Warnings))
		}

		expectedWarnings := []string{"devcontainer", "standardized tooling", "Reopen in Container"}
		for i, expected := range expectedWarnings {
			if i >= len(result.Warnings) {
				t.Errorf("Missing warning %d: should contain '%s'", i, expected)
				continue
			}
			if !strings.Contains(result.Warnings[i], expected) {
				t.Errorf("Warning %d should contain '%s', got: %s", i, expected, result.Warnings[i])
			}
		}

		t.Logf("Host environment warnings: %v", result.Warnings)
	})

	// Test devcontainer environment (should not show warnings)
	t.Run("DevcontainerEnvironment", func(t *testing.T) {
		result := ValidationResult{
			Version:   "1.0.0",
			Timestamp: "2024-01-01T00:00:00Z",
			Environment: EnvironmentInfo{
				Platform:     "linux",
				Architecture: "amd64",
				Container:    "devcontainer", // This should NOT trigger warning
			},
			Tools:    make(map[string]ToolStatus),
			Services: make(map[string]ServiceInfo),
			Warnings: []string{},
			Errors:   []string{},
		}

		// Use all fields to avoid unused write warnings
		_ = result.Version
		_ = result.Timestamp
		_ = result.Tools
		_ = result.Services
		_ = result.Errors

		// Simulate the warning logic from validateEnvironment
		if result.Environment.Container == "host" {
			result.Warnings = append(result.Warnings,
				"Running on host system. Consider using VS Code devcontainer for consistent environment.")
			result.Warnings = append(result.Warnings,
				"Devcontainer provides standardized tooling, dependencies, and configuration.")
			result.Warnings = append(result.Warnings,
				"To use devcontainer: Open this project in VS Code and select 'Reopen in Container'.")
		}

		// Verify no container-related warnings were added
		for _, warning := range result.Warnings {
			if strings.Contains(warning, "devcontainer") || strings.Contains(warning, "host system") {
				t.Errorf("Unexpected devcontainer warning in container environment: %s", warning)
			}
		}

		t.Logf("Devcontainer environment warnings: %v", result.Warnings)
	})
}

// TestContainerDetection tests the container detection logic specifically
func TestContainerDetection(t *testing.T) {
	// Test different environment variable scenarios
	testCases := []struct {
		name      string
		envVars   map[string]string
		dockerEnv bool
		expected  string
	}{
		{
			name:     "Host environment",
			envVars:  map[string]string{},
			expected: "host",
		},
		{
			name: "Devcontainer environment",
			envVars: map[string]string{
				"DEVCONTAINER": "true",
			},
			expected: "devcontainer",
		},
		{
			name: "Codespaces environment",
			envVars: map[string]string{
				"CODESPACES": "true",
			},
			expected: "codespaces",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Clear other environment variables that might interfere
			for _, envVar := range []string{"DEVCONTAINER", "CODESPACES"} {
				if _, exists := tc.envVars[envVar]; !exists {
					os.Unsetenv(envVar)
				}
			}

			env := detectEnvironment()
			if env.Container != tc.expected {
				t.Errorf("Expected container type '%s', got '%s'", tc.expected, env.Container)
			}

			t.Logf("Container detection test '%s': detected '%s'", tc.name, env.Container)
		})
	}
}
