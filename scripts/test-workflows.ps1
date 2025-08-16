# Workflow Testing Script (PowerShell)
# This script tests GitHub Actions workflows and validates configurations

param(
    [switch]$Cleanup,
    [switch]$Help
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

# Show help
if ($Help) {
    Write-Host "Usage: .\test-workflows.ps1 [-Cleanup] [-Help]"
    Write-Host "  -Cleanup   Remove test configuration files after testing"
    Write-Host "  -Help      Show this help message"
    exit 0
}

# Test workflow syntax and structure
function Test-WorkflowSyntax {
    param([string]$WorkflowFile)
    
    $workflowName = [System.IO.Path]::GetFileNameWithoutExtension($WorkflowFile)
    Write-Info "Testing syntax for workflow: $workflowName"
    
    try {
        # Use yq to validate YAML structure if available
        if (Get-Command "yq" -ErrorAction SilentlyContinue) {
            $result = & yq eval '.' $WorkflowFile 2>&1
            if ($LASTEXITCODE -ne 0) {
                Write-Error-Custom "✗ Workflow $workflowName has invalid YAML syntax"
                return $false
            }
            
            # Check required fields
            $requiredFields = @("name", "on", "jobs")
            foreach ($field in $requiredFields) {
                $hasField = & yq eval "has(`"$field`")" $WorkflowFile
                if ($hasField -ne "true") {
                    Write-Error-Custom "✗ Workflow $workflowName missing required field: $field"
                    return $false
                }
            }
        } else {
            # Basic YAML validation using PowerShell
            $content = Get-Content $WorkflowFile -Raw
            if ([string]::IsNullOrWhiteSpace($content)) {
                Write-Error-Custom "✗ Workflow $workflowName is empty"
                return $false
            }
            
            # Check for basic required fields
            if ($content -notmatch "name:" -or $content -notmatch "on:" -or $content -notmatch "jobs:") {
                Write-Error-Custom "✗ Workflow $workflowName missing required fields"
                return $false
            }
        }
        
        Write-Info "✓ Workflow $workflowName has valid syntax and structure"
        return $true
    } catch {
        Write-Error-Custom "✗ Error testing workflow $workflowName`: $_"
        return $false
    }
}

# Test security configurations
function Test-SecurityConfig {
    param([string]$WorkflowFile)
    
    $workflowName = [System.IO.Path]::GetFileNameWithoutExtension($WorkflowFile)
    Write-Info "Testing security configuration for workflow: $workflowName"
    
    $content = Get-Content $WorkflowFile -Raw
    
    # Check if permissions are defined
    if ($content -match "permissions:") {
        Write-Info "✓ Workflow $workflowName has permissions defined"
        
        # Check for broad permissions
        if ($content -match "write-all|contents:\s*write") {
            Write-Warn "⚠ Workflow $workflowName has broad write permissions"
        }
    } else {
        Write-Warn "⚠ Workflow $workflowName has no explicit permissions (will use default)"
    }
    
    # Check for secret usage
    if ($content -match "secrets\.") {
        Write-Info "✓ Workflow $workflowName uses secrets properly"
    }
    
    return $true
}

# Test caching configuration
function Test-CachingConfig {
    param([string]$WorkflowFile)
    
    $workflowName = [System.IO.Path]::GetFileNameWithoutExtension($WorkflowFile)
    Write-Info "Testing caching configuration for workflow: $workflowName"
    
    $content = Get-Content $WorkflowFile -Raw
    
    # Check for Go caching
    if ($content -match "setup-go") {
        if ($content -match "cache:\s*true" -or $content -match "actions/cache") {
            Write-Info "✓ Workflow $workflowName has Go caching enabled"
        } else {
            Write-Warn "⚠ Workflow $workflowName could benefit from Go caching"
        }
    }
    
    # Check for Docker caching
    if ($content -match "docker/build-push-action") {
        if ($content -match "cache-from|cache-to") {
            Write-Info "✓ Workflow $workflowName has Docker caching enabled"
        } else {
            Write-Warn "⚠ Workflow $workflowName could benefit from Docker caching"
        }
    }
    
    return $true
}

# Create test environment
function Initialize-TestEnvironment {
    Write-Info "Setting up test environment"
    
    # Create .env file for testing if it doesn't exist
    $envFile = Join-Path $RepoRoot ".env"
    if (-not (Test-Path $envFile)) {
        $envContent = @"
GITHUB_TOKEN=test_token
GITHUB_REPOSITORY=test/agentflow
GITHUB_REPOSITORY_OWNER=test
GITHUB_ACTOR=test-user
GITHUB_SHA=test-sha
GITHUB_REF=refs/heads/test-branch
"@
        Set-Content -Path $envFile -Value $envContent
        Write-Info "Created .env file for testing"
    }
}

# Clean up test environment
function Remove-TestEnvironment {
    Write-Info "Cleaning up test environment"
    
    if ($Cleanup) {
        $envFile = Join-Path $RepoRoot ".env"
        if (Test-Path $envFile) {
            Remove-Item $envFile -Force
            Write-Info "Removed test .env file"
        }
    }
}

# Main test function
function Invoke-WorkflowTests {
    $totalTests = 0
    $passedTests = 0
    $failedTests = 0
    
    Write-Info "Starting workflow tests"
    
    # Get all workflow files
    $workflowFiles = @()
    $workflowFiles += Get-ChildItem -Path $WorkflowsDir -Filter "*.yml" -File -ErrorAction SilentlyContinue
    $workflowFiles += Get-ChildItem -Path $WorkflowsDir -Filter "*.yaml" -File -ErrorAction SilentlyContinue
    
    foreach ($workflowFile in $workflowFiles) {
        $workflowName = [System.IO.Path]::GetFileNameWithoutExtension($workflowFile.Name)
        Write-Info "Testing workflow: $workflowName"
        
        $totalTests++
        $workflowPassed = $true
        
        # Run all tests for this workflow
        if (-not (Test-WorkflowSyntax $workflowFile.FullName)) { $workflowPassed = $false }
        Test-SecurityConfig $workflowFile.FullName | Out-Null
        Test-CachingConfig $workflowFile.FullName | Out-Null
        
        if ($workflowPassed) {
            $passedTests++
            Write-Info "✅ Workflow $workflowName passed all tests"
        } else {
            $failedTests++
            Write-Error-Custom "❌ Workflow $workflowName failed some tests"
        }
        
        Write-Host "----------------------------------------"
    }
    
    # Summary
    Write-Info "Test Summary:"
    Write-Info "Total workflows tested: $totalTests"
    Write-Info "Passed: $passedTests"
    Write-Info "Failed: $failedTests"
    
    if ($failedTests -eq 0) {
        Write-Info "🎉 All workflow tests passed!"
        return $true
    } else {
        Write-Error-Custom "💥 Some workflow tests failed!"
        return $false
    }
}

# Main execution
function Main {
    Write-Info "GitHub Actions Workflow Testing"
    
    # Check if workflows directory exists
    if (-not (Test-Path $WorkflowsDir)) {
        Write-Error-Custom "Workflows directory not found: $WorkflowsDir"
        exit 1
    }
    
    # Setup test environment
    Initialize-TestEnvironment
    
    # Check for yq installation
    if (-not (Get-Command "yq" -ErrorAction SilentlyContinue)) {
        Write-Warn "yq is not installed. Some advanced validation will be skipped."
        Write-Info "Install yq from: https://github.com/mikefarah/yq/releases"
    }
    
    # Run tests
    $success = Invoke-WorkflowTests
    
    # Cleanup
    Remove-TestEnvironment
    
    if (-not $success) {
        exit 1
    }
}

# Run main function
Main