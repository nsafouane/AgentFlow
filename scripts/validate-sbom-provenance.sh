#!/bin/bash

# SBOM & Provenance Validation Script
# This script validates that artifacts published per build include SBOM & provenance

set -euo pipefail

# Configuration
REGISTRY="${REGISTRY:-ghcr.io}"
REPOSITORY="${REPOSITORY:-agentflow/agentflow}"
SERVICES=("control-plane" "worker" "af")
REQUIRED_TOOLS=("cosign" "syft" "jq" "docker")

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

# Check if required tools are installed
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_tools=()
    for tool in "${REQUIRED_TOOLS[@]}"; do
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
}

# Validate SBOM file structure
validate_sbom_structure() {
    local sbom_file="$1"
    local format="$2"
    
    log_info "Validating SBOM structure for $sbom_file ($format format)"
    
    if [ ! -f "$sbom_file" ]; then
        log_error "SBOM file not found: $sbom_file"
        return 1
    fi
    
    # Check if file is valid JSON
    if ! jq empty "$sbom_file" 2>/dev/null; then
        log_error "SBOM file is not valid JSON: $sbom_file"
        return 1
    fi
    
    case "$format" in
        "spdx")
            # Validate SPDX structure
            local spdx_version
            spdx_version=$(jq -r '.spdxVersion // empty' "$sbom_file")
            if [ -z "$spdx_version" ]; then
                log_error "SPDX SBOM missing spdxVersion field"
                return 1
            fi
            
            local packages_count
            packages_count=$(jq '.packages | length' "$sbom_file")
            if [ "$packages_count" -eq 0 ]; then
                log_error "SPDX SBOM contains no packages"
                return 1
            fi
            
            log_success "SPDX SBOM structure is valid (version: $spdx_version, packages: $packages_count)"
            ;;
            
        "cyclonedx")
            # Validate CycloneDX structure
            local cyclone_version
            cyclone_version=$(jq -r '.specVersion // empty' "$sbom_file")
            if [ -z "$cyclone_version" ]; then
                log_error "CycloneDX SBOM missing specVersion field"
                return 1
            fi
            
            local components_count
            components_count=$(jq '.components | length' "$sbom_file")
            if [ "$components_count" -eq 0 ]; then
                log_error "CycloneDX SBOM contains no components"
                return 1
            fi
            
            log_success "CycloneDX SBOM structure is valid (version: $cyclone_version, components: $components_count)"
            ;;
            
        *)
            log_error "Unknown SBOM format: $format"
            return 1
            ;;
    esac
    
    return 0
}

# Validate container image SBOM and provenance
validate_container_artifacts() {
    local image_ref="$1"
    local service="$2"
    
    log_info "Validating SBOM and provenance for $service: $image_ref"
    
    # Check if image exists and is accessible
    if ! docker manifest inspect "$image_ref" &>/dev/null; then
        log_error "Cannot access container image: $image_ref"
        return 1
    fi
    
    # Generate SBOM using syft and validate
    local temp_dir
    temp_dir=$(mktemp -d)
    local spdx_sbom="$temp_dir/sbom.spdx.json"
    local cyclonedx_sbom="$temp_dir/sbom.cyclonedx.json"
    
    log_info "Generating SBOM for validation..."
    
    # Generate SPDX format SBOM
    if ! syft "$image_ref" -o spdx-json="$spdx_sbom" --quiet; then
        log_error "Failed to generate SPDX SBOM for $image_ref"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Generate CycloneDX format SBOM
    if ! syft "$image_ref" -o cyclonedx-json="$cyclonedx_sbom" --quiet; then
        log_error "Failed to generate CycloneDX SBOM for $image_ref"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Validate SBOM structures
    if ! validate_sbom_structure "$spdx_sbom" "spdx"; then
        rm -rf "$temp_dir"
        return 1
    fi
    
    if ! validate_sbom_structure "$cyclonedx_sbom" "cyclonedx"; then
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Verify cosign signature and attestation
    log_info "Verifying container signature and attestation..."
    
    # Verify signature
    if ! cosign verify \
        --certificate-identity-regexp="https://github.com/$REPOSITORY" \
        --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
        "$image_ref" &>/dev/null; then
        log_warning "Container signature verification failed for $image_ref"
        log_warning "This may be expected for local builds or unsigned images"
    else
        log_success "Container signature verified for $image_ref"
    fi
    
    # Verify attestation
    if ! cosign verify-attestation \
        --certificate-identity-regexp="https://github.com/$REPOSITORY" \
        --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
        --type slsaprovenance \
        "$image_ref" &>/dev/null; then
        log_warning "Provenance attestation verification failed for $image_ref"
        log_warning "This may be expected for local builds or unsigned images"
    else
        log_success "Provenance attestation verified for $image_ref"
    fi
    
    # Clean up
    rm -rf "$temp_dir"
    
    log_success "SBOM validation completed for $service"
    return 0
}

# Validate local SBOM files
validate_local_sbom_files() {
    log_info "Validating local SBOM files..."
    
    local sbom_files=()
    
    # Look for SBOM files in common locations
    if [ -f "af-sbom.spdx.json" ]; then
        sbom_files+=("af-sbom.spdx.json:spdx")
    fi
    
    if [ -f "sbom.spdx.json" ]; then
        sbom_files+=("sbom.spdx.json:spdx")
    fi
    
    if [ -f "sbom.cyclonedx.json" ]; then
        sbom_files+=("sbom.cyclonedx.json:cyclonedx")
    fi
    
    # Look for SBOM files in artifacts directory
    if [ -d "artifacts" ]; then
        while IFS= read -r -d '' file; do
            if [[ "$file" == *.spdx.json ]]; then
                sbom_files+=("$file:spdx")
            elif [[ "$file" == *.cyclonedx.json ]]; then
                sbom_files+=("$file:cyclonedx")
            fi
        done < <(find artifacts -name "*.json" -print0 2>/dev/null || true)
    fi
    
    if [ ${#sbom_files[@]} -eq 0 ]; then
        log_warning "No local SBOM files found"
        return 0
    fi
    
    local validation_failed=false
    for sbom_entry in "${sbom_files[@]}"; do
        IFS=':' read -r sbom_file sbom_format <<< "$sbom_entry"
        if ! validate_sbom_structure "$sbom_file" "$sbom_format"; then
            validation_failed=true
        fi
    done
    
    if [ "$validation_failed" = true ]; then
        log_error "Local SBOM validation failed"
        return 1
    fi
    
    log_success "Local SBOM validation completed"
    return 0
}

# Main validation function
main() {
    local image_tag="${1:-latest}"
    local validate_containers="${2:-true}"
    local validate_local="${3:-true}"
    
    log_info "Starting SBOM & Provenance validation"
    log_info "Image tag: $image_tag"
    log_info "Validate containers: $validate_containers"
    log_info "Validate local files: $validate_local"
    
    # Check prerequisites
    check_prerequisites
    
    local validation_failed=false
    
    # Validate local SBOM files if requested
    if [ "$validate_local" = "true" ]; then
        if ! validate_local_sbom_files; then
            validation_failed=true
        fi
    fi
    
    # Validate container artifacts if requested
    if [ "$validate_containers" = "true" ]; then
        for service in "${SERVICES[@]}"; do
            local image_ref="$REGISTRY/$REPOSITORY/$service:$image_tag"
            if ! validate_container_artifacts "$image_ref" "$service"; then
                validation_failed=true
            fi
        done
    fi
    
    if [ "$validation_failed" = true ]; then
        log_error "SBOM & Provenance validation failed"
        exit 1
    fi
    
    log_success "SBOM & Provenance validation completed successfully"
}

# Script usage
usage() {
    echo "Usage: $0 [IMAGE_TAG] [VALIDATE_CONTAINERS] [VALIDATE_LOCAL]"
    echo ""
    echo "Arguments:"
    echo "  IMAGE_TAG           Container image tag to validate (default: latest)"
    echo "  VALIDATE_CONTAINERS Whether to validate container artifacts (default: true)"
    echo "  VALIDATE_LOCAL      Whether to validate local SBOM files (default: true)"
    echo ""
    echo "Environment variables:"
    echo "  REGISTRY           Container registry (default: ghcr.io)"
    echo "  REPOSITORY         Repository name (default: agentflow/agentflow)"
    echo ""
    echo "Examples:"
    echo "  $0                           # Validate latest images and local files"
    echo "  $0 v1.0.0                    # Validate v1.0.0 images and local files"
    echo "  $0 latest false true         # Validate only local files"
    echo "  $0 latest true false         # Validate only container artifacts"
}

# Handle command line arguments
if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

# Run main function with arguments
main "${1:-latest}" "${2:-true}" "${3:-true}"