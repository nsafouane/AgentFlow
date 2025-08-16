package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	_ "testing"
)

// CrossPlatformBuildTest represents a cross-platform build test configuration
type CrossPlatformBuildTest struct {
	GOOS     string
	GOARCH   string
	Services []string
	BinDir   string
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Service  string
	Platform string
	Success  bool
	Error    error
	BinPath  string
}

func TestCrossPlatformBuilds(t *testing.T) {
	// Define build configurations
	buildConfigs := []CrossPlatformBuildTest{
		{
			GOOS:     "linux",
			GOARCH:   "amd64",
			Services: []string{"control-plane", "worker", "af"},
			BinDir:   "bin/linux",
		},
		{
			GOOS:     "windows",
			GOARCH:   "amd64",
			Services: []string{"control-plane", "worker", "af"},
			BinDir:   "bin/windows",
		},
		{
			GOOS:     "darwin",
			GOARCH:   "amd64",
			Services: []string{"control-plane", "worker", "af"},
			BinDir:   "bin/darwin",
		},
	}

	for _, config := range buildConfigs {
		t.Run(fmt.Sprintf("build-%s-%s", config.GOOS, config.GOARCH), func(t *testing.T) {
			results := testPlatformBuild(t, config)

			// Verify all builds succeeded
			for _, result := range results {
				if !result.Success {
					t.Errorf("Build failed for %s on %s: %v", result.Service, result.Platform, result.Error)
				} else {
					t.Logf("✅ Build succeeded for %s on %s: %s", result.Service, result.Platform, result.BinPath)
				}
			}
		})
	}
}

func TestMakefileCrossPlatformTargets(t *testing.T) {
	// Test Makefile cross-platform targets
	targets := []string{"build-linux", "build-windows", "build-all"}

	for _, target := range targets {
		t.Run(fmt.Sprintf("makefile-%s", target), func(t *testing.T) {
			cmd := exec.Command("make", target)
			cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Makefile target %s failed: %v\nOutput: %s", target, err, string(output))
			} else {
				t.Logf("✅ Makefile target %s succeeded", target)
			}
		})
	}
}

func TestTaskfileCrossPlatformTargets(t *testing.T) {
	// Check if task is available
	if _, err := exec.LookPath("task"); err != nil {
		t.Skip("Task not available, skipping Taskfile tests")
	}

	// Test Taskfile cross-platform targets
	targets := []string{"build-linux", "build-windows", "build-all"}

	for _, target := range targets {
		t.Run(fmt.Sprintf("taskfile-%s", target), func(t *testing.T) {
			cmd := exec.Command("task", target)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Taskfile target %s failed: %v\nOutput: %s", target, err, string(output))
			} else {
				t.Logf("✅ Taskfile target %s succeeded", target)
			}
		})
	}
}

func TestWSL2Compatibility(t *testing.T) {
	// Only run on Windows or if WSL is detected
	if runtime.GOOS != "windows" && !isWSL() {
		t.Skip("Not running on Windows or WSL, skipping WSL2 compatibility test")
	}

	// Test that builds work in WSL2 environment
	t.Run("wsl2-build", func(t *testing.T) {
		// Test basic Go build in WSL2 context
		cmd := exec.Command("go", "build", "-o", "../../bin/test-wsl2")
		cmd.Dir = "cmd/af"
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("WSL2 build failed: %v\nOutput: %s", err, string(output))
		} else {
			t.Logf("✅ WSL2 build succeeded")

			// Clean up test binary
			os.Remove("bin/test-wsl2")
		}
	})
}

func TestBuildArtifactValidation(t *testing.T) {
	// Test that built artifacts are valid executables
	buildConfigs := []struct {
		platform string
		binDir   string
		ext      string
	}{
		{"linux", "bin/linux", ""},
		{"windows", "bin/windows", ".exe"},
	}

	services := []string{"control-plane", "worker", "af"}

	for _, config := range buildConfigs {
		for _, service := range services {
			t.Run(fmt.Sprintf("validate-%s-%s", config.platform, service), func(t *testing.T) {
				binPath := filepath.Join(config.binDir, service+config.ext)

				// Check if binary exists
				if _, err := os.Stat(binPath); os.IsNotExist(err) {
					t.Skipf("Binary %s not found, skipping validation", binPath)
					return
				}

				// Check if binary is executable (on Unix-like systems)
				if config.platform != "windows" {
					info, err := os.Stat(binPath)
					if err != nil {
						t.Errorf("Failed to stat binary %s: %v", binPath, err)
						return
					}

					if info.Mode()&0111 == 0 {
						t.Errorf("Binary %s is not executable", binPath)
						return
					}
				}

				// Try to get version info (if the binary supports --version)
				if runtime.GOOS == config.platform || (config.platform == "linux" && isWSL()) {
					cmd := exec.Command(binPath, "--version")
					output, err := cmd.CombinedOutput()
					if err != nil {
						// Not all binaries may support --version, so this is not a hard failure
						t.Logf("Binary %s doesn't support --version: %v", binPath, err)
					} else {
						t.Logf("✅ Binary %s version info: %s", binPath, strings.TrimSpace(string(output)))
					}
				}

				t.Logf("✅ Binary validation passed for %s", binPath)
			})
		}
	}
}

func testPlatformBuild(t *testing.T, config CrossPlatformBuildTest) []BuildResult {
	var results []BuildResult

	// Create bin directory
	if err := os.MkdirAll(config.BinDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory %s: %v", config.BinDir, err)
	}

	for _, service := range config.Services {
		result := BuildResult{
			Service:  service,
			Platform: fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
		}

		// Determine binary extension
		ext := ""
		if config.GOOS == "windows" {
			ext = ".exe"
		}

		// Build binary path
		binPath := filepath.Join(config.BinDir, service+ext)
		result.BinPath = binPath

		// Execute build - each cmd is a separate module
		cmd := exec.Command("go", "build", "-o", fmt.Sprintf("../../%s", binPath))
		cmd.Dir = fmt.Sprintf("cmd/%s", service)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GOOS=%s", config.GOOS),
			fmt.Sprintf("GOARCH=%s", config.GOARCH),
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			result.Error = fmt.Errorf("build failed: %v\nOutput: %s", err, string(output))
		} else {
			result.Success = true
		}

		results = append(results, result)
	}

	return results
}

// isWSL detects if running in Windows Subsystem for Linux
func isWSL() bool {
	// Check for WSL-specific environment variables or files
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}

	// Check /proc/version for WSL signature
	if data, err := os.ReadFile("/proc/version"); err == nil {
		version := string(data)
		return strings.Contains(strings.ToLower(version), "microsoft") ||
			strings.Contains(strings.ToLower(version), "wsl")
	}

	return false
}

// This file contains cross-platform build validation tests.
// Run with: go test scripts/test-cross-platform-build.go
