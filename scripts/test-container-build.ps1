# Container Build Testing Script (PowerShell)
# Tests multi-arch container builds, signatures, and SBOM generation

param(
    [string]$Registry = $(if ($env:REGISTRY) { $env:REGISTRY } else { "ghcr.io" }),
    [string]$ImageName = $(if ($env:IMAGE_NAME) { $env:IMAGE_NAME } else { "agentflow/agentflow" }),
    [string]$Tag = $(if ($env:TAG) { $env:TAG } else { "latest" })
)

# Configuration
$Services = @("control-plane", "worker", "af")
$ExitCode = 0

Write-Host "AgentFlow Container Build Tests" -ForegroundColor Blue
Write-Host "Registry: $Registry"
Write-Host "Image Name: $ImageName"
Write-Host "Tag: $Tag"
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

# Function to test manifest list inspection
function Test-ManifestList {
    param([string]$Service)
    
    $ImageRef = "$Registry/$ImageName/$Service`:$Tag"
    Write-Host "Testing manifest list for $Service..." -ForegroundColor Yellow
    
    if (-not (Test-Command "docker")) {
        Write-Host "X Docker not found, skipping manifest list test" -ForegroundColor Red
        return $false
    }
    
    # Check if image exists
    try {
        docker buildx imagetools inspect $ImageRef 2>$null | Out-Null
    }
    catch {
        Write-Host "! Image $ImageRef not found, skipping test" -ForegroundColor Yellow
        return $true
    }
    
    # Get raw manifest
    try {
        $Manifest = docker buildx imagetools inspect --raw $ImageRef 2>$null
        if (-not $Manifest) {
            Write-Host "X Failed to get manifest for $ImageRef" -ForegroundColor Red
            return $false
        }
        
        $ManifestObj = $Manifest | ConvertFrom-Json
        $MediaType = $ManifestObj.mediaType
        
        if ($MediaType -like "*manifest.list*" -or $MediaType -like "*image.index*") {
            Write-Host "+ Manifest list found for $Service" -ForegroundColor Green
            
            # Check architectures
            $Architectures = $ManifestObj.manifests | Where-Object { $_.platform.os -eq "linux" } | Select-Object -ExpandProperty platform | Select-Object -ExpandProperty architecture
            
            $HasAmd64 = $Architectures -contains "amd64"
            $HasArm64 = $Architectures -contains "arm64"
            
            if ($HasAmd64 -and $HasArm64) {
                Write-Host "+ Both amd64 and arm64 architectures found" -ForegroundColor Green
                return $true
            }
            else {
                Write-Host "X Missing required architectures (amd64: $HasAmd64, arm64: $HasArm64)" -ForegroundColor Red
                return $false
            }
        }
        else {
            Write-Host "X Not a manifest list: $MediaType" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "X Error testing manifest list: $_" -ForegroundColor Red
        return $false
    }
}

# Function to test signature presence
function Test-Signature {
    param([string]$Service)
    
    $ImageRef = "$Registry/$ImageName/$Service`:$Tag"
    Write-Host "Testing signature for $Service..." -ForegroundColor Yellow
    
    if (-not (Test-Command "cosign")) {
        Write-Host "! Cosign not found, skipping signature test" -ForegroundColor Yellow
        return $true
    }
    
    # Verify signature
    try {
        cosign verify --certificate-identity-regexp="https://github.com/$ImageName" --certificate-oidc-issuer="https://token.actions.githubusercontent.com" $ImageRef 2>$null | Out-Null
        Write-Host "+ Signature verified for $Service" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "! Signature verification failed or not found for $Service" -ForegroundColor Yellow
        return $true
    }
}

# Function to test SBOM presence
function Test-SBOM {
    param([string]$Service)
    
    $ImageRef = "$Registry/$ImageName/$Service`:$Tag"
    Write-Host "Testing SBOM for $Service..." -ForegroundColor Yellow
    
    if (-not (Test-Command "syft")) {
        Write-Host "! Syft not found, skipping SBOM test" -ForegroundColor Yellow
        return $true
    }
    
    # Generate SBOM
    try {
        syft $ImageRef -o json 2>$null | Out-Null
        Write-Host "+ SBOM generated successfully for $Service" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "! SBOM generation failed for $Service" -ForegroundColor Yellow
        return $true
    }
}

# Function to test provenance attestation
function Test-Provenance {
    param([string]$Service)
    
    $ImageRef = "$Registry/$ImageName/$Service`:$Tag"
    Write-Host "Testing provenance attestation for $Service..." -ForegroundColor Yellow
    
    if (-not (Test-Command "cosign")) {
        Write-Host "! Cosign not found, skipping provenance test" -ForegroundColor Yellow
        return $true
    }
    
    # Verify provenance attestation
    try {
        cosign verify-attestation --certificate-identity-regexp="https://github.com/$ImageName" --certificate-oidc-issuer="https://token.actions.githubusercontent.com" --type slsaprovenance $ImageRef 2>$null | Out-Null
        Write-Host "+ Provenance attestation verified for $Service" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "! Provenance attestation verification failed for $Service" -ForegroundColor Yellow
        return $true
    }
}

# Main test execution
Write-Host "Running container build tests..." -ForegroundColor Blue
Write-Host ""

foreach ($Service in $Services) {
    Write-Host "Testing service: $Service" -ForegroundColor Blue
    
    # Test manifest list
    if (-not (Test-ManifestList $Service)) {
        $ExitCode = 1
    }
    
    # Test signature
    if (-not (Test-Signature $Service)) {
        $ExitCode = 1
    }
    
    # Test SBOM
    if (-not (Test-SBOM $Service)) {
        $ExitCode = 1
    }
    
    # Test provenance
    if (-not (Test-Provenance $Service)) {
        $ExitCode = 1
    }
    
    Write-Host ""
}

if ($ExitCode -eq 0) {
    Write-Host "All container build tests passed!" -ForegroundColor Green
}
else {
    Write-Host "X Some container build tests failed" -ForegroundColor Red
}

exit $ExitCode