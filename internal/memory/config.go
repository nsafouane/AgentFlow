// Package memory provides configuration and dependency injection for memory store
package memory

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds memory store configuration
type Config struct {
	// Enabled controls whether memory store is available (experimental feature flag)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Implementation specifies the memory store implementation to use
	Implementation string `json:"implementation" yaml:"implementation"`

	// MaxEntries limits the number of entries in memory (0 = unlimited)
	MaxEntries int `json:"max_entries" yaml:"max_entries"`

	// Debug enables debug logging for memory operations
	Debug bool `json:"debug" yaml:"debug"`
}

// DefaultConfig returns the default memory store configuration
func DefaultConfig() Config {
	return Config{
		Enabled:        false, // Experimental feature - disabled by default
		Implementation: "in_memory",
		MaxEntries:     10000, // Reasonable default for development
		Debug:          false,
	}
}

// LoadConfigFromEnv loads memory store configuration from environment variables
func LoadConfigFromEnv() Config {
	config := DefaultConfig()

	// AF_MEMORY_ENABLED - experimental feature flag
	if enabled := os.Getenv("AF_MEMORY_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			config.Enabled = val
		}
	}

	// AF_MEMORY_IMPLEMENTATION - memory store implementation
	if impl := os.Getenv("AF_MEMORY_IMPLEMENTATION"); impl != "" {
		config.Implementation = strings.ToLower(impl)
	}

	// AF_MEMORY_MAX_ENTRIES - maximum number of entries
	if maxEntries := os.Getenv("AF_MEMORY_MAX_ENTRIES"); maxEntries != "" {
		if val, err := strconv.Atoi(maxEntries); err == nil && val >= 0 {
			config.MaxEntries = val
		}
	}

	// AF_MEMORY_DEBUG - debug logging
	if debug := os.Getenv("AF_MEMORY_DEBUG"); debug != "" {
		if val, err := strconv.ParseBool(debug); err == nil {
			config.Debug = val
		}
	}

	return config
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.Enabled {
		switch c.Implementation {
		case "in_memory":
			// Valid implementation
		default:
			return fmt.Errorf("unsupported memory implementation: %s", c.Implementation)
		}

		if c.MaxEntries < 0 {
			return fmt.Errorf("max_entries cannot be negative: %d", c.MaxEntries)
		}
	}

	return nil
}

// Container provides dependency injection for memory store
type Container struct {
	config Config
	store  MemoryStore
}

// NewContainer creates a new dependency injection container
func NewContainer(config Config) (*Container, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid memory config: %w", err)
	}

	container := &Container{
		config: config,
	}

	// Initialize store if enabled
	if config.Enabled {
		store, err := container.createStore()
		if err != nil {
			return nil, fmt.Errorf("failed to create memory store: %w", err)
		}
		container.store = store
	}

	return container, nil
}

// GetStore returns the memory store instance if enabled
func (c *Container) GetStore() (MemoryStore, error) {
	if !c.config.Enabled {
		return nil, fmt.Errorf("memory store is disabled (experimental feature flag AF_MEMORY_ENABLED=false)")
	}

	if c.store == nil {
		return nil, fmt.Errorf("memory store not initialized")
	}

	return c.store, nil
}

// IsEnabled returns whether memory store is enabled
func (c *Container) IsEnabled() bool {
	return c.config.Enabled
}

// GetConfig returns the current configuration
func (c *Container) GetConfig() Config {
	return c.config
}

// createStore creates the appropriate memory store implementation
func (c *Container) createStore() (MemoryStore, error) {
	switch c.config.Implementation {
	case "in_memory":
		return NewInMemoryStore(), nil
	default:
		return nil, fmt.Errorf("unsupported memory implementation: %s", c.config.Implementation)
	}
}

// HealthCheck performs a health check on the memory store
func (c *Container) HealthCheck(ctx context.Context) error {
	if !c.config.Enabled {
		return nil // Not enabled, so no health check needed
	}

	store, err := c.GetStore()
	if err != nil {
		return fmt.Errorf("memory store not available: %w", err)
	}

	// Perform a simple save/query operation to verify functionality
	testKey := "health_check_test"
	testData := map[string]interface{}{
		"timestamp": "health_check",
		"test":      true,
	}

	if err := store.Save(ctx, testKey, testData); err != nil {
		return fmt.Errorf("memory store save failed: %w", err)
	}

	query := QueryRequest{Key: testKey}
	response, err := store.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("memory store query failed: %w", err)
	}

	if len(response.Entries) != 1 {
		return fmt.Errorf("memory store health check failed: expected 1 entry, got %d", len(response.Entries))
	}

	return nil
}
