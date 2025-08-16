# Security Tools Validation Script (PowerShell)
# Validates that security tools are properly configured and working

param(
    [switch]$Verbose,
    [switch]$Help
)

# Get script and repository root directories
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir

# Validation results
$ValidationsPassed = 0
$ValidationsFailed = 0

# Logging functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Check if command exists
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

# Validate a tool
function Test-Tool {
    param(
        [string]$ToolName,
        [string]$ToolCommand,
        [string]$ExpectedOutput = ""
    )
    
    Write-Info "Validating $ToolName..."
    
    if (Test-Command $ToolCommand) {
        Write-Info "✓ $ToolName is available"
        $script:ValidationsPassed++
    }
    else {
        Write-Error-Custom "✗ $ToolName is not available"
        $script:ValidationsFailed++
    }
}

# Validate configuration files
function Test-Config {
    param(
        [string]$ConfigFile,
        [string]$Description
    )
    
    Write-Info "Validating $Description..."
    
    $FullPath = Join-Path $RepoRoot $ConfigFile
    if (Test-Path $FullPath) {
        Write-Info "✓ $Description exists: $ConfigFile"
        $script:ValidationsPassed++
    }
    else {
        Write-Error-Custom "✗ $Description missing: $ConfigFile"
        $script:ValidationsFailed++
    }
}

# Validate scripts
function Test-Script {
    param(
        [string]$ScriptFile,
        [string]$Description
    )
    
    Write-Info "Validating $Description..."
    
    $FullPath = Join-Path $RepoRoot $ScriptFile
    if (Test-Path $FullPath) {
        Write-Info "✓ $Description exists: $ScriptFile"
        $script:ValidationsPassed++
    }
    else {
        Write-Error-Custom "✗ $Description missing: $ScriptFile"
        $script:ValidationsFailed++
    }
}

# Show usage information
function Show-Usage {
    Write-Host "Usage: .\validate-security-tools.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Security tools validation script."
    Write-Host ""
    Write-Host "OPTIONS:"
    Write-Host "    -Verbose                 Enable verbose output"
    Write-Host "    -Help                    Show this help message"
    Write-Host ""
    Write-Host "EXAMPLES:"
    Write-Host "    .\validate-security-tools.ps1           # Run validation"
    Write-Host "    .\validate-security-tools.ps1 -Verbose  # Run with verbose output"
}

# Main validation
function Main {
    if ($Help) {
        Show-Usage
        exit 0
    }
    
    Write-Info "Starting security tools validation"
    Write-Host ""
    
    # Validate core Go tools
    Test-Tool "Go compiler" "go" "go version"
    
    # Validate security tools (these may not be installed yet)
    Test-Tool "gosec" "gosec" ""
    Test-Tool "govulncheck" "govulncheck" ""
    Test-Tool "gitleaks" "gitleaks" ""
    Test-Tool "syft" "syft" ""
    Test-Tool "grype" "grype" ""
    
    Write-Host ""
    
    # Validate configuration files
    Test-Config ".security-config.yml" "Security configuration"
    Test-Config "docs/security-baseline.md" "Security baseline documentation"
    
    Write-Host ""
    
    # Validate scripts
    Test-Script "scripts/security-scan.sh" "Security scan script (Bash)"
    Test-Script "scripts/security-scan.ps1" "Security scan script (PowerShell)"
    Test-Script "scripts/test-security-failure-enhanced.sh" "Security failure test script"
    Test-Script "scripts/security_test.go" "Security unit tests"
    Test-Script "scripts/validate-security-tools.sh" "Security validation script (Bash)"
    Test-Script "scripts/validate-security-tools.ps1" "Security validation script (PowerShell)"
    
    Write-Host ""
    
    # Test unit tests
    Write-Info "Running security unit tests..."
    Set-Location $ScriptDir
    
    $TestOutput = go test -v ./security_test.go 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Info "✓ Security unit tests pass"
        $script:ValidationsPassed++
    }
    else {
        Write-Error-Custom "✗ Security unit tests fail"
        if ($Verbose) {
            Write-Host $TestOutput
        }
        $script:ValidationsFailed++
    }
    
    Write-Host ""
    
    # Print summary
    Write-Info "=== Validation Summary ==="
    Write-Info "Validations passed: $ValidationsPassed"
    Write-Info "Validations failed: $ValidationsFailed"
    Write-Info "Total validations: $($ValidationsPassed + $ValidationsFailed)"
    
    if ($ValidationsFailed -eq 0) {
        Write-Host ""
        Write-Info "All validations passed! Security tooling is properly configured."
        exit 0
    }
    else {
        Write-Host ""
        Write-Error-Custom "Some validations failed. Please review and fix the issues."
        exit 1
    }
}

# Execute main function
Main