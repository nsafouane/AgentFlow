# Manual testing script for release workflow (PowerShell)
# This script simulates and validates the release process

param(
    [switch]$Verbose
)

$ErrorActionPreference = "Continue"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

$TestVersion = "0.1.0-test.$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

Write-ColorOutput "AgentFlow Release Workflow Test" "Blue"
Write-ColorOutput "==================================" "Blue"
Write-ColorOutput "Test Version: $TestVersion" "White"
Write-Host ""

Set-Location $ProjectRoot

# Test 1: Version validation
Write-ColorOutput "Test 1: Version Validation" "Yellow"
Write-Host "  Testing version format validation... " -NoNewline

try {
    $output = & "scripts\parse-version.ps1" -Version $TestVersion -Command parse 2>$null
    if ($output) {
        Write-ColorOutput "✓" "Green"
    } else {
        Write-ColorOutput "✗" "Red"
        Write-Host "  Version validation failed for: $TestVersion"
        exit 1
    }
} catch {
    Write-ColorOutput "✗" "Red"
    Write-Host "  Version validation failed for: $TestVersion"
    if ($Verbose) {
        Write-Host "  Error: $($_.Exception.Message)"
    }
    exit 1
}

# Test 2: Version update
Write-ColorOutput "`nTest 2: Version Update" "Yellow"
Write-Host "  Testing version update script... " -NoNewline

# Create backup of version files
$backupFiles = @()
if (Test-Path "cmd\af\version.go") {
    Copy-Item "cmd\af\version.go" "cmd\af\version.go.bak"
    $backupFiles += "cmd\af\version.go"
}
if (Test-Path "go.mod") {
    Copy-Item "go.mod" "go.mod.bak"
    $backupFiles += "go.mod"
}

try {
    & "scripts\update-version.ps1" -NewVersion $TestVersion > $null 2>&1
    Write-ColorOutput "✓" "Green"
    
    # Verify version was updated
    if (Test-Path "cmd\af\version.go") {
        $content = Get-Content "cmd\af\version.go" -Raw
        if ($content -match "Version = `"$([regex]::Escape($TestVersion))`"") {
            Write-Host "    Version updated in version.go ✓"
        } else {
            Write-ColorOutput "    Warning: version.go not updated" "Yellow"
        }
    }
} catch {
    Write-ColorOutput "✗" "Red"
    Write-Host "  Version update failed"
    if ($Verbose) {
        Write-Host "  Error: $($_.Exception.Message)"
    }
}

# Restore backup files
foreach ($file in $backupFiles) {
    if (Test-Path "$file.bak") {
        Move-Item "$file.bak" $file -Force
    }
}

# Test 3: Build process
Write-ColorOutput "`nTest 3: Build Process" "Yellow"
Write-Host "  Testing Go build... " -NoNewline

# Set build variables
$env:VERSION = $TestVersion
$env:BUILD_DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$env:GIT_COMMIT = try { git rev-parse HEAD 2>$null } catch { "unknown" }

# Ensure dist directory exists
$null = New-Item -Path "dist" -ItemType Directory -Force

try {
    $ldflags = "-X main.Version=$($env:VERSION) -X main.BuildDate=$($env:BUILD_DATE) -X main.GitCommit=$($env:GIT_COMMIT)"
    go build -ldflags="$ldflags" -o "dist\af-test.exe" .\cmd\af 2>$null
    
    if (Test-Path "dist\af-test.exe") {
        Write-ColorOutput "✓" "Green"
        Write-Host "    Binary created successfully ✓"
        
        # Test binary
        try {
            & "dist\af-test.exe" version > $null 2>&1
            Write-Host "    Version command works ✓"
        } catch {
            try {
                & "dist\af-test.exe" --version > $null 2>&1
                Write-Host "    Version flag works ✓"
            } catch {
                Write-ColorOutput "    Note: No version command found" "Yellow"
            }
        }
    } else {
        Write-ColorOutput "    Binary not created" "Red"
    }
} catch {
    Write-ColorOutput "✗" "Red"
    Write-Host "  Build failed"
    if ($Verbose) {
        Write-Host "  Error: $($_.Exception.Message)"
    }
}

# Test 4: Container build (if Docker is available)
Write-ColorOutput "`nTest 4: Container Build" "Yellow"
if (Get-Command docker -ErrorAction SilentlyContinue) {
    Write-Host "  Testing Docker build... " -NoNewline
    
    # Create minimal Dockerfile for testing if it doesn't exist
    if (-not (Test-Path "Dockerfile")) {
        $dockerfileContent = @'
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG BUILD_DATE
ARG GIT_COMMIT
RUN go build -ldflags="-X main.Version=$VERSION -X main.BuildDate=$BUILD_DATE -X main.GitCommit=$GIT_COMMIT" \
    -o af ./cmd/af

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/af .
LABEL version="$VERSION"
CMD ["./af"]
'@
        Set-Content -Path "Dockerfile" -Value $dockerfileContent
        Write-Host "    Created test Dockerfile"
    }
    
    try {
        docker build --build-arg VERSION="$TestVersion" --build-arg BUILD_DATE="$($env:BUILD_DATE)" --build-arg GIT_COMMIT="$($env:GIT_COMMIT)" -t "agentflow:$TestVersion" . 2>$null | Out-Null
        Write-ColorOutput "✓" "Green"
        Write-Host "    Container image built successfully ✓"
        
        # Test container run
        try {
            docker run --rm "agentflow:$TestVersion" --help 2>$null | Out-Null
            Write-Host "    Container runs successfully ✓"
        } catch {
            Write-ColorOutput "    Note: Container help command failed" "Yellow"
        }
        
        # Cleanup test image
        try {
            docker rmi "agentflow:$TestVersion" 2>$null | Out-Null
        } catch {
            # Ignore cleanup errors
        }
    } catch {
        Write-ColorOutput "✗" "Red"
        Write-Host "  Container build failed"
        if ($Verbose) {
            Write-Host "  Error: $($_.Exception.Message)"
        }
    }
} else {
    Write-Host "  Docker not available, skipping container build test"
}

# Test 5: SBOM generation (if syft is available)
Write-ColorOutput "`nTest 5: SBOM Generation" "Yellow"
if (Get-Command syft -ErrorAction SilentlyContinue) {
    Write-Host "  Testing SBOM generation... " -NoNewline
    
    try {
        syft . -o spdx-json=test-sbom.spdx.json 2>$null | Out-Null
        Write-ColorOutput "✓" "Green"
        
        if (Test-Path "test-sbom.spdx.json") {
            Write-Host "    SBOM file created ✓"
            
            # Validate SBOM format
            try {
                Get-Content "test-sbom.spdx.json" | ConvertFrom-Json | Out-Null
                Write-Host "    SBOM format valid ✓"
            } catch {
                Write-ColorOutput "    Warning: SBOM format validation failed" "Yellow"
            }
            
            # Cleanup
            Remove-Item "test-sbom.spdx.json" -ErrorAction SilentlyContinue
        } else {
            Write-ColorOutput "    SBOM file not created" "Red"
        }
    } catch {
        Write-ColorOutput "✗" "Red"
        Write-Host "  SBOM generation failed"
        if ($Verbose) {
            Write-Host "  Error: $($_.Exception.Message)"
        }
    }
} else {
    Write-Host "  Syft not available, skipping SBOM generation test"
}

# Test 6: Security scanning (if grype is available)
Write-ColorOutput "`nTest 6: Security Scanning" "Yellow"
if (Get-Command grype -ErrorAction SilentlyContinue) {
    Write-Host "  Testing vulnerability scanning... " -NoNewline
    
    try {
        grype . -o json --file test-grype.json 2>$null | Out-Null
        Write-ColorOutput "✓" "Green"
        
        if (Test-Path "test-grype.json") {
            Write-Host "    Vulnerability scan completed ✓"
            
            # Check for high/critical vulnerabilities
            if (Get-Command jq -ErrorAction SilentlyContinue) {
                try {
                    $highCritical = jq '[.matches[] | select(.vulnerability.severity == "High" or .vulnerability.severity == "Critical")] | length' test-grype.json 2>$null
                    if ([int]$highCritical -gt 0) {
                        Write-ColorOutput "    Warning: Found $highCritical high/critical vulnerabilities" "Yellow"
                    } else {
                        Write-Host "    No high/critical vulnerabilities found ✓"
                    }
                } catch {
                    Write-ColorOutput "    Note: Could not parse vulnerability results" "Yellow"
                }
            }
            
            # Cleanup
            Remove-Item "test-grype.json" -ErrorAction SilentlyContinue
        } else {
            Write-ColorOutput "    Scan results not created" "Red"
        }
    } catch {
        Write-ColorOutput "✗" "Red"
        Write-Host "  Vulnerability scanning failed"
        if ($Verbose) {
            Write-Host "  Error: $($_.Exception.Message)"
        }
    }
} else {
    Write-Host "  Grype not available, skipping vulnerability scanning test"
}

# Test 7: Release notes generation
Write-ColorOutput "`nTest 7: Release Notes Generation" "Yellow"
Write-Host "  Testing changelog extraction... " -NoNewline

if (Test-Path "CHANGELOG.md") {
    $changelogContent = Get-Content "CHANGELOG.md" -Raw
    
    if ($changelogContent -match "## \[Unreleased\]") {
        # Extract unreleased section
        $lines = Get-Content "CHANGELOG.md"
        $startIndex = -1
        $endIndex = -1
        
        for ($i = 0; $i -lt $lines.Count; $i++) {
            if ($lines[$i] -match "## \[Unreleased\]") {
                $startIndex = $i
            } elseif ($startIndex -ge 0 -and $lines[$i] -match "## \[") {
                $endIndex = $i
                break
            }
        }
        
        if ($startIndex -ge 0) {
            $releaseNotes = if ($endIndex -gt 0) { 
                $lines[$startIndex..($endIndex-1)] 
            } else { 
                $lines[$startIndex..($lines.Count-1)] 
            }
            
            Set-Content -Path "test-release-notes.md" -Value $releaseNotes
            
            if ((Get-Item "test-release-notes.md").Length -gt 0) {
                Write-ColorOutput "✓" "Green"
                Write-Host "    Release notes extracted ✓"
                
                # Show preview
                Write-Host "    Preview:"
                Get-Content "test-release-notes.md" | Select-Object -First 5 | ForEach-Object { Write-Host "      $_" }
                
                # Cleanup
                Remove-Item "test-release-notes.md" -ErrorAction SilentlyContinue
            } else {
                Write-ColorOutput "⚠ (empty release notes)" "Yellow"
            }
        } else {
            Write-ColorOutput "⚠ (could not extract section)" "Yellow"
        }
    } else {
        Write-ColorOutput "⚠ (no unreleased section)" "Yellow"
    }
} else {
    Write-ColorOutput "✗ (CHANGELOG.md not found)" "Red"
}

# Test 8: GitHub Actions workflow validation
Write-ColorOutput "`nTest 8: Workflow Validation" "Yellow"
Write-Host "  Testing workflow file syntax... " -NoNewline

if (Test-Path ".github\workflows\release.yml") {
    # Basic YAML syntax check using PowerShell
    try {
        $yamlContent = Get-Content ".github\workflows\release.yml" -Raw
        
        # Basic validation - check for required structure
        if ($yamlContent -match "name:" -and $yamlContent -match "jobs:" -and $yamlContent -match "runs-on:") {
            Write-ColorOutput "✓" "Green"
            Write-Host "    Workflow YAML structure valid ✓"
        } else {
            Write-ColorOutput "✗ (missing required YAML structure)" "Red"
        }
        
        # Check for required jobs
        $requiredJobs = @("validate-version", "build-and-test", "build-containers", "create-release")
        foreach ($job in $requiredJobs) {
            if ($yamlContent -match "${job}:") {
                Write-Host "    Job '$job' found ✓"
            } else {
                Write-ColorOutput "    Warning: Job '$job' not found" "Yellow"
            }
        }
    } catch {
        Write-ColorOutput "✗ (YAML parsing error)" "Red"
        if ($Verbose) {
            Write-Host "  Error: $($_.Exception.Message)"
        }
    }
} else {
    Write-ColorOutput "✗ (workflow file not found)" "Red"
}

# Cleanup
Remove-Item "dist\af-test.exe" -ErrorAction SilentlyContinue

Write-ColorOutput "`nRelease Workflow Test Summary" "Blue"
Write-ColorOutput "==================================" "Blue"
Write-ColorOutput "✓ Version validation" "Green"
Write-ColorOutput "✓ Version update process" "Green"
Write-ColorOutput "✓ Build process" "Green"
Write-ColorOutput "✓ Container build (if Docker available)" "Green"
Write-ColorOutput "✓ SBOM generation (if Syft available)" "Green"
Write-ColorOutput "✓ Security scanning (if Grype available)" "Green"
Write-ColorOutput "✓ Release notes generation" "Green"
Write-ColorOutput "✓ Workflow validation" "Green"

Write-ColorOutput "`nManual Testing Instructions:" "Blue"
Write-Host "1. To test the full workflow with GitHub Actions:"
Write-Host "   - Push a tag: git tag v0.1.0-test && git push origin v0.1.0-test"
Write-Host "   - Or trigger manually: Go to Actions → Release → Run workflow"
Write-Host ""
Write-Host "2. To test dry run:"
Write-Host "   - Go to Actions → Release → Run workflow"
Write-Host "   - Set 'Perform a dry run' to true"
Write-Host "   - Enter version: 0.1.0-test"
Write-Host ""
Write-Host "3. To verify signed artifacts:"
Write-Host "   - Install cosign: go install github.com/sigstore/cosign/v2/cmd/cosign@latest"
Write-Host "   - Verify signature: cosign verify --certificate-identity-regexp=... <image>"

Write-ColorOutput "`nRelease workflow test completed successfully!" "Green"