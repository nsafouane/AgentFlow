package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateGateG0 tests the comprehensive Gate G0 validation logic
func TestValidateGateG0(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() error
		cleanupFunc    func() error
		expectedErrors int
		expectedPass   bool
	}{
		{
			name: "All Gate G0 criteria pass",
			setupFunc: func() error {
				return setupValidProject()
			},
			cleanupFunc: func() error {
				return cleanupTestProject()
			},
			expectedErrors: 0,
			expectedPass:   true,
		},
		{
			name: "Missing CI workflows",
			setupFunc: func() error {
				if err := setupValidProject(); err != nil {
					return err
				}
				// Remove CI workflow to simulate failure
				return os.Remove(".github/workflows/ci.yml")
			},
			cleanupFunc: func() error {
				return cleanupTestProject()
			},
			expectedErrors: 1,
			expectedPass:   false,
		},
		{
			name: "Insufficient risks in register",
			setupFunc: func() error {
				if err := setupValidProject(); err != nil {
					return err
				}
				// Create risk register with insufficient risks
				return createMinimalRiskRegister()
			},
			cleanupFunc: func() error {
				return cleanupTestProject()
			},
			expectedErrors: 1,
			expectedPass:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Cleanup after test
			defer func() {
				if tt.cleanupFunc != nil {
					if err := tt.cleanupFunc(); err != nil {
						t.Errorf("Cleanup failed: %v", err)
					}
				}
			}()

			// Run validation logic (simulated)
			errors := runGateG0Validation()

			if errors != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d", tt.expectedErrors, errors)
			}

			passed := errors == 0
			if passed != tt.expectedPass {
				t.Errorf("Expected pass=%v, got pass=%v", tt.expectedPass, passed)
			}
		})
	}
}

// TestSignatureVerification tests container image signature verification
func TestSignatureVerification(t *testing.T) {
	tests := []struct {
		name     string
		imageRef string
		expected bool
	}{
		{
			name:     "Valid signed image",
			imageRef: "ghcr.io/agentflow/agentflow/control-plane:latest",
			expected: true,
		},
		{
			name:     "Unsigned image",
			imageRef: "alpine:latest",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock signature verification
			result := mockCosignVerify(tt.imageRef)
			if result != tt.expected {
				t.Errorf("Expected signature verification %v for %s, got %v",
					tt.expected, tt.imageRef, result)
			}
		})
	}
}

// TestGovernanceArtifactCompliance tests governance artifact validation
func TestGovernanceArtifactCompliance(t *testing.T) {
	tests := []struct {
		name     string
		artifact string
		expected bool
	}{
		{
			name:     "Valid risk register",
			artifact: "docs/risk-register.yaml",
			expected: true,
		},
		{
			name:     "Valid ADR baseline",
			artifact: "docs/adr/ADR-0001-architecture-baseline.md",
			expected: true,
		},
		{
			name:     "Valid RELEASE.md",
			artifact: "RELEASE.md",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateGovernanceArtifact(tt.artifact)
			if result != tt.expected {
				t.Errorf("Expected governance artifact validation %v for %s, got %v",
					tt.expected, tt.artifact, result)
			}
		})
	}
}

// TestCIPolicyReferences tests CI policy reference validation
func TestCIPolicyReferences(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		expected bool
	}{
		{
			name:     "CI workflow with security scans",
			workflow: ".github/workflows/ci.yml",
			expected: true,
		},
		{
			name:     "Release workflow with versioning",
			workflow: ".github/workflows/release.yml",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCIPolicyReferences(tt.workflow)
			if result != tt.expected {
				t.Errorf("Expected CI policy validation %v for %s, got %v",
					tt.expected, tt.workflow, result)
			}
		})
	}
}

// TestInterfaceDocumentationCompleteness tests interface documentation validation
func TestInterfaceDocumentationCompleteness(t *testing.T) {
	requiredSections := []string{
		"Agent Runtime Interfaces",
		"Planning Interfaces",
		"Tool Execution Interfaces",
		"Memory Interfaces",
		"Messaging Interfaces",
	}

	for _, section := range requiredSections {
		t.Run("Section: "+section, func(t *testing.T) {
			result := validateInterfaceSection(section)
			if !result {
				t.Errorf("Interface section '%s' validation failed", section)
			}
		})
	}
}

// TestThreatModelingEntryValidation tests threat modeling validation
func TestThreatModelingEntryValidation(t *testing.T) {
	requiredFields := []string{
		"session_date",
		"owner",
		"participants",
		"scope",
	}

	for _, field := range requiredFields {
		t.Run("Field: "+field, func(t *testing.T) {
			result := validateThreatModelingField(field)
			if !result {
				t.Errorf("Threat modeling field '%s' validation failed", field)
			}
		})
	}
}

// Helper functions for test setup and validation

func setupValidProject() error {
	// Create necessary directories
	dirs := []string{
		".github/workflows",
		"docs/adr",
		"docs/interfaces",
		"cmd/af",
		".devcontainer",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create minimal valid files
	files := map[string]string{
		".github/workflows/ci.yml": `
name: CI
on: [push, pull_request]
jobs:
  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Run gosec
      run: gosec ./...
    - name: Run gitleaks
      run: gitleaks detect
    - name: Run osv-scanner
      run: osv-scanner .
    - name: Run grype
      run: grype .
`,
		".github/workflows/security-scan.yml": `
name: Security Scan
on: [push]
jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
    - name: Security scan with HIGH/CRITICAL thresholds
      run: echo "HIGH CRITICAL thresholds configured"
`,
		".github/workflows/container-build.yml": `
name: Container Build
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - name: Build multi-arch
      run: docker buildx build --platform linux/amd64,linux/arm64 .
    - name: Sign with cosign
      run: cosign sign image
    - name: Verify signature
      run: cosign verify image
    - name: Generate SBOM
      run: syft image -o spdx-json
    - name: Attest provenance
      run: echo "provenance true"
`,
		"Makefile": `
GOOS ?= linux
GOARCH ?= amd64
build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build ./...
`,
		"Taskfile.yml": `
version: '3'
tasks:
  build:
    cmds:
    - go build ./...
    env:
      GOOS: "{{.GOOS | default \"linux\"}}"
      GOARCH: "{{.GOARCH | default \"amd64\"}}"
`,
		"docs/cross-platform-build-troubleshooting.md": "# Cross-platform build troubleshooting",
		".devcontainer/devcontainer.json":              `{"name": "AgentFlow Dev"}`,
		"cmd/af/main.go": `
package main
import "fmt"
func main() {
	if len(os.Args) > 1 && os.Args[1] == "validate" {
		fmt.Println("Validation command")
	}
}
`,
		"docs/devcontainer-adoption-guide.md":  "# Devcontainer adoption guide",
		"docs/sbom-provenance-verification.md": "# SBOM and provenance verification",
		"docs/security-baseline.md":            "# Security baseline with cosign signing and supply chain security",
		"CONTRIBUTING.md":                      "# Contributing guide with ADR and decision record process",
		".github/workflows/release.yml": `
name: Release
on:
  push:
    tags: ['v*']
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - name: Parse version tag
      run: echo "Parsing version from tag"
`,
		"scripts/update-version.sh": "#!/bin/bash\necho 'Version update script'",
		"README.md":                 "# AgentFlow with interface and API documentation",
		"docs/ARCHITECTURE.md":      "# Architecture with interface and API references",
	}

	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func cleanupTestProject() error {
	// Remove test artifacts
	paths := []string{
		".github",
		"docs",
		"cmd",
		".devcontainer",
		"Makefile",
		"Taskfile.yml",
		"CONTRIBUTING.md",
		"README.md",
		"scripts",
	}

	for _, path := range paths {
		os.RemoveAll(path)
	}

	return nil
}

func createMinimalRiskRegister() error {
	content := `
metadata:
  version: "1.0"
risks:
  - id: "RISK-001"
    title: "Test risk"
    severity: "low"
`
	return os.WriteFile("docs/risk-register.yaml", []byte(content), 0644)
}

func runGateG0Validation() int {
	// Simulate validation logic - count missing artifacts
	errors := 0

	requiredFiles := []string{
		".github/workflows/ci.yml",
		".github/workflows/security-scan.yml",
		".github/workflows/container-build.yml",
		"docs/risk-register.yaml",
		"docs/adr/ADR-0001-architecture-baseline.md",
		"RELEASE.md",
		"docs/interfaces/README.md",
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			errors++
		}
	}

	// Check risk register has sufficient risks
	if content, err := os.ReadFile("docs/risk-register.yaml"); err == nil {
		riskCount := strings.Count(string(content), "- id:")
		if riskCount < 8 {
			errors++
		}
	}

	return errors
}

func mockCosignVerify(imageRef string) bool {
	// Mock cosign verification - return true for AgentFlow images
	return strings.Contains(imageRef, "agentflow")
}

func validateGovernanceArtifact(artifact string) bool {
	// Check if governance artifact exists and has required content
	if _, err := os.Stat(artifact); os.IsNotExist(err) {
		return false
	}

	content, err := os.ReadFile(artifact)
	if err != nil {
		return false
	}

	switch artifact {
	case "docs/risk-register.yaml":
		return strings.Contains(string(content), "risks:") &&
			strings.Contains(string(content), "threat_modeling:")
	case "docs/adr/ADR-0001-architecture-baseline.md":
		return strings.Contains(string(content), "## Status") &&
			strings.Contains(string(content), "## Context") &&
			strings.Contains(string(content), "## Decision")
	case "RELEASE.md":
		return strings.Contains(string(content), "Versioning Scheme") &&
			strings.Contains(string(content), "semantic")
	}

	return true
}

func validateCIPolicyReferences(workflow string) bool {
	if _, err := os.Stat(workflow); os.IsNotExist(err) {
		return false
	}

	content, err := os.ReadFile(workflow)
	if err != nil {
		return false
	}

	switch workflow {
	case ".github/workflows/ci.yml":
		return strings.Contains(string(content), "gosec") &&
			strings.Contains(string(content), "gitleaks")
	case ".github/workflows/release.yml":
		return strings.Contains(string(content), "tag") ||
			strings.Contains(string(content), "version")
	}

	return true
}

func validateInterfaceSection(section string) bool {
	content, err := os.ReadFile("docs/interfaces/README.md")
	if err != nil {
		return false
	}

	return strings.Contains(string(content), section)
}

func validateThreatModelingField(field string) bool {
	content, err := os.ReadFile("docs/risk-register.yaml")
	if err != nil {
		return false
	}

	return strings.Contains(string(content), field+":")
}
