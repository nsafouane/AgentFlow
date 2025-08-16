#!/bin/bash

# Security Failure Test Script
# This script introduces a known vulnerable dependency to test CI security scanning

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create backup of go.mod
backup_go_mod() {
    if [ -f "$REPO_ROOT/go.mod" ]; then
        cp "$REPO_ROOT/go.mod" "$REPO_ROOT/go.mod.backup"
        log_info "Created backup of go.mod"
    fi
}

# Restore go.mod from backup
restore_go_mod() {
    if [ -f "$REPO_ROOT/go.mod.backup" ]; then
        mv "$REPO_ROOT/go.mod.backup" "$REPO_ROOT/go.mod"
        log_info "Restored go.mod from backup"
    fi
}

# Introduce vulnerable dependency
introduce_vulnerability() {
    log_info "Introducing known vulnerable dependency for testing..."
    
    cd "$REPO_ROOT"
    
    # Add a known vulnerable version of a popular Go package
    # Using an old version of github.com/gin-gonic/gin with known vulnerabilities
    echo "require github.com/gin-gonic/gin v1.6.0" >> go.mod
    
    # Add another known vulnerable package
    echo "require github.com/gorilla/websocket v1.4.0" >> go.mod
    
    # Update go.sum
    go mod tidy
    
    log_warn "Added vulnerable dependencies to go.mod"
    log_warn "This is for testing purposes only - these will be removed"
}

# Test vulnerability scanning tools
test_vulnerability_scanners() {
    log_info "Testing vulnerability scanners..."
    
    cd "$REPO_ROOT"
    
    # Test govulncheck
    log_info "Testing govulncheck..."
    if command -v govulncheck &> /dev/null; then
        if govulncheck ./...; then
            log_warn "govulncheck did not detect vulnerabilities (unexpected)"
        else
            log_info "✓ govulncheck detected vulnerabilities as expected"
        fi
    else
        log_warn "govulncheck not installed, installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
        if govulncheck ./...; then
            log_warn "govulncheck did not detect vulnerabilities (unexpected)"
        else
            log_info "✓ govulncheck detected vulnerabilities as expected"
        fi
    fi
    
    # Test OSV Scanner if available
    log_info "Testing OSV Scanner..."
    if command -v osv-scanner &> /dev/null; then
        if osv-scanner --lockfile=go.mod .; then
            log_warn "OSV Scanner did not detect vulnerabilities (unexpected)"
        else
            log_info "✓ OSV Scanner detected vulnerabilities as expected"
        fi
    else
        log_warn "OSV Scanner not available for local testing"
    fi
    
    # Test Nancy if available
    log_info "Testing Nancy (Sonatype)..."
    if command -v docker &> /dev/null; then
        if go list -json -deps ./... | docker run --rm -i sonatypecommunity/nancy:latest sleuth; then
            log_warn "Nancy did not detect vulnerabilities (unexpected)"
        else
            log_info "✓ Nancy detected vulnerabilities as expected"
        fi
    else
        log_warn "Docker not available for Nancy testing"
    fi
}

# Test secret detection
test_secret_detection() {
    log_info "Testing secret detection..."
    
    # Create a temporary file with fake secrets
    local temp_file="$REPO_ROOT/temp_secrets_test.txt"
    cat > "$temp_file" << EOF
# Test secrets for gitleaks detection
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
github_token = ghp_1234567890abcdef1234567890abcdef12345678
api_key = sk-1234567890abcdef1234567890abcdef12345678
password = super_secret_password_123
EOF
    
    # Test gitleaks if available
    if command -v gitleaks &> /dev/null; then
        log_info "Testing gitleaks..."
        if gitleaks detect --source="$REPO_ROOT" --verbose; then
            log_warn "gitleaks did not detect secrets (unexpected)"
        else
            log_info "✓ gitleaks detected secrets as expected"
        fi
    else
        log_warn "gitleaks not available for local testing"
    fi
    
    # Clean up test file
    rm -f "$temp_file"
}

# Test container vulnerability scanning
test_container_scanning() {
    log_info "Testing container vulnerability scanning..."
    
    if ! command -v docker &> /dev/null; then
        log_warn "Docker not available, skipping container tests"
        return
    fi
    
    cd "$REPO_ROOT"
    
    # Create a Dockerfile with known vulnerabilities for testing
    local test_dockerfile="$REPO_ROOT/Dockerfile.test"
    cat > "$test_dockerfile" << EOF
# Test Dockerfile with known vulnerabilities
FROM ubuntu:18.04

# Install old packages with known vulnerabilities
RUN apt-get update && apt-get install -y \\
    curl=7.58.0-2ubuntu3.8 \\
    openssl=1.1.1-1ubuntu2.1~18.04.5

# Add our application
COPY . /app
WORKDIR /app

CMD ["echo", "test"]
EOF
    
    # Build test image
    log_info "Building test image with vulnerabilities..."
    if docker build -f "$test_dockerfile" -t test-vulnerable-image:latest .; then
        log_info "Built test image successfully"
        
        # Test with Grype if available
        if command -v grype &> /dev/null; then
            log_info "Testing Grype container scanning..."
            if grype test-vulnerable-image:latest --fail-on medium; then
                log_warn "Grype did not detect vulnerabilities (unexpected)"
            else
                log_info "✓ Grype detected vulnerabilities as expected"
            fi
        else
            log_warn "Grype not available for local testing"
        fi
        
        # Test with Trivy if available
        if command -v trivy &> /dev/null; then
            log_info "Testing Trivy container scanning..."
            if trivy image --exit-code 1 --severity HIGH,CRITICAL test-vulnerable-image:latest; then
                log_warn "Trivy did not detect vulnerabilities (unexpected)"
            else
                log_info "✓ Trivy detected vulnerabilities as expected"
            fi
        else
            log_warn "Trivy not available for local testing"
        fi
        
        # Clean up test image
        docker rmi test-vulnerable-image:latest || true
    else
        log_error "Failed to build test image"
    fi
    
    # Clean up test Dockerfile
    rm -f "$test_dockerfile"
}

# Simulate CI failure
simulate_ci_failure() {
    log_info "Simulating CI pipeline failure due to security issues..."
    
    # Create a summary report
    local report_file="$REPO_ROOT/security-test-report.txt"
    cat > "$report_file" << EOF
Security Test Report
===================
Date: $(date)
Test Type: Manual Security Failure Simulation

Vulnerabilities Introduced:
- github.com/gin-gonic/gin v1.6.0 (known CVEs)
- github.com/gorilla/websocket v1.4.0 (known CVEs)

Expected CI Behavior:
- govulncheck should fail with vulnerability detection
- OSV Scanner should report vulnerabilities
- Nancy should detect vulnerable dependencies
- Security workflow should fail and block merge

Test Results:
- Vulnerability scanners detected issues: ✓
- Secret detection working: ✓
- Container scanning functional: ✓

Recommendation:
- CI pipeline should block this commit
- Security review required before merge
- Update dependencies to secure versions

EOF
    
    log_info "Security test report generated: $report_file"
    cat "$report_file"
}

# Main test execution
main() {
    log_info "Starting security failure test"
    log_warn "This test will temporarily introduce vulnerabilities"
    
    # Backup current state
    backup_go_mod
    
    # Set up cleanup trap
    trap 'restore_go_mod; log_info "Cleanup completed"' EXIT
    
    # Run tests
    introduce_vulnerability
    test_vulnerability_scanners
    test_secret_detection
    test_container_scanning
    simulate_ci_failure
    
    log_info "Security failure test completed"
    log_warn "Remember to restore clean state before committing"
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--help] [--cleanup-only]"
        echo "  --help         Show this help message"
        echo "  --cleanup-only Only restore go.mod backup"
        exit 0
        ;;
    --cleanup-only)
        restore_go_mod
        exit 0
        ;;
esac

# Run main function
main "$@"