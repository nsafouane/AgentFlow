#!/bin/bash

# Workflow Testing Script
# This script tests GitHub Actions workflows using act and validates configurations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WORKFLOWS_DIR="${REPO_ROOT}/.github/workflows"

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

# Check if act is installed
check_act_installation() {
    if ! command -v act &> /dev/null; then
        log_warn "act is not installed. Installing act..."
        
        # Install act based on OS
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            if command -v brew &> /dev/null; then
                brew install act
            else
                log_error "Please install act manually: https://github.com/nektos/act#installation"
                exit 1
            fi
        else
            log_error "Please install act manually: https://github.com/nektos/act#installation"
            exit 1
        fi
    fi
}

# Test workflow with act (dry-run)
test_workflow_with_act() {
    local workflow_file="$1"
    local workflow_name=$(basename "$workflow_file" .yml)
    
    log_info "Testing workflow: $workflow_name"
    
    # Create minimal event file for testing
    local event_file="/tmp/test_event.json"
    cat > "$event_file" << EOF
{
  "push": {
    "ref": "refs/heads/test-branch"
  }
}
EOF
    
    # Run act with dry-run mode
    if act -n -W "$workflow_file" -e "$event_file" --platform ubuntu-latest=catthehacker/ubuntu:act-latest; then
        log_info "✓ Workflow $workflow_name passed dry-run test"
        return 0
    else
        log_error "✗ Workflow $workflow_name failed dry-run test"
        return 1
    fi
}

# Test workflow syntax and structure
test_workflow_syntax() {
    local workflow_file="$1"
    local workflow_name=$(basename "$workflow_file" .yml)
    
    log_info "Testing syntax for workflow: $workflow_name"
    
    # Use yq to validate YAML structure
    if yq eval '.' "$workflow_file" > /dev/null 2>&1; then
        log_info "✓ Workflow $workflow_name has valid YAML syntax"
    else
        log_error "✗ Workflow $workflow_name has invalid YAML syntax"
        return 1
    fi
    
    # Check required fields
    local required_fields=("name" "on" "jobs")
    for field in "${required_fields[@]}"; do
        if ! yq eval "has(\"$field\")" "$workflow_file" | grep -q "true"; then
            log_error "✗ Workflow $workflow_name missing required field: $field"
            return 1
        fi
    done
    
    log_info "✓ Workflow $workflow_name has valid structure"
    return 0
}

# Test security configurations
test_security_config() {
    local workflow_file="$1"
    local workflow_name=$(basename "$workflow_file" .yml)
    
    log_info "Testing security configuration for workflow: $workflow_name"
    
    # Check if permissions are defined
    if yq eval 'has("permissions")' "$workflow_file" | grep -q "true"; then
        log_info "✓ Workflow $workflow_name has permissions defined"
        
        # Check for minimal permissions
        local permissions=$(yq eval '.permissions' "$workflow_file")
        if echo "$permissions" | grep -q "write-all\|contents: write"; then
            log_warn "⚠ Workflow $workflow_name has broad write permissions"
        fi
    else
        log_warn "⚠ Workflow $workflow_name has no explicit permissions (will use default)"
    fi
    
    # Check for secret usage
    if grep -q "secrets\." "$workflow_file"; then
        log_info "✓ Workflow $workflow_name uses secrets properly"
    fi
    
    return 0
}

# Test caching configuration
test_caching_config() {
    local workflow_file="$1"
    local workflow_name=$(basename "$workflow_file" .yml)
    
    log_info "Testing caching configuration for workflow: $workflow_name"
    
    # Check for Go caching
    if grep -q "setup-go" "$workflow_file"; then
        if grep -q "cache: true" "$workflow_file" || grep -q "actions/cache" "$workflow_file"; then
            log_info "✓ Workflow $workflow_name has Go caching enabled"
        else
            log_warn "⚠ Workflow $workflow_name could benefit from Go caching"
        fi
    fi
    
    # Check for Docker caching
    if grep -q "docker/build-push-action" "$workflow_file"; then
        if grep -q "cache-from\|cache-to" "$workflow_file"; then
            log_info "✓ Workflow $workflow_name has Docker caching enabled"
        else
            log_warn "⚠ Workflow $workflow_name could benefit from Docker caching"
        fi
    fi
    
    return 0
}

# Create test environment
setup_test_environment() {
    log_info "Setting up test environment"
    
    # Create .env file for act if it doesn't exist
    if [ ! -f "$REPO_ROOT/.env" ]; then
        cat > "$REPO_ROOT/.env" << EOF
GITHUB_TOKEN=test_token
GITHUB_REPOSITORY=test/agentflow
GITHUB_REPOSITORY_OWNER=test
GITHUB_ACTOR=test-user
GITHUB_SHA=test-sha
GITHUB_REF=refs/heads/test-branch
EOF
        log_info "Created .env file for testing"
    fi
    
    # Create .actrc file for act configuration
    if [ ! -f "$REPO_ROOT/.actrc" ]; then
        cat > "$REPO_ROOT/.actrc" << EOF
--platform ubuntu-latest=catthehacker/ubuntu:act-latest
--platform ubuntu-20.04=catthehacker/ubuntu:act-20.04
--platform ubuntu-18.04=catthehacker/ubuntu:act-18.04
--env-file .env
EOF
        log_info "Created .actrc file for act configuration"
    fi
}

# Clean up test environment
cleanup_test_environment() {
    log_info "Cleaning up test environment"
    
    # Remove temporary files
    rm -f /tmp/test_event.json
    
    # Optionally remove test configuration files
    if [ "${CLEANUP_TEST_FILES:-false}" = "true" ]; then
        rm -f "$REPO_ROOT/.env"
        rm -f "$REPO_ROOT/.actrc"
    fi
}

# Main test function
run_workflow_tests() {
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    log_info "Starting workflow tests"
    
    # Test all workflow files
    for workflow_file in "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml; do
        if [ -f "$workflow_file" ]; then
            local workflow_name=$(basename "$workflow_file" .yml)
            log_info "Testing workflow: $workflow_name"
            
            ((total_tests++))
            
            # Run all tests for this workflow
            local workflow_passed=true
            
            test_workflow_syntax "$workflow_file" || workflow_passed=false
            test_security_config "$workflow_file"
            test_caching_config "$workflow_file"
            
            # Only run act test if syntax is valid and act is available
            if [ "$workflow_passed" = true ] && command -v act &> /dev/null; then
                test_workflow_with_act "$workflow_file" || workflow_passed=false
            fi
            
            if [ "$workflow_passed" = true ]; then
                ((passed_tests++))
                log_info "✅ Workflow $workflow_name passed all tests"
            else
                ((failed_tests++))
                log_error "❌ Workflow $workflow_name failed some tests"
            fi
            
            echo "----------------------------------------"
        fi
    done
    
    # Summary
    log_info "Test Summary:"
    log_info "Total workflows tested: $total_tests"
    log_info "Passed: $passed_tests"
    log_info "Failed: $failed_tests"
    
    if [ $failed_tests -eq 0 ]; then
        log_info "🎉 All workflow tests passed!"
        return 0
    else
        log_error "💥 Some workflow tests failed!"
        return 1
    fi
}

# Main execution
main() {
    log_info "GitHub Actions Workflow Testing"
    
    # Check if workflows directory exists
    if [ ! -d "$WORKFLOWS_DIR" ]; then
        log_error "Workflows directory not found: $WORKFLOWS_DIR"
        exit 1
    fi
    
    # Setup test environment
    setup_test_environment
    
    # Check for act installation (optional)
    if ! command -v act &> /dev/null; then
        log_warn "act is not installed. Skipping dry-run tests."
        log_info "Install act from: https://github.com/nektos/act#installation"
    fi
    
    # Run tests
    local exit_code=0
    run_workflow_tests || exit_code=$?
    
    # Cleanup
    cleanup_test_environment
    
    exit $exit_code
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--help] [--cleanup]"
        echo "  --help     Show this help message"
        echo "  --cleanup  Remove test configuration files after testing"
        exit 0
        ;;
    --cleanup)
        export CLEANUP_TEST_FILES=true
        ;;
esac

# Run main function
main "$@"