#!/bin/bash

# Workflow Validation Script
# This script validates GitHub Actions workflow files for syntax and best practices

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

# Check if required tools are installed
check_dependencies() {
    local missing_deps=()
    
    if ! command -v yq &> /dev/null; then
        missing_deps+=("yq")
    fi
    
    if ! command -v yamllint &> /dev/null; then
        missing_deps+=("yamllint")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_info "Install with: sudo apt-get install yq yamllint"
        exit 1
    fi
}

# Validate YAML syntax
validate_yaml_syntax() {
    local file="$1"
    log_info "Validating YAML syntax for $(basename "$file")"
    
    if ! yamllint -d relaxed "$file"; then
        log_error "YAML syntax validation failed for $file"
        return 1
    fi
    
    return 0
}

# Validate workflow structure
validate_workflow_structure() {
    local file="$1"
    local filename=$(basename "$file")
    log_info "Validating workflow structure for $filename"
    
    # Check required top-level keys
    local required_keys=("name" "on" "jobs")
    for key in "${required_keys[@]}"; do
        if ! yq eval "has(\"$key\")" "$file" | grep -q "true"; then
            log_error "Missing required key '$key' in $filename"
            return 1
        fi
    done
    
    # Check that jobs exist
    local job_count=$(yq eval '.jobs | length' "$file")
    if [ "$job_count" -eq 0 ]; then
        log_error "No jobs defined in $filename"
        return 1
    fi
    
    # Check job structure
    local jobs=$(yq eval '.jobs | keys | .[]' "$file")
    while IFS= read -r job; do
        if ! yq eval ".jobs.$job | has(\"runs-on\")" "$file" | grep -q "true"; then
            log_error "Job '$job' missing 'runs-on' in $filename"
            return 1
        fi
        
        if ! yq eval ".jobs.$job | has(\"steps\")" "$file" | grep -q "true"; then
            log_error "Job '$job' missing 'steps' in $filename"
            return 1
        fi
    done <<< "$jobs"
    
    return 0
}

# Validate security best practices
validate_security_practices() {
    local file="$1"
    local filename=$(basename "$file")
    log_info "Validating security practices for $filename"
    
    # Check for pinned action versions
    local unpinned_actions=$(yq eval '.jobs.*.steps[].uses | select(. != null) | select(. | test("@[a-f0-9]{40}$") | not) | select(. | test("@v[0-9]+$") | not)' "$file" 2>/dev/null || true)
    if [ -n "$unpinned_actions" ]; then
        log_warn "Found unpinned actions in $filename (consider pinning to specific versions):"
        echo "$unpinned_actions"
    fi
    
    # Check for secrets in workflow files
    if grep -i "password\|secret\|token" "$file" | grep -v "secrets\." | grep -v "github.token" > /dev/null; then
        log_warn "Potential hardcoded secrets found in $filename"
    fi
    
    # Check for proper permissions
    if yq eval 'has("permissions")' "$file" | grep -q "true"; then
        log_info "Permissions defined in $filename"
    else
        log_warn "No permissions defined in $filename (consider adding explicit permissions)"
    fi
    
    return 0
}

# Validate caching strategy
validate_caching() {
    local file="$1"
    local filename=$(basename "$file")
    
    # Check if Go workflows have proper caching
    if grep -q "setup-go" "$file"; then
        if ! grep -q "cache: true" "$file" && ! grep -q "actions/cache" "$file"; then
            log_warn "Go workflow $filename should include caching for better performance"
        fi
    fi
    
    # Check if Docker workflows have proper caching
    if grep -q "docker/build-push-action" "$file"; then
        if ! grep -q "cache-from\|cache-to" "$file"; then
            log_warn "Docker workflow $filename should include build caching"
        fi
    fi
}

# Main validation function
validate_workflow() {
    local file="$1"
    local errors=0
    
    log_info "Validating workflow: $(basename "$file")"
    
    validate_yaml_syntax "$file" || ((errors++))
    validate_workflow_structure "$file" || ((errors++))
    validate_security_practices "$file"
    validate_caching "$file"
    
    return $errors
}

# Main execution
main() {
    log_info "Starting GitHub Actions workflow validation"
    
    check_dependencies
    
    if [ ! -d "$WORKFLOWS_DIR" ]; then
        log_error "Workflows directory not found: $WORKFLOWS_DIR"
        exit 1
    fi
    
    local total_errors=0
    local workflow_count=0
    
    # Validate all workflow files
    for workflow_file in "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml; do
        if [ -f "$workflow_file" ]; then
            ((workflow_count++))
            validate_workflow "$workflow_file" || ((total_errors++))
            echo
        fi
    done
    
    # Summary
    log_info "Validation complete"
    log_info "Workflows validated: $workflow_count"
    
    if [ $total_errors -eq 0 ]; then
        log_info "All workflows passed validation!"
        exit 0
    else
        log_error "Validation failed with $total_errors errors"
        exit 1
    fi
}

# Run main function
main "$@"