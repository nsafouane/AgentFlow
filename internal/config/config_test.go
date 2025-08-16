package config

import (
	"testing"
)

// TestLoad tests the configuration loading functionality
func TestLoad(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Errorf("Load() returned error: %v", err)
	}

	if config == nil {
		t.Error("Load() returned nil config")
	}

	t.Log("Configuration loading test completed")
}
