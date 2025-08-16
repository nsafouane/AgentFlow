# Cross-Platform Build Testing Script (PowerShell)
# Tests builds on Linux, Windows, and WSL2 environments

param(
    [string]$BinDir = "bin"
)

# Configuration
$Services = @("control-plane", "worker", "af")
$Platforms = @(
    @{OS="linux"; Arch="amd64"},
    @{OS="windows"; Arch="amd64"},
    @{OS="darwin"; Arch="amd64"}
)
$ExitCode = 0

Write-Host "AgentFlow Cross-Platform Build Tests" -ForegroundColor Blue
Write-Host ""

# Function to check if a command exists
function Test-Command {
    param([string]$Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

# Function to detect WSL
function Test-WSL {
    if ($env:WSL_DISTRO_NAME) {
        return $true
    }
    
    if (Test-Path "/proc/version") {
        $version = Get-Content "/proc/version" -ErrorAction SilentlyContinue
        if ($version -and ($version -match "microsoft|wsl")) {
            return $true
        }
    }
    
    return $false
}

# Function to get current platform info
function Get-PlatformInfo {
    $osName = switch ($true) {
        ($IsWindows -or $env:OS -eq "Windows_NT") { "Windows" }
        ($IsLinux) { 
            if (Test-WSL) { "WSL2" } else { "Linux" }
        }
        ($IsMacOS) { "macOS" }
        default { "Unknown" }
    }
    
    $archName = switch ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture) {
        "X64" { "amd64" }
        "Arm64" { "arm64" }
        default { "unknown" }
    }
    
    return "$osName/$archName"
}

# Function to test cross-platform build
function Test-CrossPlatformBuild {
    param(
        [hashtable]$Platform
    )
    
    $platformName = "$($Platform.OS)/$($Platform.Arch)"
    $platformDir = Join-Path $BinDir $Platform.OS
    
    Write-Host "Testing build for $platformName..." -ForegroundColor Yellow
    
    # Create platform directory
    if (-not (Test-Path $platformDir)) {
        New-Item -ItemType Directory -Path $platformDir -Force | Out-Null
    }
    
    $success = $true
    $ext = if ($Platform.OS -eq "windows") { ".exe" } else { "" }
    
    foreach ($service in $Services) {
        $binPath = Join-Path $platformDir "$service$ext"
        
        Write-Host "  Building $service for $platformName..."
        
        # Set environment variables for cross-compilation
        $env:GOOS = $Platform.OS
        $env:GOARCH = $Platform.Arch
        
        try {
            # Build the service (each cmd is a separate module)
            Push-Location "cmd/$service"
            $relativeBinPath = "../../$binPath"
            
            try {
                & go build -o $relativeBinPath . 2>$null
                $buildSuccess = $LASTEXITCODE -eq 0
            }
            catch {
                $buildSuccess = $false
            }
            finally {
                Pop-Location
            }
            
            if ($buildSuccess) {
                Write-Host "  + $service build succeeded" -ForegroundColor Green
                
                # Validate binary exists
                if (Test-Path $binPath) {
                    Write-Host "  + Binary created: $binPath" -ForegroundColor Green
                }
                else {
                    Write-Host "  X Binary not found: $binPath" -ForegroundColor Red
                    $success = $false
                }
            }
            else {
                Write-Host "  X $service build failed" -ForegroundColor Red
                $success = $false
            }
        }
        catch {
            Write-Host "  X $service build failed: $_" -ForegroundColor Red
            $success = $false
        }
        finally {
            # Reset environment variables
            Remove-Item Env:GOOS -ErrorAction SilentlyContinue
            Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        }
    }
    
    if ($success) {
        Write-Host "+ All builds succeeded for $platformName" -ForegroundColor Green
        return $true
    }
    else {
        Write-Host "X Some builds failed for $platformName" -ForegroundColor Red
        return $false
    }
}

# Function to test Makefile targets
function Test-MakefileTargets {
    Write-Host "Testing Makefile cross-platform targets..." -ForegroundColor Yellow
    
    if (-not (Test-Command "make")) {
        Write-Host "! Make not found, skipping Makefile tests" -ForegroundColor Yellow
        return $true
    }
    
    $targets = @("build-linux", "build-windows", "build-all")
    $success = $true
    
    foreach ($target in $targets) {
        Write-Host "  Testing make $target..."
        
        try {
            & make $target 2>$null | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "  + make $target succeeded" -ForegroundColor Green
            }
            else {
                Write-Host "  X make $target failed" -ForegroundColor Red
                $success = $false
            }
        }
        catch {
            Write-Host "  X make $target failed: $_" -ForegroundColor Red
            $success = $false
        }
    }
    
    if ($success) {
        Write-Host "+ All Makefile targets succeeded" -ForegroundColor Green
        return $true
    }
    else {
        Write-Host "X Some Makefile targets failed" -ForegroundColor Red
        return $false
    }
}

# Function to test Taskfile targets
function Test-TaskfileTargets {
    Write-Host "Testing Taskfile cross-platform targets..." -ForegroundColor Yellow
    
    if (-not (Test-Command "task")) {
        Write-Host "! Task not found, skipping Taskfile tests" -ForegroundColor Yellow
        return $true
    }
    
    $targets = @("build-linux", "build-windows", "build-all")
    $success = $true
    
    foreach ($target in $targets) {
        Write-Host "  Testing task $target..."
        
        try {
            & task $target 2>$null | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "  + task $target succeeded" -ForegroundColor Green
            }
            else {
                Write-Host "  X task $target failed" -ForegroundColor Red
                $success = $false
            }
        }
        catch {
            Write-Host "  X task $target failed: $_" -ForegroundColor Red
            $success = $false
        }
    }
    
    if ($success) {
        Write-Host "+ All Taskfile targets succeeded" -ForegroundColor Green
        return $true
    }
    else {
        Write-Host "X Some Taskfile targets failed" -ForegroundColor Red
        return $false
    }
}

# Function to test WSL2 compatibility
function Test-WSL2Compatibility {
    if (-not (Test-WSL)) {
        Write-Host "! Not running in WSL2, skipping WSL2-specific tests" -ForegroundColor Yellow
        return $true
    }
    
    Write-Host "Testing WSL2 compatibility..." -ForegroundColor Yellow
    
    # Test that we can build Linux binaries in WSL2
    $testBin = "bin/test-wsl2"
    
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    
    try {
        Push-Location "cmd/af"
        $relativeBinPath = "../../$testBin"
        
        & go build -o $relativeBinPath . 2>$null
        $buildSuccess = $LASTEXITCODE -eq 0
        Pop-Location
        
        if ($buildSuccess) {
            Write-Host "+ WSL2 Linux build succeeded" -ForegroundColor Green
            
            # Test that the binary exists
            if (Test-Path $testBin) {
                Write-Host "+ WSL2 binary created successfully" -ForegroundColor Green
            }
            else {
                Write-Host "X WSL2 binary not found" -ForegroundColor Red
                return $false
            }
            
            # Clean up
            Remove-Item $testBin -ErrorAction SilentlyContinue
            return $true
        }
        else {
            Write-Host "X WSL2 Linux build failed" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "X WSL2 Linux build failed: $_" -ForegroundColor Red
        return $false
    }
    finally {
        # Reset environment variables
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    }
}

# Function to validate build artifacts
function Test-BuildArtifacts {
    Write-Host "Validating build artifacts..." -ForegroundColor Yellow
    
    $success = $true
    
    # Check Linux binaries
    $linuxDir = Join-Path $BinDir "linux"
    if (Test-Path $linuxDir) {
        foreach ($service in $Services) {
            $binPath = Join-Path $linuxDir $service
            if (Test-Path $binPath) {
                Write-Host "  + Linux $service exists" -ForegroundColor Green
            }
            else {
                Write-Host "  ! Linux $service not found" -ForegroundColor Yellow
            }
        }
    }
    
    # Check Windows binaries
    $windowsDir = Join-Path $BinDir "windows"
    if (Test-Path $windowsDir) {
        foreach ($service in $Services) {
            $binPath = Join-Path $windowsDir "$service.exe"
            if (Test-Path $binPath) {
                Write-Host "  + Windows $service.exe exists" -ForegroundColor Green
            }
            else {
                Write-Host "  ! Windows $service.exe not found" -ForegroundColor Yellow
            }
        }
    }
    
    if ($success) {
        Write-Host "+ Build artifact validation passed" -ForegroundColor Green
        return $true
    }
    else {
        Write-Host "X Build artifact validation failed" -ForegroundColor Red
        return $false
    }
}

# Function to test Go module compatibility
function Test-GoModules {
    Write-Host "Testing Go module compatibility..." -ForegroundColor Yellow
    
    $modules = @("cmd/control-plane", "cmd/worker", "cmd/af", "sdk/go")
    $success = $true
    
    # Test root module with go mod tidy instead of build
    Write-Host "  Testing root module dependencies..."
    try {
        & go mod tidy 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  + Root module dependencies are valid" -ForegroundColor Green
        }
        else {
            Write-Host "  X Root module dependencies failed" -ForegroundColor Red
            $success = $false
        }
    }
    catch {
        Write-Host "  X Root module dependencies failed: $_" -ForegroundColor Red
        $success = $false
    }
    
    foreach ($module in $modules) {
        if (Test-Path (Join-Path $module "go.mod")) {
            Write-Host "  Testing module: $module"
            
            try {
                Push-Location $module
                & go build . 2>$null | Out-Null
                
                if ($LASTEXITCODE -eq 0) {
                    Write-Host "  + Module $module builds successfully" -ForegroundColor Green
                }
                else {
                    Write-Host "  X Module $module build failed" -ForegroundColor Red
                    $success = $false
                }
            }
            catch {
                Write-Host "  X Module $module build failed: $_" -ForegroundColor Red
                $success = $false
            }
            finally {
                Pop-Location
            }
        }
    }
    
    if ($success) {
        Write-Host "+ All Go modules build successfully" -ForegroundColor Green
        return $true
    }
    else {
        Write-Host "X Some Go modules failed to build" -ForegroundColor Red
        return $false
    }
}

# Main test execution
Write-Host "Current platform: $(Get-PlatformInfo)" -ForegroundColor Blue
Write-Host "Go version: $(go version)" -ForegroundColor Blue
Write-Host ""

# Test Go modules first
if (-not (Test-GoModules)) {
    $ExitCode = 1
}
Write-Host ""

# Test cross-platform builds
foreach ($platform in $Platforms) {
    if (-not (Test-CrossPlatformBuild $platform)) {
        $ExitCode = 1
    }
    Write-Host ""
}

# Test build tools
if (-not (Test-MakefileTargets)) {
    $ExitCode = 1
}
Write-Host ""

if (-not (Test-TaskfileTargets)) {
    $ExitCode = 1
}
Write-Host ""

# Test WSL2 compatibility if applicable
if (-not (Test-WSL2Compatibility)) {
    $ExitCode = 1
}
Write-Host ""

# Validate build artifacts
if (-not (Test-BuildArtifacts)) {
    $ExitCode = 1
}
Write-Host ""

if ($ExitCode -eq 0) {
    Write-Host "All cross-platform build tests passed!" -ForegroundColor Green
}
else {
    Write-Host "X Some cross-platform build tests failed" -ForegroundColor Red
}

exit $ExitCode