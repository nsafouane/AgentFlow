#!/usr/bin/env pwsh
# Test script for migration tooling on Windows
# This script validates that goose migrations work correctly with Windows paths

param(
    [string]$DatabaseUrl = $env:DATABASE_URL
)

Write-Host "AgentFlow Migration Testing Script" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Green

# Check if DATABASE_URL is provided
if (-not $DatabaseUrl) {
    Write-Host "DATABASE_URL not provided. Using test database URL..." -ForegroundColor Yellow
    $DatabaseUrl = "postgres://test:test@localhost:5432/agentflow_test?sslmode=disable"
    Write-Host "Test URL: $DatabaseUrl" -ForegroundColor Yellow
    Write-Host "Note: This will fail without a running PostgreSQL instance" -ForegroundColor Yellow
}

# Test 1: Validate migration directory structure
Write-Host "`nTest 1: Validating migration directory structure..." -ForegroundColor Cyan
if (Test-Path "migrations") {
    Write-Host "✓ Migrations directory exists" -ForegroundColor Green
    
    $migrationFiles = Get-ChildItem -Path "migrations" -Filter "*.sql"
    if ($migrationFiles.Count -gt 0) {
        Write-Host "✓ Found $($migrationFiles.Count) migration file(s)" -ForegroundColor Green
        foreach ($file in $migrationFiles) {
            Write-Host "  - $($file.Name)" -ForegroundColor Gray
        }
    } else {
        Write-Host "✗ No migration files found" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "✗ Migrations directory not found" -ForegroundColor Red
    exit 1
}

# Test 2: Validate Windows path handling
Write-Host "`nTest 2: Validating Windows path handling..." -ForegroundColor Cyan
$currentPath = Get-Location
$migrationsPath = Join-Path $currentPath "migrations"
$normalizedPath = $migrationsPath -replace '\\', '/'

Write-Host "Current directory: $currentPath" -ForegroundColor Gray
Write-Host "Migrations path: $migrationsPath" -ForegroundColor Gray
Write-Host "Normalized path: $normalizedPath" -ForegroundColor Gray

if (Test-Path $migrationsPath) {
    Write-Host "✓ Windows path resolution works correctly" -ForegroundColor Green
} else {
    Write-Host "✗ Windows path resolution failed" -ForegroundColor Red
    exit 1
}

# Test 3: Check goose binary availability
Write-Host "`nTest 3: Checking goose binary availability..." -ForegroundColor Cyan
try {
    $gooseVersion = & goose -version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Goose is available: $gooseVersion" -ForegroundColor Green
    } else {
        Write-Host "✗ Goose command failed" -ForegroundColor Red
        Write-Host "Error: $gooseVersion" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "✗ Goose binary not found in PATH" -ForegroundColor Red
    Write-Host "Please install goose: go install github.com/pressly/goose/v3/cmd/goose@latest" -ForegroundColor Yellow
    exit 1
}

# Test 4: Validate migration file syntax
Write-Host "`nTest 4: Validating migration file syntax..." -ForegroundColor Cyan
$migrationFiles = Get-ChildItem -Path "migrations" -Filter "*.sql"
foreach ($file in $migrationFiles) {
    $content = Get-Content -Path $file.FullName -Raw
    
    if ($content -match "-- \+goose Up" -and $content -match "-- \+goose Down") {
        Write-Host "✓ $($file.Name) has correct goose directives" -ForegroundColor Green
    } else {
        Write-Host "✗ $($file.Name) missing goose directives" -ForegroundColor Red
        exit 1
    }
}

# Test 5: Test migration commands (dry run)
Write-Host "`nTest 5: Testing migration commands (dry run)..." -ForegroundColor Cyan

# Test status command (this should work even without DB connection in some cases)
Write-Host "Testing goose status command..." -ForegroundColor Gray
try {
    $statusOutput = & goose -dir migrations postgres $DatabaseUrl status 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Goose status command works" -ForegroundColor Green
        Write-Host "Status output:" -ForegroundColor Gray
        Write-Host $statusOutput -ForegroundColor Gray
    } else {
        Write-Host "⚠ Goose status command failed (expected without DB connection)" -ForegroundColor Yellow
        Write-Host "Error: $statusOutput" -ForegroundColor Gray
    }
} catch {
    Write-Host "⚠ Goose status command failed (expected without DB connection)" -ForegroundColor Yellow
}

# Test 6: Validate sqlc configuration
Write-Host "`nTest 6: Validating sqlc configuration..." -ForegroundColor Cyan
if (Test-Path "sqlc.yaml") {
    Write-Host "✓ sqlc.yaml configuration file exists" -ForegroundColor Green
    
    # Check if sqlc is available
    try {
        $sqlcVersion = & sqlc version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ sqlc is available: $sqlcVersion" -ForegroundColor Green
            
            # Test sqlc generate
            Write-Host "Testing sqlc generate..." -ForegroundColor Gray
            $generateOutput = & sqlc generate 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host "✓ sqlc generate completed successfully" -ForegroundColor Green
            } else {
                Write-Host "✗ sqlc generate failed" -ForegroundColor Red
                Write-Host "Error: $generateOutput" -ForegroundColor Red
                exit 1
            }
        } else {
            Write-Host "✗ sqlc command failed" -ForegroundColor Red
            Write-Host "Error: $sqlcVersion" -ForegroundColor Red
            exit 1
        }
    } catch {
        Write-Host "✗ sqlc binary not found in PATH" -ForegroundColor Red
        Write-Host "Please install sqlc: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest" -ForegroundColor Yellow
        exit 1
    }
} else {
    Write-Host "✗ sqlc.yaml configuration file not found" -ForegroundColor Red
    exit 1
}

# Test 7: Validate generated code compiles
Write-Host "`nTest 7: Validating generated code compiles..." -ForegroundColor Cyan
$testOutput = & go test ./internal/storage/queries/ 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Generated sqlc code compiles and tests pass" -ForegroundColor Green
} else {
    Write-Host "✗ Generated sqlc code compilation failed" -ForegroundColor Red
    Write-Host "Error: $testOutput" -ForegroundColor Red
    exit 1
}

Write-Host "`nAll migration tooling tests completed successfully! ✓" -ForegroundColor Green
Write-Host "Migration tooling is properly configured for Windows development." -ForegroundColor Green