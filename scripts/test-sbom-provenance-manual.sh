#!/bin/bash

# Manual Testing Script for SBOM & Provenance Validation
# This script provides step-by-step manual testing procedures

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REGISTRY="${REGISTRY:-ghcr.io}"
REPOSITORY="${REPOSITORY:-agentflow/agentflow}"
SERVICES=("control-plane" "worker" "af")

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

log_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

# Wait for user confirmation
wait_for_confirmation() {
    local message="$1"
    echo -e "${BLUE}[CONFIRM]${NC} $message"
    read -p "Press Enter to continue or Ctrl+C to abort..."
}

# Test 1: Verify published artifacts contain SBOM metadata
test_published_artifacts_sbom() {
    log_step "Test 1: Verify published artifacts contain SBOM metadata"
    
    local image_tag="${1:-latest}"
    
    for service in "${SERVICES[@]}"; do
        local image_ref="$REGISTRY/$REPOSITORY/$service:$image_tag"
        
        log_info "Testing service: $service"
        log_info "Image reference: $image_ref"
        
        wait_for_confirmation "About to pull and inspect image manifest for $service"
        
        # Pull the image
        if docker pull "$image_ref"; then
            log_success "Successfully pulled $image_ref"
        else
            log_error "Failed to pull $image_ref"
            continue
        fi
        
        # Inspect the image
        log_info "Inspecting image manifest..."
        docker manifest inspect "$image_ref" | jq '.' || {
            log_error "Failed to inspect manifest for $image_ref"
            continue
        }
        
        # Generate SBOM using syft
        log_info "Generating SBOM for verification..."
        local temp_sbom="/tmp/test-sbom-$service.spdx.json"
        
        if syft "$image_ref" -o spdx-json="$temp_sbom" --quiet; then
            log_success "Generated SBOM for $service"
            
            # Display SBOM summary
            log_info "SBOM Summary:"
            echo "  - SPDX Version: $(jq -r '.spdxVersion' "$temp_sbom")"
            echo "  - Packages: $(jq '.packages | length' "$temp_sbom")"
            echo "  - Files: $(jq '.files | length' "$temp_sbom")"
            echo "  - Relationships: $(jq '.relationships | length' "$temp_sbom")"
            
            # Show first few packages
            log_info "First 3 packages in SBOM:"
            jq -r '.packages[0:3][] | "  - \(.name) (\(.versionInfo // "no version"))"' "$temp_sbom"
            
            rm -f "$temp_sbom"
        else
            log_error "Failed to generate SBOM for $service"
        fi
        
        echo ""
    done
}

# Test 2: Verify provenance attestation
test_provenance_attestation() {
    log_step "Test 2: Verify provenance attestation"
    
    local image_tag="${1:-latest}"
    
    for service in "${SERVICES[@]}"; do
        local image_ref="$REGISTRY/$REPOSITORY/$service:$image_tag"
        
        log_info "Testing provenance for service: $service"
        log_info "Image reference: $image_ref"
        
        wait_for_confirmation "About to verify provenance attestation for $service"
        
        # Verify attestation exists
        log_info "Checking for provenance attestation..."
        if cosign verify-attestation \
            --certificate-identity-regexp="https://github.com/$REPOSITORY" \
            --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
            --type slsaprovenance \
            "$image_ref" 2>/dev/null; then
            log_success "Provenance attestation verified for $service"
            
            # Display attestation details
            log_info "Attestation details:"
            cosign verify-attestation \
                --certificate-identity-regexp="https://github.com/$REPOSITORY" \
                --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
                --type slsaprovenance \
                "$image_ref" 2>/dev/null | jq -r '.payload' | base64 -d | jq '.'
        else
            log_warning "Provenance attestation verification failed for $service"
            log_warning "This may be expected for local builds or unsigned images"
        fi
        
        echo ""
    done
}

# Test 3: Verify container signatures
test_container_signatures() {
    log_step "Test 3: Verify container signatures"
    
    local image_tag="${1:-latest}"
    
    for service in "${SERVICES[@]}"; do
        local image_ref="$REGISTRY/$REPOSITORY/$service:$image_tag"
        
        log_info "Testing signature for service: $service"
        log_info "Image reference: $image_ref"
        
        wait_for_confirmation "About to verify container signature for $service"
        
        # Verify signature
        log_info "Verifying container signature..."
        if cosign verify \
            --certificate-identity-regexp="https://github.com/$REPOSITORY" \
            --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
            "$image_ref" 2>/dev/null; then
            log_success "Container signature verified for $service"
            
            # Display signature details
            log_info "Signature verification output:"
            cosign verify \
                --certificate-identity-regexp="https://github.com/$REPOSITORY" \
                --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
                "$image_ref" 2>/dev/null | jq '.'
        else
            log_warning "Container signature verification failed for $service"
            log_warning "This may be expected for local builds or unsigned images"
        fi
        
        echo ""
    done
}

# Test 4: Verify local SBOM files
test_local_sbom_files() {
    log_step "Test 4: Verify local SBOM files"
    
    wait_for_confirmation "About to check for local SBOM files"
    
    # Check for existing SBOM files
    local sbom_files=()
    
    if [ -f "af-sbom.spdx.json" ]; then
        sbom_files+=("af-sbom.spdx.json")
        log_info "Found: af-sbom.spdx.json"
    fi
    
    if [ -f "sbom.spdx.json" ]; then
        sbom_files+=("sbom.spdx.json")
        log_info "Found: sbom.spdx.json"
    fi
    
    if [ -f "sbom.cyclonedx.json" ]; then
        sbom_files+=("sbom.cyclonedx.json")
        log_info "Found: sbom.cyclonedx.json"
    fi
    
    # Look in artifacts directory
    if [ -d "artifacts" ]; then
        while IFS= read -r -d '' file; do
            if [[ "$file" == *.spdx.json ]] || [[ "$file" == *.cyclonedx.json ]]; then
                sbom_files+=("$file")
                log_info "Found: $file"
            fi
        done < <(find artifacts -name "*.json" -print0 2>/dev/null || true)
    fi
    
    if [ ${#sbom_files[@]} -eq 0 ]; then
        log_warning "No local SBOM files found"
        return 0
    fi
    
    # Validate each SBOM file
    for sbom_file in "${sbom_files[@]}"; do
        log_info "Validating: $sbom_file"
        
        # Check if file is valid JSON
        if jq empty "$sbom_file" 2>/dev/null; then
            log_success "$sbom_file is valid JSON"
            
            # Determine format and validate structure
            if jq -e '.spdxVersion' "$sbom_file" >/dev/null 2>&1; then
                log_info "Detected SPDX format"
                local spdx_version
                spdx_version=$(jq -r '.spdxVersion' "$sbom_file")
                local packages_count
                packages_count=$(jq '.packages | length' "$sbom_file")
                
                echo "  - SPDX Version: $spdx_version"
                echo "  - Packages: $packages_count"
                
                if [ "$packages_count" -gt 0 ]; then
                    log_success "SPDX SBOM structure is valid"
                else
                    log_error "SPDX SBOM contains no packages"
                fi
                
            elif jq -e '.specVersion' "$sbom_file" >/dev/null 2>&1; then
                log_info "Detected CycloneDX format"
                local spec_version
                spec_version=$(jq -r '.specVersion' "$sbom_file")
                local components_count
                components_count=$(jq '.components | length' "$sbom_file")
                
                echo "  - Spec Version: $spec_version"
                echo "  - Components: $components_count"
                
                if [ "$components_count" -gt 0 ]; then
                    log_success "CycloneDX SBOM structure is valid"
                else
                    log_error "CycloneDX SBOM contains no components"
                fi
            else
                log_warning "Unknown SBOM format for $sbom_file"
            fi
        else
            log_error "$sbom_file is not valid JSON"
        fi
        
        echo ""
    done
}

# Test 5: Generate and validate new SBOM
test_sbom_generation() {
    log_step "Test 5: Generate and validate new SBOM"
    
    wait_for_confirmation "About to generate new SBOM for current project"
    
    local temp_dir
    temp_dir=$(mktemp -d)
    local spdx_sbom="$temp_dir/new-sbom.spdx.json"
    local cyclonedx_sbom="$temp_dir/new-sbom.cyclonedx.json"
    
    # Generate SPDX SBOM
    log_info "Generating SPDX SBOM..."
    if syft . -o spdx-json="$spdx_sbom" --quiet; then
        log_success "Generated SPDX SBOM"
        
        # Validate structure
        local spdx_version
        spdx_version=$(jq -r '.spdxVersion' "$spdx_sbom")
        local packages_count
        packages_count=$(jq '.packages | length' "$spdx_sbom")
        
        echo "  - SPDX Version: $spdx_version"
        echo "  - Packages: $packages_count"
        
        # Show sample packages
        log_info "Sample packages:"
        jq -r '.packages[0:5][] | "  - \(.name) (\(.versionInfo // "no version"))"' "$spdx_sbom"
    else
        log_error "Failed to generate SPDX SBOM"
    fi
    
    # Generate CycloneDX SBOM
    log_info "Generating CycloneDX SBOM..."
    if syft . -o cyclonedx-json="$cyclonedx_sbom" --quiet; then
        log_success "Generated CycloneDX SBOM"
        
        # Validate structure
        local spec_version
        spec_version=$(jq -r '.specVersion' "$cyclonedx_sbom")
        local components_count
        components_count=$(jq '.components | length' "$cyclonedx_sbom")
        
        echo "  - Spec Version: $spec_version"
        echo "  - Components: $components_count"
        
        # Show sample components
        log_info "Sample components:"
        jq -r '.components[0:5][] | "  - \(.name) (\(.version // "no version"))"' "$cyclonedx_sbom"
    else
        log_error "Failed to generate CycloneDX SBOM"
    fi
    
    # Clean up
    rm -rf "$temp_dir"
}

# Test 6: Verify CI/CD integration
test_cicd_integration() {
    log_step "Test 6: Verify CI/CD integration"
    
    wait_for_confirmation "About to check CI/CD workflow configuration"
    
    # Check GitHub Actions workflows
    local workflows_dir=".github/workflows"
    
    if [ ! -d "$workflows_dir" ]; then
        log_error "GitHub Actions workflows directory not found"
        return 1
    fi
    
    # Check for SBOM and provenance in workflows
    local workflow_files=("ci.yml" "container-build.yml" "release.yml")
    
    for workflow in "${workflow_files[@]}"; do
        local workflow_file="$workflows_dir/$workflow"
        
        if [ -f "$workflow_file" ]; then
            log_info "Checking workflow: $workflow"
            
            # Check for SBOM generation
            if grep -q "sbom.*true\|syft" "$workflow_file"; then
                log_success "SBOM generation found in $workflow"
            else
                log_warning "SBOM generation not found in $workflow"
            fi
            
            # Check for provenance
            if grep -q "provenance.*true\|attestation" "$workflow_file"; then
                log_success "Provenance attestation found in $workflow"
            else
                log_warning "Provenance attestation not found in $workflow"
            fi
            
            # Check for cosign signing
            if grep -q "cosign" "$workflow_file"; then
                log_success "Cosign signing found in $workflow"
            else
                log_warning "Cosign signing not found in $workflow"
            fi
        else
            log_warning "Workflow file not found: $workflow_file"
        fi
        
        echo ""
    done
}

# Main test execution
main() {
    local image_tag="${1:-latest}"
    local test_selection="${2:-all}"
    
    log_info "Starting manual SBOM & Provenance validation tests"
    log_info "Image tag: $image_tag"
    log_info "Test selection: $test_selection"
    
    echo ""
    
    case "$test_selection" in
        "all")
            test_published_artifacts_sbom "$image_tag"
            test_provenance_attestation "$image_tag"
            test_container_signatures "$image_tag"
            test_local_sbom_files
            test_sbom_generation
            test_cicd_integration
            ;;
        "artifacts")
            test_published_artifacts_sbom "$image_tag"
            ;;
        "provenance")
            test_provenance_attestation "$image_tag"
            ;;
        "signatures")
            test_container_signatures "$image_tag"
            ;;
        "local")
            test_local_sbom_files
            ;;
        "generation")
            test_sbom_generation
            ;;
        "cicd")
            test_cicd_integration
            ;;
        *)
            log_error "Unknown test selection: $test_selection"
            echo "Available options: all, artifacts, provenance, signatures, local, generation, cicd"
            exit 1
            ;;
    esac
    
    log_success "Manual testing completed"
}

# Show usage
usage() {
    echo "Usage: $0 [IMAGE_TAG] [TEST_SELECTION]"
    echo ""
    echo "Arguments:"
    echo "  IMAGE_TAG       Container image tag to test (default: latest)"
    echo "  TEST_SELECTION  Which tests to run (default: all)"
    echo ""
    echo "Test selections:"
    echo "  all         Run all tests"
    echo "  artifacts   Test published artifacts SBOM"
    echo "  provenance  Test provenance attestation"
    echo "  signatures  Test container signatures"
    echo "  local       Test local SBOM files"
    echo "  generation  Test SBOM generation"
    echo "  cicd        Test CI/CD integration"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run all tests with latest tag"
    echo "  $0 v1.0.0             # Run all tests with v1.0.0 tag"
    echo "  $0 latest artifacts   # Test only published artifacts"
    echo "  $0 latest local       # Test only local SBOM files"
}

# Handle command line arguments
if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

# Check prerequisites
log_info "Checking prerequisites..."
required_tools=("docker" "jq" "syft" "cosign")
missing_tools=()

for tool in "${required_tools[@]}"; do
    if ! command -v "$tool" &> /dev/null; then
        missing_tools+=("$tool")
    fi
done

if [ ${#missing_tools[@]} -ne 0 ]; then
    log_error "Missing required tools: ${missing_tools[*]}"
    log_info "Please install the missing tools and try again."
    exit 1
fi

log_success "All required tools are available"
echo ""

# Run main function
main "${1:-latest}" "${2:-all}"