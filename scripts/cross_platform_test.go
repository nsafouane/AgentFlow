package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCrossPlatformBinariesExist(t *testing.T) {
	// Test that cross-platform binaries exist after build
	testCases := []struct {
		platform string
		binDir   string
		services []string
		ext      string
	}{
		{"linux", "bin/linux", []string{"control-plane", "worker", "af"}, ""},
		{"windows", "bin/windows", []string{"control-plane", "worker", "af"}, ".exe"},
		{"darwin", "bin/darwin", []string{"control-plane", "worker", "af"}, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.platform, func(t *testing.T) {
			for _, service := range tc.services {
				binPath := filepath.Join("..", tc.binDir, service+tc.ext)

				if _, err := os.Stat(binPath); os.IsNotExist(err) {
					t.Errorf("Binary %s does not exist for platform %s", binPath, tc.platform)
				} else {
					t.Logf("✅ Binary %s exists for platform %s", binPath, tc.platform)
				}
			}
		})
	}
}

func TestBuildDirectoriesExist(t *testing.T) {
	// Test that build directories exist
	buildDirs := []string{"../bin/linux", "../bin/windows", "../bin/darwin"}

	for _, dir := range buildDirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				t.Errorf("Build directory %s does not exist", dir)
			} else {
				t.Logf("✅ Build directory %s exists", dir)
			}
		})
	}
}
