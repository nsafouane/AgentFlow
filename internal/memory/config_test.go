package memory

import (
	"context"
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Error("DefaultConfig() Enabled should be false")
	}

	if config.Implementation != "in_memory" {
		t.Errorf("DefaultConfig() Implementation = %s, expected in_memory", config.Implementation)
	}

	if config.MaxEntries != 10000 {
		t.Errorf("DefaultConfig() MaxEntries = %d, expected 10000", config.MaxEntries)
	}

	if config.Debug {
		t.Error("DefaultConfig() Debug should be false")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"AF_MEMORY_ENABLED":        os.Getenv("AF_MEMORY_ENABLED"),
		"AF_MEMORY_IMPLEMENTATION": os.Getenv("AF_MEMORY_IMPLEMENTATION"),
		"AF_MEMORY_MAX_ENTRIES":    os.Getenv("AF_MEMORY_MAX_ENTRIES"),
		"AF_MEMORY_DEBUG":          os.Getenv("AF_MEMORY_DEBUG"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			expected: Config{
				Enabled:        false,
				Implementation: "in_memory",
				MaxEntries:     10000,
				Debug:          false,
			},
		},
		{
			name: "enabled with custom values",
			envVars: map[string]string{
				"AF_MEMORY_ENABLED":        "true",
				"AF_MEMORY_IMPLEMENTATION": "in_memory",
				"AF_MEMORY_MAX_ENTRIES":    "5000",
				"AF_MEMORY_DEBUG":          "true",
			},
			expected: Config{
				Enabled:        true,
				Implementation: "in_memory",
				MaxEntries:     5000,
				Debug:          true,
			},
		},
		{
			name: "invalid boolean values use defaults",
			envVars: map[string]string{
				"AF_MEMORY_ENABLED":        "invalid",
				"AF_MEMORY_DEBUG":          "not_a_bool",
				"AF_MEMORY_IMPLEMENTATION": "", // Clear previous value
				"AF_MEMORY_MAX_ENTRIES":    "", // Clear previous value
			},
			expected: Config{
				Enabled:        false, // Default when parsing fails
				Implementation: "in_memory",
				MaxEntries:     10000,
				Debug:          false, // Default when parsing fails
			},
		},
		{
			name: "invalid max entries uses default",
			envVars: map[string]string{
				"AF_MEMORY_MAX_ENTRIES": "invalid",
			},
			expected: Config{
				Enabled:        false,
				Implementation: "in_memory",
				MaxEntries:     10000, // Default when parsing fails
				Debug:          false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}

			config := LoadConfigFromEnv()

			if config.Enabled != tt.expected.Enabled {
				t.Errorf("LoadConfigFromEnv() Enabled = %v, expected %v", config.Enabled, tt.expected.Enabled)
			}

			if config.Implementation != tt.expected.Implementation {
				t.Errorf("LoadConfigFromEnv() Implementation = %s, expected %s", config.Implementation, tt.expected.Implementation)
			}

			if config.MaxEntries != tt.expected.MaxEntries {
				t.Errorf("LoadConfigFromEnv() MaxEntries = %d, expected %d", config.MaxEntries, tt.expected.MaxEntries)
			}

			if config.Debug != tt.expected.Debug {
				t.Errorf("LoadConfigFromEnv() Debug = %v, expected %v", config.Debug, tt.expected.Debug)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid disabled config",
			config: Config{
				Enabled:        false,
				Implementation: "invalid", // Should not matter when disabled
				MaxEntries:     -1,        // Should not matter when disabled
			},
			wantErr: false,
		},
		{
			name: "valid enabled config",
			config: Config{
				Enabled:        true,
				Implementation: "in_memory",
				MaxEntries:     1000,
			},
			wantErr: false,
		},
		{
			name: "invalid implementation",
			config: Config{
				Enabled:        true,
				Implementation: "unsupported",
				MaxEntries:     1000,
			},
			wantErr: true,
		},
		{
			name: "negative max entries",
			config: Config{
				Enabled:        true,
				Implementation: "in_memory",
				MaxEntries:     -1,
			},
			wantErr: true,
		},
		{
			name: "zero max entries (unlimited)",
			config: Config{
				Enabled:        true,
				Implementation: "in_memory",
				MaxEntries:     0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewContainer(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid disabled container",
			config: Config{
				Enabled:        false,
				Implementation: "in_memory",
				MaxEntries:     1000,
			},
			wantErr: false,
		},
		{
			name: "valid enabled container",
			config: Config{
				Enabled:        true,
				Implementation: "in_memory",
				MaxEntries:     1000,
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: Config{
				Enabled:        true,
				Implementation: "unsupported",
				MaxEntries:     1000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, err := NewContainer(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if container == nil {
					t.Error("NewContainer() returned nil container")
					return
				}

				if container.IsEnabled() != tt.config.Enabled {
					t.Errorf("Container.IsEnabled() = %v, expected %v", container.IsEnabled(), tt.config.Enabled)
				}

				config := container.GetConfig()
				if config.Enabled != tt.config.Enabled {
					t.Errorf("Container.GetConfig().Enabled = %v, expected %v", config.Enabled, tt.config.Enabled)
				}
			}
		})
	}
}

func TestContainer_GetStore(t *testing.T) {
	// Test disabled container
	disabledConfig := Config{
		Enabled:        false,
		Implementation: "in_memory",
		MaxEntries:     1000,
	}

	disabledContainer, err := NewContainer(disabledConfig)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	store, err := disabledContainer.GetStore()
	if err == nil {
		t.Error("GetStore() on disabled container should return error")
	}
	if store != nil {
		t.Error("GetStore() on disabled container should return nil store")
	}

	// Test enabled container
	enabledConfig := Config{
		Enabled:        true,
		Implementation: "in_memory",
		MaxEntries:     1000,
	}

	enabledContainer, err := NewContainer(enabledConfig)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	store, err = enabledContainer.GetStore()
	if err != nil {
		t.Errorf("GetStore() on enabled container error = %v", err)
	}
	if store == nil {
		t.Error("GetStore() on enabled container should return store")
	}
}

func TestContainer_HealthCheck(t *testing.T) {
	ctx := context.Background()

	// Test disabled container (should pass)
	disabledConfig := Config{
		Enabled:        false,
		Implementation: "in_memory",
		MaxEntries:     1000,
	}

	disabledContainer, err := NewContainer(disabledConfig)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	err = disabledContainer.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() on disabled container error = %v", err)
	}

	// Test enabled container
	enabledConfig := Config{
		Enabled:        true,
		Implementation: "in_memory",
		MaxEntries:     1000,
	}

	enabledContainer, err := NewContainer(enabledConfig)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	err = enabledContainer.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() on enabled container error = %v", err)
	}

	// Verify health check actually performed operations
	store, err := enabledContainer.GetStore()
	if err != nil {
		t.Fatalf("GetStore() error = %v", err)
	}

	// The health check should have left a test entry
	query := QueryRequest{Key: "health_check_test"}
	response, err := store.Query(ctx, query)
	if err != nil {
		t.Errorf("Query() after health check error = %v", err)
	}

	if len(response.Entries) != 1 {
		t.Errorf("Health check test entry count = %d, expected 1", len(response.Entries))
	}
}

func TestContainer_ExperimentalFeatureFlag(t *testing.T) {
	// Test that memory store is disabled by default (experimental feature)
	defaultConfig := DefaultConfig()
	if defaultConfig.Enabled {
		t.Error("Memory store should be disabled by default (experimental feature)")
	}

	// Test that it can be enabled via environment variable
	os.Setenv("AF_MEMORY_ENABLED", "true")
	defer os.Unsetenv("AF_MEMORY_ENABLED")

	envConfig := LoadConfigFromEnv()
	if !envConfig.Enabled {
		t.Error("Memory store should be enabled when AF_MEMORY_ENABLED=true")
	}

	// Test container creation with experimental flag
	container, err := NewContainer(envConfig)
	if err != nil {
		t.Fatalf("NewContainer() with experimental flag error = %v", err)
	}

	if !container.IsEnabled() {
		t.Error("Container should be enabled when experimental flag is set")
	}

	store, err := container.GetStore()
	if err != nil {
		t.Errorf("GetStore() with experimental flag error = %v", err)
	}
	if store == nil {
		t.Error("GetStore() should return store when experimental flag is enabled")
	}
}
