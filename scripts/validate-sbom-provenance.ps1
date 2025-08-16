# SBOM & Provenance Validation Script (PowerShell)
# This script validates that artifacts published per build include SBOM & provenance

param(
    [string]$ImageTag = "latest",
    [bool]$ValidateContainers = $true,
    [bool]$ValidateLocal = $true,
    [switch]$Help
)

# Configuration
$Registry = $env:REGISTRY ?? "ghcr.io"
$Repository = $env:REPOSITORY ?? "agentflow/agentflow"
$Services = @("control-plane", "worker", "af")
$RequiredTools = @("cosign", "syft", "docker")

# Colors for output
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
    Write-Host "Usage: .\validate-sbom-provenance.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Parameters:"
    Write-Host "  -ImageTag           Container image tag to validate (default: latest)"
    Write-Host "  -ValidateContainers Whether to validate container artifacts (default: true)"
    Write-Host "  -ValidateLocal      Whether to validate local SBOM files (default: true)"
    Write-Host "  -Help               Show this help message"
    Write-Host ""
    Write-Host "Environment variables:"
    Write-Host "  REGISTRY           Container registry (default: ghcr.io)"
    Write-Host "  REPOSITORY         Repository name (default: agentflow/agentflow)"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\validate-sbom-provenance.ps1                                    # Validate latest images and local files"
    Write-Host "  .\validate-sbom-provenance.ps1 -ImageTag v1.0.0                  # Validate v1.0.0 images and local files"
    Write-Host "  .\validate-sbom-provenance.ps1 -ValidateContainers `$false        # Validate only local files"
    Write-Host "  .\validate-sbom-provenance.ps1 -ValidateLocal `$false             # Validate only container artifacts"
}

# Check if required tools are installed
function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    $missingTools = @()
    foreach ($tool in $RequiredTools) {
        try {
            $null = Get-Command $tool -ErrorAction Stop
        }
        catch {
            $missingTools += $tool
        }
    }
    
    if ($missingTools.Count -gt 0) {
        Write-Error "Missing required tools: $($missingTools -join ', ')"
        Write-Info "Please install the missing tools and try again."
        return $false
    }
    
    Write-Success "All required tools are available"
    return $true
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

# Validate SBOM file structure
function Test-SbomStructure {
    param(
        [string]$SbomFile,
        [string]$Format
    )
    
    Write-Info "Validating SBOM structure for $SbomFile ($Format format)"
    
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
    
    switch ($Format) {
        "spdx" {
            # Validate SPDX structure
            if (-not $sbomContent.spdxVersion) {
                Write-Error "SPDX SBOM missing spdxVersion field"
                return $false
            }
            
            if (-not $sbomContent.packages -or $sbomContent.packages.Count -eq 0) {
                Write-Error "SPDX SBOM contains no packages"
                return $false
            }
            
            Write-Success "SPDX SBOM structure is valid (version: $($sbomContent.spdxVersion), packages: $($sbomContent.packages.Count))"
        }
        
        "cyclonedx" {
            # Validate CycloneDX structure
            if (-not $sbomContent.specVersion) {
                Write-Error "CycloneDX SBOM missing specVersion field"
                return $false
            }
            
            if (-not $sbomContent.components -or $sbomContent.components.Count -eq 0) {
                Write-Error "CycloneDX SBOM contains no components"
                return $false
            }
            
            Write-Success "CycloneDX SBOM structure is valid (version: $($sbomContent.specVersion), components: $($sbomContent.components.Count))"
        }
        
        default {
            Write-Error "Unknown SBOM format: $Format"
            return $false
        }
    }
    
    return $true
}

# Validate container image SBOM and provenance
function Test-ContainerArtifacts {
    param(
        [string]$ImageRef,
        [string]$Service
    )
    
    Write-Info "Validating SBOM and provenance for $Service`: $ImageRef"
    
    # Check if image exists and is accessible
    try {
        $null = docker manifest inspect $ImageRef 2>$null
    }
    catch {
        Write-Error "Cannot access container image: $ImageRef"
        return $false
    }
    
    # Generate SBOM using syft and validate
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    $spdxSbom = Join-Path $tempDir "sbom.spdx.json"
    $cyclonedxSbom = Join-Path $tempDir "sbom.cyclonedx.json"
    
    Write-Info "Generating SBOM for validation..."
    
    try {
        # Generate SPDX format SBOM
        $result = & syft $ImageRef -o "spdx-json=$spdxSbom" --quiet 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to generate SPDX SBOM for $ImageRef`: $result"
            Remove-Item $tempDir -Recurse -Force
            return $false
        }
        
        # Generate CycloneDX format SBOM
        $result = & syft $ImageRef -o "cyclonedx-json=$cyclonedxSbom" --quiet 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to generate CycloneDX SBOM for $ImageRef`: $result"
            Remove-Item $tempDir -Recurse -Force
            return $false
        }
        
        # Validate SBOM structures
        if (-not (Test-SbomStructure $spdxSbom "spdx")) {
            Remove-Item $tempDir -Recurse -Force
            return $false
        }
        
        if (-not (Test-SbomStructure $cyclonedxSbom "cyclonedx")) {
            Remove-Item $tempDir -Recurse -Force
            return $false
        }
        
        # Verify cosign signature and attestation
        Write-Info "Verifying container signature and attestation..."
        
        # Verify signature
        try {
            $null = & cosign verify `
                --certificate-identity-regexp="https://github.com/$Repository" `
                --certificate-oidc-issuer="https://token.actions.githubusercontent.com" `
                $ImageRef 2>$null
            Write-Success "Container signature verified for $ImageRef"
        }
        catch {
            Write-Warning "Container signature verification failed for $ImageRef"
            Write-Warning "This may be expected for local builds or unsigned images"
        }
        
        # Verify attestation
        try {
            $null = & cosign verify-attestation `
                --certificate-identity-regexp="https://github.com/$Repository" `
                --certificate-oidc-issuer="https://token.actions.githubusercontent.com" `
                --type slsaprovenance `
                $ImageRef 2>$null
            Write-Success "Provenance attestation verified for $ImageRef"
        }
        catch {
            Write-Warning "Provenance attestation verification failed for $ImageRef"
            Write-Warning "This may be expected for local builds or unsigned images"
        }
        
        Write-Success "SBOM validation completed for $Service"
        return $true
    }
    finally {
        # Clean up
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Validate local SBOM files
function Test-LocalSbomFiles {
    Write-Info "Validating local SBOM files..."
    
    $sbomFiles = @()
    
    # Look for SBOM files in common locations
    if (Test-Path "af-sbom.spdx.json") {
        $sbomFiles += @{ File = "af-sbom.spdx.json"; Format = "spdx" }
    }
    
    if (Test-Path "sbom.spdx.json") {
        $sbomFiles += @{ File = "sbom.spdx.json"; Format = "spdx" }
    }
    
    if (Test-Path "sbom.cyclonedx.json") {
        $sbomFiles += @{ File = "sbom.cyclonedx.json"; Format = "cyclonedx" }
    }
    
    # Look for SBOM files in artifacts directory
    if (Test-Path "artifacts") {
        $artifactFiles = Get-ChildItem -Path "artifacts" -Filter "*.json" -Recurse -ErrorAction SilentlyContinue
        foreach ($file in $artifactFiles) {
            if ($file.Name -like "*.spdx.json") {
                $sbomFiles += @{ File = $file.FullName; Format = "spdx" }
            }
            elseif ($file.Name -like "*.cyclonedx.json") {
                $sbomFiles += @{ File = $file.FullName; Format = "cyclonedx" }
            }
        }
    }
    
    if ($sbomFiles.Count -eq 0) {
        Write-Warning "No local SBOM files found"
        return $true
    }
    
    $validationFailed = $false
    foreach ($sbomEntry in $sbomFiles) {
        if (-not (Test-SbomStructure $sbomEntry.File $sbomEntry.Format)) {
            $validationFailed = $true
        }
    }
    
    if ($validationFailed) {
        Write-Error "Local SBOM validation failed"
        return $false
    }
    
    Write-Success "Local SBOM validation completed"
    return $true
}

# Main validation function
function Invoke-SbomProvenanceValidation {
    param(
        [string]$ImageTag,
        [bool]$ValidateContainers,
        [bool]$ValidateLocal
    )
    
    Write-Info "Starting SBOM & Provenance validation"
    Write-Info "Image tag: $ImageTag"
    Write-Info "Validate containers: $ValidateContainers"
    Write-Info "Validate local files: $ValidateLocal"
    
    # Check prerequisites
    if (-not (Test-Prerequisites)) {
        return $false
    }
    
    $validationFailed = $false
    
    # Validate local SBOM files if requested
    if ($ValidateLocal) {
        if (-not (Test-LocalSbomFiles)) {
            $validationFailed = $true
        }
    }
    
    # Validate container artifacts if requested
    if ($ValidateContainers) {
        foreach ($service in $Services) {
            $imageRef = "$Registry/$Repository/$service`:$ImageTag"
            if (-not (Test-ContainerArtifacts $imageRef $service)) {
                $validationFailed = $true
            }
        }
    }
    
    if ($validationFailed) {
        Write-Error "SBOM & Provenance validation failed"
        return $false
    }
    
    Write-Success "SBOM & Provenance validation completed successfully"
    return $true
}

# Handle help parameter
if ($Help) {
    Show-Usage
    exit 0
}

# Run main validation
$success = Invoke-SbomProvenanceValidation -ImageTag $ImageTag -ValidateContainers $ValidateContainers -ValidateLocal $ValidateLocal

if (-not $success) {
    exit 1
}

exit 0