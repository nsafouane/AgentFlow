package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/agentflow/agentflow/internal/health"
)

// ValidationResult represents the JSON output structure for af validate
type ValidationResult struct {
	Version     string                 `json:"version"`
	Timestamp   string                 `json:"timestamp"`
	Environment EnvironmentInfo        `json:"environment"`
	Tools       map[string]ToolStatus  `json:"tools"`
	Services    map[string]ServiceInfo `json:"services"`
	Warnings    []string               `json:"warnings"`
	Errors      []string               `json:"errors"`
}

type EnvironmentInfo struct {
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	Container    string `json:"container"`
}

type ToolStatus struct {
	Version string `json:"version"`
	Status  string `json:"status"` // ok, warning, error
	Message string `json:"message,omitempty"`
}

type ServiceInfo struct {
	Status     string `json:"status"` // available, unavailable, unknown
	Connection string `json:"connection,omitempty"`
}

// Progress: IN PROGRESS - 2025-08-24
// Implementation: ✓ CLI stub created and functional, validate command added, audit verify command added
// Unit Tests: ✓ Basic test coverage implemented
// Manual Testing: ✓ Cross-platform build validation passed
// Documentation: ✓ Architecture documentation updated
func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "validate":
		validateEnvironment()
	case "audit":
		handleAuditCommand()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("AgentFlow CLI")
	fmt.Println("Usage:")
	fmt.Println("  af validate       Validate development environment")
	fmt.Println("  af audit verify   Verify audit hash-chain integrity")
	log.Println("CLI tool - ready for implementation")
}

func validateEnvironment() {
	result := ValidationResult{
		Version:     "1.0.0",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Environment: detectEnvironment(),
		Tools:       make(map[string]ToolStatus),
		Services:    make(map[string]ServiceInfo),
		Warnings:    []string{},
		Errors:      []string{},
	}

	// Validate tools
	validateGo(&result)
	validateDocker(&result)
	validateTask(&result)
	validatePostgreSQL(&result)
	validateNATS(&result)
	validateGolangciLint(&result)
	validateGoose(&result)
	validateSqlc(&result)
	validateGosec(&result)
	validateGitleaks(&result)
	validatePreCommit(&result)

	// Validate services
	validatePostgreSQLService(&result)
	validateNATSService(&result)
	validateRedisService(&result)
	validateQdrantService(&result)

	// Add container warning if not in container
	if result.Environment.Container == "host" {
		result.Warnings = append(result.Warnings,
			"Running on host system. Consider using VS Code devcontainer for consistent environment.")
		result.Warnings = append(result.Warnings,
			"Devcontainer provides standardized tooling, dependencies, and configuration.")
		result.Warnings = append(result.Warnings,
			"To use devcontainer: Open this project in VS Code and select 'Reopen in Container'.")
	}

	// Output JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal validation result: %v", err)
	}

	fmt.Println(string(output))

	// Exit with error code if there are errors
	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

func detectEnvironment() EnvironmentInfo {
	env := EnvironmentInfo{
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Container:    "host",
	}

	// Detect container environment
	if os.Getenv("DEVCONTAINER") != "" {
		env.Container = "devcontainer"
	} else if os.Getenv("CODESPACES") != "" {
		env.Container = "codespaces"
	} else if _, err := os.Stat("/.dockerenv"); err == nil {
		env.Container = "docker"
	}

	return env
}

func validateGo(result *ValidationResult) {
	if !commandExists("go") {
		result.Tools["go"] = ToolStatus{
			Status:  "error",
			Message: "Go is not installed",
		}
		result.Errors = append(result.Errors, "Go is not installed")
		return
	}

	output, err := exec.Command("go", "version").Output()
	if err != nil {
		result.Tools["go"] = ToolStatus{
			Status:  "error",
			Message: "Failed to get Go version",
		}
		result.Errors = append(result.Errors, "Failed to get Go version")
		return
	}

	version := strings.Fields(string(output))[2] // "go1.22.0" -> "1.22.0"
	version = strings.TrimPrefix(version, "go")

	result.Tools["go"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validateDocker(result *ValidationResult) {
	if !commandExists("docker") {
		result.Tools["docker"] = ToolStatus{
			Status:  "warning",
			Message: "Docker is not installed",
		}
		result.Warnings = append(result.Warnings, "Docker is not installed")
		return
	}

	output, err := exec.Command("docker", "--version").Output()
	if err != nil {
		result.Tools["docker"] = ToolStatus{
			Status:  "warning",
			Message: "Failed to get Docker version",
		}
		result.Warnings = append(result.Warnings, "Failed to get Docker version")
		return
	}

	// Parse "Docker version 24.0.7, build afdd53b" -> "24.0.7"
	fields := strings.Fields(string(output))
	var version string
	for i, field := range fields {
		if field == "version" && i+1 < len(fields) {
			version = strings.TrimSuffix(fields[i+1], ",")
			break
		}
	}

	result.Tools["docker"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validateTask(result *ValidationResult) {
	if !commandExists("task") {
		result.Tools["task"] = ToolStatus{
			Status:  "warning",
			Message: "Task is not installed",
		}
		result.Warnings = append(result.Warnings, "Task is not installed")
		return
	}

	output, err := exec.Command("task", "--version").Output()
	if err != nil {
		result.Tools["task"] = ToolStatus{
			Status:  "warning",
			Message: "Failed to get Task version",
		}
		result.Warnings = append(result.Warnings, "Failed to get Task version")
		return
	}

	// Parse "Task version: v3.35.1 (h1:...)" -> "3.35.1"
	version := strings.TrimSpace(string(output))
	if strings.Contains(version, "version:") {
		parts := strings.Split(version, "version:")
		if len(parts) > 1 {
			version = strings.TrimSpace(parts[1])
			version = strings.TrimPrefix(version, "v")
			if spaceIndex := strings.Index(version, " "); spaceIndex != -1 {
				version = version[:spaceIndex]
			}
		}
	}

	result.Tools["task"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validatePostgreSQL(result *ValidationResult) {
	if !commandExists("psql") {
		result.Tools["psql"] = ToolStatus{
			Status:  "warning",
			Message: "PostgreSQL client (psql) is not installed",
		}
		result.Warnings = append(result.Warnings, "PostgreSQL client (psql) is not installed")
		return
	}

	output, err := exec.Command("psql", "--version").Output()
	if err != nil {
		result.Tools["psql"] = ToolStatus{
			Status:  "warning",
			Message: "Failed to get PostgreSQL version",
		}
		result.Warnings = append(result.Warnings, "Failed to get PostgreSQL version")
		return
	}

	// Parse "psql (PostgreSQL) 15.4" -> "15.4"
	fields := strings.Fields(string(output))
	var version string
	for _, field := range fields {
		if strings.Contains(field, ".") && len(field) < 10 {
			version = field
			break
		}
	}

	result.Tools["psql"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validateNATS(result *ValidationResult) {
	if !commandExists("nats") {
		result.Tools["nats"] = ToolStatus{
			Status:  "warning",
			Message: "NATS CLI is not installed",
		}
		result.Warnings = append(result.Warnings, "NATS CLI is not installed")
		return
	}

	// NATS CLI version output varies, just check if it's functional
	err := exec.Command("nats", "--help").Run()
	if err != nil {
		result.Tools["nats"] = ToolStatus{
			Status:  "warning",
			Message: "NATS CLI is not functional",
		}
		result.Warnings = append(result.Warnings, "NATS CLI is not functional")
		return
	}

	result.Tools["nats"] = ToolStatus{
		Version: "installed",
		Status:  "ok",
	}
}

func validateGolangciLint(result *ValidationResult) {
	if !commandExists("golangci-lint") {
		result.Tools["golangci-lint"] = ToolStatus{
			Status:  "warning",
			Message: "golangci-lint is not installed",
		}
		result.Warnings = append(result.Warnings, "golangci-lint is not installed")
		return
	}

	output, err := exec.Command("golangci-lint", "version").Output()
	if err != nil {
		result.Tools["golangci-lint"] = ToolStatus{
			Status:  "warning",
			Message: "Failed to get golangci-lint version",
		}
		result.Warnings = append(result.Warnings, "Failed to get golangci-lint version")
		return
	}

	// Parse version from output
	lines := strings.Split(string(output), "\n")
	var version string
	for _, line := range lines {
		if strings.Contains(line, "version") {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.HasPrefix(field, "v") && strings.Contains(field, ".") {
					version = strings.TrimPrefix(field, "v")
					break
				}
			}
			break
		}
	}

	result.Tools["golangci-lint"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validateGoose(result *ValidationResult) {
	if !commandExists("goose") {
		result.Tools["goose"] = ToolStatus{
			Status:  "warning",
			Message: "goose is not installed",
		}
		result.Warnings = append(result.Warnings, "goose is not installed")
		return
	}

	result.Tools["goose"] = ToolStatus{
		Version: "installed",
		Status:  "ok",
	}
}

func validateSqlc(result *ValidationResult) {
	if !commandExists("sqlc") {
		result.Tools["sqlc"] = ToolStatus{
			Status:  "warning",
			Message: "sqlc is not installed",
		}
		result.Warnings = append(result.Warnings, "sqlc is not installed")
		return
	}

	output, err := exec.Command("sqlc", "version").Output()
	if err != nil {
		result.Tools["sqlc"] = ToolStatus{
			Status:  "warning",
			Message: "Failed to get sqlc version",
		}
		result.Warnings = append(result.Warnings, "Failed to get sqlc version")
		return
	}

	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "v")

	result.Tools["sqlc"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validateGosec(result *ValidationResult) {
	if !commandExists("gosec") {
		result.Tools["gosec"] = ToolStatus{
			Status:  "warning",
			Message: "gosec is not installed",
		}
		result.Warnings = append(result.Warnings, "gosec is not installed")
		return
	}

	result.Tools["gosec"] = ToolStatus{
		Version: "installed",
		Status:  "ok",
	}
}

func validateGitleaks(result *ValidationResult) {
	if !commandExists("gitleaks") {
		result.Tools["gitleaks"] = ToolStatus{
			Status:  "warning",
			Message: "gitleaks is not installed",
		}
		result.Warnings = append(result.Warnings, "gitleaks is not installed")
		return
	}

	output, err := exec.Command("gitleaks", "version").Output()
	if err != nil {
		result.Tools["gitleaks"] = ToolStatus{
			Status:  "warning",
			Message: "Failed to get gitleaks version",
		}
		result.Warnings = append(result.Warnings, "Failed to get gitleaks version")
		return
	}

	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "v")

	result.Tools["gitleaks"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validatePreCommit(result *ValidationResult) {
	if !commandExists("pre-commit") {
		result.Tools["pre-commit"] = ToolStatus{
			Status:  "warning",
			Message: "pre-commit is not installed",
		}
		result.Warnings = append(result.Warnings, "pre-commit is not installed")
		return
	}

	output, err := exec.Command("pre-commit", "--version").Output()
	if err != nil {
		result.Tools["pre-commit"] = ToolStatus{
			Status:  "warning",
			Message: "Failed to get pre-commit version",
		}
		result.Warnings = append(result.Warnings, "Failed to get pre-commit version")
		return
	}

	// Parse "pre-commit 3.6.0" -> "3.6.0"
	fields := strings.Fields(string(output))
	var version string
	if len(fields) >= 2 {
		version = fields[1]
	}

	result.Tools["pre-commit"] = ToolStatus{
		Version: version,
		Status:  "ok",
	}
}

func validatePostgreSQLService(result *ValidationResult) {
	// Try to connect to default PostgreSQL service
	err := exec.Command("psql", "-h", "localhost", "-p", "5432", "-U", "agentflow", "-d", "agentflow_dev", "-c", "SELECT 1;").Run()
	if err != nil {
		result.Services["postgres"] = ServiceInfo{
			Status:     "unavailable",
			Connection: "Failed to connect to PostgreSQL at localhost:5432",
		}
	} else {
		result.Services["postgres"] = ServiceInfo{
			Status:     "available",
			Connection: "postgresql://agentflow@localhost:5432/agentflow_dev",
		}
	}
}

func validateNATSService(result *ValidationResult) {
	// Try to connect to NATS service
	err := exec.Command("nats", "server", "check", "--server=localhost:4222").Run()
	if err != nil {
		result.Services["nats"] = ServiceInfo{
			Status:     "unavailable",
			Connection: "Failed to connect to NATS at localhost:4222",
		}
	} else {
		result.Services["nats"] = ServiceInfo{
			Status:     "available",
			Connection: "nats://localhost:4222",
		}
	}
}

func validateRedisService(result *ValidationResult) {
	checker := health.NewServiceChecker(5 * time.Second)
	ctx := context.Background()

	status := checker.CheckRedis(ctx, "localhost:6379")
	result.Services["redis"] = ServiceInfo{
		Status:     status.Status,
		Connection: status.Connection,
	}

	// Add guidance message for Windows users
	if status.Status == "unavailable" && runtime.GOOS == "windows" {
		if status.Message != "" {
			result.Warnings = append(result.Warnings, status.Message)
		}
	}
}

func validateQdrantService(result *ValidationResult) {
	checker := health.NewServiceChecker(5 * time.Second)
	ctx := context.Background()

	status := checker.CheckQdrant(ctx, "localhost:6333")
	result.Services["qdrant"] = ServiceInfo{
		Status:     status.Status,
		Connection: status.Connection,
	}

	// Add guidance message for Windows users
	if status.Status == "unavailable" && runtime.GOOS == "windows" {
		if status.Message != "" {
			result.Warnings = append(result.Warnings, status.Message)
		}
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
