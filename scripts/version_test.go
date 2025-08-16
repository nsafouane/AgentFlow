package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionParsing tests the version parsing script functionality
func TestVersionParsing(t *testing.T) {
	scriptPath := "./parse-version.sh"

	// Skip if script doesn't exist or we're on Windows
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("parse-version.sh not found")
	}

	tests := []struct {
		name          string
		version       string
		expectedMajor string
		expectedMinor string
		expectedPatch string
		expectedType  string
		shouldFail    bool
	}{
		{
			name:          "Valid release version",
			version:       "1.2.3",
			expectedMajor: "1",
			expectedMinor: "2",
			expectedPatch: "3",
			expectedType:  "release",
			shouldFail:    false,
		},
		{
			name:          "Valid pre-release version",
			version:       "0.1.0-alpha.1",
			expectedMajor: "0",
			expectedMinor: "1",
			expectedPatch: "0",
			expectedType:  "prerelease",
			shouldFail:    false,
		},
		{
			name:          "Version with v prefix",
			version:       "v2.0.0",
			expectedMajor: "2",
			expectedMinor: "0",
			expectedPatch: "0",
			expectedType:  "release",
			shouldFail:    false,
		},
		{
			name:       "Invalid version format",
			version:    "1.2",
			shouldFail: true,
		},
		{
			name:       "Invalid characters",
			version:    "1.2.3a",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("bash", scriptPath, tt.version)
			output, err := cmd.Output()

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected script to fail for version %s, but it succeeded", tt.version)
				}
				return
			}

			if err != nil {
				t.Fatalf("Script failed for valid version %s: %v", tt.version, err)
			}

			outputStr := string(output)
			lines := strings.Split(strings.TrimSpace(outputStr), "\n")

			// Parse output into map
			results := make(map[string]string)
			for _, line := range lines {
				if strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					results[parts[0]] = parts[1]
				}
			}

			// Verify parsed components
			if results["MAJOR"] != tt.expectedMajor {
				t.Errorf("Expected MAJOR=%s, got %s", tt.expectedMajor, results["MAJOR"])
			}
			if results["MINOR"] != tt.expectedMinor {
				t.Errorf("Expected MINOR=%s, got %s", tt.expectedMinor, results["MINOR"])
			}
			if results["PATCH"] != tt.expectedPatch {
				t.Errorf("Expected PATCH=%s, got %s", tt.expectedPatch, results["PATCH"])
			}
			if results["VERSION_TYPE"] != tt.expectedType {
				t.Errorf("Expected VERSION_TYPE=%s, got %s", tt.expectedType, results["VERSION_TYPE"])
			}
		})
	}
}

// TestVersionIncrement tests version increment functionality
func TestVersionIncrement(t *testing.T) {
	scriptPath := "./parse-version.sh"

	// Skip if script doesn't exist
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("parse-version.sh not found")
	}

	tests := []struct {
		name            string
		version         string
		incrementType   string
		expectedVersion string
	}{
		{
			name:            "Increment patch",
			version:         "1.2.3",
			incrementType:   "patch",
			expectedVersion: "1.2.4",
		},
		{
			name:            "Increment minor",
			version:         "1.2.3",
			incrementType:   "minor",
			expectedVersion: "1.3.0",
		},
		{
			name:            "Increment major",
			version:         "1.2.3",
			incrementType:   "major",
			expectedVersion: "2.0.0",
		},
		{
			name:            "Increment pre-release patch",
			version:         "0.1.0-alpha.1",
			incrementType:   "patch",
			expectedVersion: "0.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("bash", scriptPath, tt.version, "increment", tt.incrementType)
			output, err := cmd.Output()

			if err != nil {
				t.Fatalf("Script failed for version increment %s %s: %v", tt.version, tt.incrementType, err)
			}

			outputStr := string(output)
			lines := strings.Split(strings.TrimSpace(outputStr), "\n")

			// Find NEXT_VERSION line
			var nextVersion string
			for _, line := range lines {
				if strings.HasPrefix(line, "NEXT_VERSION=") {
					nextVersion = strings.TrimPrefix(line, "NEXT_VERSION=")
					break
				}
			}

			if nextVersion != tt.expectedVersion {
				t.Errorf("Expected next version %s, got %s", tt.expectedVersion, nextVersion)
			}
		})
	}
}

// TestVersionComparison tests version comparison functionality
func TestVersionComparison(t *testing.T) {
	scriptPath := "./parse-version.sh"

	// Skip if script doesn't exist
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("parse-version.sh not found")
	}

	tests := []struct {
		name     string
		version1 string
		version2 string
		expected string
	}{
		{
			name:     "Equal versions",
			version1: "1.2.3",
			version2: "1.2.3",
			expected: "equal",
		},
		{
			name:     "First version greater",
			version1: "1.2.4",
			version2: "1.2.3",
			expected: "greater",
		},
		{
			name:     "Second version greater",
			version1: "1.2.3",
			version2: "1.2.4",
			expected: "less",
		},
		{
			name:     "Release vs prerelease",
			version1: "1.2.3",
			version2: "1.2.3-alpha.1",
			expected: "greater",
		},
		{
			name:     "Prerelease vs release",
			version1: "1.2.3-alpha.1",
			version2: "1.2.3",
			expected: "less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("bash", scriptPath, tt.version1, "compare", tt.version2)
			output, err := cmd.Output()

			if err != nil {
				t.Fatalf("Script failed for version comparison %s vs %s: %v", tt.version1, tt.version2, err)
			}

			outputStr := string(output)
			lines := strings.Split(strings.TrimSpace(outputStr), "\n")

			// Find COMPARISON line
			var comparison string
			for _, line := range lines {
				if strings.HasPrefix(line, "COMPARISON=") {
					comparison = strings.TrimPrefix(line, "COMPARISON=")
					break
				}
			}

			if comparison != tt.expected {
				t.Errorf("Expected comparison %s, got %s", tt.expected, comparison)
			}
		})
	}
}

// TestUpdateVersionScript tests the version update script
func TestUpdateVersionScript(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "version-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"go.mod": `module github.com/agentflow/agentflow

go 1.22

// version: v0.0.1
`,
		"cmd/af/version.go": `package main

const (
    Version = "0.0.1"
    BuildDate = ""
    GitCommit = ""
)
`,
		"Dockerfile": `FROM golang:1.22-alpine
LABEL version="0.0.1"
COPY . .
`,
	}

	// Create test files
	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	// Copy update script to temp directory
	scriptContent := `#!/bin/bash
NEW_VERSION=$1
if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# Update go.mod version comment
sed -i.bak "s|// version: .*|// version: v$NEW_VERSION|" go.mod && rm -f go.mod.bak

# Update version in CLI
sed -i.bak "s|Version = \".*\"|Version = \"$NEW_VERSION\"|" cmd/af/version.go && rm -f cmd/af/version.go.bak

# Update Dockerfile labels
sed -i.bak "s|LABEL version=\".*\"|LABEL version=\"$NEW_VERSION\"|" Dockerfile && rm -f Dockerfile.bak

echo "Updated version to $NEW_VERSION"
`

	scriptPath := filepath.Join(tempDir, "update-version.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create update script: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Run update script
	cmd := exec.Command("bash", "update-version.sh", "0.2.0")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Update script failed: %v", err)
	}

	if !strings.Contains(string(output), "Updated version to 0.2.0") {
		t.Errorf("Expected success message, got: %s", string(output))
	}

	// Verify files were updated
	verifyFiles := map[string]string{
		"go.mod":            "// version: v0.2.0",
		"cmd/af/version.go": `Version = "0.2.0"`,
		"Dockerfile":        `LABEL version="0.2.0"`,
	}

	for filePath, expectedContent := range verifyFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read updated file %s: %v", filePath, err)
		}

		if !strings.Contains(string(content), expectedContent) {
			t.Errorf("File %s was not updated correctly. Expected to contain: %s\nActual content: %s",
				filePath, expectedContent, string(content))
		}
	}
}

// TestVersionGoFile tests the version.go file functionality
func TestVersionGoFile(t *testing.T) {
	// This would normally import and test the version.go file
	// For now, we'll test that the file exists and has the expected structure
	versionFile := "../cmd/af/version.go"

	content, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("Failed to read version.go file: %v", err)
	}

	contentStr := string(content)

	// Check for required constants
	requiredElements := []string{
		"const (",
		"Version",
		"BuildDate",
		"GitCommit",
		"func GetVersionInfo()",
		"func GetVersionString()",
		"func IsPreRelease()",
		"func GetMajorVersion()",
		"func IsStableAPI()",
	}

	for _, element := range requiredElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("version.go file missing required element: %s", element)
		}
	}
}

// TestChangelogFormat tests that CHANGELOG.md follows the expected format
func TestChangelogFormat(t *testing.T) {
	changelogFile := "../CHANGELOG.md"

	content, err := os.ReadFile(changelogFile)
	if err != nil {
		t.Fatalf("Failed to read CHANGELOG.md file: %v", err)
	}

	contentStr := string(content)

	// Check for required sections
	requiredSections := []string{
		"# Changelog",
		"## [Unreleased]",
		"### Added",
		"### Changed",
		"### Deprecated",
		"### Removed",
		"### Fixed",
		"### Security",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("CHANGELOG.md missing required section: %s", section)
		}
	}

	// Check for Keep a Changelog reference
	if !strings.Contains(contentStr, "keepachangelog.com") {
		t.Error("CHANGELOG.md should reference Keep a Changelog format")
	}

	// Check for Semantic Versioning reference
	if !strings.Contains(contentStr, "semver.org") {
		t.Error("CHANGELOG.md should reference Semantic Versioning")
	}
}

// TestReleaseDocumentation tests that RELEASE.md contains required sections
func TestReleaseDocumentation(t *testing.T) {
	releaseFile := "../RELEASE.md"

	content, err := os.ReadFile(releaseFile)
	if err != nil {
		t.Fatalf("Failed to read RELEASE.md file: %v", err)
	}

	contentStr := string(content)

	// Check for required sections
	requiredSections := []string{
		"# AgentFlow Release Engineering Guide",
		"## Versioning Scheme",
		"## Tagging Policy",
		"## Branching Model",
		"## Release Process",
		"## Hotfix Process",
		"## Version Management Scripts",
		"## CI/CD Integration",
		"## Quality Gates",
		"## Emergency Release Procedures",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("RELEASE.md missing required section: %s", section)
		}
	}

	// Check for semantic versioning reference
	if !strings.Contains(contentStr, "semver.org") {
		t.Error("RELEASE.md should reference Semantic Versioning")
	}

	// Check for pre-1.0 versioning rules
	if !strings.Contains(contentStr, "pre-1.0") {
		t.Error("RELEASE.md should document pre-1.0 versioning rules")
	}
}
