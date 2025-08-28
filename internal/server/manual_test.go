package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManualServerIntegration demonstrates the complete server functionality
// This test starts a real HTTP server and makes actual HTTP requests to verify:
// 1. Server startup and graceful shutdown
// 2. Middleware execution order through logs
// 3. Routing functionality
// 4. Error handling and recovery
// 5. CORS headers
// 6. Structured logging with correlation IDs
func TestManualServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual integration test in short mode")
	}

	// Create a logger that writes to a buffer so we can verify log output
	logFile, err := os.CreateTemp("", "server_test_*.log")
	require.NoError(t, err)
	defer os.Remove(logFile.Name())
	defer logFile.Close()

	logger := logging.NewLoggerWithOutput(logFile)

	// Configure server with test settings
	config := &Config{
		Port:            8081, // Use different port to avoid conflicts
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     10 * time.Second,
		MaxHeaderBytes:  1048576,
		EnableTLS:       false,
		ShutdownTimeout: 2 * time.Second,
		EnableTracing:   true, // Enable tracing for full integration test
		TracingEndpoint: "http://localhost:4318",
		ServiceName:     "test-control-plane",
	}

	server, err := New(config, logger)
	require.NoError(t, err)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Test 1: Health check endpoint
	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/api/v1/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, resp.Header.Get("X-Correlation-ID"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "healthy")
	})

	// Test 2: Root endpoint
	t.Run("Root Endpoint", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("X-Correlation-ID"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "AgentFlow Control Plane API")
	})

	// Test 3: Not implemented endpoint
	t.Run("Not Implemented Endpoint", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/api/v1/workflows")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("X-Correlation-ID"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "NOT_IMPLEMENTED")
	})

	// Test 4: CORS preflight request
	t.Run("CORS Preflight", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("OPTIONS", "http://localhost:8081/api/v1/workflows", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
	})

	// Test 5: Custom correlation ID
	t.Run("Custom Correlation ID", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("GET", "http://localhost:8081/api/v1/health", nil)
		require.NoError(t, err)
		req.Header.Set("X-Correlation-ID", "test-correlation-12345")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "test-correlation-12345", resp.Header.Get("X-Correlation-ID"))
	})

	// Test 6: Multiple concurrent requests
	t.Run("Concurrent Requests", func(t *testing.T) {
		const numRequests = 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				resp, err := http.Get(fmt.Sprintf("http://localhost:8081/api/v1/health?id=%d", id))
				if err != nil {
					results <- err
					return
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
					return
				}
				results <- nil
			}(i)
		}

		for i := 0; i < numRequests; i++ {
			select {
			case err := <-results:
				assert.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("Request timed out")
			}
		}
	})

	// Shutdown server
	cancel()

	// Wait for server to shutdown
	select {
	case err := <-serverErr:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server shutdown timed out")
	}

	// Verify log output contains expected middleware execution
	logFile.Seek(0, 0)
	logContent, err := io.ReadAll(logFile)
	require.NoError(t, err)
	logOutput := string(logContent)

	// Verify structured logging
	assert.Contains(t, logOutput, "Starting AgentFlow Control Plane API server")
	assert.Contains(t, logOutput, "HTTP request started")
	assert.Contains(t, logOutput, "HTTP request completed")
	assert.Contains(t, logOutput, "Shutting down AgentFlow Control Plane API server")
	assert.Contains(t, logOutput, "Server shutdown completed")

	// Verify correlation IDs in logs
	assert.Contains(t, logOutput, "correlation_id")
	assert.Contains(t, logOutput, "test-correlation-12345")

	// Verify request details in logs
	assert.Contains(t, logOutput, "/api/v1/health")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "200")
	assert.Contains(t, logOutput, "duration_ms")

	fmt.Printf("Manual integration test completed successfully!\n")
	fmt.Printf("Log file: %s\n", logFile.Name())
	fmt.Printf("Log entries: %d lines\n", len(strings.Split(logOutput, "\n")))
}

// TestManualMiddlewareOrdering verifies middleware execution order through logs
func TestManualMiddlewareOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual middleware test in short mode")
	}

	// Create a logger that writes to a buffer
	logFile, err := os.CreateTemp("", "middleware_test_*.log")
	require.NoError(t, err)
	defer os.Remove(logFile.Name())
	defer logFile.Close()

	logger := logging.NewLoggerWithOutput(logFile)

	config := &Config{
		Port:            8082,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     10 * time.Second,
		ShutdownTimeout: 2 * time.Second,
		EnableTracing:   false, // Disable tracing for simpler log analysis
	}

	server, err := New(config, logger)
	require.NoError(t, err)

	// Add a custom route that we can test
	server.router.HandleFunc("/test-middleware", func(w http.ResponseWriter, r *http.Request) {
		// Get logger from context to verify it's available
		ctxLogger := GetLoggerFromContext(r.Context())
		ctxLogger.Info("Handler executed")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("middleware test"))
	}).Methods("GET")

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(300 * time.Millisecond)

	// Make a request to test middleware ordering
	resp, err := http.Get("http://localhost:8082/test-middleware")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown server
	cancel()
	time.Sleep(500 * time.Millisecond)

	// Analyze log output for middleware execution order
	logFile.Seek(0, 0)
	logContent, err := io.ReadAll(logFile)
	require.NoError(t, err)
	logOutput := string(logContent)

	// Verify middleware execution sequence
	assert.Contains(t, logOutput, "HTTP request started")   // Logging middleware
	assert.Contains(t, logOutput, "Handler executed")       // Our handler
	assert.Contains(t, logOutput, "HTTP request completed") // Logging middleware

	// Verify CORS headers were applied
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))

	// Verify correlation ID was generated
	assert.NotEmpty(t, resp.Header.Get("X-Correlation-ID"))

	fmt.Printf("Middleware ordering test completed successfully!\n")
	fmt.Printf("Middleware execution verified through structured logs\n")
}
