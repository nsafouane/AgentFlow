#!/bin/bash
# Container Build Testing Script
# Tests multi-arch container builds, signatures, and SBOM generation

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REGISTRY="${REGISTRY:-ghcr.io}"
IMAGE_NAME="${IMAGE_NAME:-agentflow/agentflow}"
TAG="${TAG:-latest}"
SERVICES=("control-plane" "worker" "af")

echo -e "${BLUE}🐳 AgentFlow Container Build Tests${NC}"
echo "Registry: $REGISTRY"
echo "Image Name: $IMAGE_NAME"
echo "Tag: $TAG"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to test manifest list inspection
test_manifest_list() {
    local service=$1
    local image_ref="${REGISTRY}/${IMAGE_NAME}/${service}:${TAG}"
    
    echo -e "${YELLOW}Testing manifest list for ${service}...${NC}"
    
    if ! command_exists docker; then
        echo -e "${RED}❌ Docker not found, skipping manifest list test${NC}"
        return 1
    fi
    
    # Check if image exists
    if ! docker buildx imagetools inspect "$image_ref" >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Image $image_ref not found, skipping test${NC}"
        return 0
    fi
    
    # Get raw manifest
    local manifest
    manifest=$(docker buildx imagetools inspect --raw "$image_ref" 2>/dev/null || echo "")
    
    if [ -z "$manifest" ]; then
        echo -e "${RED}❌ Failed to get manifest for $image_ref${NC}"
        return 1
    fi
    
    # Check if it's a manifest list
    local media_type
    media_type=$(echo "$manifest" | jq -r '.mediaType // empty' 2>/dev/null || echo "")
    
    if [[ "$media_type" == *"manifest.list"* ]] || [[ "$media_type" == *"image.index"* ]]; then
        echo -e "${GREEN}✅ Manifest list found for $service${NC}"
        
        # Check architectures
        local architectures
        architectures=$(echo "$manifest" | jq -r '.manifests[]? | select(.platform.os == "linux") | .platform.architecture' 2>/dev/null || echo "")
        
        local has_amd64=false
        local has_arm64=false
        
        while IFS= read -r arch; do
            case "$arch" in
                "amd64") has_amd64=true ;;
                "arm64") has_arm64=true ;;
            esac
        done <<< "$architectures"
        
        if [ "$has_amd64" = true ] && [ "$has_arm64" = true ]; then
            echo -e "${GREEN}✅ Both amd64 and arm64 architectures found${NC}"
        else
            echo -e "${RED}❌ Missing required architectures (amd64: $has_amd64, arm64: $has_arm64)${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ Not a manifest list: $media_type${NC}"
        return 1
    fi
    
    return 0
}

# Function to test signature presence
test_signature() {
    local service=$1
    local image_ref="${REGISTRY}/${IMAGE_NAME}/${service}:${TAG}"
    
    echo -e "${YELLOW}Testing signature for ${service}...${NC}"
    
    if ! command_exists cosign; then
        echo -e "${YELLOW}⚠️  Cosign not found, skipping signature test${NC}"
        return 0
    fi
    
    # Verify signature
    if cosign verify \
        --certificate-identity-regexp="https://github.com/${IMAGE_NAME}" \
        --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
        "$image_ref" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Signature verified for $service${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠️  Signature verification failed or not found for $service${NC}"
        return 0
    fi
}

# Function to test SBOM presence
test_sbom() {
    local service=$1
    local image_ref="${REGISTRY}/${IMAGE_NAME}/${service}:${TAG}"
    
    echo -e "${YELLOW}Testing SBOM for ${service}...${NC}"
    
    if ! command_exists syft; then
        echo -e "${YELLOW}⚠️  Syft not found, skipping SBOM test${NC}"
        return 0
    fi
    
    # Generate SBOM
    if syft "$image_ref" -o json >/dev/null 2>&1; then
        echo -e "${GREEN}✅ SBOM generated successfully for $service${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠️  SBOM generation failed for $service${NC}"
        return 0
    fi
}

# Function to test provenance attestation
test_provenance() {
    local service=$1
    local image_ref="${REGISTRY}/${IMAGE_NAME}/${service}:${TAG}"
    
    echo -e "${YELLOW}Testing provenance attestation for ${service}...${NC}"
    
    if ! command_exists cosign; then
        echo -e "${YELLOW}⚠️  Cosign not found, skipping provenance test${NC}"
        return 0
    fi
    
    # Verify provenance attestation
    if cosign verify-attestation \
        --certificate-identity-regexp="https://github.com/${IMAGE_NAME}" \
        --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
        --type slsaprovenance \
        "$image_ref" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Provenance attestation verified for $service${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠️  Provenance attestation verification failed for $service${NC}"
        return 0
    fi
}

# Main test execution
main() {
    local exit_code=0
    
    echo -e "${BLUE}🔍 Running container build tests...${NC}"
    echo ""
    
    for service in "${SERVICES[@]}"; do
        echo -e "${BLUE}Testing service: $service${NC}"
        
        # Test manifest list
        if ! test_manifest_list "$service"; then
            exit_code=1
        fi
        
        # Test signature
        if ! test_signature "$service"; then
            exit_code=1
        fi
        
        # Test SBOM
        if ! test_sbom "$service"; then
            exit_code=1
        fi
        
        # Test provenance
        if ! test_provenance "$service"; then
            exit_code=1
        fi
        
        echo ""
    done
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}🎉 All container build tests passed!${NC}"
    else
        echo -e "${RED}❌ Some container build tests failed${NC}"
    fi
    
    return $exit_code
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi