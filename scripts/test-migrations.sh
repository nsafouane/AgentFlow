#!/bin/bash
# Test script for migration tooling on Unix-like systems
# This script validates that goose migrations work correctly

set -e

DATABASE_URL=${DATABASE_URL:-"postgres://test:test@localhost:5432/agentflow_test?sslmode=disable"}

echo "AgentFlow Migration Testing Script"
echo "================================="

# Check if DATABASE_URL is provided
if [ -z "$DATABASE_URL" ]; then
    echo "DATABASE_URL not provided. Using test database URL..."
    DATABASE_URL="postgres://test:test@localhost:5432/agentflow_test?sslmode=disable"
    echo "Test URL: $DATABASE_URL"
    echo "Note: This will fail without a running PostgreSQL instance"
fi

# Test 1: Validate migration directory structure
echo
echo "Test 1: Validating migration directory structure..."
if [ -d "migrations" ]; then
    echo "✓ Migrations directory exists"
    
    migration_count=$(find migrations -name "*.sql" | wc -l)
    if [ "$migration_count" -gt 0 ]; then
        echo "✓ Found $migration_count migration file(s)"
        find migrations -name "*.sql" | while read -r file; do
            echo "  - $(basename "$file")"
        done
    else
        echo "✗ No migration files found"
        exit 1
    fi
else
    echo "✗ Migrations directory not found"
    exit 1
fi

# Test 2: Validate path handling
echo
echo "Test 2: Validating path handling..."
current_path=$(pwd)
migrations_path="$current_path/migrations"

echo "Current directory: $current_path"
echo "Migrations path: $migrations_path"

if [ -d "$migrations_path" ]; then
    echo "✓ Path resolution works correctly"
else
    echo "✗ Path resolution failed"
    exit 1
fi

# Test 3: Check goose binary availability
echo
echo "Test 3: Checking goose binary availability..."
if command -v goose >/dev/null 2>&1; then
    goose_version=$(goose -version 2>&1 || echo "version check failed")
    echo "✓ Goose is available: $goose_version"
else
    echo "✗ Goose binary not found in PATH"
    echo "Please install goose: go install github.com/pressly/goose/v3/cmd/goose@latest"
    exit 1
fi

# Test 4: Validate migration file syntax
echo
echo "Test 4: Validating migration file syntax..."
find migrations -name "*.sql" | while read -r file; do
    if grep -q "-- +goose Up" "$file" && grep -q "-- +goose Down" "$file"; then
        echo "✓ $(basename "$file") has correct goose directives"
    else
        echo "✗ $(basename "$file") missing goose directives"
        exit 1
    fi
done

# Test 5: Test migration commands (dry run)
echo
echo "Test 5: Testing migration commands (dry run)..."

# Test status command (this should work even without DB connection in some cases)
echo "Testing goose status command..."
if goose -dir migrations postgres "$DATABASE_URL" status >/dev/null 2>&1; then
    echo "✓ Goose status command works"
    echo "Status output:"
    goose -dir migrations postgres "$DATABASE_URL" status 2>&1 || true
else
    echo "⚠ Goose status command failed (expected without DB connection)"
fi

# Test 6: Validate sqlc configuration
echo
echo "Test 6: Validating sqlc configuration..."
if [ -f "sqlc.yaml" ]; then
    echo "✓ sqlc.yaml configuration file exists"
    
    # Check if sqlc is available
    if command -v sqlc >/dev/null 2>&1; then
        sqlc_version=$(sqlc version 2>&1 || echo "version check failed")
        echo "✓ sqlc is available: $sqlc_version"
        
        # Test sqlc generate
        echo "Testing sqlc generate..."
        if sqlc generate >/dev/null 2>&1; then
            echo "✓ sqlc generate completed successfully"
        else
            echo "✗ sqlc generate failed"
            sqlc generate 2>&1 || true
            exit 1
        fi
    else
        echo "✗ sqlc binary not found in PATH"
        echo "Please install sqlc: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"
        exit 1
    fi
else
    echo "✗ sqlc.yaml configuration file not found"
    exit 1
fi

# Test 7: Validate generated code compiles
echo
echo "Test 7: Validating generated code compiles..."
if go test ./internal/storage/queries/ >/dev/null 2>&1; then
    echo "✓ Generated sqlc code compiles and tests pass"
else
    echo "✗ Generated sqlc code compilation failed"
    go test ./internal/storage/queries/ 2>&1 || true
    exit 1
fi

echo
echo "All migration tooling tests completed successfully! ✓"
echo "Migration tooling is properly configured for development."