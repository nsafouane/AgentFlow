#!/bin/bash
# Test script for version management functionality

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Testing AgentFlow Version Management${NC}"
echo "========================================"

cd "$SCRIPT_DIR"

# Test 1: Version parsing script functionality
echo -e "\n${YELLOW}Test 1: Version Parsing Script${NC}"
if [ -f "parse-version.sh" ]; then
    echo "Testing valid version parsing..."
    
    # Test valid versions
    test_versions=("1.2.3" "v0.1.0" "2.0.0-alpha.1" "0.5.0-beta.2" "1.0.0-rc.1")
    
    for version in "${test_versions[@]}"; do
        echo -n "  Testing $version... "
        if output=$(bash parse-version.sh "$version" 2>/dev/null); then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
            echo "    Failed to parse version: $version"
        fi
    done
    
    # Test invalid versions
    echo "Testing invalid version rejection..."
    invalid_versions=("1.2" "1.2.3.4" "v1.2.3a" "invalid")
    
    for version in "${invalid_versions[@]}"; do
        echo -n "  Testing $version (should fail)... "
        if bash parse-version.sh "$version" >/dev/null 2>&1; then
            echo -e "${RED}✗ (should have failed)${NC}"
        else
            echo -e "${GREEN}✓${NC}"
        fi
    done
else
    echo -e "${RED}✗ parse-version.sh not found${NC}"
fi

# Test 2: Version increment functionality
echo -e "\n${YELLOW}Test 2: Version Increment${NC}"
if [ -f "parse-version.sh" ]; then
    test_cases=(
        "1.2.3:patch:1.2.4"
        "1.2.3:minor:1.3.0"
        "1.2.3:major:2.0.0"
        "0.1.0:patch:0.1.1"
    )
    
    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r version increment expected <<< "$test_case"
        echo -n "  Testing $version + $increment = $expected... "
        
        if output=$(bash parse-version.sh "$version" increment "$increment" 2>/dev/null); then
            actual=$(echo "$output" | grep "NEXT_VERSION=" | cut -d'=' -f2)
            if [ "$actual" = "$expected" ]; then
                echo -e "${GREEN}✓${NC}"
            else
                echo -e "${RED}✗ (got $actual)${NC}"
            fi
        else
            echo -e "${RED}✗ (script failed)${NC}"
        fi
    done
else
    echo -e "${RED}✗ parse-version.sh not found${NC}"
fi

# Test 3: Version comparison
echo -e "\n${YELLOW}Test 3: Version Comparison${NC}"
if [ -f "parse-version.sh" ]; then
    test_cases=(
        "1.2.3:1.2.3:equal"
        "1.2.4:1.2.3:greater"
        "1.2.3:1.2.4:less"
        "1.2.3:1.2.3-alpha.1:greater"
        "1.2.3-alpha.1:1.2.3:less"
    )
    
    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r version1 version2 expected <<< "$test_case"
        echo -n "  Testing $version1 vs $version2 = $expected... "
        
        if output=$(bash parse-version.sh "$version1" compare "$version2" 2>/dev/null); then
            actual=$(echo "$output" | grep "COMPARISON=" | cut -d'=' -f2)
            if [ "$actual" = "$expected" ]; then
                echo -e "${GREEN}✓${NC}"
            else
                echo -e "${RED}✗ (got $actual)${NC}"
            fi
        else
            echo -e "${RED}✗ (script failed)${NC}"
        fi
    done
else
    echo -e "${RED}✗ parse-version.sh not found${NC}"
fi

# Test 4: Update version script
echo -e "\n${YELLOW}Test 4: Version Update Script${NC}"
if [ -f "update-version.sh" ]; then
    # Create temporary test environment
    temp_dir=$(mktemp -d)
    echo "  Using temporary directory: $temp_dir"
    
    # Create test files
    mkdir -p "$temp_dir/cmd/af"
    
    cat > "$temp_dir/go.mod" << 'EOF'
module github.com/agentflow/agentflow

go 1.22

// version: v0.0.1
EOF
    
    cat > "$temp_dir/cmd/af/version.go" << 'EOF'
package main

const (
    Version = "0.0.1"
    BuildDate = ""
    GitCommit = ""
)
EOF
    
    cat > "$temp_dir/Dockerfile" << 'EOF'
FROM golang:1.22-alpine
LABEL version="0.0.1"
COPY . .
EOF
    
    # Copy update script
    cp update-version.sh "$temp_dir/"
    
    # Test version update
    cd "$temp_dir"
    echo -n "  Testing version update to 0.2.0... "
    
    if bash update-version.sh 0.2.0 >/dev/null 2>&1; then
        # Verify updates
        if grep -q "// version: v0.2.0" go.mod && \
           grep -q 'Version = "0.2.0"' cmd/af/version.go && \
           grep -q 'LABEL version="0.2.0"' Dockerfile; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗ (files not updated correctly)${NC}"
        fi
    else
        echo -e "${RED}✗ (script failed)${NC}"
    fi
    
    # Cleanup
    cd "$SCRIPT_DIR"
    rm -rf "$temp_dir"
else
    echo -e "${RED}✗ update-version.sh not found${NC}"
fi

# Test 5: Go unit tests
echo -e "\n${YELLOW}Test 5: Go Unit Tests${NC}"
if [ -f "version_test.go" ]; then
    echo -n "  Running Go unit tests... "
    if go test -v . >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        echo "  Running tests with output:"
        go test -v .
    fi
else
    echo -e "${RED}✗ version_test.go not found${NC}"
fi

# Test 6: Documentation validation
echo -e "\n${YELLOW}Test 6: Documentation Validation${NC}"

# Check RELEASE.md
echo -n "  Checking RELEASE.md... "
if [ -f "$PROJECT_ROOT/RELEASE.md" ]; then
    required_sections=(
        "Versioning Scheme"
        "Tagging Policy"
        "Branching Model"
        "Release Process"
        "Hotfix Process"
    )
    
    all_found=true
    for section in "${required_sections[@]}"; do
        if ! grep -q "$section" "$PROJECT_ROOT/RELEASE.md"; then
            all_found=false
            break
        fi
    done
    
    if $all_found; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (missing required sections)${NC}"
    fi
else
    echo -e "${RED}✗ (file not found)${NC}"
fi

# Check CHANGELOG.md
echo -n "  Checking CHANGELOG.md... "
if [ -f "$PROJECT_ROOT/CHANGELOG.md" ]; then
    required_sections=(
        "# Changelog"
        "## [Unreleased]"
        "### Added"
        "### Changed"
        "### Fixed"
        "### Security"
    )
    
    all_found=true
    for section in "${required_sections[@]}"; do
        if ! grep -q "$section" "$PROJECT_ROOT/CHANGELOG.md"; then
            all_found=false
            break
        fi
    done
    
    if $all_found; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (missing required sections)${NC}"
    fi
else
    echo -e "${RED}✗ (file not found)${NC}"
fi

# Check version.go
echo -n "  Checking cmd/af/version.go... "
if [ -f "$PROJECT_ROOT/cmd/af/version.go" ]; then
    required_functions=(
        "GetVersionInfo"
        "GetVersionString"
        "IsPreRelease"
        "GetMajorVersion"
        "IsStableAPI"
    )
    
    all_found=true
    for func in "${required_functions[@]}"; do
        if ! grep -q "func $func" "$PROJECT_ROOT/cmd/af/version.go"; then
            all_found=false
            break
        fi
    done
    
    if $all_found; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (missing required functions)${NC}"
    fi
else
    echo -e "${RED}✗ (file not found)${NC}"
fi

echo -e "\n${BLUE}Version Management Tests Complete${NC}"
echo "========================================"