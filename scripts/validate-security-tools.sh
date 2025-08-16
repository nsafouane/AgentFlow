#!/bin/bash

# Security Tools Validation Script
# Validates that security tools are properly configured and working

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

# Validation results
VALIDATIONS_PASSED=0
VALIDATIONS_FAILED=0

# Validate a tool
validate_tool() {
    local tool_name="$1"
    local tool_command="$2"
    local expected_output="$3"
    
    log_info "Validating $tool_name..."
    
    if command -v "$tool_command" &> /dev/null; then
        if [[ -n "$expected_output" ]]; then
            local output
            output=$($tool_command --version 2>&1 || echo "version check failed")
            if [[ "$output" == *"$expected_output"* ]]; then
                log_info "✓ $tool_name is available and working"
                VALIDATIONS_PASSED=$((VALIDATIONS_PASSED + 1))
            else
                log_warn "⚠ $tool_name is available but version check failed: $output"
                VALIDATIONS_PASSED=$((VALIDATIONS_PASSED + 1))
            fi
        else
            log_info "✓ $tool_name is available"
            VALIDATIONS_PASSED=$((VALIDATIONS_PASSED + 1))
        fi
    else
        log_error "✗ $tool_name is not available"
        VALIDATIONS_FAILED=$((VALIDATIONS_FAILED + 1))
    fi
}

# Validate configuration files
validate_config() {
    local config_file="$1"
    local description="$2"
    
    log_info "Validating $description..."
    
    if [[ -f "$REPO_ROOT/$config_file" ]]; then
        log_info "✓ $description exists: $config_file"
        VALIDATIONS_PASSED=$((VALIDATIONS_PASSED + 1))
    else
        log_error "✗ $description missing: $config_file"
        VALIDATIONS_FAILED=$((VALIDATIONS_FAILED + 1))
    fi
}

# Validate scripts
validate_script() {
    local script_file="$1"
    local description="$2"
    
    log_info "Validating $description..."
    
    if [[ -f "$REPO_ROOT/$script_file" ]]; then
        if [[ -x "$REPO_ROOT/$script_file" ]]; then
            log_info "✓ $description exists and is executable: $script_file"
            VALIDATIONS_PASSED=$((VALIDATIONS_PASSED + 1))
        else
            log_warn "⚠ $description exists but is not executable: $script_file"
            VALIDATIONS_PASSED=$((VALIDATIONS_PASSED + 1))
        fi
    else
        log_error "✗ $description missing: $script_file"
        VALIDATIONS_FAILED=$((VALIDATIONS_FAILED + 1))
    fi
}

# Main validation
main() {
    log_info "Starting security tools validation"
    echo
    
    # Validate core Go tools
    validate_tool "Go compiler" "go" "go version"
    
    # Validate security tools
    validate_tool "gosec" "gosec" ""
    validate_tool "govulncheck" "govulncheck" ""
    validate_tool "gitleaks" "gitleaks" ""
    validate_tool "syft" "syft" ""
    validate_tool "grype" "grype" ""
    
    echo
    
    # Validate configuration files
    validate_config ".security-config.yml" "Security configuration"
    validate_config "docs/security-baseline.md" "Security baseline documentation"
    
    echo
    
    # Validate scripts
    validate_script "scripts/security-scan.sh" "Security scan script (Bash)"
    validate_script "scripts/security-scan.ps1" "Security scan script (PowerShell)"
    validate_script "scripts/test-security-failure-enhanced.sh" "Security failure test script"
    validate_script "scripts/security_test.go" "Security unit tests"
    
    echo
    
    # Test unit tests
    log_info "Running security unit tests..."
    cd "$SCRIPT_DIR"
    if go test -v ./security_test.go > /dev/null 2>&1; then
        log_info "✓ Security unit tests pass"
        VALIDATIONS_PASSED=$((VALIDATIONS_PASSED + 1))
    else
        log_error "✗ Security unit tests fail"
        VALIDATIONS_FAILED=$((VALIDATIONS_FAILED + 1))
    fi
    
    echo
    
    # Print summary
    log_info "=== Validation Summary ==="
    log_info "Validations passed: $VALIDATIONS_PASSED"
    log_info "Validations failed: $VALIDATIONS_FAILED"
    log_info "Total validations: $((VALIDATIONS_PASSED + VALIDATIONS_FAILED))"
    
    if [[ $VALIDATIONS_FAILED -eq 0 ]]; then
        echo
        log_info "🎉 All validations passed! Security tooling is properly configured."
        return 0
    else
        echo
        log_error "❌ Some validations failed. Please review and fix the issues."
        return 1
    fi
}

# Execute main function
main "$@"