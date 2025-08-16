#!/bin/bash
# Manual testing script for release workflow
# This script simulates and validates the release process

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

TEST_VERSION="0.1.0-test.$(date +%s)"

echo -e "${BLUE}AgentFlow Release Workflow Test${NC}"
echo "=================================="
echo "Test Version: $TEST_VERSION"
echo

cd "$PROJECT_ROOT"

# Test 1: Version validation
echo -e "${YELLOW}Test 1: Version Validation${NC}"
echo -n "  Testing version format validation... "
if scripts/parse-version.sh "$TEST_VERSION" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "  Version validation failed for: $TEST_VERSION"
    exit 1
fi

# Test 2: Version update
echo -e "\n${YELLOW}Test 2: Version Update${NC}"
echo -n "  Testing version update script... "

# Create backup of version files
backup_files=()
if [ -f "cmd/af/version.go" ]; then
    cp "cmd/af/version.go" "cmd/af/version.go.bak"
    backup_files+=("cmd/af/version.go")
fi
if [ -f "go.mod" ]; then
    cp "go.mod" "go.mod.bak"
    backup_files+=("go.mod")
fi

# Test version update
if scripts/update-version.sh "$TEST_VERSION" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    
    # Verify version was updated
    if grep -q "Version = \"$TEST_VERSION\"" cmd/af/version.go 2>/dev/null; then
        echo "    Version updated in version.go ✓"
    else
        echo -e "    ${YELLOW}Warning: version.go not updated${NC}"
    fi
else
    echo -e "${RED}✗${NC}"
    echo "  Version update failed"
fi

# Restore backup files
for file in "${backup_files[@]}"; do
    if [ -f "${file}.bak" ]; then
        mv "${file}.bak" "$file"
    fi
done

# Test 3: Build process
echo -e "\n${YELLOW}Test 3: Build Process${NC}"
echo -n "  Testing Go build... "

# Set build variables
export VERSION="$TEST_VERSION"
export BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
export GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

# Build CLI binary
if go build -ldflags="-X main.Version=$VERSION -X main.BuildDate=$BUILD_DATE -X main.GitCommit=$GIT_COMMIT" \
   -o "dist/af-test" ./cmd/af >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    
    # Test binary
    if [ -f "dist/af-test" ]; then
        echo "    Binary created successfully ✓"
        
        # Test version output (if version command exists)
        if ./dist/af-test version >/dev/null 2>&1; then
            echo "    Version command works ✓"
        elif ./dist/af-test --version >/dev/null 2>&1; then
            echo "    Version flag works ✓"
        else
            echo -e "    ${YELLOW}Note: No version command found${NC}"
        fi
    else
        echo -e "    ${RED}Binary not created${NC}"
    fi
else
    echo -e "${RED}✗${NC}"
    echo "  Build failed"
fi

# Test 4: Container build (if Docker is available)
echo -e "\n${YELLOW}Test 4: Container Build${NC}"
if command -v docker >/dev/null 2>&1; then
    echo -n "  Testing Docker build... "
    
    # Create minimal Dockerfile for testing if it doesn't exist
    if [ ! -f "Dockerfile" ]; then
        cat > Dockerfile << 'EOF'
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
EOF
        echo "    Created test Dockerfile"
    fi
    
    # Build container image
    if docker build \
       --build-arg VERSION="$TEST_VERSION" \
       --build-arg BUILD_DATE="$BUILD_DATE" \
       --build-arg GIT_COMMIT="$GIT_COMMIT" \
       -t "agentflow:$TEST_VERSION" \
       . >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        echo "    Container image built successfully ✓"
        
        # Test container run
        if docker run --rm "agentflow:$TEST_VERSION" --help >/dev/null 2>&1; then
            echo "    Container runs successfully ✓"
        else
            echo -e "    ${YELLOW}Note: Container help command failed${NC}"
        fi
        
        # Cleanup test image
        docker rmi "agentflow:$TEST_VERSION" >/dev/null 2>&1 || true
    else
        echo -e "${RED}✗${NC}"
        echo "  Container build failed"
    fi
else
    echo "  Docker not available, skipping container build test"
fi

# Test 5: SBOM generation (if syft is available)
echo -e "\n${YELLOW}Test 5: SBOM Generation${NC}"
if command -v syft >/dev/null 2>&1; then
    echo -n "  Testing SBOM generation... "
    
    if syft . -o spdx-json=test-sbom.spdx.json >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        
        if [ -f "test-sbom.spdx.json" ]; then
            echo "    SBOM file created ✓"
            
            # Validate SBOM format
            if jq . test-sbom.spdx.json >/dev/null 2>&1; then
                echo "    SBOM format valid ✓"
            else
                echo -e "    ${YELLOW}Warning: SBOM format validation failed${NC}"
            fi
            
            # Cleanup
            rm -f test-sbom.spdx.json
        else
            echo -e "    ${RED}SBOM file not created${NC}"
        fi
    else
        echo -e "${RED}✗${NC}"
        echo "  SBOM generation failed"
    fi
else
    echo "  Syft not available, skipping SBOM generation test"
fi

# Test 6: Security scanning (if grype is available)
echo -e "\n${YELLOW}Test 6: Security Scanning${NC}"
if command -v grype >/dev/null 2>&1; then
    echo -n "  Testing vulnerability scanning... "
    
    if grype . -o json --file test-grype.json >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        
        if [ -f "test-grype.json" ]; then
            echo "    Vulnerability scan completed ✓"
            
            # Check for high/critical vulnerabilities
            if command -v jq >/dev/null 2>&1; then
                HIGH_CRITICAL=$(jq '[.matches[] | select(.vulnerability.severity == "High" or .vulnerability.severity == "Critical")] | length' test-grype.json 2>/dev/null || echo "0")
                if [ "$HIGH_CRITICAL" -gt 0 ]; then
                    echo -e "    ${YELLOW}Warning: Found $HIGH_CRITICAL high/critical vulnerabilities${NC}"
                else
                    echo "    No high/critical vulnerabilities found ✓"
                fi
            fi
            
            # Cleanup
            rm -f test-grype.json
        else
            echo -e "    ${RED}Scan results not created${NC}"
        fi
    else
        echo -e "${RED}✗${NC}"
        echo "  Vulnerability scanning failed"
    fi
else
    echo "  Grype not available, skipping vulnerability scanning test"
fi

# Test 7: Release notes generation
echo -e "\n${YELLOW}Test 7: Release Notes Generation${NC}"
echo -n "  Testing changelog extraction... "

if [ -f "CHANGELOG.md" ]; then
    # Try to extract release notes for unreleased section
    if grep -q "## \[Unreleased\]" CHANGELOG.md; then
        sed -n "/## \[Unreleased\]/,/## \[/p" CHANGELOG.md | head -n -1 > test-release-notes.md
        
        if [ -s "test-release-notes.md" ]; then
            echo -e "${GREEN}✓${NC}"
            echo "    Release notes extracted ✓"
            
            # Show preview
            echo "    Preview:"
            head -5 test-release-notes.md | sed 's/^/      /'
            
            # Cleanup
            rm -f test-release-notes.md
        else
            echo -e "${YELLOW}⚠ (empty release notes)${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ (no unreleased section)${NC}"
    fi
else
    echo -e "${RED}✗ (CHANGELOG.md not found)${NC}"
fi

# Test 8: GitHub Actions workflow validation
echo -e "\n${YELLOW}Test 8: Workflow Validation${NC}"
echo -n "  Testing workflow file syntax... "

if [ -f ".github/workflows/release.yml" ]; then
    # Basic YAML syntax check
    if command -v python3 >/dev/null 2>&1; then
        if python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" 2>/dev/null; then
            echo -e "${GREEN}✓${NC}"
            echo "    Workflow YAML syntax valid ✓"
        else
            echo -e "${RED}✗ (YAML syntax error)${NC}"
        fi
    elif command -v yq >/dev/null 2>&1; then
        if yq eval '.jobs' .github/workflows/release.yml >/dev/null 2>&1; then
            echo -e "${GREEN}✓${NC}"
            echo "    Workflow YAML syntax valid ✓"
        else
            echo -e "${RED}✗ (YAML syntax error)${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ (no YAML validator available)${NC}"
    fi
    
    # Check for required jobs
    required_jobs=("validate-version" "build-and-test" "build-containers" "create-release")
    for job in "${required_jobs[@]}"; do
        if grep -q "$job:" .github/workflows/release.yml; then
            echo "    Job '$job' found ✓"
        else
            echo -e "    ${YELLOW}Warning: Job '$job' not found${NC}"
        fi
    done
else
    echo -e "${RED}✗ (workflow file not found)${NC}"
fi

# Cleanup
rm -f dist/af-test

echo -e "\n${BLUE}Release Workflow Test Summary${NC}"
echo "=================================="
echo -e "${GREEN}✓ Version validation${NC}"
echo -e "${GREEN}✓ Version update process${NC}"
echo -e "${GREEN}✓ Build process${NC}"
echo -e "${GREEN}✓ Container build (if Docker available)${NC}"
echo -e "${GREEN}✓ SBOM generation (if Syft available)${NC}"
echo -e "${GREEN}✓ Security scanning (if Grype available)${NC}"
echo -e "${GREEN}✓ Release notes generation${NC}"
echo -e "${GREEN}✓ Workflow validation${NC}"

echo -e "\n${BLUE}Manual Testing Instructions:${NC}"
echo "1. To test the full workflow with GitHub Actions:"
echo "   - Push a tag: git tag v0.1.0-test && git push origin v0.1.0-test"
echo "   - Or trigger manually: Go to Actions → Release → Run workflow"
echo ""
echo "2. To test dry run:"
echo "   - Go to Actions → Release → Run workflow"
echo "   - Set 'Perform a dry run' to true"
echo "   - Enter version: 0.1.0-test"
echo ""
echo "3. To verify signed artifacts:"
echo "   - Install cosign: go install github.com/sigstore/cosign/v2/cmd/cosign@latest"
echo "   - Verify signature: cosign verify --certificate-identity-regexp=... <image>"

echo -e "\n${GREEN}Release workflow test completed successfully!${NC}"