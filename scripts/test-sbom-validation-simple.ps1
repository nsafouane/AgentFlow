# Simple SBOM Validation Test (PowerShell)
# Tests SBOM structure validation without requiring external tools

param(
    [switch]$Help
)

# Colors for output
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
}

# Logging functions
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
    Write-Host "Usage: .\test-sbom-validation-simple.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Parameters:"
    Write-Host "  -Help               Show this help message"
    Write-Host ""
    Write-Host "This script validates SBOM structure without requiring external tools."
}

# Test if JSON is valid
function Test-JsonValid {
    param([string]$FilePath)
    
    try {
        $null = Get-Content $FilePath -Raw | ConvertFrom-Json
        return $true
    }
    catch {
        return $false
    }
}

# Validate SPDX SBOM structure
function Test-SpdxSbomStructure {
    param([string]$SbomFile)
    
    Write-Info "Validating SPDX SBOM structure: $SbomFile"
    
    if (-not (Test-Path $SbomFile)) {
        Write-Error "SBOM file not found: $SbomFile"
        return $false
    }
    
    # Check if file is valid JSON
    if (-not (Test-JsonValid $SbomFile)) {
        Write-Error "SBOM file is not valid JSON: $SbomFile"
        return $false
    }
    
    $sbomContent = Get-Content $SbomFile -Raw | ConvertFrom-Json
    
    # Validate SPDX structure
    if (-not $sbomContent.spdxVersion) {
        Write-Error "SPDX SBOM missing spdxVersion field"
        return $false
    }
    
    if (-not $sbomContent.packages -or $sbomContent.packages.Count -eq 0) {
        Write-Error "SPDX SBOM contains no packages"
        return $false
    }
    
    if (-not $sbomContent.SPDXID) {
        Write-Error "SPDX SBOM missing SPDXID field"
        return $false
    }
    
    if (-not $sbomContent.documentNamespace) {
        Write-Error "SPDX SBOM missing documentNamespace field"
        return $false
    }
    
    Write-Success "SPDX SBOM structure is valid"
    Write-Info "  - SPDX Version: $($sbomContent.spdxVersion)"
    Write-Info "  - Packages: $($sbomContent.packages.Count)"
    Write-Info "  - Files: $($sbomContent.files.Count)"
    Write-Info "  - Relationships: $($sbomContent.relationships.Count)"
    
    # Show sample packages
    Write-Info "Sample packages:"
    $sampleCount = [Math]::Min(3, $sbomContent.packages.Count)
    for ($i = 0; $i -lt $sampleCount; $i++) {
        $pkg = $sbomContent.packages[$i]
        $version = if ($pkg.versionInfo) { $pkg.versionInfo } else { "no version" }
        Write-Info "  - $($pkg.name) ($version)"
    }
    
    return $true
}

# Test local SBOM files
function Test-LocalSbomFiles {
    Write-Info "Testing local SBOM files..."
    
    $sbomFiles = @()
    
    # Look for SBOM files in common locations
    if (Test-Path "af-sbom.spdx.json") {
        $sbomFiles += "af-sbom.spdx.json"
        Write-Info "Found: af-sbom.spdx.json"
    }
    
    if (Test-Path "sbom.spdx.json") {
        $sbomFiles += "sbom.spdx.json"
        Write-Info "Found: sbom.spdx.json"
    }
    
    if (Test-Path "sbom.cyclonedx.json") {
        $sbomFiles += "sbom.cyclonedx.json"
        Write-Info "Found: sbom.cyclonedx.json"
    }
    
    # Look in artifacts directory
    if (Test-Path "artifacts") {
        $artifactFiles = Get-ChildItem -Path "artifacts" -Filter "*.json" -Recurse -ErrorAction SilentlyContinue
        foreach ($file in $artifactFiles) {
            if ($file.Name -like "*.spdx.json" -or $file.Name -like "*.cyclonedx.json") {
                $sbomFiles += $file.FullName
                Write-Info "Found: $($file.FullName)"
            }
        }
    }
    
    if ($sbomFiles.Count -eq 0) {
        Write-Warning "No local SBOM files found"
        return $true
    }
    
    $validationFailed = $false
    foreach ($sbomFile in $sbomFiles) {
        if ($sbomFile -like "*.spdx.json") {
            if (-not (Test-SpdxSbomStructure $sbomFile)) {
                $validationFailed = $true
            }
        } else {
            Write-Info "Skipping non-SPDX SBOM file: $sbomFile"
        }
        Write-Host ""
    }
    
    if ($validationFailed) {
        Write-Error "Local SBOM validation failed"
        return $false
    }
    
    Write-Success "Local SBOM validation completed successfully"
    return $true
}

# Test CI/CD workflow configuration
function Test-CicdConfiguration {
    Write-Info "Testing CI/CD workflow configuration..."
    
    $workflowsDir = ".github/workflows"
    
    if (-not (Test-Path $workflowsDir)) {
        Write-Error "GitHub Actions workflows directory not found"
        return $false
    }
    
    $workflowFiles = @("ci.yml", "container-build.yml", "release.yml")
    $foundSbom = $false
    $foundProvenance = $false
    $foundSigning = $false
    
    foreach ($workflow in $workflowFiles) {
        $workflowFile = Join-Path $workflowsDir $workflow
        
        if (Test-Path $workflowFile) {
            Write-Info "Checking workflow: $workflow"
            $content = Get-Content $workflowFile -Raw
            
            # Check for SBOM generation
            if ($content -match "sbom.*true|syft") {
                Write-Success "SBOM generation found in $workflow"
                $foundSbom = $true
            }
            
            # Check for provenance
            if ($content -match "provenance.*true|attestation") {
                Write-Success "Provenance attestation found in $workflow"
                $foundProvenance = $true
            }
            
            # Check for cosign signing
            if ($content -match "cosign") {
                Write-Success "Cosign signing found in $workflow"
                $foundSigning = $true
            }
        } else {
            Write-Warning "Workflow file not found: $workflowFile"
        }
    }
    
    $success = $foundSbom -and $foundProvenance -and $foundSigning
    
    if ($success) {
        Write-Success "CI/CD configuration validation completed successfully"
    } else {
        Write-Error "CI/CD configuration validation failed"
        if (-not $foundSbom) { Write-Error "SBOM generation not found in workflows" }
        if (-not $foundProvenance) { Write-Error "Provenance attestation not found in workflows" }
        if (-not $foundSigning) { Write-Error "Cosign signing not found in workflows" }
    }
    
    return $success
}

# Main function
function Invoke-SimpleSbomValidation {
    Write-Info "Starting simple SBOM validation (no external tools required)"
    Write-Host ""
    
    $allPassed = $true
    
    # Test local SBOM files
    if (-not (Test-LocalSbomFiles)) {
        $allPassed = $false
    }
    
    Write-Host ""
    
    # Test CI/CD configuration
    if (-not (Test-CicdConfiguration)) {
        $allPassed = $false
    }
    
    Write-Host ""
    
    if ($allPassed) {
        Write-Success "All simple SBOM validation tests passed!"
        return $true
    } else {
        Write-Error "Some simple SBOM validation tests failed"
        return $false
    }
}

# Handle help parameter
if ($Help) {
    Show-Usage
    exit 0
}

# Run validation
$success = Invoke-SimpleSbomValidation

if (-not $success) {
    exit 1
}

exit 0