package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()

	server, err := New(config, logger)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, config, server.config)
	assert.Equal(t, logger, server.logger)
	assert.NotNil(t, server.router)
	assert.NotNil(t, server.middleware)
	assert.NotNil(t, server.httpServer)
}

func TestNewWithNilConfig(t *testing.T) {
	logger := logging.NewLogger()

	server, err := New(nil, logger)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.config)
	// Should use default config
	assert.Equal(t, 8080, server.config.Port)
}

func TestServerRoutes(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()
	config.EnableTracing = false // Disable tracing for simpler testing

	server, err := New(config, logger)
	require.NoError(t, err)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Root endpoint",
			method:         "GET",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "AgentFlow Control Plane API",
		},
		{
			name:           "API root endpoint",
			method:         "GET",
			path:           "/api",
			expectedStatus: http.StatusOK,
			expectedBody:   "v1",
		},
		{
			name:           "Health endpoint",
			method:         "GET",
			path:           "/api/v1/health",
			expectedStatus: http.StatusOK,
			expectedBody:   "healthy",
		},
		{
			name:           "Workflows endpoint (not implemented)",
			method:         "GET",
			path:           "/api/v1/workflows",
			expectedStatus: http.StatusNotImplemented,
			expectedBody:   "NOT_IMPLEMENTED",
		},
		{
			name:           "Individual workflow endpoint (not implemented)",
			method:         "GET",
			path:           "/api/v1/workflows/123",
			expectedStatus: http.StatusNotImplemented,
			expectedBody:   "NOT_IMPLEMENTED",
		},
		{
			name:           "Agents endpoint (not implemented)",
			method:         "GET",
			path:           "/api/v1/agents",
			expectedStatus: http.StatusNotImplemented,
			expectedBody:   "NOT_IMPLEMENTED",
		},
		{
			name:           "Tools endpoint (not implemented)",
			method:         "GET",
			path:           "/api/v1/tools",
			expectedStatus: http.StatusNotImplemented,
			expectedBody:   "NOT_IMPLEMENTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			server.httpServer.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			// Verify CORS headers are present
			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

			// Verify correlation ID is present
			assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()
	config.EnableTracing = false

	server, err := New(config, logger)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, "agentflow-control-plane", data["service"])
	assert.Equal(t, "1.0.0", data["version"])
	assert.NotEmpty(t, data["timestamp"])
}

func TestCORSPreflightRequest(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()
	config.EnableTracing = false

	server, err := New(config, logger)
	require.NoError(t, err)

	req := httptest.NewRequest("OPTIONS", "/api/v1/workflows", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

	w := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestServerShutdown(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()
	config.Port = 0 // Use random available port
	config.ShutdownTimeout = 1 * time.Second
	config.EnableTracing = false

	server, err := New(config, logger)
	require.NoError(t, err)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for server to shutdown
	select {
	case err := <-serverErr:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server shutdown timed out")
	}
}

func TestServerStartError(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()
	config.EnableTLS = true
	config.TLSCertPath = "" // Missing cert path should cause error
	config.TLSKeyPath = ""  // Missing key path should cause error
	config.EnableTracing = false

	server, err := New(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = server.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TLS enabled but cert or key path not provided")
}

func TestMiddlewareOrdering(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()
	config.EnableTracing = false

	server, err := New(config, logger)
	require.NoError(t, err)

	// Create a test handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Replace one of the routes with panic handler to test recovery
	server.router.HandleFunc("/panic", panicHandler).Methods("GET")

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Should not panic due to recovery middleware
	require.NotPanics(t, func() {
		server.httpServer.Handler.ServeHTTP(w, req)
	})

	// Should return 500 status
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Should have CORS headers (applied after recovery)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Should have correlation ID (applied after recovery)
	assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
}

func TestWriteJSONResponse(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()

	server, err := New(config, logger)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}

	server.writeJSONResponse(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "success")
	assert.Contains(t, w.Body.String(), "test")
	assert.Contains(t, w.Body.String(), "value")
}

func TestWriteNotImplemented(t *testing.T) {
	logger := logging.NewLogger()
	config := DefaultConfig()

	server, err := New(config, logger)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	message := "Test endpoint not implemented"

	server.writeNotImplemented(w, message)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "NOT_IMPLEMENTED")
	assert.Contains(t, w.Body.String(), message)
	assert.Contains(t, w.Body.String(), `"success":false`)
}

func TestServerConfiguration(t *testing.T) {
	logger := logging.NewLogger()
	config := &Config{
		Port:            9999,
		ReadTimeout:     45 * time.Second,
		WriteTimeout:    60 * time.Second,
		IdleTimeout:     180 * time.Second,
		MaxHeaderBytes:  2097152,
		ShutdownTimeout: 45 * time.Second,
		EnableTracing:   false,
	}

	server, err := New(config, logger)
	require.NoError(t, err)

	assert.Equal(t, ":9999", server.httpServer.Addr)
	assert.Equal(t, 45*time.Second, server.httpServer.ReadTimeout)
	assert.Equal(t, 60*time.Second, server.httpServer.WriteTimeout)
	assert.Equal(t, 180*time.Second, server.httpServer.IdleTimeout)
	assert.Equal(t, 2097152, server.httpServer.MaxHeaderBytes)
}
