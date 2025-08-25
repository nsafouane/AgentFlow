package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceChecker_CheckRedis(t *testing.T) {
	checker := NewServiceChecker(2 * time.Second)
	ctx := context.Background()

	t.Run("unavailable service", func(t *testing.T) {
		// Use a non-existent address
		status := checker.CheckRedis(ctx, "localhost:9999")

		assert.Equal(t, "unavailable", status.Status)
		assert.Equal(t, "redis://localhost:9999", status.Connection)
		// On Windows, we get a different message due to conditional skipping
		if runtime.GOOS == "windows" {
			assert.Contains(t, status.Message, "Redis service unavailable on Windows")
		} else {
			assert.Contains(t, status.Message, "Failed to connect to Redis")
		}
	})

	t.Run("timeout handling", func(t *testing.T) {
		shortChecker := NewServiceChecker(1 * time.Millisecond)

		status := shortChecker.CheckRedis(ctx, "localhost:9999")

		assert.Equal(t, "unavailable", status.Status)
		// On Windows, we get a different message due to conditional skipping
		if runtime.GOOS == "windows" {
			assert.Contains(t, status.Message, "Redis service unavailable on Windows")
		} else {
			assert.Contains(t, status.Message, "Failed to connect to Redis")
		}
	})
}

func TestServiceChecker_CheckQdrant(t *testing.T) {
	checker := NewServiceChecker(2 * time.Second)
	ctx := context.Background()

	t.Run("healthy service", func(t *testing.T) {
		// Create a mock Qdrant health endpoint
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"title":"qdrant - vector search engine","version":"1.7.4"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Extract host:port from server URL
		addr := server.URL[7:] // Remove "http://" prefix

		status := checker.CheckQdrant(ctx, addr)

		assert.Equal(t, "available", status.Status)
		assert.Equal(t, "http://"+addr, status.Connection)
		assert.Empty(t, status.Message)
	})

	t.Run("unhealthy service", func(t *testing.T) {
		// Create a mock server that returns 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		addr := server.URL[7:] // Remove "http://" prefix

		status := checker.CheckQdrant(ctx, addr)

		assert.Equal(t, "unavailable", status.Status)
		assert.Equal(t, "http://"+addr, status.Connection)
		assert.Contains(t, status.Message, "health check failed with status: 500")
	})

	t.Run("unavailable service", func(t *testing.T) {
		// Use a non-existent address
		status := checker.CheckQdrant(ctx, "localhost:9999")

		assert.Equal(t, "unavailable", status.Status)
		assert.Equal(t, "http://localhost:9999", status.Connection)
		// On Windows, we get a different message due to conditional skipping
		if runtime.GOOS == "windows" {
			assert.Contains(t, status.Message, "Qdrant service unavailable on Windows")
		} else {
			assert.Contains(t, status.Message, "Failed to connect to Qdrant")
		}
	})

	t.Run("timeout handling", func(t *testing.T) {
		shortChecker := NewServiceChecker(1 * time.Millisecond)

		status := shortChecker.CheckQdrant(ctx, "localhost:9999")

		assert.Equal(t, "unavailable", status.Status)
		// On Windows, we get a different message due to conditional skipping
		if runtime.GOOS == "windows" {
			assert.Contains(t, status.Message, "Qdrant service unavailable on Windows")
		} else {
			assert.Contains(t, status.Message, "Failed to connect to Qdrant")
		}
	})
}

func TestNewServiceChecker(t *testing.T) {
	timeout := 5 * time.Second
	checker := NewServiceChecker(timeout)

	require.NotNil(t, checker)
	assert.Equal(t, timeout, checker.timeout)
}

// Integration test that can be run manually with real Redis
func TestServiceChecker_CheckRedis_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := NewServiceChecker(5 * time.Second)
	ctx := context.Background()

	// This test will pass if Redis is running on localhost:6379
	// and fail gracefully if it's not
	status := checker.CheckRedis(ctx, "localhost:6379")

	// We don't assert the status since Redis may or may not be running
	// Just verify the structure is correct
	assert.NotEmpty(t, status.Status)
	assert.Equal(t, "redis://localhost:6379", status.Connection)

	if status.Status == "available" {
		t.Log("Redis is available for integration testing")
	} else {
		t.Log("Redis is not available - this is expected if not running locally")
	}
}

// Integration test that can be run manually with real Qdrant
func TestServiceChecker_CheckQdrant_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := NewServiceChecker(5 * time.Second)
	ctx := context.Background()

	// This test will pass if Qdrant is running on localhost:6333
	// and fail gracefully if it's not
	status := checker.CheckQdrant(ctx, "localhost:6333")

	// We don't assert the status since Qdrant may or may not be running
	// Just verify the structure is correct
	assert.NotEmpty(t, status.Status)
	assert.Equal(t, "http://localhost:6333", status.Connection)

	if status.Status == "available" {
		t.Log("Qdrant is available for integration testing")
	} else {
		t.Log("Qdrant is not available - this is expected if not running locally")
	}
}
