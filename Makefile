# AgentFlow Makefile
# Cross-platform build automation for AgentFlow

# Container registry configuration
REGISTRY ?= ghcr.io
IMAGE_NAME ?= agentflow/agentflow
TAG ?= latest

.PHONY: help build test lint clean dev deps security-scan containers containers-local containers-push

# Default target
help: ## Show this help message
	@echo "AgentFlow Build System"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build all components
build: ## Build all AgentFlow components
	@echo "Building AgentFlow components..."
	go build -o bin/control-plane ./cmd/control-plane
	go build -o bin/worker ./cmd/worker
	go build -o bin/af ./cmd/af
	@echo "Build complete"

# Run tests
test: ## Run all tests
	@echo "Running tests..."
	go test ./...
	cd cmd/control-plane && go test ./...
	cd cmd/worker && go test ./...
	cd cmd/af && go test ./...
	cd sdk/go && go test ./...
	@echo "Tests complete"

# Run linting
lint: ## Run golangci-lint on all modules
	@echo "Running linting..."
	golangci-lint run ./...
	cd cmd/control-plane && golangci-lint run ./...
	cd cmd/worker && golangci-lint run ./...
	cd cmd/af && golangci-lint run ./...
	cd sdk/go && golangci-lint run ./...
	@echo "Linting complete"

# Install dependencies
deps: ## Install/update dependencies
	@echo "Installing dependencies..."
	go mod tidy
	cd cmd/control-plane && go mod tidy
	cd cmd/worker && go mod tidy
	cd cmd/af && go mod tidy
	cd sdk/go && go mod tidy
	@echo "Dependencies updated"

# Development environment
dev: ## Start development environment
	@echo "Starting development environment..."
	@echo "Development environment setup complete"

# Security scanning
security-scan: ## Run security scans
	@echo "Running security scans..."
	@echo "Security scanning will be implemented with gosec, osv-scanner, etc."

# Governance validation
validate-governance: ## Validate risk register and ADR structure
	@echo "Validating governance artifacts..."
	go run scripts/validate-governance.go all
	@echo "Governance validation complete"

validate-risks: ## Validate risk register schema
	@echo "Validating risk register..."
	go run scripts/validate-governance.go risk-schema

validate-adrs: ## Validate ADR filename patterns
	@echo "Validating ADR filenames..."
	go run scripts/validate-governance.go adr-filenames

# Container builds
containers: ## Build container images
	@echo "Building multi-arch containers..."
	docker buildx build --platform linux/amd64,linux/arm64 -t agentflow/control-plane:latest -f cmd/control-plane/Dockerfile .
	docker buildx build --platform linux/amd64,linux/arm64 -t agentflow/worker:latest -f cmd/worker/Dockerfile .
	docker buildx build --platform linux/amd64,linux/arm64 -t agentflow/af:latest -f cmd/af/Dockerfile .
	@echo "Multi-arch container builds complete"

containers-local: ## Build container images for local platform
	@echo "Building containers for local platform..."
	docker build -t agentflow/control-plane:latest -f cmd/control-plane/Dockerfile .
	docker build -t agentflow/worker:latest -f cmd/worker/Dockerfile .
	docker build -t agentflow/af:latest -f cmd/af/Dockerfile .
	@echo "Local container builds complete"

containers-push: ## Build and push container images
	@echo "Building and pushing multi-arch containers..."
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $(REGISTRY)/$(IMAGE_NAME)/control-plane:$(TAG) -f cmd/control-plane/Dockerfile .
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $(REGISTRY)/$(IMAGE_NAME)/worker:$(TAG) -f cmd/worker/Dockerfile .
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $(REGISTRY)/$(IMAGE_NAME)/af:$(TAG) -f cmd/af/Dockerfile .
	@echo "Multi-arch container push complete"

test-containers: ## Test container builds, signatures, and SBOM
	@echo "Testing container builds..."
	@if command -v bash >/dev/null 2>&1; then \
		bash scripts/test-container-build.sh; \
	else \
		powershell -ExecutionPolicy Bypass -File scripts/test-container-build.ps1; \
	fi

# Database migrations
migrate-up: ## Run database migrations up
	@echo "Running database migrations up..."
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down: ## Rollback last database migration
	@echo "Rolling back last database migration..."
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status: ## Show migration status
	@echo "Checking migration status..."
	goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-create: ## Create new migration (usage: make migrate-create NAME=migration_name)
	@echo "Creating new migration: $(NAME)"
	goose -dir migrations create $(NAME) sql

# Generate type-safe queries
sqlc-generate: ## Generate type-safe Go code from SQL queries
	@echo "Generating type-safe queries with sqlc..."
	sqlc generate

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean ./...
	cd cmd/control-plane && go clean ./...
	cd cmd/worker && go clean ./...
	cd cmd/af && go clean ./...
	cd sdk/go && go clean ./...
	@echo "Clean complete"

# Cross-platform builds
build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@mkdir -p bin/linux
	cd cmd/control-plane && GOOS=linux GOARCH=amd64 go build -o ../../bin/linux/control-plane .
	cd cmd/worker && GOOS=linux GOARCH=amd64 go build -o ../../bin/linux/worker .
	cd cmd/af && GOOS=linux GOARCH=amd64 go build -o ../../bin/linux/af .
	@echo "Linux builds complete"

build-windows: ## Build for Windows
	@echo "Building for Windows..."
	@mkdir -p bin/windows
	cd cmd/control-plane && GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/control-plane.exe .
	cd cmd/worker && GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/worker.exe .
	cd cmd/af && GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/af.exe .
	@echo "Windows builds complete"

build-darwin: ## Build for macOS
	@echo "Building for macOS..."
	@mkdir -p bin/darwin
	cd cmd/control-plane && GOOS=darwin GOARCH=amd64 go build -o ../../bin/darwin/control-plane .
	cd cmd/worker && GOOS=darwin GOARCH=amd64 go build -o ../../bin/darwin/worker .
	cd cmd/af && GOOS=darwin GOARCH=amd64 go build -o ../../bin/darwin/af .
	@echo "macOS builds complete"

build-all: build-linux build-windows build-darwin ## Build for all platforms

# Cross-platform build validation
test-cross-platform: ## Test cross-platform builds
	@echo "Running cross-platform build tests..."
	@if command -v bash >/dev/null 2>&1; then \
		bash scripts/test-cross-platform-build.sh; \
	else \
		powershell -ExecutionPolicy Bypass -File scripts/test-cross-platform-build.ps1; \
	fi

validate-cross-platform: ## Validate cross-platform build compatibility
	@echo "Validating cross-platform build compatibility..."
	@if command -v bash >/dev/null 2>&1; then \
		bash scripts/test-cross-platform-build.sh; \
	else \
		powershell -ExecutionPolicy Bypass -File scripts/test-cross-platform-build.ps1; \
	fi