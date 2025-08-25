package health

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-redis/redis/v8"
)

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Status     string `json:"status"` // available, unavailable, unknown
	Connection string `json:"connection,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ServiceChecker provides health check functionality for various services
type ServiceChecker struct {
	timeout time.Duration
}

// NewServiceChecker creates a new service checker with the specified timeout
func NewServiceChecker(timeout time.Duration) *ServiceChecker {
	return &ServiceChecker{
		timeout: timeout,
	}
}

// CheckRedis checks Redis connectivity
func (sc *ServiceChecker) CheckRedis(ctx context.Context, addr string) ServiceStatus {
	// Skip on Windows if service is unavailable (conditional test skipping)
	if runtime.GOOS == "windows" {
		// First try a quick connection test
		quickCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		client := redis.NewClient(&redis.Options{
			Addr:         addr,
			DialTimeout:  2 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		})
		defer client.Close()

		_, err := client.Ping(quickCtx).Result()
		if err != nil {
			return ServiceStatus{
				Status:     "unavailable",
				Connection: fmt.Sprintf("redis://%s", addr),
				Message:    "Redis service unavailable on Windows. Start with: docker-compose up redis",
			}
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, sc.timeout)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  sc.timeout,
		ReadTimeout:  sc.timeout,
		WriteTimeout: sc.timeout,
	})
	defer client.Close()

	_, err := client.Ping(timeoutCtx).Result()
	if err != nil {
		return ServiceStatus{
			Status:     "unavailable",
			Connection: fmt.Sprintf("redis://%s", addr),
			Message:    fmt.Sprintf("Failed to connect to Redis: %v", err),
		}
	}

	return ServiceStatus{
		Status:     "available",
		Connection: fmt.Sprintf("redis://%s", addr),
	}
}

// CheckQdrant checks Qdrant vector database connectivity
func (sc *ServiceChecker) CheckQdrant(ctx context.Context, addr string) ServiceStatus {
	// Skip on Windows if service is unavailable (conditional test skipping)
	if runtime.GOOS == "windows" {
		// First try a quick health check
		quickCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		client := &http.Client{Timeout: 2 * time.Second}
		req, err := http.NewRequestWithContext(quickCtx, "GET", fmt.Sprintf("http://%s/", addr), nil)
		if err != nil {
			return ServiceStatus{
				Status:     "unavailable",
				Connection: fmt.Sprintf("http://%s", addr),
				Message:    "Qdrant service unavailable on Windows. Start with: docker-compose up qdrant",
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return ServiceStatus{
				Status:     "unavailable",
				Connection: fmt.Sprintf("http://%s", addr),
				Message:    "Qdrant service unavailable on Windows. Start with: docker-compose up qdrant",
			}
		}
		resp.Body.Close()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, sc.timeout)
	defer cancel()

	client := &http.Client{Timeout: sc.timeout}
	req, err := http.NewRequestWithContext(timeoutCtx, "GET", fmt.Sprintf("http://%s/", addr), nil)
	if err != nil {
		return ServiceStatus{
			Status:     "unavailable",
			Connection: fmt.Sprintf("http://%s", addr),
			Message:    fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return ServiceStatus{
			Status:     "unavailable",
			Connection: fmt.Sprintf("http://%s", addr),
			Message:    fmt.Sprintf("Failed to connect to Qdrant: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ServiceStatus{
			Status:     "unavailable",
			Connection: fmt.Sprintf("http://%s", addr),
			Message:    fmt.Sprintf("Qdrant health check failed with status: %d", resp.StatusCode),
		}
	}

	return ServiceStatus{
		Status:     "available",
		Connection: fmt.Sprintf("http://%s", addr),
	}
}

// CheckPostgreSQL checks PostgreSQL connectivity using existing logic
func (sc *ServiceChecker) CheckPostgreSQL(ctx context.Context, connStr string) ServiceStatus {
	// This will be implemented using the existing logic from main.go
	// For now, return a placeholder
	return ServiceStatus{
		Status:     "unknown",
		Connection: connStr,
		Message:    "PostgreSQL check not yet implemented in health package",
	}
}

// CheckNATS checks NATS connectivity using existing logic
func (sc *ServiceChecker) CheckNATS(ctx context.Context, addr string) ServiceStatus {
	// This will be implemented using the existing logic from main.go
	// For now, return a placeholder
	return ServiceStatus{
		Status:     "unknown",
		Connection: addr,
		Message:    "NATS check not yet implemented in health package",
	}
}
