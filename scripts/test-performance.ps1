# Performance testing script for CI/CD pipeline (PowerShell version)
# This script runs performance tests and enforces thresholds

param(
    [string]$Mode = $env:AF_PERFORMANCE_MODE ?? "ci",
    [int]$P95ThresholdMs = $env:AF_PERF_P95_THRESHOLD_MS ?? 15,
    [int]$P50ThresholdMs = $env:AF_PERF_P50_THRESHOLD_MS ?? 5,
    [int]$MinThroughput = $env:AF_PERF_MIN_THROUGHPUT ?? 100,
    [bool]$SkipPerformance = ($env:AF_SKIP_PERFORMANCE -eq "true"),
    [switch]$Help
)

# Configuration
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$TestPackage = "./pkg/messaging"
$ResultsDir = Join-Path $ProjectRoot "performance-results"
$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"

# Colors for output (if supported)
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
    White = "White"
}

# Logging functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Colors.Blue
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor $Colors.Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Colors.Red
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Colors.Green
}

# Show help
if ($Help) {
    Write-Host "Usage: .\test-performance.ps1 [options]"
    Write-Host ""
    Write-Host "Parameters:"
    Write-Host "  -Mode <string>           Test mode: ci, local, or baseline (default: ci)"
    Write-Host "  -P95ThresholdMs <int>    P95 latency threshold in milliseconds (default: 15)"
    Write-Host "  -P50ThresholdMs <int>    P50 latency threshold in milliseconds (default: 5)"
    Write-Host "  -MinThroughput <int>     Minimum throughput in msg/sec (default: 100)"
    Write-Host "  -SkipPerformance         Skip performance tests"
    Write-Host "  -Help                    Show this help message"
    Write-Host ""
    Write-Host "Environment variables:"
    Write-Host "  AF_PERF_P95_THRESHOLD_MS  P95 latency threshold in milliseconds"
    Write-Host "  AF_PERF_P50_THRESHOLD_MS  P50 latency threshold in milliseconds"
    Write-Host "  AF_PERF_MIN_THROUGHPUT    Minimum throughput in msg/sec"
    Write-Host "  AF_SKIP_PERFORMANCE       Skip performance tests (true/false)"
    Write-Host "  AF_PERFORMANCE_MODE       Test mode: ci, local, or baseline"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\test-performance.ps1                           # Run CI performance tests"
    Write-Host "  .\test-performance.ps1 -Mode local               # Run local performance tests"
    Write-Host "  .\test-performance.ps1 -Mode baseline            # Record performance baseline"
    Write-Host "  .\test-performance.ps1 -P95ThresholdMs 25        # Use custom P95 threshold"
    exit 0
}

# Print configuration
function Show-Config {
    Write-Info "Performance Test Configuration:"
    Write-Info "  Mode: $Mode"
    Write-Info "  P95 Threshold: ${P95ThresholdMs}ms"
    Write-Info "  P50 Threshold: ${P50ThresholdMs}ms"
    Write-Info "  Min Throughput: $MinThroughput msg/sec"
    Write-Info "  Results Directory: $ResultsDir"
    Write-Info "  Skip Performance: $SkipPerformance"
    Write-Host ""
}

# Check if performance tests should be skipped
function Test-Skip {
    if ($SkipPerformance) {
        Write-Warn "Performance tests skipped (AF_SKIP_PERFORMANCE=true)"
        exit 0
    }
}

# Setup results directory
function Initialize-ResultsDir {
    if (-not (Test-Path $ResultsDir)) {
        New-Item -ItemType Directory -Path $ResultsDir -Force | Out-Null
    }
    Write-Info "Created results directory: $ResultsDir"
}

# Check prerequisites
function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check if Go is available
    try {
        $goVersion = go version 2>$null
        if (-not $goVersion) {
            throw "Go command failed"
        }
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        exit 1
    }
    
    # Check if Docker is available
    try {
        $dockerVersion = docker --version 2>$null
        if (-not $dockerVersion) {
            throw "Docker command failed"
        }
    }
    catch {
        Write-Error "Docker is not installed or not in PATH"
        exit 1
    }
    
    # Check if Docker daemon is running
    try {
        docker info 2>$null | Out-Null
    }
    catch {
        Write-Error "Docker daemon is not running"
        exit 1
    }
    
    Write-Success "Prerequisites check passed"
}

# Run performance threshold tests
function Invoke-ThresholdTests {
    Write-Info "Running performance threshold tests..."
    
    Set-Location $ProjectRoot
    
    # Set environment variables for the test
    $env:AF_PERF_P95_THRESHOLD_MS = $P95ThresholdMs.ToString()
    $env:AF_PERF_P50_THRESHOLD_MS = $P50ThresholdMs.ToString()
    
    # Run the performance threshold test
    $testOutput = ""
    $testExitCode = 0
    
    try {
        $testOutput = go test -v $TestPackage -run TestPerformanceThresholds 2>&1
    }
    catch {
        $testExitCode = $LASTEXITCODE
    }
    
    # Save test output
    $testLogFile = Join-Path $ResultsDir "threshold_test_${Timestamp}.log"
    $testOutput | Out-File -FilePath $testLogFile -Encoding UTF8
    
    if ($testExitCode -eq 0) {
        Write-Success "Performance threshold tests PASSED"
        
        # Extract key metrics from output
        $p95Latency = ($testOutput | Select-String "Latency P95: ([0-9.]*[a-z]*)" | Select-Object -First 1).Matches.Groups[1].Value
        $throughput = ($testOutput | Select-String "Throughput: ([0-9.]*)" | Select-Object -First 1).Matches.Groups[1].Value
        
        if ($p95Latency) {
            Write-Info "  P95 Latency: $p95Latency"
        }
        if ($throughput) {
            Write-Info "  Throughput: ${throughput} msg/sec"
        }
        
        return $true
    }
    else {
        Write-Error "Performance threshold tests FAILED"
        Write-Error "Test output saved to: $testLogFile"
        
        # Show relevant error lines
        $testOutput | Select-String "(FAIL|ERROR|exceeds|below)" | Select-Object -First 10 | ForEach-Object {
            Write-Host $_.Line
        }
        
        return $false
    }
}

# Run benchmark tests
function Invoke-Benchmarks {
    Write-Info "Running benchmark tests..."
    
    Set-Location $ProjectRoot
    
    $benchmarkOutput = ""
    $benchmarkExitCode = 0
    
    try {
        $benchmarkOutput = go test -bench=BenchmarkPingPong -benchmem -benchtime=10s $TestPackage 2>&1
    }
    catch {
        $benchmarkExitCode = $LASTEXITCODE
    }
    
    # Save benchmark output
    $benchmarkLogFile = Join-Path $ResultsDir "benchmark_${Timestamp}.log"
    $benchmarkOutput | Out-File -FilePath $benchmarkLogFile -Encoding UTF8
    
    if ($benchmarkExitCode -eq 0) {
        Write-Success "Benchmark tests completed"
        
        # Extract and display key metrics
        $benchmarkOutput | Select-String "(BenchmarkPingPong|p95_latency_ms|throughput_msg_per_sec)" | ForEach-Object {
            Write-Info "  $($_.Line)"
        }
        
        return $true
    }
    else {
        Write-Error "Benchmark tests failed"
        Write-Error "Benchmark output saved to: $benchmarkLogFile"
        return $false
    }
}

# Run regression detection
function Invoke-RegressionTests {
    Write-Info "Running regression detection tests..."
    
    Set-Location $ProjectRoot
    
    $regressionOutput = ""
    $regressionExitCode = 0
    
    try {
        $regressionOutput = go test -v $TestPackage -run TestPerformanceRegression 2>&1
    }
    catch {
        $regressionExitCode = $LASTEXITCODE
    }
    
    # Save regression test output
    $regressionLogFile = Join-Path $ResultsDir "regression_test_${Timestamp}.log"
    $regressionOutput | Out-File -FilePath $regressionLogFile -Encoding UTF8
    
    if ($regressionExitCode -eq 0) {
        Write-Success "Regression detection completed"
        return $true
    }
    else {
        Write-Warn "Regression detection had issues (this may be expected if no baseline exists)"
        Write-Info "Regression output saved to: $regressionLogFile"
        return $true  # Don't fail CI for regression tests yet
    }
}

# Generate performance report
function New-PerformanceReport {
    Write-Info "Generating performance report..."
    
    $reportFile = Join-Path $ResultsDir "performance_report_${Timestamp}.md"
    
    $reportContent = @"
# Performance Test Report

**Date:** $(Get-Date -Format "yyyy-MM-dd HH:mm:ss UTC")
**Mode:** $Mode
**Commit:** $($env:GITHUB_SHA ?? (git rev-parse HEAD 2>$null) ?? "unknown")
**Branch:** $($env:GITHUB_REF_NAME ?? (git branch --show-current 2>$null) ?? "unknown")

## Configuration

- P95 Threshold: ${P95ThresholdMs}ms
- P50 Threshold: ${P50ThresholdMs}ms
- Min Throughput: $MinThroughput msg/sec

## Test Results

"@

    # Add threshold test results if available
    $thresholdLogFile = Join-Path $ResultsDir "threshold_test_${Timestamp}.log"
    if (Test-Path $thresholdLogFile) {
        $reportContent += "`n### Threshold Tests`n`n"
        $reportContent += "``````n"
        $thresholdResults = Get-Content $thresholdLogFile | Select-String "(PASS|FAIL|Latency|Throughput|Performance)" | Select-Object -First 20
        $reportContent += ($thresholdResults -join "`n")
        $reportContent += "`n``````n`n"
    }
    
    # Add benchmark results if available
    $benchmarkLogFile = Join-Path $ResultsDir "benchmark_${Timestamp}.log"
    if (Test-Path $benchmarkLogFile) {
        $reportContent += "### Benchmark Results`n`n"
        $reportContent += "``````n"
        $benchmarkResults = Get-Content $benchmarkLogFile | Select-String "BenchmarkPingPong"
        $reportContent += ($benchmarkResults -join "`n")
        $reportContent += "`n``````n`n"
    }
    
    # Add system information
    $reportContent += "## System Information`n`n"
    $reportContent += "- OS: $($env:OS ?? (Get-CimInstance Win32_OperatingSystem).Caption)`n"
    $reportContent += "- Architecture: $env:PROCESSOR_ARCHITECTURE`n"
    $reportContent += "- Go Version: $(go version)`n"
    $reportContent += "- CPU Cores: $env:NUMBER_OF_PROCESSORS`n"
    
    $reportContent | Out-File -FilePath $reportFile -Encoding UTF8
    
    Write-Success "Performance report generated: $reportFile"
}

# Main execution
function Main {
    Write-Info "Starting AgentFlow Performance Tests"
    Write-Host ""
    
    Show-Config
    Test-Skip
    Test-Prerequisites
    Initialize-ResultsDir
    
    $overallSuccess = $true
    
    # Run tests based on mode
    switch ($Mode) {
        "ci" {
            Write-Info "Running CI performance tests..."
            if (-not (Invoke-ThresholdTests)) {
                $overallSuccess = $false
            }
            Invoke-Benchmarks | Out-Null  # Don't fail CI on benchmark issues
            Invoke-RegressionTests | Out-Null  # Don't fail CI on regression issues yet
        }
        "local" {
            Write-Info "Running local performance tests..."
            Invoke-ThresholdTests | Out-Null
            Invoke-Benchmarks | Out-Null
            Invoke-RegressionTests | Out-Null
        }
        "baseline" {
            Write-Info "Running baseline recording..."
            # This would run the manual baseline recording test
            Set-Location $ProjectRoot
            go test -tags=manual -v $TestPackage -run TestManualBaselineRecording | Out-Null
        }
        default {
            Write-Error "Unknown performance mode: $Mode"
            exit 1
        }
    }
    
    New-PerformanceReport
    
    if ($overallSuccess) {
        Write-Success "All performance tests completed successfully"
        exit 0
    }
    else {
        Write-Error "Some performance tests failed"
        exit 1
    }
}

# Run main function
Main