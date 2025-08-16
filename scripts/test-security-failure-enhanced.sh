#!/bin/bash

# Enhanced Security Failure Test Script
# This script introduces known vulnerable dependencies and secrets to test CI security scanning
# It validates that security tools properly detect and fail on vulnerabilities above threshold

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_BRANCH="security-test-$(date +%s)"
BACKUP_DIR="${REPO_ROOT}/.security-test-backup"
TEST_RESULTS_FILE="${REPO_ROOT}/security-test-results.json"

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

log_debug() {
    if [[ "${DEBUG:-}" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# Test results tracking
declare -A TEST_RESULTS
TESTS_PASSED=0
TESTS_FAILED=0

# Record test result
record_test_result() {
    local test_name="$1"
    local result="$2"
    local details="$3"
    
    TEST_RESULTS["$test_name"]="$result:$details"
    
    if [[ "$result" == "PASS" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        log_info "✓ $test_name: PASSED - $details"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        log_error "✗ $test_name: FAILED - $details"
    fi
}

# Create backup of current state
create_backup() {
    log_info "Creating backup of current state..."
    
    mkdir -p "$BACKUP_DIR"
    
    # Backup important files
    if [[ -f "$REPO_ROOT/go.mod" ]]; then
        cp "$REPO_ROOT/go.mod" "$BACKUP_DIR/go.mod.backup"
    fi
    
    if [[ -f "$REPO_ROOT/go.sum" ]]; then
        cp "$REPO_ROOT/go.sum" "$BACKUP_DIR/go.sum.backup"
    fi
    
    # Backup any existing test files
    find "$REPO_ROOT" -name "*_vulnerable_test.go" -exec cp {} "$BACKUP_DIR/" \; 2>/dev/null || true
    
    log_info "Backup created in $BACKUP_DIR"
}

# Restore from backup
restore_backup() {
    log_info "Restoring from backup..."
    
    if [[ -d "$BACKUP_DIR" ]]; then
        # Restore go.mod and go.sum
        if [[ -f "$BACKUP_DIR/go.mod.backup" ]]; then
            cp "$BACKUP_DIR/go.mod.backup" "$REPO_ROOT/go.mod"
        fi
        
        if [[ -f "$BACKUP_DIR/go.sum.backup" ]]; then
            cp "$BACKUP_DIR/go.sum.backup" "$REPO_ROOT/go.sum"
        fi
        
        # Clean up test files
        find "$REPO_ROOT" -name "*_vulnerable_test.go" -delete 2>/dev/null || true
        find "$REPO_ROOT" -name "vulnerable_*" -delete 2>/dev/null || true
        find "$REPO_ROOT" -name "test_secrets_*" -delete 2>/dev/null || true
        
        # Remove backup directory
        rm -rf "$BACKUP_DIR"
        
        log_info "Backup restored and cleaned up"
    fi
}

# Create vulnerable Go module
create_vulnerable_go_module() {
    log_info "Creating vulnerable Go module for testing..."
    
    local vulnerable_file="$REPO_ROOT/cmd/af/vulnerable_test_module.go"
    
    cat > "$vulnerable_file" << 'EOF'
//go:build security_test
// +build security_test

package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"unsafe"
)

// VulnerableFunction contains intentional security vulnerabilities for testing
func VulnerableFunction() {
	// G101: Potential hardcoded credentials (HIGH severity)
	password := "super_secret_password_123"
	apiKey := "sk-1234567890abcdef1234567890abcdef12345678"
	
	// G401: Use of weak cryptographic primitive (MEDIUM severity)
	hash := md5.Sum([]byte("test"))
	fmt.Printf("Hash: %x\n", hash)
	
	// G103: Use of unsafe calls (HIGH severity)
	data := "test"
	ptr := unsafe.Pointer(&data)
	fmt.Printf("Pointer: %v\n", ptr)
	
	// G102: Bind to all interfaces (MEDIUM severity)
	http.ListenAndServe(":8080", nil)
	
	// G104: Audit errors not checked (LOW severity)
	os.Setenv("TEST", "value")
	
	// Use the variables to avoid unused variable warnings
	_ = password
	_ = apiKey
}

// AnotherVulnerableFunction with more security issues
func AnotherVulnerableFunction() {
	// G201: SQL query construction using format string (HIGH severity)
	query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", "1")
	
	// G301: Poor file permissions (MEDIUM severity)
	os.OpenFile("/tmp/test", os.O_CREATE|os.O_WRONLY, 0777)
	
	// G302: Poor file permissions for chmod (MEDIUM severity)
	os.Chmod("/tmp/test", 0777)
	
	// Use variables
	_ = query
}
EOF
    
    log_info "Created vulnerable Go module: $vulnerable_file"
}

# Introduce vulnerable dependencies
introduce_vulnerable_dependencies() {
    log_info "Introducing known vulnerable dependencies..."
    
    cd "$REPO_ROOT"
    
    # Create a temporary go.mod entry for testing
    local temp_mod_file="$REPO_ROOT/go_vulnerable.mod"
    
    cat > "$temp_mod_file" << EOF
module agentflow-security-test

go 1.22

require (
    // Known vulnerable versions for testing
    github.com/gin-gonic/gin v1.6.0  // CVE-2020-28483
    github.com/gorilla/websocket v1.4.0  // CVE-2020-27813
    github.com/dgrijalva/jwt-go v3.2.0+incompatible  // CVE-2020-26160
    gopkg.in/yaml.v2 v2.2.2  // CVE-2019-11254
    github.com/opencontainers/runc v1.0.0-rc8  // CVE-2019-5736
)
EOF
    
    # Temporarily replace go.mod for testing
    if [[ -f "$REPO_ROOT/go.mod" ]]; then
        mv "$REPO_ROOT/go.mod" "$REPO_ROOT/go.mod.original"
    fi
    
    mv "$temp_mod_file" "$REPO_ROOT/go.mod"
    
    # Update go.sum
    go mod tidy || true  # Allow this to fail as some packages might not be available
    
    log_warn "Introduced vulnerable dependencies (temporary for testing)"
}

# Create test secrets file
create_test_secrets() {
    log_info "Creating test secrets file..."
    
    local secrets_file="$REPO_ROOT/test_secrets_file.txt"
    
    cat > "$secrets_file" << EOF
# Test secrets for gitleaks detection - DO NOT COMMIT TO PRODUCTION
# These are intentionally fake credentials for security testing

# AWS Credentials
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# GitHub Token
GITHUB_TOKEN=ghp_1234567890abcdef1234567890abcdef12345678

# API Keys
STRIPE_API_KEY=sk_test_1234567890abcdef1234567890abcdef12345678
OPENAI_API_KEY=sk-1234567890abcdef1234567890abcdef12345678

# Database URLs
DATABASE_URL=postgresql://user:password123@localhost:5432/database
REDIS_URL=redis://:password123@localhost:6379/0

# JWT Secrets
JWT_SECRET=super_secret_jwt_key_that_should_not_be_hardcoded
API_SECRET=api_secret_key_12345

# Private Keys (fake)
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1234567890abcdef1234567890abcdef1234567890abcdef
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
-----END RSA PRIVATE KEY-----"

# Slack Webhook
SLACK_WEBHOOK=https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX

# Generic passwords
PASSWORD=super_secret_password_123
DB_PASSWORD=database_password_456
ADMIN_PASSWORD=admin_password_789
EOF
    
    log_info "Created test secrets file: $secrets_file"
}

# Create vulnerable container configuration
create_vulnerable_dockerfile() {
    log_info "Creating vulnerable Dockerfile for container scanning..."
    
    local dockerfile="$REPO_ROOT/Dockerfile.vulnerable"
    
    cat > "$dockerfile" << EOF
# Vulnerable Dockerfile for security testing
# Uses outdated base image with known vulnerabilities

FROM ubuntu:18.04

# Install packages with known vulnerabilities
RUN apt-get update && apt-get install -y \\
    curl=7.58.0-2ubuntu3.8 \\
    openssl=1.1.1-1ubuntu2.1~18.04.5 \\
    git=1:2.17.1-1ubuntu0.7 \\
    wget=1.19.4-1ubuntu2.2

# Run as root (security issue)
USER root

# Expose all ports (security issue)
EXPOSE 1-65535

# Copy application
COPY . /app
WORKDIR /app

# Set insecure permissions
RUN chmod 777 /app

# Use insecure environment variables
ENV SECRET_KEY=hardcoded_secret_key
ENV DB_PASSWORD=insecure_password

CMD ["echo", "vulnerable container for testing"]
EOF
    
    log_info "Created vulnerable Dockerfile: $dockerfile"
}

# Test gosec scanner
test_gosec_scanner() {
    log_info "Testing gosec scanner..."
    
    if ! command -v gosec &> /dev/null; then
        record_test_result "gosec_availability" "FAIL" "gosec not available"
        return
    fi
    
    # Run gosec on vulnerable code
    local output_file="$REPO_ROOT/security-reports/test-gosec.json"
    mkdir -p "$(dirname "$output_file")"
    
    if gosec -fmt json -out "$output_file" ./...; then
        record_test_result "gosec_execution" "FAIL" "gosec should have failed on vulnerable code"
    else
        # Check if issues were found
        if [[ -f "$output_file" ]]; then
            local issues_count
            issues_count=$(jq -r '.Issues // [] | length' "$output_file" 2>/dev/null || echo "0")
            
            if [[ "$issues_count" -gt 0 ]]; then
                record_test_result "gosec_detection" "PASS" "detected $issues_count security issues"
            else
                record_test_result "gosec_detection" "FAIL" "no security issues detected"
            fi
        else
            record_test_result "gosec_output" "FAIL" "no output file generated"
        fi
    fi
}

# Test gitleaks scanner
test_gitleaks_scanner() {
    log_info "Testing gitleaks scanner..."
    
    if ! command -v gitleaks &> /dev/null; then
        record_test_result "gitleaks_availability" "FAIL" "gitleaks not available"
        return
    fi
    
    # Run gitleaks on repository with secrets
    local output_file="$REPO_ROOT/security-reports/test-gitleaks.json"
    mkdir -p "$(dirname "$output_file")"
    
    if gitleaks detect --source="$REPO_ROOT" --report-format json --report-path "$output_file" --verbose; then
        record_test_result "gitleaks_execution" "FAIL" "gitleaks should have failed when secrets are present"
    else
        # Check if secrets were found
        if [[ -f "$output_file" ]]; then
            local secrets_count
            secrets_count=$(jq '. | length' "$output_file" 2>/dev/null || echo "0")
            
            if [[ "$secrets_count" -gt 0 ]]; then
                record_test_result "gitleaks_detection" "PASS" "detected $secrets_count secrets"
            else
                record_test_result "gitleaks_detection" "FAIL" "no secrets detected"
            fi
        else
            record_test_result "gitleaks_output" "FAIL" "no output file generated"
        fi
    fi
}

# Test govulncheck scanner
test_govulncheck_scanner() {
    log_info "Testing govulncheck scanner..."
    
    if ! command -v govulncheck &> /dev/null; then
        log_warn "govulncheck not available, installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
    
    # Run govulncheck on vulnerable dependencies
    local output_file="$REPO_ROOT/security-reports/test-govulncheck.json"
    mkdir -p "$(dirname "$output_file")"
    
    if govulncheck -json ./... > "$output_file" 2>&1; then
        record_test_result "govulncheck_execution" "FAIL" "govulncheck should have failed on vulnerable dependencies"
    else
        # Check if vulnerabilities were found
        if [[ -f "$output_file" ]]; then
            local vuln_count
            vuln_count=$(grep -c '"finding"' "$output_file" 2>/dev/null || echo "0")
            
            if [[ "$vuln_count" -gt 0 ]]; then
                record_test_result "govulncheck_detection" "PASS" "detected $vuln_count vulnerabilities"
            else
                record_test_result "govulncheck_detection" "FAIL" "no vulnerabilities detected"
            fi
        else
            record_test_result "govulncheck_output" "FAIL" "no output file generated"
        fi
    fi
}

# Test grype scanner
test_grype_scanner() {
    log_info "Testing grype scanner..."
    
    if ! command -v grype &> /dev/null; then
        log_warn "grype not available, installing..."
        curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin
    fi
    
    # Build vulnerable container for testing
    if command -v docker &> /dev/null && [[ -f "$REPO_ROOT/Dockerfile.vulnerable" ]]; then
        log_info "Building vulnerable container for grype testing..."
        
        if docker build -f "$REPO_ROOT/Dockerfile.vulnerable" -t test-vulnerable:latest "$REPO_ROOT"; then
            # Run grype on vulnerable container
            local output_file="$REPO_ROOT/security-reports/test-grype.json"
            mkdir -p "$(dirname "$output_file")"
            
            if grype test-vulnerable:latest -o json --file "$output_file" --fail-on high; then
                record_test_result "grype_execution" "FAIL" "grype should have failed on vulnerable container"
            else
                # Check if vulnerabilities were found
                if [[ -f "$output_file" ]]; then
                    local vuln_count
                    vuln_count=$(jq -r '.matches // [] | length' "$output_file" 2>/dev/null || echo "0")
                    
                    if [[ "$vuln_count" -gt 0 ]]; then
                        record_test_result "grype_detection" "PASS" "detected $vuln_count vulnerabilities"
                    else
                        record_test_result "grype_detection" "FAIL" "no vulnerabilities detected"
                    fi
                else
                    record_test_result "grype_output" "FAIL" "no output file generated"
                fi
            fi
            
            # Clean up test container
            docker rmi test-vulnerable:latest || true
        else
            record_test_result "grype_container_build" "FAIL" "failed to build vulnerable container"
        fi
    else
        record_test_result "grype_availability" "FAIL" "docker not available or Dockerfile missing"
    fi
}

# Test security scan script
test_security_scan_script() {
    log_info "Testing security scan script..."
    
    local scan_script="$SCRIPT_DIR/security-scan.sh"
    
    if [[ -f "$scan_script" ]]; then
        # Make script executable
        chmod +x "$scan_script"
        
        # Run security scan script (should fail due to vulnerabilities)
        if "$scan_script" --threshold high --output "$REPO_ROOT/security-reports/test-scan"; then
            record_test_result "security_scan_script" "FAIL" "security scan should have failed on vulnerable code"
        else
            record_test_result "security_scan_script" "PASS" "security scan correctly failed on vulnerable code"
        fi
    else
        record_test_result "security_scan_script" "FAIL" "security scan script not found"
    fi
}

# Test threshold logic
test_threshold_logic() {
    log_info "Testing threshold logic..."
    
    # Run Go tests for threshold logic
    cd "$SCRIPT_DIR"
    
    if go test -v -run TestMeetsThreshold ./security_test.go; then
        record_test_result "threshold_logic" "PASS" "threshold logic tests passed"
    else
        record_test_result "threshold_logic" "FAIL" "threshold logic tests failed"
    fi
    
    # Test parsing logic
    if go test -v -run TestParseSeverity ./security_test.go; then
        record_test_result "severity_parsing" "PASS" "severity parsing tests passed"
    else
        record_test_result "severity_parsing" "FAIL" "severity parsing tests failed"
    fi
    
    # Test report parsing
    if go test -v -run TestParse ./security_test.go; then
        record_test_result "report_parsing" "PASS" "report parsing tests passed"
    else
        record_test_result "report_parsing" "FAIL" "report parsing tests failed"
    fi
}

# Generate test report
generate_test_report() {
    log_info "Generating test report..."
    
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    cat > "$TEST_RESULTS_FILE" << EOF
{
  "timestamp": "$timestamp",
  "test_summary": {
    "total_tests": $((TESTS_PASSED + TESTS_FAILED)),
    "passed": $TESTS_PASSED,
    "failed": $TESTS_FAILED,
    "success_rate": "$(( TESTS_PASSED * 100 / (TESTS_PASSED + TESTS_FAILED) ))%"
  },
  "test_results": {
EOF

    local first=true
    for test_name in "${!TEST_RESULTS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo "," >> "$TEST_RESULTS_FILE"
        fi
        
        local result_data="${TEST_RESULTS[$test_name]}"
        local result="${result_data%%:*}"
        local details="${result_data#*:}"
        
        cat >> "$TEST_RESULTS_FILE" << EOF
    "$test_name": {
      "result": "$result",
      "details": "$details"
    }
EOF
    done

    cat >> "$TEST_RESULTS_FILE" << EOF
  },
  "recommendations": [
EOF

    if [[ $TESTS_FAILED -gt 0 ]]; then
        cat >> "$TEST_RESULTS_FILE" << EOF
    "Review failed tests and ensure security tools are properly configured",
    "Verify that vulnerable dependencies and secrets are correctly detected",
    "Check that security thresholds are enforced in CI/CD pipeline"
EOF
    else
        cat >> "$TEST_RESULTS_FILE" << EOF
    "All security tests passed - security tooling is working correctly",
    "Consider running this test regularly to validate security controls",
    "Review and update vulnerable test cases as new threats emerge"
EOF
    fi

    cat >> "$TEST_RESULTS_FILE" << EOF
  ]
}
EOF

    log_info "Test report generated: $TEST_RESULTS_FILE"
}

# Print test summary
print_test_summary() {
    echo
    log_info "=== Security Test Summary ==="
    echo
    
    for test_name in "${!TEST_RESULTS[@]}"; do
        local result_data="${TEST_RESULTS[$test_name]}"
        local result="${result_data%%:*}"
        local details="${result_data#*:}"
        
        if [[ "$result" == "PASS" ]]; then
            echo -e "  ${GREEN}✓${NC} $test_name: $details"
        else
            echo -e "  ${RED}✗${NC} $test_name: $details"
        fi
    done
    
    echo
    log_info "Tests passed: $TESTS_PASSED"
    log_info "Tests failed: $TESTS_FAILED"
    log_info "Total tests: $((TESTS_PASSED + TESTS_FAILED))"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo
        log_info "🎉 All security tests passed! Security tooling is working correctly."
        return 0
    else
        echo
        log_error "❌ Some security tests failed. Review the results and fix issues."
        return 1
    fi
}

# Show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Enhanced security failure test script that validates security tooling.

OPTIONS:
    --skip-deps          Skip vulnerable dependency injection
    --skip-secrets       Skip secret file creation
    --skip-container     Skip container vulnerability testing
    --cleanup-only       Only perform cleanup operations
    -v, --verbose        Enable verbose output
    -h, --help           Show this help message

EXAMPLES:
    $0                   # Run all security tests
    $0 --skip-container  # Skip container testing
    $0 --cleanup-only    # Only cleanup test artifacts
    $0 -v                # Run with verbose output

EXIT CODES:
    0    All tests passed
    1    Some tests failed
    2    Invalid arguments or setup error
EOF
}

# Parse command line arguments
parse_arguments() {
    SKIP_DEPS=false
    SKIP_SECRETS=false
    SKIP_CONTAINER=false
    CLEANUP_ONLY=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-deps)
                SKIP_DEPS=true
                shift
                ;;
            --skip-secrets)
                SKIP_SECRETS=true
                shift
                ;;
            --skip-container)
                SKIP_CONTAINER=true
                shift
                ;;
            --cleanup-only)
                CLEANUP_ONLY=true
                shift
                ;;
            -v|--verbose)
                DEBUG=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 2
                ;;
        esac
    done
}

# Main execution
main() {
    log_info "Starting enhanced security failure test"
    
    # Set up cleanup trap
    trap 'restore_backup; exit 1' EXIT
    
    # Create backup
    create_backup
    
    if [[ "$CLEANUP_ONLY" == "true" ]]; then
        log_info "Cleanup-only mode - restoring backup and exiting"
        restore_backup
        trap - EXIT
        exit 0
    fi
    
    # Set up test environment
    create_vulnerable_go_module
    
    if [[ "$SKIP_DEPS" != "true" ]]; then
        introduce_vulnerable_dependencies
    fi
    
    if [[ "$SKIP_SECRETS" != "true" ]]; then
        create_test_secrets
    fi
    
    if [[ "$SKIP_CONTAINER" != "true" ]]; then
        create_vulnerable_dockerfile
    fi
    
    # Run security tests
    test_threshold_logic
    test_gosec_scanner
    test_gitleaks_scanner
    test_govulncheck_scanner
    
    if [[ "$SKIP_CONTAINER" != "true" ]]; then
        test_grype_scanner
    fi
    
    test_security_scan_script
    
    # Generate reports
    generate_test_report
    
    # Clean up
    restore_backup
    trap - EXIT
    
    # Print results
    print_test_summary
}

# Handle script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_arguments "$@"
    main
fi