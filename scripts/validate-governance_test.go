package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidateGovernance(t *testing.T) {
	// Change to project root for tests
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	tests := []struct {
		name     string
		command  string
		wantCode int
	}{
		{
			name:     "risk schema validation",
			command:  "risk-schema",
			wantCode: 0,
		},
		{
			name:     "ADR filename validation",
			command:  "adr-filenames",
			wantCode: 0,
		},
		{
			name:     "all validations",
			command:  "all",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", "scripts/validate-governance.go", tt.command)
			err := cmd.Run()

			var exitCode int
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("Failed to run validation command: %v", err)
				}
			}

			if exitCode != tt.wantCode {
				t.Errorf("Expected exit code %d, got %d", tt.wantCode, exitCode)
			}
		})
	}
}

func TestRiskRegisterExists(t *testing.T) {
	// Change to project root for tests
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	riskFile := "docs/risk-register.yaml"
	if _, err := os.Stat(riskFile); os.IsNotExist(err) {
		t.Errorf("Risk register file does not exist: %s", riskFile)
	}
}

func TestADRDirectoryStructure(t *testing.T) {
	// Change to project root for tests
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	adrDir := "docs/adr"
	if _, err := os.Stat(adrDir); os.IsNotExist(err) {
		t.Errorf("ADR directory does not exist: %s", adrDir)
		return
	}

	// Check for template
	templateFile := filepath.Join(adrDir, "template.md")
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		t.Errorf("ADR template does not exist: %s", templateFile)
	}

	// Check for at least one ADR
	files, err := os.ReadDir(adrDir)
	if err != nil {
		t.Fatalf("Failed to read ADR directory: %v", err)
	}

	adrCount := 0
	for _, file := range files {
		if !file.IsDir() && file.Name() != "template.md" && filepath.Ext(file.Name()) == ".md" {
			adrCount++
		}
	}

	if adrCount == 0 {
		t.Error("No ADR files found in ADR directory")
	}
}
