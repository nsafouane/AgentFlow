package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestValidationReport represents the structure of the validation report
type TestValidationReport struct {
	Timestamp         string   `json:"timestamp"`
	ValidationStatus  string   `json:"validation_status"`
	SecurityThreshold string   `json:"security_threshold"`
	TotalValidations  int      `json:"total_validations"`
	Validations       []string `json:"validations"`
	ReportsDirectory  string   `json:"reports_directory"`
	GitHubRepository  string   `json:"github_repository"`
	GitCommit         string   `json:"git_commit"`
	GitBranch         string   `json:"git_branch"`
}

// TestValidationReportGeneration tests validation report generation and parsing
func TestValidationReportGeneration(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "report-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock validation report
	report := TestValidationReport{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		ValidationStatus:  "PASSED",
		SecurityThreshold: "high",
		TotalValidations:  3,
		Validations: []string{
			"Security Thresholds: PASSED - Test",
			"Security Tools: PASSED - Test",
			"SARIF Upload: PASSED - Test",
		},
		ReportsDirectory: tempDir,
		GitHubRepository: "test/repo",
		GitCommit:        "abc123",
		GitBranch:        "main",
	}

	// Write report to file
	reportPath := filepath.Join(tempDir, "test-report.json")
	reportData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal report: %v", err)
	}

	if err := ioutil.WriteFile(reportPath, reportData, 0644); err != nil {
		t.Fatalf("Failed to write report: %v", err)
	}

	// Read and validate report
	readData, err := ioutil.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}

	var readReport TestValidationReport
	if err := json.Unmarshal(readData, &readReport); err != nil {
		t.Fatalf("Failed to unmarshal report: %v", err)
	}

	// Validate report fields
	if readReport.ValidationStatus != "PASSED" {
		t.Errorf("Expected validation status 'PASSED', got '%s'", readReport.ValidationStatus)
	}

	if readReport.TotalValidations != 3 {
		t.Errorf("Expected 3 total validations, got %d", readReport.TotalValidations)
	}

	// Validate timestamp format
	if _, err := time.Parse(time.RFC3339, readReport.Timestamp); err != nil {
		t.Errorf("Invalid timestamp format: %s", readReport.Timestamp)
	}
}

// TestSecurityThresholdValidation tests security threshold validation logic
func TestSecurityThresholdValidation(t *testing.T) {
	testCases := []struct {
		name            string
		content         string
		expectThreshold bool
	}{
		{
			name:            "Workflow with high threshold",
			content:         "run: grype . --fail-on high",
			expectThreshold: true,
		},
		{
			name:            "Workflow with critical severity",
			content:         "run: scanner --severity CRITICAL",
			expectThreshold: true,
		},
		{
			name:            "Workflow without threshold",
			content:         "run: make build",
			expectThreshold: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasThreshold := strings.Contains(tc.content, "fail-on") && (strings.Contains(tc.content, "high") || strings.Contains(tc.content, "HIGH")) ||
				strings.Contains(tc.content, "severity") && (strings.Contains(tc.content, "high") || strings.Contains(tc.content, "HIGH") || strings.Contains(tc.content, "CRITICAL"))

			if hasThreshold != tc.expectThreshold {
				t.Errorf("Expected threshold detection %v, got %v for content: %s", tc.expectThreshold, hasThreshold, tc.content)
			}
		})
	}
}

// TestSecurityToolsValidation tests security tools validation
func TestSecurityToolsValidation(t *testing.T) {
	requiredTools := []string{"gosec", "gitleaks", "grype", "syft", "govulncheck"}
	workflowContent := `
name: Security
jobs:
  security:
    steps:
      - name: Run gosec
        run: gosec ./...
      - name: Run gitleaks
        run: gitleaks detect
      - name: Run grype
        run: grype .
      - name: Run syft
        run: syft packages .
      - name: Run govulncheck
        run: govulncheck ./...
`

	toolsFound := 0
	for _, tool := range requiredTools {
		if strings.Contains(workflowContent, tool) {
			toolsFound++
		}
	}

	if toolsFound != len(requiredTools) {
		t.Errorf("Expected to find %d tools, found %d", len(requiredTools), toolsFound)
	}
}

// TestScriptExistence tests that the validation scripts exist
func TestScriptExistence(t *testing.T) {
	psScript := "validate-ci-green.ps1"
	bashScript := "validate-ci-green.sh"

	var scriptFound bool

	if _, err := os.Stat(psScript); err == nil {
		scriptFound = true
		t.Logf("Found PowerShell script: %s", psScript)
	}

	if _, err := os.Stat(bashScript); err == nil {
		scriptFound = true
		t.Logf("Found bash script: %s", bashScript)
	}

	if !scriptFound {
		t.Errorf("No validation script found (expected %s or %s)", psScript, bashScript)
	}
}

// TestValidationLogic tests the core validation logic components
func TestValidationLogic(t *testing.T) {
	validThresholds := []string{"critical", "high", "medium", "low", "info"}
	invalidThresholds := []string{"invalid", "unknown", ""}

	for _, threshold := range validThresholds {
		if !isValidThreshold(threshold) {
			t.Errorf("Expected '%s' to be a valid threshold", threshold)
		}
	}

	for _, threshold := range invalidThresholds {
		if isValidThreshold(threshold) {
			t.Errorf("Expected '%s' to be an invalid threshold", threshold)
		}
	}
}

// isValidThreshold simulates threshold validation logic
func isValidThreshold(threshold string) bool {
	validThresholds := []string{"critical", "high", "medium", "low", "info"}
	for _, valid := range validThresholds {
		if threshold == valid {
			return true
		}
	}
	return false
}
