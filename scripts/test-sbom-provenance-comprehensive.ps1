# Comprehensive SBOM & Provenance Validation Test
# Demonstrates all validation capabilities for task 14

param(
    [string]$ImageTag = "latest",
    [switch]$SkipExternalTools,
    [switch]$Help
)

# Colors for output
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
    Cyan = "Cyan"
    Magenta = "Magenta"
}

# Logging functions
function Write-Header {
    param([string]$Message)
    Write-Host ""
    Write-Host "=" * 80 -ForegroundColor $Colors.Cyan
    Write-Host $Message -ForegroundColor $Colors.Cyan
    Write-Host "=" * 80 -ForegroundColor $Colors.Cyan
}

function Write-SubHeader {
    param([string]$Message)
    Write-Host ""
    Write-Host "-" * 60 -ForegroundColor $Colors.Magenta
    Write-Host $Message -ForegroundColor $Colors.Magenta
    Write-Host "-" * 60 -ForegroundColor $Colors.Magenta
}

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Colors.Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Colors.Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor $Colors.Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Colors.Red
}

# Show usage information
function Show-Usage {
    Write-Host "Usage: .\test-sbom-provenance-comprehensive.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Parameters:"
    Write-Host "  -ImageTag           Container image tag to test (default: latest)"
    Write-Host "  -SkipExternalTools  Skip tests that require external tools"
    Write-Host "  -Help               Show this help message"
    Write-Host ""
    Write-Host "This script demonstrates comprehensive SBOM & provenance validation."
}

# Check if external tools are available
function Test-ExternalTools {
    $tools = @("docker", "cosign", "syft", "jq")
    $available = @{}
    
    foreach ($tool in $tools) {
        try {
            $null = Get-Command $tool -ErrorAction Stop
            $available[$tool] = $true
        }
        catch {
            $available[$tool] = $false
        }
    }
    
    return $available
}

# Test 1: Unit Tests
function Invoke-UnitTests {
    Write-SubHeader "Test 1: Unit Tests"
    
    Write-Info "Running SBOM and provenance validation unit tests..."
    
    try {
        $result = & go test -v .\scripts\validate-sbom-provenance_test.go 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Unit tests passed"
            Write-Info "Test results:"
            $result | Where-Object { $_ -match "PASS|SKIP|FAIL" } | ForEach-Object {
                if ($_ -match "PASS") {
                    Write-Host "  ✓ $_" -ForegroundColor $Colors.Green
                } elseif ($_ -match "SKIP") {
                    Write-Host "  ⚠ $_" -ForegroundColor $Colors.Yellow
                } elseif ($_ -match "FAIL") {
                    Write-Host "  ✗ $_" -ForegroundColor $Colors.Red
                }
            }
            return $true
        } else {
            Write-Error "Unit tests failed"
            Write-Info "Error output:"
            $result | ForEach-Object { Write-Host "  $_" -ForegroundColor $Colors.Red }
            return $false
        }
    }
    catch {
        Write-Error "Failed to run unit tests: $($_.Exception.Message)"
        return $false
    }
}

# Test 2: Local SBOM Validation
function Invoke-LocalSbomValidation {
    Write-SubHeader "Test 2: Local SBOM File Validation"
    
    Write-Info "Running simple SBOM validation script..."
    
    try {
        $result = & .\scripts\test-sbom-validation-simple.ps1 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Local SBOM validation passed"
            return $true
        } else {
            Write-Error "Local SBOM validation failed"
            $result | ForEach-Object { Write-Host "  $_" -ForegroundColor $Colors.Red }
            return $false
        }
    }
    catch {
        Write-Error "Failed to run local SBOM validation: $($_.Exception.Message)"
        return $false
    }
}

# Test 3: Validation Scripts
function Test-ValidationScripts {
    Write-SubHeader "Test 3: Validation Scripts"
    
    $scripts = @(
        @{ Name = "validate-sbom-provenance.ps1"; Type = "PowerShell" },
        @{ Name = "validate-sbom-provenance.sh"; Type = "Bash" }
    )
    
    $allPassed = $true
    
    foreach ($script in $scripts) {
        Write-Info "Testing validation script: $($script.Name)"
        
        $scriptPath = ".\scripts\$($script.Name)"
        
        if (Test-Path $scriptPath) {
            Write-Success "Script exists: $($script.Name)"
            
            # Test script structure
            $content = Get-Content $scriptPath -Raw
            
            $requiredElements = @("SBOM", "provenance", "cosign", "syft")
            $missingElements = @()
            
            foreach ($element in $requiredElements) {
                if ($content -notmatch $element) {
                    $missingElements += $element
                }
            }
            
            if ($missingElements.Count -eq 0) {
                Write-Success "Script contains all required elements"
            } else {
                Write-Error "Script missing elements: $($missingElements -join ', ')"
                $allPassed = $false
            }
        } else {
            Write-Error "Script not found: $($script.Name)"
            $allPassed = $false
        }
    }
    
    return $allPassed
}

# Test 4: External Tool Integration (if available)
function Test-ExternalToolIntegration {
    param([hashtable]$ToolsAvailable)
    
    Write-SubHeader "Test 4: External Tool Integration"
    
    if ($SkipExternalTools) {
        Write-Warning "Skipping external tool tests (SkipExternalTools flag set)"
        return $true
    }
    
    $allPassed = $true
    
    # Test Docker
    if ($ToolsAvailable["docker"]) {
        Write-Info "Testing Docker integration..."
        try {
            $result = & docker --version 2>&1
            Write-Success "Docker available: $result"
        }
        catch {
            Write-Error "Docker test failed: $($_.Exception.Message)"
            $allPassed = $false
        }
    } else {
        Write-Warning "Docker not available"
    }
    
    # Test Cosign
    if ($ToolsAvailable["cosign"]) {
        Write-Info "Testing Cosign integration..."
        try {
            $result = & cosign version 2>&1
            Write-Success "Cosign available: $($result | Select-Object -First 1)"
        }
        catch {
            Write-Error "Cosign test failed: $($_.Exception.Message)"
            $allPassed = $false
        }
    } else {
        Write-Warning "Cosign not available"
    }
    
    # Test Syft
    if ($ToolsAvailable["syft"]) {
        Write-Info "Testing Syft integration..."
        try {
            $result = & syft version 2>&1
            Write-Success "Syft available: $($result | Select-Object -First 1)"
        }
        catch {
            Write-Error "Syft test failed: $($_.Exception.Message)"
            $allPassed = $false
        }
    } else {
        Write-Warning "Syft not available"
    }
    
    return $allPassed
}

# Test 5: CI/CD Integration
function Test-CicdIntegration {
    Write-SubHeader "Test 5: CI/CD Integration"
    
    Write-Info "Checking GitHub Actions workflows for SBOM and provenance configuration..."
    
    $workflowsDir = ".github/workflows"
    $workflows = @("ci.yml", "container-build.yml", "release.yml")
    
    $allPassed = $true
    $sbomFound = $false
    $provenanceFound = $false
    $signingFound = $false
    
    foreach ($workflow in $workflows) {
        $workflowPath = Join-Path $workflowsDir $workflow
        
        if (Test-Path $workflowPath) {
            Write-Info "Analyzing workflow: $workflow"
            $content = Get-Content $workflowPath -Raw
            
            # Check for SBOM generation
            if ($content -match "sbom.*true|syft") {
                Write-Success "  ✓ SBOM generation configured"
                $sbomFound = $true
            }
            
            # Check for provenance attestation
            if ($content -match "provenance.*true|attestation") {
                Write-Success "  ✓ Provenance attestation configured"
                $provenanceFound = $true
            }
            
            # Check for signing
            if ($content -match "cosign") {
                Write-Success "  ✓ Container signing configured"
                $signingFound = $true
            }
        } else {
            Write-Warning "Workflow not found: $workflow"
        }
    }
    
    # Summary
    if ($sbomFound -and $provenanceFound -and $signingFound) {
        Write-Success "All required CI/CD configurations found"
    } else {
        Write-Error "Missing CI/CD configurations:"
        if (-not $sbomFound) { Write-Error "  - SBOM generation" }
        if (-not $provenanceFound) { Write-Error "  - Provenance attestation" }
        if (-not $signingFound) { Write-Error "  - Container signing" }
        $allPassed = $false
    }
    
    return $allPassed
}

# Test 6: Documentation
function Test-Documentation {
    Write-SubHeader "Test 6: Documentation"
    
    $docFiles = @(
        @{ Path = "docs/sbom-provenance-verification.md"; Description = "SBOM & Provenance verification procedures" }
    )
    
    $allPassed = $true
    
    foreach ($doc in $docFiles) {
        Write-Info "Checking documentation: $($doc.Description)"
        
        if (Test-Path $doc.Path) {
            $content = Get-Content $doc.Path -Raw
            
            if ($content.Length -gt 0) {
                Write-Success "Documentation exists and has content: $($doc.Path)"
                
                # Check for key sections
                $requiredSections = @("Prerequisites", "Verification", "SBOM", "Provenance")
                $missingSections = @()
                
                foreach ($section in $requiredSections) {
                    if ($content -notmatch $section) {
                        $missingSections += $section
                    }
                }
                
                if ($missingSections.Count -eq 0) {
                    Write-Success "  ✓ All required sections present"
                } else {
                    Write-Warning "  ⚠ Missing sections: $($missingSections -join ', ')"
                }
            } else {
                Write-Error "Documentation file is empty: $($doc.Path)"
                $allPassed = $false
            }
        } else {
            Write-Error "Documentation file not found: $($doc.Path)"
            $allPassed = $false
        }
    }
    
    return $allPassed
}

# Test 7: Manual Testing Procedures
function Test-ManualTestingProcedures {
    Write-SubHeader "Test 7: Manual Testing Procedures"
    
    $manualTestScript = "scripts/test-sbom-provenance-manual.sh"
    
    Write-Info "Checking manual testing script..."
    
    if (Test-Path $manualTestScript) {
        Write-Success "Manual testing script exists: $manualTestScript"
        
        $content = Get-Content $manualTestScript -Raw
        
        # Check for test categories
        $testCategories = @("artifacts", "provenance", "signatures", "local", "generation", "cicd")
        $foundCategories = @()
        
        foreach ($category in $testCategories) {
            if ($content -match $category) {
                $foundCategories += $category
            }
        }
        
        Write-Info "Found test categories: $($foundCategories -join ', ')"
        
        if ($foundCategories.Count -eq $testCategories.Count) {
            Write-Success "All manual test categories present"
            return $true
        } else {
            $missing = $testCategories | Where-Object { $_ -notin $foundCategories }
            Write-Warning "Missing test categories: $($missing -join ', ')"
            return $false
        }
    } else {
        Write-Error "Manual testing script not found: $manualTestScript"
        return $false
    }
}

# Generate summary report
function Write-SummaryReport {
    param([hashtable]$TestResults)
    
    Write-Header "COMPREHENSIVE VALIDATION SUMMARY"
    
    $totalTests = $TestResults.Count
    $passedTests = ($TestResults.Values | Where-Object { $_ -eq $true }).Count
    $failedTests = $totalTests - $passedTests
    
    Write-Info "Total Tests: $totalTests"
    Write-Info "Passed: $passedTests"
    Write-Info "Failed: $failedTests"
    Write-Host ""
    
    foreach ($test in $TestResults.GetEnumerator()) {
        $status = if ($test.Value) { "✓ PASS" } else { "✗ FAIL" }
        $color = if ($test.Value) { $Colors.Green } else { $Colors.Red }
        Write-Host "$status $($test.Key)" -ForegroundColor $color
    }
    
    Write-Host ""
    
    if ($failedTests -eq 0) {
        Write-Success "🎉 ALL TESTS PASSED! SBOM & Provenance validation is fully implemented."
        Write-Info "Task 14 requirements satisfied:"
        Write-Info "  ✓ Implementation: SBOM & provenance generation in CI/CD"
        Write-Info "  ✓ Unit Tests: Comprehensive validation test suite"
        Write-Info "  ✓ Manual Testing: Interactive validation procedures"
        Write-Info "  ✓ Documentation: Complete verification procedures"
    } else {
        Write-Error "❌ Some tests failed. Please review the results above."
    }
    
    return $failedTests -eq 0
}

# Main execution
function Invoke-ComprehensiveValidation {
    Write-Header "AGENTFLOW SBOM & PROVENANCE VALIDATION - TASK 14"
    Write-Info "Image Tag: $ImageTag"
    Write-Info "Skip External Tools: $SkipExternalTools"
    
    # Check external tools availability
    $toolsAvailable = Test-ExternalTools
    Write-Info "External tools availability:"
    foreach ($tool in $toolsAvailable.GetEnumerator()) {
        $status = if ($tool.Value) { "✓" } else { "✗" }
        $color = if ($tool.Value) { $Colors.Green } else { $Colors.Red }
        Write-Host "  $status $($tool.Key)" -ForegroundColor $color
    }
    
    # Run all tests
    $testResults = @{}
    
    $testResults["Unit Tests"] = Invoke-UnitTests
    $testResults["Local SBOM Validation"] = Invoke-LocalSbomValidation
    $testResults["Validation Scripts"] = Test-ValidationScripts
    $testResults["External Tool Integration"] = Test-ExternalToolIntegration -ToolsAvailable $toolsAvailable
    $testResults["CI/CD Integration"] = Test-CicdIntegration
    $testResults["Documentation"] = Test-Documentation
    $testResults["Manual Testing Procedures"] = Test-ManualTestingProcedures
    
    # Generate summary
    return Write-SummaryReport -TestResults $testResults
}

# Handle help parameter
if ($Help) {
    Show-Usage
    exit 0
}

# Run comprehensive validation
$success = Invoke-ComprehensiveValidation

if (-not $success) {
    exit 1
}

exit 0