# Workflow Validation Script (PowerShell)
# This script validates GitHub Actions workflow files for syntax and best practices

param(
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

# Get script directory and repo root
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$WorkflowsDir = Join-Path $RepoRoot ".github\workflows"

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

# Check if required tools are installed
function Test-Dependencies {
    $missingDeps = @()
    
    try {
        $null = Get-Command "yq" -ErrorAction Stop
    } catch {
        $missingDeps += "yq"
    }
    
    if ($missingDeps.Count -gt 0) {
        Write-Error-Custom "Missing required dependencies: $($missingDeps -join ', ')"
        Write-Info "Install yq from: https://github.com/mikefarah/yq/releases"
        exit 1
    }
}

# Validate YAML syntax using PowerShell YAML module or yq
function Test-YamlSyntax {
    param([string]$FilePath)
    
    $filename = Split-Path -Leaf $FilePath
    Write-Info "Validating YAML syntax for $filename"
    
    try {
        # Use yq to validate YAML syntax
        $result = & yq eval '.' $FilePath 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error-Custom "YAML syntax validation failed for $filename"
            return $false
        }
        return $true
    } catch {
        Write-Error-Custom "Failed to validate YAML syntax for $filename`: $_"
        return $false
    }
}

# Validate workflow structure
function Test-WorkflowStructure {
    param([string]$FilePath)
    
    $filename = Split-Path -Leaf $FilePath
    Write-Info "Validating workflow structure for $filename"
    
    # Check required top-level keys
    $requiredKeys = @("name", "on", "jobs")
    foreach ($key in $requiredKeys) {
        $hasKey = & yq eval "has(`"$key`")" $FilePath
        if ($hasKey -ne "true") {
            Write-Error-Custom "Missing required key '$key' in $filename"
            return $false
        }
    }
    
    # Check that jobs exist
    $jobCount = & yq eval '.jobs | length' $FilePath
    if ([int]$jobCount -eq 0) {
        Write-Error-Custom "No jobs defined in $filename"
        return $false
    }
    
    # Check job structure
    $jobs = & yq eval '.jobs | keys | .[]' $FilePath
    foreach ($job in $jobs) {
        $hasRunsOn = & yq eval ".jobs.$job | has(`"runs-on`")" $FilePath
        if ($hasRunsOn -ne "true") {
            Write-Error-Custom "Job '$job' missing 'runs-on' in $filename"
            return $false
        }
        
        $hasSteps = & yq eval ".jobs.$job | has(`"steps`")" $FilePath
        if ($hasSteps -ne "true") {
            Write-Error-Custom "Job '$job' missing 'steps' in $filename"
            return $false
        }
    }
    
    return $true
}

# Validate security best practices
function Test-SecurityPractices {
    param([string]$FilePath)
    
    $filename = Split-Path -Leaf $FilePath
    Write-Info "Validating security practices for $filename"
    
    $content = Get-Content $FilePath -Raw
    
    # Check for potential hardcoded secrets (excluding proper secret references)
    if ($content -match "(?i)(password|secret|token)" -and 
        $content -notmatch "secrets\." -and 
        $content -notmatch "github\.token") {
        Write-Warn "Potential hardcoded secrets found in $filename"
    }
    
    # Check for proper permissions
    $hasPermissions = & yq eval 'has("permissions")' $FilePath
    if ($hasPermissions -eq "true") {
        Write-Info "Permissions defined in $filename"
    } else {
        Write-Warn "No permissions defined in $filename (consider adding explicit permissions)"
    }
    
    return $true
}

# Validate caching strategy
function Test-Caching {
    param([string]$FilePath)
    
    $filename = Split-Path -Leaf $FilePath
    $content = Get-Content $FilePath -Raw
    
    # Check if Go workflows have proper caching
    if ($content -match "setup-go") {
        if ($content -notmatch "cache: true" -and $content -notmatch "actions/cache") {
            Write-Warn "Go workflow $filename should include caching for better performance"
        }
    }
    
    # Check if Docker workflows have proper caching
    if ($content -match "docker/build-push-action") {
        if ($content -notmatch "cache-from|cache-to") {
            Write-Warn "Docker workflow $filename should include build caching"
        }
    }
}

# Main validation function
function Test-Workflow {
    param([string]$FilePath)
    
    $filename = Split-Path -Leaf $FilePath
    Write-Info "Validating workflow: $filename"
    
    $errors = 0
    
    if (-not (Test-YamlSyntax $FilePath)) { $errors++ }
    if (-not (Test-WorkflowStructure $FilePath)) { $errors++ }
    Test-SecurityPractices $FilePath
    Test-Caching $FilePath
    
    return $errors
}

# Main execution
function Main {
    Write-Info "Starting GitHub Actions workflow validation"
    
    Test-Dependencies
    
    if (-not (Test-Path $WorkflowsDir)) {
        Write-Error-Custom "Workflows directory not found: $WorkflowsDir"
        exit 1
    }
    
    $totalErrors = 0
    $workflowCount = 0
    
    # Validate all workflow files
    $workflowFiles = Get-ChildItem -Path $WorkflowsDir -Filter "*.yml" -File
    $workflowFiles += Get-ChildItem -Path $WorkflowsDir -Filter "*.yaml" -File
    
    foreach ($workflowFile in $workflowFiles) {
        $workflowCount++
        $errors = Test-Workflow $workflowFile.FullName
        $totalErrors += $errors
        Write-Host ""
    }
    
    # Summary
    Write-Info "Validation complete"
    Write-Info "Workflows validated: $workflowCount"
    
    if ($totalErrors -eq 0) {
        Write-Info "All workflows passed validation!"
        exit 0
    } else {
        Write-Error-Custom "Validation failed with $totalErrors errors"
        exit 1
    }
}

# Run main function
Main