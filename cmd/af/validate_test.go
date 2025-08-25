package main

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEnvironment_Structure(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run validation (this will exit with code 1 if there are errors, but we'll capture output)
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Ignore panics from os.Exit calls
			}
		}()
		validateEnvironment()
	}()

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON output
	var result ValidationResult
	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "Output should be valid JSON")

	// Verify structure
	assert.NotEmpty(t, result.Version)
	assert.NotEmpty(t, result.Timestamp)
	assert.NotEmpty(t, result.Environment.Platform)
	assert.NotEmpty(t, result.Environment.Architecture)
	assert.NotEmpty(t, result.Environment.Container)

	// Verify services are included
	assert.Contains(t, result.Services, "postgres")
	assert.Contains(t, result.Services, "nats")
	assert.Contains(t, result.Services, "redis")
	assert.Contains(t, result.Services, "qdrant")

	// Verify service structure
	for serviceName, service := range result.Services {
		assert.NotEmpty(t, service.Status, "Service %s should have status", serviceName)
		assert.Contains(t, []string{"available", "unavailable", "unknown"}, service.Status,
			"Service %s should have valid status", serviceName)
	}
}

func TestDetectEnvironment(t *testing.T) {
	env := detectEnvironment()

	assert.Equal(t, runtime.GOOS, env.Platform)
	assert.Equal(t, runtime.GOARCH, env.Architecture)
	assert.NotEmpty(t, env.Container)
	assert.Contains(t, []string{"host", "devcontainer", "codespaces", "docker"}, env.Container)
}

func TestValidateRedisService_WindowsConditionalSkipping(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("This test is specific to Windows conditional skipping")
	}

	result := &ValidationResult{
		Services: make(map[string]ServiceInfo),
		Warnings: []string{},
	}

	validateRedisService(result)

	// Verify Redis service is checked
	assert.Contains(t, result.Services, "redis")
	redisService := result.Services["redis"]

	// On Windows, if Redis is unavailable, should have helpful message
	if redisService.Status == "unavailable" {
		// Should have guidance in warnings
		found := false
		for _, warning := range result.Warnings {
			if strings.Contains(warning, "docker-compose up redis") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should provide guidance for starting Redis on Windows")
	}
}

func TestValidateQdrantService_WindowsConditionalSkipping(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("This test is specific to Windows conditional skipping")
	}

	result := &ValidationResult{
		Services: make(map[string]ServiceInfo),
		Warnings: []string{},
	}

	validateQdrantService(result)

	// Verify Qdrant service is checked
	assert.Contains(t, result.Services, "qdrant")
	qdrantService := result.Services["qdrant"]

	// On Windows, if Qdrant is unavailable, should have helpful message
	if qdrantService.Status == "unavailable" {
		// Should have guidance in warnings
		found := false
		for _, warning := range result.Warnings {
			if strings.Contains(warning, "docker-compose up qdrant") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should provide guidance for starting Qdrant on Windows")
	}
}

func TestServiceValidation_NonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("This test is for non-Windows platforms")
	}

	result := &ValidationResult{
		Services: make(map[string]ServiceInfo),
		Warnings: []string{},
	}

	validateRedisService(result)
	validateQdrantService(result)

	// Verify services are checked
	assert.Contains(t, result.Services, "redis")
	assert.Contains(t, result.Services, "qdrant")

	// Verify connection strings are properly formatted
	redisService := result.Services["redis"]
	assert.Equal(t, "redis://localhost:6379", redisService.Connection)

	qdrantService := result.Services["qdrant"]
	assert.Equal(t, "http://localhost:6333", qdrantService.Connection)
}

// Manual test function for service validation
func TestManualServiceValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual test in short mode")
	}

	t.Log("=== Manual Service Validation Test ===")
	t.Log("This test validates the service checking functionality.")
	t.Log("Start services with: docker-compose up -d")
	t.Log("")

	result := &ValidationResult{
		Services: make(map[string]ServiceInfo),
		Warnings: []string{},
		Errors:   []string{},
	}

	// Test all services
	validatePostgreSQLService(result)
	validateNATSService(result)
	validateRedisService(result)
	validateQdrantService(result)

	// Print results
	for serviceName, service := range result.Services {
		t.Logf("Service: %s", serviceName)
		t.Logf("  Status: %s", service.Status)
		t.Logf("  Connection: %s", service.Connection)
		t.Logf("")
	}

	if len(result.Warnings) > 0 {
		t.Log("Warnings:")
		for _, warning := range result.Warnings {
			t.Logf("  - %s", warning)
		}
	}

	if len(result.Errors) > 0 {
		t.Log("Errors:")
		for _, err := range result.Errors {
			t.Logf("  - %s", err)
		}
	}

	t.Log("=== End Manual Test ===")
}
