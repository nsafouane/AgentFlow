package server

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.WriteTimeout)
	assert.Equal(t, 120*time.Second, config.IdleTimeout)
	assert.Equal(t, 1048576, config.MaxHeaderBytes)
	assert.False(t, config.EnableTLS)
	assert.Equal(t, "", config.TLSCertPath)
	assert.Equal(t, "", config.TLSKeyPath)
	assert.Equal(t, 30*time.Second, config.ShutdownTimeout)
	assert.True(t, config.EnableTracing)
	assert.Equal(t, "http://localhost:4318", config.TracingEndpoint)
	assert.Equal(t, "agentflow-control-plane", config.ServiceName)
}

func TestLoadFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"AF_API_PORT",
		"AF_API_READ_TIMEOUT",
		"AF_API_WRITE_TIMEOUT",
		"AF_API_IDLE_TIMEOUT",
		"AF_API_MAX_HEADER_BYTES",
		"AF_API_TLS_ENABLED",
		"AF_API_TLS_CERT_PATH",
		"AF_API_TLS_KEY_PATH",
		"AF_API_SHUTDOWN_TIMEOUT",
		"AF_TRACING_ENABLED",
		"AF_OTEL_EXPORTER_OTLP_ENDPOINT",
		"AF_SERVICE_NAME",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
	}

	// Clean up after test
	defer func() {
		for _, envVar := range envVars {
			if originalEnv[envVar] == "" {
				os.Unsetenv(envVar)
			} else {
				os.Setenv(envVar, originalEnv[envVar])
			}
		}
	}()

	// Set test environment variables
	os.Setenv("AF_API_PORT", "9090")
	os.Setenv("AF_API_READ_TIMEOUT", "45s")
	os.Setenv("AF_API_WRITE_TIMEOUT", "60s")
	os.Setenv("AF_API_IDLE_TIMEOUT", "180s")
	os.Setenv("AF_API_MAX_HEADER_BYTES", "2097152")
	os.Setenv("AF_API_TLS_ENABLED", "true")
	os.Setenv("AF_API_TLS_CERT_PATH", "/path/to/cert.pem")
	os.Setenv("AF_API_TLS_KEY_PATH", "/path/to/key.pem")
	os.Setenv("AF_API_SHUTDOWN_TIMEOUT", "45s")
	os.Setenv("AF_TRACING_ENABLED", "false")
	os.Setenv("AF_OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4318")
	os.Setenv("AF_SERVICE_NAME", "test-service")

	config := LoadFromEnv()

	assert.Equal(t, 9090, config.Port)
	assert.Equal(t, 45*time.Second, config.ReadTimeout)
	assert.Equal(t, 60*time.Second, config.WriteTimeout)
	assert.Equal(t, 180*time.Second, config.IdleTimeout)
	assert.Equal(t, 2097152, config.MaxHeaderBytes)
	assert.True(t, config.EnableTLS)
	assert.Equal(t, "/path/to/cert.pem", config.TLSCertPath)
	assert.Equal(t, "/path/to/key.pem", config.TLSKeyPath)
	assert.Equal(t, 45*time.Second, config.ShutdownTimeout)
	assert.False(t, config.EnableTracing)
	assert.Equal(t, "http://jaeger:4318", config.TracingEndpoint)
	assert.Equal(t, "test-service", config.ServiceName)
}

func TestLoadFromEnvWithInvalidValues(t *testing.T) {
	// Save original environment
	originalPort := os.Getenv("AF_API_PORT")
	originalTimeout := os.Getenv("AF_API_READ_TIMEOUT")

	defer func() {
		if originalPort == "" {
			os.Unsetenv("AF_API_PORT")
		} else {
			os.Setenv("AF_API_PORT", originalPort)
		}
		if originalTimeout == "" {
			os.Unsetenv("AF_API_READ_TIMEOUT")
		} else {
			os.Setenv("AF_API_READ_TIMEOUT", originalTimeout)
		}
	}()

	// Set invalid values
	os.Setenv("AF_API_PORT", "invalid")
	os.Setenv("AF_API_READ_TIMEOUT", "invalid")

	config := LoadFromEnv()

	// Should fall back to defaults for invalid values
	assert.Equal(t, 8080, config.Port)                  // default value
	assert.Equal(t, 30*time.Second, config.ReadTimeout) // default value
}

func TestLoadFromEnvBooleanValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true string", "true", true},
		{"1 string", "1", true},
		{"false string", "false", false},
		{"0 string", "0", false},
		{"empty string", "", false},
		{"other string", "yes", false},
	}

	originalTLS := os.Getenv("AF_API_TLS_ENABLED")
	defer func() {
		if originalTLS == "" {
			os.Unsetenv("AF_API_TLS_ENABLED")
		} else {
			os.Setenv("AF_API_TLS_ENABLED", originalTLS)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AF_API_TLS_ENABLED", tt.value)
			config := LoadFromEnv()
			assert.Equal(t, tt.expected, config.EnableTLS)
		})
	}
}
