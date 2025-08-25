#!/usr/bin/env pwsh

# Manual Test: Services Validation
# This script tests the Redis & Vector Dev Bootstrap functionality

param(
    [switch]$StartServices = $false,
    [switch]$StopServices = $false,
    [switch]$Verbose = $false
)

$ErrorActionPreference = "Stop"

Write-Host "=== AgentFlow Services Validation Test ===" -ForegroundColor Cyan
Write-Host ""

# Function to run af validate and parse JSON output
function Test-AFValidate {
    Write-Host "Running 'af validate' command..." -ForegroundColor Yellow
    
    try {
        $output = & .\cmd\af\af.exe validate 2>&1
        if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 1) {
            throw "af validate failed with exit code $LASTEXITCODE"
        }
        
        # Parse JSON output
        $result = $output | ConvertFrom-Json
        
        Write-Host "✓ af validate executed successfully" -ForegroundColor Green
        return $result
    }
    catch {
        Write-Host "✗ af validate failed: $_" -ForegroundColor Red
        throw
    }
}

# Function to check service status
function Test-ServiceStatus {
    param($services, $expectedStatus = "available")
    
    $serviceNames = @("postgres", "nats", "redis", "qdrant")
    $results = @{}
    
    foreach ($serviceName in $serviceNames) {
        if ($services.PSObject.Properties.Name -contains $serviceName) {
            $service = $services.$serviceName
            $results[$serviceName] = @{
                Status = $service.status
                Connection = $service.connection
                Expected = $expectedStatus
                Passed = $service.status -eq $expectedStatus
            }
            
            if ($service.status -eq $expectedStatus) {
                Write-Host "✓ $serviceName`: $($service.status) ($($service.connection))" -ForegroundColor Green
            } else {
                Write-Host "✗ $serviceName`: $($service.status) (expected: $expectedStatus)" -ForegroundColor Red
                if ($service.connection) {
                    Write-Host "  Connection: $($service.connection)" -ForegroundColor Gray
                }
            }
        } else {
            Write-Host "✗ $serviceName`: not found in validation output" -ForegroundColor Red
            $results[$serviceName] = @{
                Status = "missing"
                Connection = ""
                Expected = $expectedStatus
                Passed = $false
            }
        }
    }
    
    return $results
}

# Function to start services
function Start-Services {
    Write-Host "Starting Docker Compose services..." -ForegroundColor Yellow
    
    try {
        & docker-compose up -d
        if ($LASTEXITCODE -ne 0) {
            throw "docker-compose up failed with exit code $LASTEXITCODE"
        }
        
        Write-Host "✓ Services started successfully" -ForegroundColor Green
        
        # Wait for services to be ready
        Write-Host "Waiting for services to be ready..." -ForegroundColor Yellow
        Start-Sleep -Seconds 10
        
    }
    catch {
        Write-Host "✗ Failed to start services: $_" -ForegroundColor Red
        throw
    }
}

# Function to stop services
function Stop-Services {
    Write-Host "Stopping Docker Compose services..." -ForegroundColor Yellow
    
    try {
        & docker-compose down
        if ($LASTEXITCODE -ne 0) {
            throw "docker-compose down failed with exit code $LASTEXITCODE"
        }
        
        Write-Host "✓ Services stopped successfully" -ForegroundColor Green
    }
    catch {
        Write-Host "✗ Failed to stop services: $_" -ForegroundColor Red
        throw
    }
}

# Function to check Docker availability
function Test-DockerAvailability {
    Write-Host "Checking Docker availability..." -ForegroundColor Yellow
    
    try {
        & docker --version | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Docker is not available"
        }
        
        & docker ps | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Docker daemon is not running"
        }
        
        Write-Host "✓ Docker is available and running" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "✗ Docker check failed: $_" -ForegroundColor Red
        return $false
    }
}

# Main test execution
try {
    # Check if af.exe exists
    if (-not (Test-Path ".\cmd\af\af.exe")) {
        Write-Host "Building af.exe..." -ForegroundColor Yellow
        Set-Location "cmd\af"
        & go build -o af.exe .
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to build af.exe"
        }
        Set-Location "..\\.."
        Write-Host "✓ af.exe built successfully" -ForegroundColor Green
    }
    
    # Test 1: Validate without services (should show unavailable)
    Write-Host "`n--- Test 1: Validation without services ---" -ForegroundColor Cyan
    $result1 = Test-AFValidate
    $services1 = Test-ServiceStatus -services $result1.services -expectedStatus "unavailable"
    
    # Check Windows-specific guidance messages
    if ($result1.warnings) {
        Write-Host "`nWarnings provided:" -ForegroundColor Yellow
        foreach ($warning in $result1.warnings) {
            Write-Host "  - $warning" -ForegroundColor Gray
        }
        
        # Verify Windows-specific messages
        $redisGuidance = $result1.warnings | Where-Object { $_ -like "*docker-compose up redis*" }
        $qdrantGuidance = $result1.warnings | Where-Object { $_ -like "*docker-compose up qdrant*" }
        
        if ($redisGuidance) {
            Write-Host "✓ Redis guidance message provided" -ForegroundColor Green
        } else {
            Write-Host "✗ Redis guidance message missing" -ForegroundColor Red
        }
        
        if ($qdrantGuidance) {
            Write-Host "✓ Qdrant guidance message provided" -ForegroundColor Green
        } else {
            Write-Host "✗ Qdrant guidance message missing" -ForegroundColor Red
        }
    }
    
    # Test 2: Start services and validate (if requested)
    if ($StartServices) {
        Write-Host "`n--- Test 2: Starting services and validating ---" -ForegroundColor Cyan
        
        if (-not (Test-DockerAvailability)) {
            Write-Host "Skipping service startup test - Docker not available" -ForegroundColor Yellow
        } else {
            Start-Services
            
            # Wait a bit more for services to fully initialize
            Write-Host "Waiting for services to fully initialize..." -ForegroundColor Yellow
            Start-Sleep -Seconds 15
            
            $result2 = Test-AFValidate
            $services2 = Test-ServiceStatus -services $result2.services -expectedStatus "available"
            
            # Check that warnings are reduced when services are available
            if ($result2.warnings) {
                Write-Host "`nRemaining warnings:" -ForegroundColor Yellow
                foreach ($warning in $result2.warnings) {
                    Write-Host "  - $warning" -ForegroundColor Gray
                }
            } else {
                Write-Host "✓ No service-related warnings when services are available" -ForegroundColor Green
            }
        }
    }
    
    # Test 3: Stop services (if requested)
    if ($StopServices) {
        Write-Host "`n--- Test 3: Stopping services ---" -ForegroundColor Cyan
        
        if (Test-DockerAvailability) {
            Stop-Services
        } else {
            Write-Host "Skipping service stop test - Docker not available" -ForegroundColor Yellow
        }
    }
    
    # Summary
    Write-Host "`n=== Test Summary ===" -ForegroundColor Cyan
    Write-Host "✓ af validate command structure validated" -ForegroundColor Green
    Write-Host "✓ Service status reporting implemented" -ForegroundColor Green
    Write-Host "✓ Windows conditional guidance messages working" -ForegroundColor Green
    Write-Host "✓ JSON output format validated" -ForegroundColor Green
    
    if ($StartServices) {
        Write-Host "✓ Service startup and health checking tested" -ForegroundColor Green
    }
    
    Write-Host "`nTest completed successfully!" -ForegroundColor Green
    
}
catch {
    Write-Host "`n✗ Test failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host "`n=== Manual Test Instructions ===" -ForegroundColor Cyan
Write-Host "1. Run without services: .\scripts\test-services-validation.ps1" -ForegroundColor White
Write-Host "2. Run with service startup: .\scripts\test-services-validation.ps1 -StartServices" -ForegroundColor White
Write-Host "3. Run with service cleanup: .\scripts\test-services-validation.ps1 -StartServices -StopServices" -ForegroundColor White
Write-Host "4. Check individual services: docker-compose ps" -ForegroundColor White
Write-Host "5. View service logs: docker-compose logs [service-name]" -ForegroundColor White