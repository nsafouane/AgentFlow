package main

import (
	"fmt"
	"runtime"
)

// Version information for AgentFlow CLI
// These values are set during build time
const (
	Version   = "0.1.0"
	BuildDate = ""
	GitCommit = ""
)

// VersionInfo contains detailed version information
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	Arch      string `json:"arch"`
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	info := GetVersionInfo()

	versionStr := fmt.Sprintf("AgentFlow CLI v%s", info.Version)

	if info.GitCommit != "" {
		if len(info.GitCommit) > 7 {
			versionStr += fmt.Sprintf(" (%s)", info.GitCommit[:7])
		} else {
			versionStr += fmt.Sprintf(" (%s)", info.GitCommit)
		}
	}

	if info.BuildDate != "" {
		versionStr += fmt.Sprintf(" built on %s", info.BuildDate)
	}

	versionStr += fmt.Sprintf(" %s/%s", info.Platform, info.Arch)

	return versionStr
}

// IsPreRelease returns true if this is a pre-release version
func IsPreRelease() bool {
	// Check if version contains pre-release identifiers
	for _, char := range Version {
		if char == '-' {
			return true
		}
	}
	return false
}

// GetMajorVersion returns the major version number
func GetMajorVersion() string {
	for i, char := range Version {
		if char == '.' {
			return Version[:i]
		}
	}
	return Version
}

// IsStableAPI returns true if this version has stable API guarantees
func IsStableAPI() bool {
	major := GetMajorVersion()
	return major != "0" && !IsPreRelease()
}
