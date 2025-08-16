package main

import (
	"strings"
	"testing"
)

// TestVersionConstants tests that version constants are properly defined
func TestVersionConstants(t *testing.T) {
	if Version == "" {
		t.Error("Version constant should not be empty")
	}

	// Version should follow semantic versioning format
	if !strings.Contains(Version, ".") {
		t.Errorf("Version should contain dots for semantic versioning, got: %s", Version)
	}

	t.Logf("Version: %s", Version)
}

// TestGetVersionInfo tests the GetVersionInfo function
func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if info.Version == "" {
		t.Error("VersionInfo.Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("VersionInfo.GoVersion should not be empty")
	}

	if info.Platform == "" {
		t.Error("VersionInfo.Platform should not be empty")
	}

	if info.Arch == "" {
		t.Error("VersionInfo.Arch should not be empty")
	}

	// Verify Go version starts with "go"
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Errorf("GoVersion should start with 'go', got: %s", info.GoVersion)
	}

	t.Logf("Version Info: %+v", info)
}

// TestGetVersionString tests the formatted version string
func TestGetVersionString(t *testing.T) {
	versionStr := GetVersionString()

	if versionStr == "" {
		t.Error("Version string should not be empty")
	}

	// Should contain "AgentFlow CLI"
	if !strings.Contains(versionStr, "AgentFlow CLI") {
		t.Errorf("Version string should contain 'AgentFlow CLI', got: %s", versionStr)
	}

	// Should contain the version number
	if !strings.Contains(versionStr, Version) {
		t.Errorf("Version string should contain version %s, got: %s", Version, versionStr)
	}

	t.Logf("Version String: %s", versionStr)
}

// TestIsPreRelease tests pre-release detection
func TestIsPreRelease(t *testing.T) {
	// Test with current version
	isPreRelease := IsPreRelease()
	t.Logf("Current version %s is pre-release: %t", Version, isPreRelease)

	// The logic should detect if version contains a hyphen
	expectedPreRelease := strings.Contains(Version, "-")
	if isPreRelease != expectedPreRelease {
		t.Errorf("IsPreRelease() returned %t, expected %t for version %s",
			isPreRelease, expectedPreRelease, Version)
	}
}

// TestGetMajorVersion tests major version extraction
func TestGetMajorVersion(t *testing.T) {
	major := GetMajorVersion()

	if major == "" {
		t.Error("Major version should not be empty")
	}

	// Major version should be numeric
	if major != "0" && major != "1" && major != "2" && major != "3" && major != "4" &&
		major != "5" && major != "6" && major != "7" && major != "8" && major != "9" {
		// Allow multi-digit major versions
		allDigits := true
		for _, char := range major {
			if char < '0' || char > '9' {
				allDigits = false
				break
			}
		}
		if !allDigits {
			t.Errorf("Major version should be numeric, got: %s", major)
		}
	}

	t.Logf("Major version: %s", major)
}

// TestIsStableAPI tests stable API detection
func TestIsStableAPI(t *testing.T) {
	isStable := IsStableAPI()
	major := GetMajorVersion()
	isPreRelease := IsPreRelease()

	// API is stable if major version is not "0" and not a pre-release
	expectedStable := major != "0" && !isPreRelease

	if isStable != expectedStable {
		t.Errorf("IsStableAPI() returned %t, expected %t (major=%s, prerelease=%t)",
			isStable, expectedStable, major, isPreRelease)
	}

	t.Logf("Version %s has stable API: %t", Version, isStable)
}

// TestVersionConsistency tests that all version functions are consistent
func TestVersionConsistency(t *testing.T) {
	info := GetVersionInfo()
	versionStr := GetVersionString()
	major := GetMajorVersion()

	// Version from info should match constant
	if info.Version != Version {
		t.Errorf("VersionInfo.Version (%s) doesn't match Version constant (%s)",
			info.Version, Version)
	}

	// Version string should contain the version
	if !strings.Contains(versionStr, Version) {
		t.Errorf("Version string doesn't contain version %s: %s", Version, versionStr)
	}

	// Major version should be prefix of full version
	if !strings.HasPrefix(Version, major) {
		t.Errorf("Version %s doesn't start with major version %s", Version, major)
	}

	t.Log("All version functions are consistent")
}

// TestVersionFormat tests that version follows expected format
func TestVersionFormat(t *testing.T) {
	// Version should follow semantic versioning: MAJOR.MINOR.PATCH[-PRERELEASE]
	parts := strings.Split(Version, ".")
	if len(parts) < 3 {
		t.Errorf("Version should have at least 3 parts (MAJOR.MINOR.PATCH), got: %s", Version)
		return
	}

	// Check that first three parts are numeric (allowing for pre-release suffix on patch)
	for i, part := range parts[:3] {
		// For patch version, split on hyphen to handle pre-release
		if i == 2 && strings.Contains(part, "-") {
			part = strings.Split(part, "-")[0]
		}

		// Check if part is numeric
		allDigits := true
		for _, char := range part {
			if char < '0' || char > '9' {
				allDigits = false
				break
			}
		}

		if !allDigits {
			t.Errorf("Version part %d should be numeric, got: %s in version %s", i, part, Version)
		}
	}

	t.Logf("Version format validation passed for: %s", Version)
}
