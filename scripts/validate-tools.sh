#!/bin/bash
# validate-tools.sh - Validates required development tools and their versions
# This script is used in CI and local development to ensure consistent tooling

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Expected versions (these should match .devcontainer/post-create.sh)
EXPECTED_GO_VERSION="1.22"
EXPECTED_GOLANGCI_VERSION="1.55.2"
EXPECTED_TASK_VERSION="3.35.1"
EXPECTED_GOOSE_VERSION="3.18.0"
EXPECTED_SQLC_VERSION="1.25.0"
EXPECTED_GOSEC_VERSION="2.19.0"
EXPECTED_GITLEAKS_VERSION="8.18.1"
EXPECTED_PRECOMMIT_VERSION="3.6.0"
EXPECTED_NATS_VERSION="0.1.4"

# Validation results
VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    ((VALIDATION_WARNINGS++))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((VALIDATION_ERRORS++))
}

# Version comparison helper
version_ge() {
    printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

# Validation functions
validate_go() {
    log_info "Validating Go installation..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        return 1
    fi
    
    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local major_minor=$(echo "$go_version" | cut -d. -f1,2)
    
    if version_ge "$major_minor" "$EXPECTED_GO_VERSION"; then
        log_success "Go $go_version (>= $EXPECTED_GO_VERSION required)"
    else
        log_error "Go $go_version found, but >= $EXPECTED_GO_VERSION required"
        return 1
    fi
    
    # Validate GOPATH and GOROOT
    local gopath=$(go env GOPATH)
    local goroot=$(go env GOROOT)
    
    if [[ -z "$gopath" ]]; then
        log_warning "GOPATH is not set"
    else
        log_success "GOPATH: $gopath"
    fi
    
    if [[ -z "$goroot" ]]; then
        log_warning "GOROOT is not set"
    else
        log_success "GOROOT: $goroot"
    fi
}

validate_task() {
    log_info "Validating Task runner..."
    
    if ! command -v task &> /dev/null; then
        log_error "Task is not installed"
        return 1
    fi
    
    local task_version=$(task --version | awk '{print $3}' | sed 's/v//')
    
    if version_ge "$task_version" "$EXPECTED_TASK_VERSION"; then
        log_success "Task $task_version (>= $EXPECTED_TASK_VERSION required)"
    else
        log_warning "Task $task_version found, expected $EXPECTED_TASK_VERSION"
    fi
}

validate_postgresql() {
    log_info "Validating PostgreSQL client..."
    
    if ! command -v psql &> /dev/null; then
        log_error "PostgreSQL client (psql) is not installed"
        return 1
    fi
    
    local psql_version=$(psql --version | awk '{print $3}')
    log_success "PostgreSQL client $psql_version"
}

validate_nats() {
    log_info "Validating NATS CLI..."
    
    if ! command -v nats &> /dev/null; then
        log_error "NATS CLI is not installed"
        return 1
    fi
    
    # NATS CLI version output varies, so we just check if it runs
    if nats --version &> /dev/null || nats --help &> /dev/null; then
        log_success "NATS CLI is installed and functional"
    else
        log_error "NATS CLI is installed but not functional"
        return 1
    fi
}

validate_golangci_lint() {
    log_info "Validating golangci-lint..."
    
    if ! command -v golangci-lint &> /dev/null; then
        log_error "golangci-lint is not installed"
        return 1
    fi
    
    local golangci_version=$(golangci-lint version | head -1 | awk '{print $4}' | sed 's/v//')
    
    if version_ge "$golangci_version" "$EXPECTED_GOLANGCI_VERSION"; then
        log_success "golangci-lint $golangci_version (>= $EXPECTED_GOLANGCI_VERSION required)"
    else
        log_warning "golangci-lint $golangci_version found, expected $EXPECTED_GOLANGCI_VERSION"
    fi
}

validate_goose() {
    log_info "Validating goose..."
    
    if ! command -v goose &> /dev/null; then
        log_error "goose is not installed"
        return 1
    fi
    
    # goose version output varies, so we just check if it runs
    if goose -version &> /dev/null; then
        log_success "goose is installed and functional"
    else
        log_error "goose is installed but not functional"
        return 1
    fi
}

validate_sqlc() {
    log_info "Validating sqlc..."
    
    if ! command -v sqlc &> /dev/null; then
        log_error "sqlc is not installed"
        return 1
    fi
    
    local sqlc_version=$(sqlc version | sed 's/v//')
    
    if version_ge "$sqlc_version" "$EXPECTED_SQLC_VERSION"; then
        log_success "sqlc $sqlc_version (>= $EXPECTED_SQLC_VERSION required)"
    else
        log_warning "sqlc $sqlc_version found, expected $EXPECTED_SQLC_VERSION"
    fi
}

validate_gosec() {
    log_info "Validating gosec..."
    
    if ! command -v gosec &> /dev/null; then
        log_error "gosec is not installed"
        return 1
    fi
    
    # gosec version output varies, so we just check if it runs
    if gosec -version &> /dev/null; then
        log_success "gosec is installed and functional"
    else
        log_error "gosec is installed but not functional"
        return 1
    fi
}

validate_gitleaks() {
    log_info "Validating gitleaks..."
    
    if ! command -v gitleaks &> /dev/null; then
        log_error "gitleaks is not installed"
        return 1
    fi
    
    local gitleaks_version=$(gitleaks version | awk '{print $2}' | sed 's/v//')
    
    if version_ge "$gitleaks_version" "$EXPECTED_GITLEAKS_VERSION"; then
        log_success "gitleaks $gitleaks_version (>= $EXPECTED_GITLEAKS_VERSION required)"
    else
        log_warning "gitleaks $gitleaks_version found, expected $EXPECTED_GITLEAKS_VERSION"
    fi
}

validate_precommit() {
    log_info "Validating pre-commit..."
    
    if ! command -v pre-commit &> /dev/null; then
        log_error "pre-commit is not installed"
        return 1
    fi
    
    local precommit_version=$(pre-commit --version | awk '{print $2}')
    
    if version_ge "$precommit_version" "$EXPECTED_PRECOMMIT_VERSION"; then
        log_success "pre-commit $precommit_version (>= $EXPECTED_PRECOMMIT_VERSION required)"
    else
        log_warning "pre-commit $precommit_version found, expected $EXPECTED_PRECOMMIT_VERSION"
    fi
}

validate_docker() {
    log_info "Validating Docker..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        return 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running or not accessible"
        return 1
    fi
    
    local docker_version=$(docker --version | awk '{print $3}' | sed 's/,//')
    log_success "Docker $docker_version"
}

validate_make() {
    log_info "Validating Make..."
    
    if ! command -v make &> /dev/null; then
        log_error "Make is not installed"
        return 1
    fi
    
    local make_version=$(make --version | head -1 | awk '{print $3}')
    log_success "Make $make_version"
}

# Environment detection
detect_environment() {
    log_info "Detecting environment..."
    
    if [[ -n "${DEVCONTAINER}" ]]; then
        log_success "Running in VS Code devcontainer"
        return 0
    elif [[ -n "${CODESPACES}" ]]; then
        log_success "Running in GitHub Codespaces"
        return 0
    elif [[ -f "/.dockerenv" ]]; then
        log_success "Running in Docker container"
        return 0
    else
        log_warning "Running on host system (consider using devcontainer for consistency)"
        return 1
    fi
}

# Main validation function
main() {
    echo "🔍 AgentFlow Development Tools Validation"
    echo "========================================"
    
    # Detect environment
    detect_environment
    echo
    
    # Run all validations
    validate_go || true
    validate_task || true
    validate_postgresql || true
    validate_nats || true
    validate_golangci_lint || true
    validate_goose || true
    validate_sqlc || true
    validate_gosec || true
    validate_gitleaks || true
    validate_precommit || true
    validate_docker || true
    validate_make || true
    
    echo
    echo "========================================"
    
    # Summary
    if [[ $VALIDATION_ERRORS -eq 0 ]]; then
        if [[ $VALIDATION_WARNINGS -eq 0 ]]; then
            log_success "All tools validated successfully! 🎉"
            exit 0
        else
            log_warning "$VALIDATION_WARNINGS warning(s) found, but all required tools are present"
            exit 0
        fi
    else
        log_error "$VALIDATION_ERRORS error(s) and $VALIDATION_WARNINGS warning(s) found"
        echo
        echo "To fix these issues:"
        echo "1. Use the VS Code devcontainer for a consistent environment"
        echo "2. Run the setup script: .devcontainer/post-create.sh"
        echo "3. Check the documentation: docs/dev-environment.md"
        exit 1
    fi
}

# Run main function
main "$@"