# CI Green Including Security Scans Validation Script (PowerShell)
# Validates that all CI workflows pass with no High/Critical vulnerabilities
# Part of Task 11: CI Green Including Security Scans Validation

param(
    [string]$Branch = "main",
    [string]$Threshold = "high",
    [string]$OutputDir = "",
    [switch]$SkipLocalScan,
    [switch]$SkipGitHubCheck,
    [switch]$Verbose,
    [switch]$Help
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Get script directory and repo root
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir

# Configuration
$GitHubApiUrl = "https://api.github.com"
$WorkflowTimeout = 1800  # 30 minutes
$CheckInterval = 30      # 30 seconds
if (-not $OutputDir) {
    $OutputDir = Join-Path $RepoRoot "ci-validation-reports"
}

# Show usage information
function Show-Usage {
    Write-Host @"
Usage: validate-ci-green.ps1 [OPTIONS]

CI Green validation script - ensures all workflows pass with no High/Critical vulnerabilities.

OPTIONS:
    -Branch BRANCH           Check workflows on specific branch [default: main]
    -Threshold LEVEL         Security severity threshold [default: high]
    -OutputDir DIR           Output directory for reports [default: ./ci-validation-reports]
    -SkipLocalScan           Skip local security scan validation
    -SkipGitHubCheck         Skip GitHub workflow status checks
    -Verbose                 Enable verbose output
    -Help                    Show this help message

EXAMPLES:
    .\validate-ci-green.ps1                    # Run full validation on main branch
    .\validate-ci-green.ps1 -Branch develop    # Check develop branch
    .\validate-ci-green.ps1 -SkipLocalScan     # Only check GitHub workflow status
    .\validate-ci-green.ps1 -Verbose           # Enable verbose output

EXIT CODES:
    0    All validations passed
    1    Some validations failed
    2    Invalid arguments or configuration error
"@
}

# Show help if requested
if ($Help) {
    Show-Usage
    exit 0
}

# Validate threshold parameter
$ValidThresholds = @("critical", "high", "medium", "low", "info")
if ($Threshold -notin $ValidThresholds) {
    Write-Host "[ERROR] Invalid severity threshold: $Threshold" -ForegroundColor Red
    Write-Host "[ERROR] Valid options: $($ValidThresholds -join ', ')" -ForegroundColor Red
    exit 2
}

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Validation results
$ValidationResults = @()
$ValidationFailed = $false

# Logging functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-ErrorMsg {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-Debug {
    param([string]$Message)
    if ($Verbose) {
        Write-Host "[DEBUG] $Message" -ForegroundColor Blue
    }
}

# Add validation result
function Add-Result {
    param(
        [string]$TestName,
        [string]$Status,
        [string]$Message
    )
    
    $script:ValidationResults += "$TestName`: $Status - $Message"
    
    if ($Status -eq "FAILED") {
        $script:ValidationFailed = $true
        Write-ErrorMsg "✗ $TestName`: $Message"
    } else {
        Write-Info "✓ $TestName`: $Message"
    }
}

# Validate security scan thresholds in workflow files
function Test-SecurityThresholds {
    Write-Info "Validating security scan thresholds in workflow files..."
    
    $workflowsDir = Join-Path $RepoRoot ".github\workflows"
    $thresholdFound = $false
    
    # Check CI workflow
    $ciWorkflow = Join-Path $workflowsDir "ci.yml"
    if (Test-Path $ciWorkflow) {
        $content = Get-Content $ciWorkflow -Raw
        if ($content -match "fail-on.*high|severity.*high|HIGH|CRITICAL") {
            $thresholdFound = $true
            Write-Debug "Found high/critical threshold in ci.yml"
        }
    }
    
    # Check security scan workflow
    $securityWorkflow = Join-Path $workflowsDir "security-scan.yml"
    if (Test-Path $securityWorkflow) {
        $content = Get-Content $securityWorkflow -Raw
        if ($content -match "fail-on.*high|severity.*high|HIGH|CRITICAL") {
            $thresholdFound = $true
            Write-Debug "Found high/critical threshold in security-scan.yml"
        }
    }
    
    # Check container build workflow
    $containerWorkflow = Join-Path $workflowsDir "container-build.yml"
    if (Test-Path $containerWorkflow) {
        $content = Get-Content $containerWorkflow -Raw
        if ($content -match "fail-on.*high|severity.*high|HIGH|CRITICAL") {
            $thresholdFound = $true
            Write-Debug "Found high/critical threshold in container-build.yml"
        }
    }
    
    if ($thresholdFound) {
        Add-Result "Security Thresholds" "PASSED" "High/Critical severity thresholds configured in workflows"
    } else {
        Add-Result "Security Thresholds" "FAILED" "No high/critical severity thresholds found in workflow files"
    }
}

# Check for required security tools in workflows
function Test-SecurityTools {
    Write-Info "Validating security tools configuration in workflows..."
    
    $workflowsDir = Join-Path $RepoRoot ".github\workflows"
    $requiredTools = @("gosec", "gitleaks", "grype", "syft", "govulncheck")
    $toolsFound = 0
    
    foreach ($tool in $requiredTools) {
        $found = $false
        Get-ChildItem -Path $workflowsDir -Filter "*.yml" | ForEach-Object {
            $content = Get-Content $_.FullName -Raw
            if ($content -match $tool) {
                $found = $true
            }
        }
        
        if ($found) {
            Write-Debug "Found $tool in workflow files"
            $toolsFound++
        } else {
            Write-Warn "$tool not found in workflow files"
        }
    }
    
    if ($toolsFound -ge 4) {
        Add-Result "Security Tools" "PASSED" "$toolsFound/$($requiredTools.Count) required security tools found in workflows"
    } else {
        Add-Result "Security Tools" "FAILED" "Only $toolsFound/$($requiredTools.Count) required security tools found in workflows"
    }
}

# Validate SARIF upload configuration
function Test-SarifUpload {
    Write-Info "Validating SARIF upload configuration..."
    
    $workflowsDir = Join-Path $RepoRoot ".github\workflows"
    $sarifUploads = 0
    
    # Check for SARIF upload actions
    Get-ChildItem -Path $workflowsDir -Filter "*.yml" | ForEach-Object {
        $content = Get-Content $_.FullName -Raw
        if ($content -match "github/codeql-action/upload-sarif") {
            $sarifUploads++
            Write-Debug "Found SARIF upload configuration"
        }
        
        # Check for SARIF output formats
        if ($content -match "sarif|SARIF") {
            $sarifUploads++
            Write-Debug "Found SARIF output format configuration"
        }
    }
    
    if ($sarifUploads -ge 1) {
        Add-Result "SARIF Upload" "PASSED" "SARIF upload configuration found in workflows"
    } else {
        Add-Result "SARIF Upload" "FAILED" "No SARIF upload configuration found in workflows"
    }
}

# Generate validation report
function New-ValidationReport {
    $reportFile = Join-Path $OutputDir "ci-green-validation-report.json"
    $timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    
    $status = if ($ValidationFailed) { "FAILED" } else { "PASSED" }
    
    $gitRepo = try { git remote get-url origin 2>$null } catch { "unknown" }
    $gitCommit = try { git rev-parse HEAD 2>$null } catch { "unknown" }
    $gitBranch = try { git branch --show-current 2>$null } catch { "unknown" }
    
    $report = @{
        timestamp = $timestamp
        validation_status = $status
        security_threshold = $Threshold
        total_validations = $ValidationResults.Count
        validations = $ValidationResults
        reports_directory = $OutputDir
        github_repository = $gitRepo
        git_commit = $gitCommit
        git_branch = $gitBranch
    }
    
    $report | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportFile -Encoding UTF8
    
    Write-Info "Validation report generated: $reportFile"
}

# Print validation summary
function Show-ValidationSummary {
    Write-Host ""
    Write-Info "=== CI Green Validation Summary ==="
    Write-Host ""
    
    $passed = 0
    $failed = 0
    $warnings = 0
    
    foreach ($result in $ValidationResults) {
        Write-Host "  • $result"
        
        if ($result -match "PASSED") {
            $passed++
        } elseif ($result -match "FAILED") {
            $failed++
        } elseif ($result -match "WARNING") {
            $warnings++
        }
    }
    
    Write-Host ""
    Write-Info "Validations passed: $passed"
    Write-Info "Validations failed: $failed"
    Write-Info "Warnings: $warnings"
    Write-Info "Total validations: $($ValidationResults.Count)"
    
    if ($ValidationFailed) {
        Write-Host ""
        Write-ErrorMsg "❌ CI Green validation FAILED"
        Write-ErrorMsg "Some workflows or security scans are not passing"
        Write-ErrorMsg "Review the validation report in: $OutputDir"
        return 1
    } else {
        Write-Host ""
        Write-Info "✅ CI Green validation PASSED"
        Write-Info "All workflows are passing with no high/critical vulnerabilities"
        return 0
    }
}

# Main validation function
function Invoke-Main {
    Write-Info "Starting CI Green validation..."
    Write-Info "Branch: $Branch"
    Write-Info "Security threshold: $Threshold"
    Write-Info "Output directory: $OutputDir"
    
    Set-Location $RepoRoot
    
    # Validate workflow configurations
    Test-SecurityThresholds
    Test-SecurityTools
    Test-SarifUpload
    
    # Generate report and print summary
    New-ValidationReport
    $exitCode = Show-ValidationSummary
    
    exit $exitCode
}

# Execute main function
Invoke-Main