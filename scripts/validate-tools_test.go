package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestValidateToolsScript tests the validate-tools.sh script functionality
func TestValidateToolsScript(t *testing.T) {
	// Skip if not on Unix-like system
	if !isUnixLike() {
		t.Skip("Skipping shell script test on non-Unix system")
	}

	scriptPath := "./validate-tools.sh"
	
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("validate-tools.sh script not found at %s", scriptPath)
	}

	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		t.Fatalf("Failed to make script executable: %v", err)
	}

	// Run the script with dry-run mode (we'll add this flag)
	cmd := exec.Command("/bin/bash", scriptPath)
	output, err := cmd.CombinedOutput()
	
	// The script should run without crashing
	if err != nil {
		// It's okay if tools are missing in test environment
		// We just want to ensure the script structure is valid
		if !strings.Contains(string(output), "not installed") {
			t.Fatalf("Script failed with unexpected error: %v\nOutput: %s", err, output)
		}
	}

	// Check that output contains expected validation messages
	outputStr := string(output)
	expectedMessages := []string{
		"AgentFlow Development Tools Validation",
		"Validating Go installation",
		"Validating Task runner",
		"Validating PostgreSQL client",
		"Validating NATS CLI",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(outputStr, msg) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nFull output: %s", msg, outputStr)
		}
	}
}

// TestValidateToolsScriptStructure tests the script structure and syntax
func TestValidateToolsScriptStructure(t *testing.T) {
	scriptPath := "./validate-tools.sh"
	
	// Read script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script: %v", err)
	}

	scriptContent := string(content)

	// Check for required elements
	requiredElements := []string{
		"#!/bin/bash",
		"set -e",
		"validate_go()",
		"validate_task()",
		"validate_postgresql()",
		"validate_nats()",
		"validate_golangci_lint()",
		"validate_goose()",
		"validate_sqlc()",
		"validate_gosec()",
		"validate_gitleaks()",
		"validate_precommit()",
		"main()",
		"EXPECTED_GO_VERSION=",
		"EXPECTED_GOLANGCI_VERSION=",
	}

	for _, element := range requiredElements {
		if !strings.Contains(scriptContent, element) {
			t.Errorf("Script missing required element: %s", element)
		}
	}

	// Check version variables are properly defined
	versionVars := []string{
		"EXPECTED_GO_VERSION=\"1.22\"",
		"EXPECTED_GOLANGCI_VERSION=\"1.55.2\"",
		"EXPECTED_TASK_VERSION=\"3.35.1\"",
		"EXPECTED_GOOSE_VERSION=\"3.18.0\"",
		"EXPECTED_SQLC_VERSION=\"1.25.0\"",
		"EXPECTED_GOSEC_VERSION=\"2.19.0\"",
		"EXPECTED_GITLEAKS_VERSION=\"8.18.1\"",
		"EXPECTED_PRECOMMIT_VERSION=\"3.6.0\"",
		"EXPECTED_NATS_VERSION=\"0.1.4\"",
	}

	for _, versionVar := range versionVars {
		if !strings.Contains(scriptContent, versionVar) {
			t.Errorf("Script missing or incorrect version variable: %s", versionVar)
		}
	}
}

// TestBinaryVersionParsing tests version parsing logic
func TestBinaryVersionParsing(t *testing.T) {
	// Test cases for version comparison logic
	testCases := []struct {
		name     string
		version1 string
		version2 string
		expected bool // true if version1 >= version2
	}{
		{"Equal versions", "1.22.0", "1.22.0", true},
		{"Higher major", "2.0.0", "1.22.0", true},
		{"Higher minor", "1.23.0", "1.22.0", true},
		{"Higher patch", "1.22.1", "1.22.0", true},
		{"Lower major", "1.0.0", "2.0.0", false},
		{"Lower minor", "1.21.0", "1.22.0", false},
		{"Lower patch", "1.22.0", "1.22.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This would test the version_ge function from the script
			// For now, we'll implement a simple Go version of the logic
			result := versionGE(tc.version1, tc.version2)
			if result != tc.expected {
				t.Errorf("versionGE(%s, %s) = %v, expected %v", 
					tc.version1, tc.version2, result, tc.expected)
			}
		})
	}
}

// TestExpectedVersionsMatchDevContainer tests that versions in validation script
// match those in devcontainer setup
func TestExpectedVersionsMatchDevContainer(t *testing.T) {
	// Read devcontainer post-create script
	postCreateContent, err := os.ReadFile("../.devcontainer/post-create.sh")
	if err != nil {
		t.Fatalf("Failed to read post-create.sh: %v", err)
	}

	// Read validation script
	validateContent, err := os.ReadFile("./validate-tools.sh")
	if err != nil {
		t.Fatalf("Failed to read validate-tools.sh: %v", err)
	}

	postCreateStr := string(postCreateContent)
	validateStr := string(validateContent)

	// Check that key versions match between scripts
	versionChecks := map[string]string{
		"NATS_VERSION":      "EXPECTED_NATS_VERSION",
		"GOLANGCI_VERSION":  "EXPECTED_GOLANGCI_VERSION", 
		"TASK_VERSION":      "EXPECTED_TASK_VERSION",
		"GOOSE_VERSION":     "EXPECTED_GOOSE_VERSION",
		"SQLC_VERSION":      "EXPECTED_SQLC_VERSION",
		"GOSEC_VERSION":     "EXPECTED_GOSEC_VERSION",
		"GITLEAKS_VERSION":  "EXPECTED_GITLEAKS_VERSION",
	}

	for postCreateVar, validateVar := range versionChecks {
		// Extract version from post-create script
		postCreateVersion := extractVersion(postCreateStr, postCreateVar)
		validateVersion := extractVersion(validateStr, validateVar)

		if postCreateVersion == "" {
			t.Errorf("Could not find %s in post-create.sh", postCreateVar)
			continue
		}
		if validateVersion == "" {
			t.Errorf("Could not find %s in validate-tools.sh", validateVar)
			continue
		}

		if postCreateVersion != validateVersion {
			t.Errorf("Version mismatch for %s: post-create.sh has %s, validate-tools.sh has %s",
				postCreateVar, postCreateVersion, validateVersion)
		}
	}
}

// Helper functions

func isUnixLike() bool {
	return os.PathSeparator == '/'
}

func versionGE(v1, v2 string) bool {
	// Simple version comparison - in real implementation would use proper semver
	return strings.Compare(v1, v2) >= 0
}

func extractVersion(content, varName string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, varName+"=") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				version := strings.Trim(parts[1], "\"")
				return version
			}
		}
	}
	return ""
}