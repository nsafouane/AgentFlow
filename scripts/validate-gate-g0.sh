#!/bin/bash
# validate-gate-g0.sh - Comprehensive Gate G0 Exit Criteria Validation
# This script validates all Gate G0 criteria for the Foundations & Project Governance spec

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Global validation state
VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0

# Helper function to increment error count
increment_errors() {
    ((VALIDATION_ERRORS++))
}

# Helper function to increment warning count
increment_warnings() {
    ((VALIDATION_WARNINGS++))
}

# Gate G0 Criterion 1: CI green including security scans
validate_ci_green() {
    log_info "Validating Gate G0.1: CI green including security scans"
    
    local errors=0
    
    # Check if required workflow files exist
    local required_workflows=(
        ".github/workflows/ci.yml"
        ".github/workflows/security-scan.yml"
        ".github/workflows/container-build.yml"
    )
    
    for workflow in "${required_workflows[@]}"; do
        if [[ ! -f "$workflow" ]]; then
            log_error "Required workflow file missing: $workflow"
            ((errors++))
        else
            log_success "Found workflow: $workflow"
        fi
    done
    
    # Validate CI workflow contains security scans
    if [[ -f ".github/workflows/ci.yml" ]]; then
        local security_tools=("gosec" "gitleaks" "osv-scanner" "grype")
        for tool in "${security_tools[@]}"; do
            if grep -q "$tool" .github/workflows/ci.yml; then
                log_success "Security tool $tool found in CI workflow"
            else
                log_error "Security tool $tool missing from CI workflow"
                ((errors++))
            fi
        done
    fi
    
    # Check for security scan thresholds
    if [[ -f ".github/workflows/security-scan.yml" ]]; then
        if grep -q "HIGH\|CRITICAL" .github/workflows/security-scan.yml; then
            log_success "Security scan severity thresholds configured"
        else
            log_warning "Security scan severity thresholds not clearly configured"
            increment_warnings
        fi
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.1: CI green including security scans - PASSED"
    else
        log_error "Gate G0.1: CI green including security scans - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 2: Cross-platform builds
validate_cross_platform_builds() {
    log_info "Validating Gate G0.2: Cross-platform builds (Linux + Windows + WSL2)"
    
    local errors=0
    
    # Check for cross-platform build configuration
    if [[ -f "Makefile" ]]; then
        if grep -q "GOOS\|GOARCH" Makefile; then
            log_success "Cross-platform build configuration found in Makefile"
        else
            log_warning "Cross-platform build configuration not found in Makefile"
            increment_warnings
        fi
    else
        log_error "Makefile not found"
        ((errors++))
    fi
    
    if [[ -f "Taskfile.yml" ]]; then
        if grep -q "GOOS\|GOARCH\|windows\|linux" Taskfile.yml; then
            log_success "Cross-platform build configuration found in Taskfile.yml"
        else
            log_warning "Cross-platform build configuration not found in Taskfile.yml"
            increment_warnings
        fi
    else
        log_error "Taskfile.yml not found"
        ((errors++))
    fi
    
    # Check CI workflow for matrix builds
    if [[ -f ".github/workflows/ci.yml" ]]; then
        if grep -q "matrix:" .github/workflows/ci.yml && grep -q "windows-latest\|ubuntu-latest" .github/workflows/ci.yml; then
            log_success "Cross-platform CI matrix build configuration found"
        else
            log_error "Cross-platform CI matrix build configuration missing"
            ((errors++))
        fi
    fi
    
    # Check for cross-platform troubleshooting documentation
    if [[ -f "docs/cross-platform-build-troubleshooting.md" ]]; then
        log_success "Cross-platform build troubleshooting documentation found"
    else
        log_error "Cross-platform build troubleshooting documentation missing"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.2: Cross-platform builds - PASSED"
    else
        log_error "Gate G0.2: Cross-platform builds - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 3: Devcontainer adoption
validate_devcontainer_adoption() {
    log_info "Validating Gate G0.3: Devcontainer adoption (af validate warns outside container)"
    
    local errors=0
    
    # Check for devcontainer configuration
    if [[ -d ".devcontainer" ]]; then
        log_success "Devcontainer directory found"
        
        if [[ -f ".devcontainer/devcontainer.json" ]]; then
            log_success "Devcontainer configuration file found"
        else
            log_error "Devcontainer configuration file missing"
            ((errors++))
        fi
    else
        log_error "Devcontainer directory missing"
        ((errors++))
    fi
    
    # Check for CLI validation tool
    if [[ -f "cmd/af/main.go" ]]; then
        log_success "CLI tool source found"
        
        # Check if validate command exists
        if grep -r "validate" cmd/af/ >/dev/null 2>&1; then
            log_success "Validate command found in CLI"
        else
            log_error "Validate command missing from CLI"
            ((errors++))
        fi
    else
        log_error "CLI tool source missing"
        ((errors++))
    fi
    
    # Check for devcontainer adoption guide
    if [[ -f "docs/devcontainer-adoption-guide.md" ]]; then
        log_success "Devcontainer adoption guide found"
    else
        log_error "Devcontainer adoption guide missing"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.3: Devcontainer adoption - PASSED"
    else
        log_error "Gate G0.3: Devcontainer adoption - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 4: SBOM & provenance
validate_sbom_provenance() {
    log_info "Validating Gate G0.4: SBOM & provenance (artifacts published per build)"
    
    local errors=0
    
    # Check for SBOM generation in CI
    if [[ -f ".github/workflows/ci.yml" ]]; then
        if grep -q "syft\|sbom" .github/workflows/ci.yml; then
            log_success "SBOM generation found in CI workflow"
        else
            log_error "SBOM generation missing from CI workflow"
            ((errors++))
        fi
        
        if grep -q "provenance\|attestation" .github/workflows/ci.yml; then
            log_success "Provenance attestation found in CI workflow"
        else
            log_error "Provenance attestation missing from CI workflow"
            ((errors++))
        fi
    fi
    
    # Check container build workflow for SBOM
    if [[ -f ".github/workflows/container-build.yml" ]]; then
        if grep -q "sbom.*true\|syft" .github/workflows/container-build.yml; then
            log_success "Container SBOM generation configured"
        else
            log_error "Container SBOM generation missing"
            ((errors++))
        fi
        
        if grep -q "provenance.*true\|attest-build-provenance" .github/workflows/container-build.yml; then
            log_success "Container provenance attestation configured"
        else
            log_error "Container provenance attestation missing"
            ((errors++))
        fi
    fi
    
    # Check for SBOM verification documentation
    if [[ -f "docs/sbom-provenance-verification.md" ]]; then
        log_success "SBOM and provenance verification documentation found"
    else
        log_error "SBOM and provenance verification documentation missing"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.4: SBOM & provenance - PASSED"
    else
        log_error "Gate G0.4: SBOM & provenance - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 5: Signed multi-arch images
validate_signed_multiarch_images() {
    log_info "Validating Gate G0.5: Signed multi-arch images (amd64+arm64, cosign verify passes)"
    
    local errors=0
    
    # Check for multi-arch build configuration
    if [[ -f ".github/workflows/container-build.yml" ]]; then
        if grep -q "linux/amd64,linux/arm64\|platforms.*amd64.*arm64" .github/workflows/container-build.yml; then
            log_success "Multi-architecture build configuration found"
        else
            log_error "Multi-architecture build configuration missing"
            ((errors++))
        fi
        
        if grep -q "cosign.*sign\|sigstore/cosign-installer" .github/workflows/container-build.yml; then
            log_success "Cosign signing configuration found"
        else
            log_error "Cosign signing configuration missing"
            ((errors++))
        fi
        
        if grep -q "cosign.*verify" .github/workflows/container-build.yml; then
            log_success "Cosign verification found"
        else
            log_error "Cosign verification missing"
            ((errors++))
        fi
    else
        log_error "Container build workflow missing"
        ((errors++))
    fi
    
    # Check for signing documentation
    if [[ -f "docs/security-baseline.md" ]]; then
        if grep -q "cosign\|signing\|supply.*chain" docs/security-baseline.md; then
            log_success "Container signing documentation found"
        else
            log_warning "Container signing documentation incomplete"
            increment_warnings
        fi
    else
        log_error "Security baseline documentation missing"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.5: Signed multi-arch images - PASSED"
    else
        log_error "Gate G0.5: Signed multi-arch images - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 6: Risk register & ADR baseline
validate_risk_register_adr() {
    log_info "Validating Gate G0.6: Risk register & ADR baseline (merged)"
    
    local errors=0
    
    # Check for risk register
    if [[ -f "docs/risk-register.yaml" ]]; then
        log_success "Risk register found"
        
        # Validate risk register structure
        if grep -q "risks:" docs/risk-register.yaml && grep -q "id:" docs/risk-register.yaml; then
            log_success "Risk register has proper structure"
        else
            log_error "Risk register structure invalid"
            ((errors++))
        fi
        
        # Check for minimum number of risks (≥8)
        local risk_count=$(grep -c "^  - id:" docs/risk-register.yaml || echo "0")
        if [[ $risk_count -ge 8 ]]; then
            log_success "Risk register contains $risk_count risks (≥8 required)"
        else
            log_error "Risk register contains only $risk_count risks (≥8 required)"
            ((errors++))
        fi
        
        # Check for threat modeling session
        if grep -q "threat_modeling:" docs/risk-register.yaml; then
            log_success "Threat modeling session scheduled in risk register"
        else
            log_error "Threat modeling session not scheduled in risk register"
            ((errors++))
        fi
    else
        log_error "Risk register missing"
        ((errors++))
    fi
    
    # Check for ADR directory and baseline ADR
    if [[ -d "docs/adr" ]]; then
        log_success "ADR directory found"
        
        if [[ -f "docs/adr/ADR-0001-architecture-baseline.md" ]]; then
            log_success "Architecture baseline ADR found"
            
            # Validate ADR structure
            if grep -q "## Status" docs/adr/ADR-0001-architecture-baseline.md && 
               grep -q "## Context" docs/adr/ADR-0001-architecture-baseline.md &&
               grep -q "## Decision" docs/adr/ADR-0001-architecture-baseline.md; then
                log_success "ADR has proper structure"
            else
                log_error "ADR structure invalid"
                ((errors++))
            fi
        else
            log_error "Architecture baseline ADR missing"
            ((errors++))
        fi
        
        if [[ -f "docs/adr/template.md" ]]; then
            log_success "ADR template found"
        else
            log_error "ADR template missing"
            ((errors++))
        fi
    else
        log_error "ADR directory missing"
        ((errors++))
    fi
    
    # Check CONTRIBUTING.md references ADR process
    if [[ -f "CONTRIBUTING.md" ]]; then
        if grep -q -i "adr\|decision.*record" CONTRIBUTING.md; then
            log_success "CONTRIBUTING.md references ADR process"
        else
            log_warning "CONTRIBUTING.md should reference ADR process"
            increment_warnings
        fi
    else
        log_error "CONTRIBUTING.md missing"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.6: Risk register & ADR baseline - PASSED"
    else
        log_error "Gate G0.6: Risk register & ADR baseline - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 7: Release versioning policy
validate_release_versioning() {
    log_info "Validating Gate G0.7: Release versioning policy (RELEASE.md published & CI referenced)"
    
    local errors=0
    
    # Check for RELEASE.md
    if [[ -f "RELEASE.md" ]]; then
        log_success "RELEASE.md found"
        
        # Validate RELEASE.md content
        local required_sections=("Versioning Scheme" "Tagging Policy" "Branching Model" "Release Process")
        for section in "${required_sections[@]}"; do
            if grep -q "$section" RELEASE.md; then
                log_success "RELEASE.md contains '$section' section"
            else
                log_error "RELEASE.md missing '$section' section"
                ((errors++))
            fi
        done
        
        # Check for semantic versioning reference
        if grep -q -i "semantic.*version\|semver" RELEASE.md; then
            log_success "RELEASE.md references semantic versioning"
        else
            log_error "RELEASE.md should reference semantic versioning"
            ((errors++))
        fi
    else
        log_error "RELEASE.md missing"
        ((errors++))
    fi
    
    # Check for release workflow
    if [[ -f ".github/workflows/release.yml" ]]; then
        log_success "Release workflow found"
        
        # Check if release workflow references versioning policy
        if grep -q "tag\|version" .github/workflows/release.yml; then
            log_success "Release workflow includes versioning logic"
        else
            log_error "Release workflow missing versioning logic"
            ((errors++))
        fi
    else
        log_error "Release workflow missing"
        ((errors++))
    fi
    
    # Check for version management scripts
    if [[ -f "scripts/update-version.sh" ]] || [[ -f "scripts/parse-version.sh" ]]; then
        log_success "Version management scripts found"
    else
        log_warning "Version management scripts missing"
        increment_warnings
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.7: Release versioning policy - PASSED"
    else
        log_error "Gate G0.7: Release versioning policy - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 8: Interface freeze snapshot
validate_interface_freeze() {
    log_info "Validating Gate G0.8: Interface freeze snapshot (/docs/interfaces committed & referenced)"
    
    local errors=0
    
    # Check for interfaces documentation
    if [[ -d "docs/interfaces" ]]; then
        log_success "Interfaces documentation directory found"
        
        if [[ -f "docs/interfaces/README.md" ]]; then
            log_success "Interfaces documentation README found"
            
            # Validate interface documentation content
            local required_sections=("Agent Runtime Interfaces" "Planning Interfaces" "Tool Execution Interfaces" "Memory Interfaces" "Messaging Interfaces")
            for section in "${required_sections[@]}"; do
                if grep -q "$section" docs/interfaces/README.md; then
                    log_success "Interface documentation contains '$section'"
                else
                    log_error "Interface documentation missing '$section'"
                    ((errors++))
                fi
            done
            
            # Check for interface freeze date
            if grep -q "Interface Freeze Date\|Freeze Date" docs/interfaces/README.md; then
                log_success "Interface freeze date documented"
            else
                log_error "Interface freeze date missing"
                ((errors++))
            fi
        else
            log_error "Interfaces documentation README missing"
            ((errors++))
        fi
    else
        log_error "Interfaces documentation directory missing"
        ((errors++))
    fi
    
    # Check if interfaces are referenced in main documentation
    if [[ -f "README.md" ]]; then
        if grep -q "interface\|API" README.md; then
            log_success "Main README references interfaces"
        else
            log_warning "Main README should reference interfaces"
            increment_warnings
        fi
    fi
    
    # Check if interfaces are referenced in architecture documentation
    if [[ -f "docs/ARCHITECTURE.md" ]]; then
        if grep -q "interface\|API" docs/ARCHITECTURE.md; then
            log_success "Architecture documentation references interfaces"
        else
            log_warning "Architecture documentation should reference interfaces"
            increment_warnings
        fi
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.8: Interface freeze snapshot - PASSED"
    else
        log_error "Gate G0.8: Interface freeze snapshot - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Gate G0 Criterion 9: Threat model kickoff scheduled
validate_threat_modeling() {
    log_info "Validating Gate G0.9: Threat model kickoff scheduled (logged in risk register)"
    
    local errors=0
    
    # Check if threat modeling is scheduled in risk register
    if [[ -f "docs/risk-register.yaml" ]]; then
        if grep -q "threat_modeling:" docs/risk-register.yaml; then
            log_success "Threat modeling section found in risk register"
            
            # Check for required threat modeling fields
            local required_fields=("session_date" "owner" "participants" "scope")
            for field in "${required_fields[@]}"; do
                if grep -A 10 "threat_modeling:" docs/risk-register.yaml | grep -q "$field:"; then
                    log_success "Threat modeling has '$field' field"
                else
                    log_error "Threat modeling missing '$field' field"
                    ((errors++))
                fi
            done
            
            # Check if session date is in the future or recent past
            local session_date=$(grep -A 10 "threat_modeling:" docs/risk-register.yaml | grep "session_date:" | cut -d'"' -f2)
            if [[ -n "$session_date" ]]; then
                log_success "Threat modeling session date: $session_date"
            else
                log_error "Threat modeling session date not properly formatted"
                ((errors++))
            fi
        else
            log_error "Threat modeling section missing from risk register"
            ((errors++))
        fi
    else
        log_error "Risk register missing (required for threat modeling validation)"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Gate G0.9: Threat model kickoff scheduled - PASSED"
    else
        log_error "Gate G0.9: Threat model kickoff scheduled - FAILED ($errors errors)"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + errors))
    fi
    
    return $errors
}

# Main validation function
main() {
    log_info "Starting Gate G0 Exit Criteria Validation"
    log_info "Spec: Q1.1 Foundations & Project Governance"
    log_info "Date: $(date)"
    echo
    
    # Run all validations
    validate_ci_green
    echo
    validate_cross_platform_builds
    echo
    validate_devcontainer_adoption
    echo
    validate_sbom_provenance
    echo
    validate_signed_multiarch_images
    echo
    validate_risk_register_adr
    echo
    validate_release_versioning
    echo
    validate_interface_freeze
    echo
    validate_threat_modeling
    echo
    
    # Summary
    log_info "Gate G0 Validation Summary"
    log_info "=========================="
    
    if [[ $VALIDATION_ERRORS -eq 0 ]]; then
        log_success "✅ All Gate G0 criteria PASSED"
        log_success "✅ Foundation is ready for Q1.2 development"
        if [[ $VALIDATION_WARNINGS -gt 0 ]]; then
            log_warning "⚠️  $VALIDATION_WARNINGS warnings found (non-blocking)"
        fi
        echo
        log_success "🎉 Gate G0 VALIDATION SUCCESSFUL"
        exit 0
    else
        log_error "❌ $VALIDATION_ERRORS Gate G0 criteria FAILED"
        if [[ $VALIDATION_WARNINGS -gt 0 ]]; then
            log_warning "⚠️  $VALIDATION_WARNINGS warnings found"
        fi
        echo
        log_error "🚫 Gate G0 VALIDATION FAILED"
        log_error "Foundation is NOT ready for Q1.2 development"
        log_error "Please address the errors above before proceeding"
        exit 1
    fi
}

# Run main function
main "$@"