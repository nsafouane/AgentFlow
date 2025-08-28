package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestMiddlewareStack(t *testing.T) {
	logger := logging.NewLogger()
	stack := NewMiddlewareStack(logger)

	// Test middleware ordering
	var executionOrder []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware1_start")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware1_end")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware2_start")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware2_end")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	stack.Use(middleware1)
	stack.Use(middleware2)

	finalHandler := stack.Apply(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	finalHandler.ServeHTTP(w, req)

	// Middleware should execute in FIFO order (first added, first executed)
	expected := []string{
		"middleware1_start",
		"middleware2_start",
		"handler",
		"middleware2_end",
		"middleware1_end",
	}

	assert.Equal(t, expected, executionOrder)
}

func TestRecoveryMiddleware(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logging.NewLoggerWithWriter(&logBuffer)
	stack := NewMiddlewareStack(logger)

	// Handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	recoveryHandler := stack.RecoveryMiddleware()(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	require.NotPanics(t, func() {
		recoveryHandler.ServeHTTP(w, req)
	})

	// Should return 500 status
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Should return JSON error response
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), "INTERNAL_SERVER_ERROR")

	// Should log the panic
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "test panic")
	assert.Contains(t, logOutput, "stack_trace")
}

func TestLoggingMiddleware(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logging.NewLoggerWithWriter(&logBuffer)
	stack := NewMiddlewareStack(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	loggingHandler := stack.LoggingMiddleware()(handler)

	req := httptest.NewRequest("GET", "/test/path", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	loggingHandler.ServeHTTP(w, req)

	// Should add correlation ID header
	assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))

	// Should log request start and completion
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request started")
	assert.Contains(t, logOutput, "HTTP request completed")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test/path")
	assert.Contains(t, logOutput, "test-agent")
	assert.Contains(t, logOutput, "200")
	assert.Contains(t, logOutput, "duration_ms")
}

func TestLoggingMiddlewareWithCorrelationID(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logging.NewLoggerWithWriter(&logBuffer)
	stack := NewMiddlewareStack(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	loggingHandler := stack.LoggingMiddleware()(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "test-correlation-123")
	w := httptest.NewRecorder()

	loggingHandler.ServeHTTP(w, req)

	// Should preserve existing correlation ID
	assert.Equal(t, "test-correlation-123", w.Header().Get("X-Correlation-ID"))

	// Should log with correlation ID
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "test-correlation-123")
}

func TestTracingMiddleware(t *testing.T) {
	logger := logging.NewLogger()
	stack := NewMiddlewareStack(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify trace context is available
		span := trace.SpanFromContext(r.Context())
		assert.True(t, span.SpanContext().IsValid())
		w.WriteHeader(http.StatusOK)
	})

	tracingHandler := stack.TracingMiddleware()(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	tracingHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSMiddleware(t *testing.T) {
	logger := logging.NewLogger()
	stack := NewMiddlewareStack(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := stack.CORSMiddleware()(handler)

	t.Run("Regular request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Correlation-ID")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("OPTIONS preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)

	// Test Write
	data := []byte("test data")
	n, err := rw.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), rw.size)

	// Test multiple writes
	moreData := []byte(" more data")
	n2, err := rw.Write(moreData)
	assert.NoError(t, err)
	assert.Equal(t, len(moreData), n2)
	assert.Equal(t, len(data)+len(moreData), rw.size)
}

func TestGenerateCorrelationID(t *testing.T) {
	id1 := generateCorrelationID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := generateCorrelationID()

	// Should generate different IDs
	assert.NotEqual(t, id1, id2)

	// Should have expected format
	assert.True(t, strings.HasPrefix(id1, "req_"))
	assert.True(t, strings.HasPrefix(id2, "req_"))
}

func TestGetLoggerFromContext(t *testing.T) {
	t.Run("Logger in context", func(t *testing.T) {
		logger := logging.NewLogger()
		ctx := context.WithValue(context.Background(), "logger", logger)

		retrievedLogger := GetLoggerFromContext(ctx)
		assert.Equal(t, logger, retrievedLogger)
	})

	t.Run("No logger in context", func(t *testing.T) {
		ctx := context.Background()

		retrievedLogger := GetLoggerFromContext(ctx)
		assert.NotNil(t, retrievedLogger)
		// Should return default logger
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "logger", "not a logger")

		retrievedLogger := GetLoggerFromContext(ctx)
		assert.NotNil(t, retrievedLogger)
		// Should return default logger
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logging.NewLoggerWithWriter(&logBuffer)
	stack := NewMiddlewareStack(logger)

	// Add all middleware in correct order
	stack.Use(stack.RecoveryMiddleware())
	stack.Use(stack.LoggingMiddleware())
	stack.Use(stack.TracingMiddleware())
	stack.Use(stack.CORSMiddleware())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify logger is available in context
		ctxLogger := GetLoggerFromContext(r.Context())
		assert.NotNil(t, ctxLogger)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	finalHandler := stack.Apply(handler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	finalHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	// Verify CORS headers
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Verify correlation ID
	assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))

	// Verify logging
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request started")
	assert.Contains(t, logOutput, "HTTP request completed")
}
