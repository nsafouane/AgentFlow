package main

import (
	"os"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/agentflow/agentflow/internal/server"
)

// Progress: IN PROGRESS - Task 1: HTTP Server & Routing + Middleware Stack
// Implementation: ✓ HTTP server with /api/v1 routing and middleware stack
// Features: ✓ Recovery, logging, tracing, CORS middleware
// Configuration: ✓ Configurable timeouts, TLS support, graceful shutdown
// Integration: ✓ OpenTelemetry tracing from Q1.2, structured logging
func main() {
	// Initialize structured logger
	logger := logging.NewLogger()

	// Load server configuration from environment
	config := server.LoadFromEnv()

	// Create and configure server
	srv, err := server.New(config, logger)
	if err != nil {
		logger.Error("Failed to create server", err)
		os.Exit(1)
	}

	// Start server with graceful shutdown
	logger.Info("Starting AgentFlow Control Plane API server")
	if err := srv.StartWithGracefulShutdown(); err != nil {
		logger.Error("Server error", err)
		os.Exit(1)
	}

	logger.Info("AgentFlow Control Plane API server stopped")
}
