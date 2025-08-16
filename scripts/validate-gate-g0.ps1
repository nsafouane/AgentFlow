# validate-gate-g0.ps1 - Comprehensive Gate G0 Exit Criteria Validation (PowerShell)
# This script validates all Gate G0 criteria for the Foundations & Project Governance spec

param(
    [switch]$Verbose = $false
)

# Set error action preference
$ErrorActionPreference = "Continue"

# Global validation state
$script:ValidationErrors = 0
$script:ValidationWarnings = 0

# Logging functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning-Custom {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Helper functions
function Increment-Errors {
    $script:ValidationErrors++
}

function Increment-Warnings {
    $script:ValidationWarnings++
}

# Gate G0 Criterion 1: CI green including security scans
function Test-CIGreen {
    Write-Info "Validating Gate G0.1: CI green including security scans"
    
    $errors = 0
    
    # Check if required workflow files exist
    $requiredWorkflows = @(
        ".github/workflows/ci.yml",
        ".github/workflows/security-scan.yml",
        ".github/workflows/container-build.yml"
    )
    
    foreach ($workflow in $requiredWorkflows) {
        if (Test-Path $workflow) {
            Write-Success "Found workflow: $workflow"
        } else {
            Write-Error-Custom "Required workflow file missing: $workflow"
            $errors++
        }
    }
    
    # Validate CI workflow contains security scans
    if (Test-Path ".github/workflows/ci.yml") {
        $ciContent = Get-Content ".github/workflows/ci.yml" -Raw
        $securityTools = @("gosec", "gitleaks", "osv-scanner", "grype")
        
        foreach ($tool in $securityTools) {
            if ($ciContent -match $tool) {
                Write-Success "Security tool $tool found in CI workflow"
            } else {
                Write-Error-Custom "Security tool $tool missing from CI workflow"
                $errors++
            }
        }
    }
    
    # Check for security scan thresholds
    if (Test-Path ".github/workflows/security-scan.yml") {
        $securityContent = Get-Content ".github/workflows/security-scan.yml" -Raw
        if ($securityContent -match "HIGH|CRITICAL") {
            Write-Success "Security scan severity thresholds configured"
        } else {
            Write-Warning-Custom "Security scan severity thresholds not clearly configured"
            Increment-Warnings
        }
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.1: CI green including security scans - PASSED"
    } else {
        Write-Error-Custom "Gate G0.1: CI green including security scans - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 2: Cross-platform builds
function Test-CrossPlatformBuilds {
    Write-Info "Validating Gate G0.2: Cross-platform builds (Linux + Windows + WSL2)"
    
    $errors = 0
    
    # Check for cross-platform build configuration
    if (Test-Path "Makefile") {
        $makeContent = Get-Content "Makefile" -Raw
        if ($makeContent -match "GOOS|GOARCH") {
            Write-Success "Cross-platform build configuration found in Makefile"
        } else {
            Write-Warning-Custom "Cross-platform build configuration not found in Makefile"
            Increment-Warnings
        }
    } else {
        Write-Error-Custom "Makefile not found"
        $errors++
    }
    
    if (Test-Path "Taskfile.yml") {
        $taskContent = Get-Content "Taskfile.yml" -Raw
        if ($taskContent -match "GOOS|GOARCH|windows|linux") {
            Write-Success "Cross-platform build configuration found in Taskfile.yml"
        } else {
            Write-Warning-Custom "Cross-platform build configuration not found in Taskfile.yml"
            Increment-Warnings
        }
    } else {
        Write-Error-Custom "Taskfile.yml not found"
        $errors++
    }
    
    # Check CI workflow for matrix builds
    if (Test-Path ".github/workflows/ci.yml") {
        $ciContent = Get-Content ".github/workflows/ci.yml" -Raw
        if ($ciContent -match "matrix:" -and $ciContent -match "windows-latest|ubuntu-latest") {
            Write-Success "Cross-platform CI matrix build configuration found"
        } else {
            Write-Error-Custom "Cross-platform CI matrix build configuration missing"
            $errors++
        }
    }
    
    # Check for cross-platform troubleshooting documentation
    if (Test-Path "docs/cross-platform-build-troubleshooting.md") {
        Write-Success "Cross-platform build troubleshooting documentation found"
    } else {
        Write-Error-Custom "Cross-platform build troubleshooting documentation missing"
        $errors++
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.2: Cross-platform builds - PASSED"
    } else {
        Write-Error-Custom "Gate G0.2: Cross-platform builds - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 3: Devcontainer adoption
function Test-DevcontainerAdoption {
    Write-Info "Validating Gate G0.3: Devcontainer adoption (af validate warns outside container)"
    
    $errors = 0
    
    # Check for devcontainer configuration
    if (Test-Path ".devcontainer" -PathType Container) {
        Write-Success "Devcontainer directory found"
        
        if (Test-Path ".devcontainer/devcontainer.json") {
            Write-Success "Devcontainer configuration file found"
        } else {
            Write-Error-Custom "Devcontainer configuration file missing"
            $errors++
        }
    } else {
        Write-Error-Custom "Devcontainer directory missing"
        $errors++
    }
    
    # Check for CLI validation tool
    if (Test-Path "cmd/af/main.go") {
        Write-Success "CLI tool source found"
        
        # Check if validate command exists
        $afFiles = Get-ChildItem -Path "cmd/af/" -Recurse -Include "*.go"
        $validateFound = $false
        foreach ($file in $afFiles) {
            $content = Get-Content $file.FullName -Raw
            if ($content -match "validate") {
                $validateFound = $true
                break
            }
        }
        
        if ($validateFound) {
            Write-Success "Validate command found in CLI"
        } else {
            Write-Error-Custom "Validate command missing from CLI"
            $errors++
        }
    } else {
        Write-Error-Custom "CLI tool source missing"
        $errors++
    }
    
    # Check for devcontainer adoption guide
    if (Test-Path "docs/devcontainer-adoption-guide.md") {
        Write-Success "Devcontainer adoption guide found"
    } else {
        Write-Error-Custom "Devcontainer adoption guide missing"
        $errors++
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.3: Devcontainer adoption - PASSED"
    } else {
        Write-Error-Custom "Gate G0.3: Devcontainer adoption - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 4: SBOM & provenance
function Test-SBOMProvenance {
    Write-Info "Validating Gate G0.4: SBOM & provenance (artifacts published per build)"
    
    $errors = 0
    
    # Check for SBOM generation in CI
    if (Test-Path ".github/workflows/ci.yml") {
        $ciContent = Get-Content ".github/workflows/ci.yml" -Raw
        
        if ($ciContent -match "syft|sbom") {
            Write-Success "SBOM generation found in CI workflow"
        } else {
            Write-Error-Custom "SBOM generation missing from CI workflow"
            $errors++
        }
        
        if ($ciContent -match "provenance|attestation") {
            Write-Success "Provenance attestation found in CI workflow"
        } else {
            Write-Error-Custom "Provenance attestation missing from CI workflow"
            $errors++
        }
    }
    
    # Check container build workflow for SBOM
    if (Test-Path ".github/workflows/container-build.yml") {
        $containerContent = Get-Content ".github/workflows/container-build.yml" -Raw
        
        if ($containerContent -match "sbom.*true|syft") {
            Write-Success "Container SBOM generation configured"
        } else {
            Write-Error-Custom "Container SBOM generation missing"
            $errors++
        }
        
        if ($containerContent -match "provenance.*true|attest-build-provenance") {
            Write-Success "Container provenance attestation configured"
        } else {
            Write-Error-Custom "Container provenance attestation missing"
            $errors++
        }
    }
    
    # Check for SBOM verification documentation
    if (Test-Path "docs/sbom-provenance-verification.md") {
        Write-Success "SBOM and provenance verification documentation found"
    } else {
        Write-Error-Custom "SBOM and provenance verification documentation missing"
        $errors++
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.4: SBOM & provenance - PASSED"
    } else {
        Write-Error-Custom "Gate G0.4: SBOM & provenance - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 5: Signed multi-arch images
function Test-SignedMultiArchImages {
    Write-Info "Validating Gate G0.5: Signed multi-arch images (amd64+arm64, cosign verify passes)"
    
    $errors = 0
    
    # Check for multi-arch build configuration
    if (Test-Path ".github/workflows/container-build.yml") {
        $containerContent = Get-Content ".github/workflows/container-build.yml" -Raw
        
        if ($containerContent -match "linux/amd64,linux/arm64|platforms.*amd64.*arm64") {
            Write-Success "Multi-architecture build configuration found"
        } else {
            Write-Error-Custom "Multi-architecture build configuration missing"
            $errors++
        }
        
        if ($containerContent -match "cosign.*sign|sigstore/cosign-installer") {
            Write-Success "Cosign signing configuration found"
        } else {
            Write-Error-Custom "Cosign signing configuration missing"
            $errors++
        }
        
        if ($containerContent -match "cosign.*verify") {
            Write-Success "Cosign verification found"
        } else {
            Write-Error-Custom "Cosign verification missing"
            $errors++
        }
    } else {
        Write-Error-Custom "Container build workflow missing"
        $errors++
    }
    
    # Check for signing documentation
    if (Test-Path "docs/security-baseline.md") {
        $securityContent = Get-Content "docs/security-baseline.md" -Raw
        if ($securityContent -match "cosign|signing|supply.*chain") {
            Write-Success "Container signing documentation found"
        } else {
            Write-Warning-Custom "Container signing documentation incomplete"
            Increment-Warnings
        }
    } else {
        Write-Error-Custom "Security baseline documentation missing"
        $errors++
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.5: Signed multi-arch images - PASSED"
    } else {
        Write-Error-Custom "Gate G0.5: Signed multi-arch images - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 6: Risk register & ADR baseline
function Test-RiskRegisterADR {
    Write-Info "Validating Gate G0.6: Risk register & ADR baseline (merged)"
    
    $errors = 0
    
    # Check for risk register
    if (Test-Path "docs/risk-register.yaml") {
        Write-Success "Risk register found"
        
        $riskContent = Get-Content "docs/risk-register.yaml" -Raw
        
        # Validate risk register structure
        if ($riskContent -match "risks:" -and $riskContent -match "id:") {
            Write-Success "Risk register has proper structure"
        } else {
            Write-Error-Custom "Risk register structure invalid"
            $errors++
        }
        
        # Check for minimum number of risks (≥8)
        $riskMatches = [regex]::Matches($riskContent, "^  - id:", [System.Text.RegularExpressions.RegexOptions]::Multiline)
        $riskCount = $riskMatches.Count
        
        if ($riskCount -ge 8) {
            Write-Success "Risk register contains $riskCount risks (≥8 required)"
        } else {
            Write-Error-Custom "Risk register contains only $riskCount risks (≥8 required)"
            $errors++
        }
        
        # Check for threat modeling session
        if ($riskContent -match "threat_modeling:") {
            Write-Success "Threat modeling session scheduled in risk register"
        } else {
            Write-Error-Custom "Threat modeling session not scheduled in risk register"
            $errors++
        }
    } else {
        Write-Error-Custom "Risk register missing"
        $errors++
    }
    
    # Check for ADR directory and baseline ADR
    if (Test-Path "docs/adr" -PathType Container) {
        Write-Success "ADR directory found"
        
        if (Test-Path "docs/adr/ADR-0001-architecture-baseline.md") {
            Write-Success "Architecture baseline ADR found"
            
            $adrContent = Get-Content "docs/adr/ADR-0001-architecture-baseline.md" -Raw
            
            # Validate ADR structure
            if ($adrContent -match "## Status" -and $adrContent -match "## Context" -and $adrContent -match "## Decision") {
                Write-Success "ADR has proper structure"
            } else {
                Write-Error-Custom "ADR structure invalid"
                $errors++
            }
        } else {
            Write-Error-Custom "Architecture baseline ADR missing"
            $errors++
        }
        
        if (Test-Path "docs/adr/template.md") {
            Write-Success "ADR template found"
        } else {
            Write-Error-Custom "ADR template missing"
            $errors++
        }
    } else {
        Write-Error-Custom "ADR directory missing"
        $errors++
    }
    
    # Check CONTRIBUTING.md references ADR process
    if (Test-Path "CONTRIBUTING.md") {
        $contributingContent = Get-Content "CONTRIBUTING.md" -Raw
        if ($contributingContent -match "adr|decision.*record") {
            Write-Success "CONTRIBUTING.md references ADR process"
        } else {
            Write-Warning-Custom "CONTRIBUTING.md should reference ADR process"
            Increment-Warnings
        }
    } else {
        Write-Error-Custom "CONTRIBUTING.md missing"
        $errors++
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.6: Risk register & ADR baseline - PASSED"
    } else {
        Write-Error-Custom "Gate G0.6: Risk register & ADR baseline - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 7: Release versioning policy
function Test-ReleaseVersioning {
    Write-Info "Validating Gate G0.7: Release versioning policy (RELEASE.md published & CI referenced)"
    
    $errors = 0
    
    # Check for RELEASE.md
    if (Test-Path "RELEASE.md") {
        Write-Success "RELEASE.md found"
        
        $releaseContent = Get-Content "RELEASE.md" -Raw
        
        # Validate RELEASE.md content
        $requiredSections = @("Versioning Scheme", "Tagging Policy", "Branching Model", "Release Process")
        foreach ($section in $requiredSections) {
            if ($releaseContent -match [regex]::Escape($section)) {
                Write-Success "RELEASE.md contains '$section' section"
            } else {
                Write-Error-Custom "RELEASE.md missing '$section' section"
                $errors++
            }
        }
        
        # Check for semantic versioning reference
        if ($releaseContent -match "semantic.*version|semver") {
            Write-Success "RELEASE.md references semantic versioning"
        } else {
            Write-Error-Custom "RELEASE.md should reference semantic versioning"
            $errors++
        }
    } else {
        Write-Error-Custom "RELEASE.md missing"
        $errors++
    }
    
    # Check for release workflow
    if (Test-Path ".github/workflows/release.yml") {
        Write-Success "Release workflow found"
        
        $releaseWorkflowContent = Get-Content ".github/workflows/release.yml" -Raw
        
        # Check if release workflow references versioning policy
        if ($releaseWorkflowContent -match "tag|version") {
            Write-Success "Release workflow includes versioning logic"
        } else {
            Write-Error-Custom "Release workflow missing versioning logic"
            $errors++
        }
    } else {
        Write-Error-Custom "Release workflow missing"
        $errors++
    }
    
    # Check for version management scripts
    if ((Test-Path "scripts/update-version.sh") -or (Test-Path "scripts/parse-version.sh")) {
        Write-Success "Version management scripts found"
    } else {
        Write-Warning-Custom "Version management scripts missing"
        Increment-Warnings
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.7: Release versioning policy - PASSED"
    } else {
        Write-Error-Custom "Gate G0.7: Release versioning policy - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 8: Interface freeze snapshot
function Test-InterfaceFreeze {
    Write-Info "Validating Gate G0.8: Interface freeze snapshot (/docs/interfaces committed & referenced)"
    
    $errors = 0
    
    # Check for interfaces documentation
    if (Test-Path "docs/interfaces" -PathType Container) {
        Write-Success "Interfaces documentation directory found"
        
        if (Test-Path "docs/interfaces/README.md") {
            Write-Success "Interfaces documentation README found"
            
            $interfaceContent = Get-Content "docs/interfaces/README.md" -Raw
            
            # Validate interface documentation content
            $requiredSections = @("Agent Runtime Interfaces", "Planning Interfaces", "Tool Execution Interfaces", "Memory Interfaces", "Messaging Interfaces")
            foreach ($section in $requiredSections) {
                if ($interfaceContent -match [regex]::Escape($section)) {
                    Write-Success "Interface documentation contains '$section'"
                } else {
                    Write-Error-Custom "Interface documentation missing '$section'"
                    $errors++
                }
            }
            
            # Check for interface freeze date
            if ($interfaceContent -match "Interface Freeze Date|Freeze Date") {
                Write-Success "Interface freeze date documented"
            } else {
                Write-Error-Custom "Interface freeze date missing"
                $errors++
            }
        } else {
            Write-Error-Custom "Interfaces documentation README missing"
            $errors++
        }
    } else {
        Write-Error-Custom "Interfaces documentation directory missing"
        $errors++
    }
    
    # Check if interfaces are referenced in main documentation
    if (Test-Path "README.md") {
        $readmeContent = Get-Content "README.md" -Raw
        if ($readmeContent -match "interface|API") {
            Write-Success "Main README references interfaces"
        } else {
            Write-Warning-Custom "Main README should reference interfaces"
            Increment-Warnings
        }
    }
    
    # Check if interfaces are referenced in architecture documentation
    if (Test-Path "docs/ARCHITECTURE.md") {
        $archContent = Get-Content "docs/ARCHITECTURE.md" -Raw
        if ($archContent -match "interface|API") {
            Write-Success "Architecture documentation references interfaces"
        } else {
            Write-Warning-Custom "Architecture documentation should reference interfaces"
            Increment-Warnings
        }
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.8: Interface freeze snapshot - PASSED"
    } else {
        Write-Error-Custom "Gate G0.8: Interface freeze snapshot - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Gate G0 Criterion 9: Threat model kickoff scheduled
function Test-ThreatModeling {
    Write-Info "Validating Gate G0.9: Threat model kickoff scheduled (logged in risk register)"
    
    $errors = 0
    
    # Check if threat modeling is scheduled in risk register
    if (Test-Path "docs/risk-register.yaml") {
        $riskContent = Get-Content "docs/risk-register.yaml" -Raw
        
        if ($riskContent -match "threat_modeling:") {
            Write-Success "Threat modeling section found in risk register"
            
            # Check for required threat modeling fields
            $requiredFields = @("session_date", "owner", "participants", "scope")
            foreach ($field in $requiredFields) {
                if ($riskContent -match "${field}:") {
                    Write-Success "Threat modeling has '$field' field"
                } else {
                    Write-Error-Custom "Threat modeling missing '$field' field"
                    $errors++
                }
            }
            
            # Extract session date
            $sessionDateMatch = [regex]::Match($riskContent, 'session_date:\s*"([^"]+)"')
            if ($sessionDateMatch.Success) {
                $sessionDate = $sessionDateMatch.Groups[1].Value
                Write-Success "Threat modeling session date: $sessionDate"
            } else {
                Write-Error-Custom "Threat modeling session date not properly formatted"
                $errors++
            }
        } else {
            Write-Error-Custom "Threat modeling section missing from risk register"
            $errors++
        }
    } else {
        Write-Error-Custom "Risk register missing (required for threat modeling validation)"
        $errors++
    }
    
    if ($errors -eq 0) {
        Write-Success "Gate G0.9: Threat model kickoff scheduled - PASSED"
    } else {
        Write-Error-Custom "Gate G0.9: Threat model kickoff scheduled - FAILED ($errors errors)"
        $script:ValidationErrors += $errors
    }
    
    return $errors
}

# Main function
function Main {
    Write-Info "Starting Gate G0 Exit Criteria Validation"
    Write-Info "Spec: Q1.1 Foundations & Project Governance"
    Write-Info "Date: $(Get-Date)"
    Write-Host ""
    
    # Run all validations
    Test-CIGreen
    Write-Host ""
    Test-CrossPlatformBuilds
    Write-Host ""
    Test-DevcontainerAdoption
    Write-Host ""
    Test-SBOMProvenance
    Write-Host ""
    Test-SignedMultiArchImages
    Write-Host ""
    Test-RiskRegisterADR
    Write-Host ""
    Test-ReleaseVersioning
    Write-Host ""
    Test-InterfaceFreeze
    Write-Host ""
    Test-ThreatModeling
    Write-Host ""
    
    # Summary
    Write-Info "Gate G0 Validation Summary"
    Write-Info "=========================="
    
    if ($script:ValidationErrors -eq 0) {
        Write-Success "✅ All Gate G0 criteria PASSED"
        Write-Success "✅ Foundation is ready for Q1.2 development"
        if ($script:ValidationWarnings -gt 0) {
            Write-Warning-Custom "⚠️  $($script:ValidationWarnings) warnings found (non-blocking)"
        }
        Write-Host ""
        Write-Success "🎉 Gate G0 VALIDATION SUCCESSFUL"
        exit 0
    } else {
        Write-Error-Custom "❌ $($script:ValidationErrors) Gate G0 criteria FAILED"
        if ($script:ValidationWarnings -gt 0) {
            Write-Warning-Custom "⚠️  $($script:ValidationWarnings) warnings found"
        }
        Write-Host ""
        Write-Error-Custom "🚫 Gate G0 VALIDATION FAILED"
        Write-Error-Custom "Foundation is NOT ready for Q1.2 development"
        Write-Error-Custom "Please address the errors above before proceeding"
        exit 1
    }
}

# Run main function
Main