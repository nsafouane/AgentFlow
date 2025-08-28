// Package server provides HTTP server implementation for AgentFlow Control Plane
package server

import (
	"os"
	"strconv"
	"time"
)

// Config holds HTTP server configuration
type Config struct {
	Port            int           `env:"AF_API_PORT"`
	ReadTimeout     time.Duration `env:"AF_API_READ_TIMEOUT"`
	WriteTimeout    time.Duration `env:"AF_API_WRITE_TIMEOUT"`
	IdleTimeout     time.Duration `env:"AF_API_IDLE_TIMEOUT"`
	MaxHeaderBytes  int           `env:"AF_API_MAX_HEADER_BYTES"`
	EnableTLS       bool          `env:"AF_API_TLS_ENABLED"`
	TLSCertPath     string        `env:"AF_API_TLS_CERT_PATH"`
	TLSKeyPath      string        `env:"AF_API_TLS_KEY_PATH"`
	ShutdownTimeout time.Duration `env:"AF_API_SHUTDOWN_TIMEOUT"`
	EnableTracing   bool          `env:"AF_TRACING_ENABLED"`
	TracingEndpoint string        `env:"AF_OTEL_EXPORTER_OTLP_ENDPOINT"`
	ServiceName     string        `env:"AF_SERVICE_NAME"`
}

// DefaultConfig returns default server configuration
func DefaultConfig() *Config {
	return &Config{
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		MaxHeaderBytes:  1048576, // 1MB
		EnableTLS:       false,
		TLSCertPath:     "",
		TLSKeyPath:      "",
		ShutdownTimeout: 30 * time.Second,
		EnableTracing:   true,
		TracingEndpoint: "http://localhost:4318",
		ServiceName:     "agentflow-control-plane",
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := DefaultConfig()

	if val := os.Getenv("AF_API_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Port = port
		}
	}

	if val := os.Getenv("AF_API_READ_TIMEOUT"); val != "" {
		if timeout, err := time.ParseDuration(val); err == nil {
			config.ReadTimeout = timeout
		}
	}

	if val := os.Getenv("AF_API_WRITE_TIMEOUT"); val != "" {
		if timeout, err := time.ParseDuration(val); err == nil {
			config.WriteTimeout = timeout
		}
	}

	if val := os.Getenv("AF_API_IDLE_TIMEOUT"); val != "" {
		if timeout, err := time.ParseDuration(val); err == nil {
			config.IdleTimeout = timeout
		}
	}

	if val := os.Getenv("AF_API_MAX_HEADER_BYTES"); val != "" {
		if size, err := strconv.Atoi(val); err == nil {
			config.MaxHeaderBytes = size
		}
	}

	if val := os.Getenv("AF_API_TLS_ENABLED"); val != "" {
		config.EnableTLS = val == "true" || val == "1"
	}

	if val := os.Getenv("AF_API_TLS_CERT_PATH"); val != "" {
		config.TLSCertPath = val
	}

	if val := os.Getenv("AF_API_TLS_KEY_PATH"); val != "" {
		config.TLSKeyPath = val
	}

	if val := os.Getenv("AF_API_SHUTDOWN_TIMEOUT"); val != "" {
		if timeout, err := time.ParseDuration(val); err == nil {
			config.ShutdownTimeout = timeout
		}
	}

	if val := os.Getenv("AF_TRACING_ENABLED"); val != "" {
		config.EnableTracing = val == "true" || val == "1"
	}

	if val := os.Getenv("AF_OTEL_EXPORTER_OTLP_ENDPOINT"); val != "" {
		config.TracingEndpoint = val
	}

	if val := os.Getenv("AF_SERVICE_NAME"); val != "" {
		config.ServiceName = val
	}

	return config
}
