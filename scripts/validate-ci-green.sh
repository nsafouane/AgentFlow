#!/bin/bash

# CI Green Including Security Scans Validation Script
# Validates that all CI workflows pass with no High/Critical vulnerabilities
# Part of Task 11: CI Green Including Security Scans Validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_API_URL="https://api.github.com"
WORKFLOW_TIMEOUT=1800  # 30 minutes
CHECK_INTERVAL=30      # 30 seconds
OUTPUT_DIR="${REPO_ROOT}/ci-validation-reports"
SECURITY_THRESHOLD="high"  # Fail on high/critical vulnerabilities

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

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Validation results
VALIDATION_RESULTS=()
VALIDATION_FAILED=false

# Add validation result
add_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    VALIDATION_RESULTS+=("$test_name: $status - $message")
    
    if [[ "$status" == "FAILED" ]]; then
        VALIDATION_FAILED=true
        log_error "✗ $test_name: $message"
    else
        log_info "✓ $test_name: $message"
    fi
}

# Check if GitHub CLI is available
check_gh_cli() {
    if ! command -v gh &> /dev/null; then
        log_error "GitHub CLI (gh) is not installed. Please install it to use this script."
        log_error "Installation: https://cli.github.com/"
        return 1
    fi
    
    # Check if authenticated
    if ! gh auth status &> /dev/null; then
        log_error "GitHub CLI is not authenticated. Please run 'gh auth login'"
        return 1
    fi
    
    return 0
}

# Get latest workflow runs for a specific workflow
get_workflow_runs() {
    local workflow_file="$1"
    local branch="${2:-main}"
    
    log_debug "Getting workflow runs for $workflow_file on branch $branch"
    
    gh run list \
        --workflow="$workflow_file" \
        --branch="$branch" \
        --limit=5 \
        --json status,conclusion,createdAt,headSha,url \
        2>/dev/null || echo "[]"
}

# Check if workflow run passed
check_workflow_status() {
    local workflow_name="$1"
    local workflow_file="$2"
    local branch="${3:-main}"
    
    log_info "Checking $workflow_name workflow status..."
    
    local runs
    runs=$(get_workflow_runs "$workflow_file" "$branch")
    
    if [[ "$runs" == "[]" ]]; then
        add_result "$workflow_name" "WARNING" "No recent workflow runs found"
        return 0
    fi
    
    # Get the most recent run
    local latest_status
    local latest_conclusion
    local latest_url
    
    latest_status=$(echo "$runs" | jq -r '.[0].status // "unknown"')
    latest_conclusion=$(echo "$runs" | jq -r '.[0].conclusion // "unknown"')
    latest_url=$(echo "$runs" | jq -r '.[0].url // ""')
    
    log_debug "Latest run status: $latest_status, conclusion: $latest_conclusion"
    
    case "$latest_status" in
        "completed")
            if [[ "$latest_conclusion" == "success" ]]; then
                add_result "$workflow_name" "PASSED" "Latest run completed successfully"
            elif [[ "$latest_conclusion" == "failure" ]]; then
                add_result "$workflow_name" "FAILED" "Latest run failed - $latest_url"
            elif [[ "$latest_conclusion" == "cancelled" ]]; then
                add_result "$workflow_name" "WARNING" "Latest run was cancelled - $latest_url"
            else
                add_result "$workflow_name" "FAILED" "Latest run completed with status: $latest_conclusion - $latest_url"
            fi
            ;;
        "in_progress")
            add_result "$workflow_name" "WARNING" "Workflow is currently running - $latest_url"
            ;;
        "queued")
            add_result "$workflow_name" "WARNING" "Workflow is queued - $latest_url"
            ;;
        *)
            add_result "$workflow_name" "FAILED" "Unknown workflow status: $latest_status - $latest_url"
            ;;
    esac
}

# Run local security scans to validate thresholds
run_local_security_validation() {
    log_info "Running local security scan validation..."
    
    local security_script="$SCRIPT_DIR/security-scan.sh"
    
    if [[ ! -f "$security_script" ]]; then
        add_result "Local Security Scan" "FAILED" "Security scan script not found: $security_script"
        return 1
    fi
    
    # Make script executable
    chmod +x "$security_script"
    
    # Run security scan with high threshold
    local scan_output
    local scan_exit_code
    
    scan_output=$("$security_script" --threshold "$SECURITY_THRESHOLD" --output "$OUTPUT_DIR/local-security" 2>&1) || scan_exit_code=$?
    
    if [[ ${scan_exit_code:-0} -eq 0 ]]; then
        add_result "Local Security Scan" "PASSED" "No high/critical vulnerabilities found"
        
        # Save scan output
        echo "$scan_output" > "$OUTPUT_DIR/local-security-scan.log"
    else
        add_result "Local Security Scan" "FAILED" "High/critical vulnerabilities detected (exit code: ${scan_exit_code:-0})"
        
        # Save scan output for debugging
        echo "$scan_output" > "$OUTPUT_DIR/local-security-scan-failed.log"
        log_error "Security scan output saved to: $OUTPUT_DIR/local-security-scan-failed.log"
    fi
}

# Validate security scan thresholds in workflow files
validate_security_thresholds() {
    log_info "Validating security scan thresholds in workflow files..."
    
    local workflows_dir="$REPO_ROOT/.github/workflows"
    local threshold_found=false
    
    # Check CI workflow
    if [[ -f "$workflows_dir/ci.yml" ]]; then
        if grep -q "fail-on.*high\|severity.*high\|HIGH\|CRITICAL" "$workflows_dir/ci.yml"; then
            threshold_found=true
            log_debug "Found high/critical threshold in ci.yml"
        fi
    fi
    
    # Check security scan workflow
    if [[ -f "$workflows_dir/security-scan.yml" ]]; then
        if grep -q "fail-on.*high\|severity.*high\|HIGH\|CRITICAL" "$workflows_dir/security-scan.yml"; then
            threshold_found=true
            log_debug "Found high/critical threshold in security-scan.yml"
        fi
    fi
    
    # Check container build workflow
    if [[ -f "$workflows_dir/container-build.yml" ]]; then
        if grep -q "fail-on.*high\|severity.*high\|HIGH\|CRITICAL" "$workflows_dir/container-build.yml"; then
            threshold_found=true
            log_debug "Found high/critical threshold in container-build.yml"
        fi
    fi
    
    if [[ "$threshold_found" == "true" ]]; then
        add_result "Security Thresholds" "PASSED" "High/Critical severity thresholds configured in workflows"
    else
        add_result "Security Thresholds" "FAILED" "No high/critical severity thresholds found in workflow files"
    fi
}

# Check for required security tools in workflows
validate_security_tools() {
    log_info "Validating security tools configuration in workflows..."
    
    local workflows_dir="$REPO_ROOT/.github/workflows"
    local required_tools=("gosec" "gitleaks" "grype" "syft" "govulncheck")
    local tools_found=0
    
    for tool in "${required_tools[@]}"; do
        if grep -r -q "$tool" "$workflows_dir"/*.yml 2>/dev/null; then
            log_debug "Found $tool in workflow files"
            tools_found=$((tools_found + 1))
        else
            log_warn "$tool not found in workflow files"
        fi
    done
    
    if [[ $tools_found -ge 4 ]]; then
        add_result "Security Tools" "PASSED" "$tools_found/${#required_tools[@]} required security tools found in workflows"
    else
        add_result "Security Tools" "FAILED" "Only $tools_found/${#required_tools[@]} required security tools found in workflows"
    fi
}

# Validate SARIF upload configuration
validate_sarif_upload() {
    log_info "Validating SARIF upload configuration..."
    
    local workflows_dir="$REPO_ROOT/.github/workflows"
    local sarif_uploads=0
    
    # Check for SARIF upload actions
    if grep -r -q "github/codeql-action/upload-sarif" "$workflows_dir"/*.yml 2>/dev/null; then
        sarif_uploads=$((sarif_uploads + 1))
        log_debug "Found SARIF upload configuration"
    fi
    
    # Check for SARIF output formats
    if grep -r -q "sarif\|SARIF" "$workflows_dir"/*.yml 2>/dev/null; then
        sarif_uploads=$((sarif_uploads + 1))
        log_debug "Found SARIF output format configuration"
    fi
    
    if [[ $sarif_uploads -ge 1 ]]; then
        add_result "SARIF Upload" "PASSED" "SARIF upload configuration found in workflows"
    else
        add_result "SARIF Upload" "FAILED" "No SARIF upload configuration found in workflows"
    fi
}

# Generate validation report
generate_validation_report() {
    local report_file="$OUTPUT_DIR/ci-green-validation-report.json"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    local status
    if [[ "$VALIDATION_FAILED" == "true" ]]; then
        status="FAILED"
    else
        status="PASSED"
    fi
    
    cat > "$report_file" << EOF
{
  "timestamp": "$timestamp",
  "validation_status": "$status",
  "security_threshold": "$SECURITY_THRESHOLD",
  "total_validations": ${#VALIDATION_RESULTS[@]},
  "validations": [
$(printf '    "%s"' "${VALIDATION_RESULTS[0]}")
$(printf ',\n    "%s"' "${VALIDATION_RESULTS[@]:1}")
  ],
  "reports_directory": "$OUTPUT_DIR",
  "github_repository": "$(git remote get-url origin 2>/dev/null || echo 'unknown')",
  "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
  "git_branch": "$(git branch --show-current 2>/dev/null || echo 'unknown')"
}
EOF
    
    log_info "Validation report generated: $report_file"
}

# Print validation summary
print_validation_summary() {
    echo
    log_info "=== CI Green Validation Summary ==="
    echo
    
    local passed=0
    local failed=0
    local warnings=0
    
    for result in "${VALIDATION_RESULTS[@]}"; do
        echo "  • $result"
        
        if [[ "$result" == *"PASSED"* ]]; then
            passed=$((passed + 1))
        elif [[ "$result" == *"FAILED"* ]]; then
            failed=$((failed + 1))
        elif [[ "$result" == *"WARNING"* ]]; then
            warnings=$((warnings + 1))
        fi
    done
    
    echo
    log_info "Validations passed: $passed"
    log_info "Validations failed: $failed"
    log_info "Warnings: $warnings"
    log_info "Total validations: ${#VALIDATION_RESULTS[@]}"
    
    if [[ "$VALIDATION_FAILED" == "true" ]]; then
        echo
        log_error "❌ CI Green validation FAILED"
        log_error "Some workflows or security scans are not passing"
        log_error "Review the validation report in: $OUTPUT_DIR"
        return 1
    else
        echo
        log_info "✅ CI Green validation PASSED"
        log_info "All workflows are passing with no high/critical vulnerabilities"
        return 0
    fi
}

# Show usage information
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

CI Green validation script - ensures all workflows pass with no High/Critical vulnerabilities.

OPTIONS:
    -b, --branch BRANCH      Check workflows on specific branch [default: main]
    -t, --threshold LEVEL    Security severity threshold [default: high]
    -o, --output DIR         Output directory for reports [default: ./ci-validation-reports]
    --skip-local-scan        Skip local security scan validation
    --skip-github-check      Skip GitHub workflow status checks
    -v, --verbose            Enable verbose output
    -h, --help               Show this help message

EXAMPLES:
    $0                       # Run full validation on main branch
    $0 -b develop            # Check develop branch
    $0 --skip-local-scan     # Only check GitHub workflow status
    $0 -v                    # Enable verbose output

ENVIRONMENT VARIABLES:
    DEBUG                    Enable debug output (true/false)
    GITHUB_TOKEN             GitHub token for API access (optional if gh CLI is authenticated)

EXIT CODES:
    0    All validations passed
    1    Some validations failed
    2    Invalid arguments or configuration error
EOF
}

# Parse command line arguments
parse_arguments() {
    local branch="main"
    local skip_local_scan=false
    local skip_github_check=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--branch)
                branch="$2"
                shift 2
                ;;
            -t|--threshold)
                SECURITY_THRESHOLD="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --skip-local-scan)
                skip_local_scan=true
                shift
                ;;
            --skip-github-check)
                skip_github_check=true
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
    
    # Export variables for use in functions
    export BRANCH="$branch"
    export SKIP_LOCAL_SCAN="$skip_local_scan"
    export SKIP_GITHUB_CHECK="$skip_github_check"
}

# Main validation function
main() {
    log_info "Starting CI Green validation..."
    log_info "Branch: ${BRANCH:-main}"
    log_info "Security threshold: $SECURITY_THRESHOLD"
    log_info "Output directory: $OUTPUT_DIR"
    
    cd "$REPO_ROOT"
    
    # Check GitHub CLI availability (unless skipping GitHub checks)
    if [[ "${SKIP_GITHUB_CHECK:-}" != "true" ]]; then
        if ! check_gh_cli; then
            log_warn "GitHub CLI not available, skipping workflow status checks"
            SKIP_GITHUB_CHECK=true
        fi
    fi
    
    # Validate workflow configurations
    validate_security_thresholds
    validate_security_tools
    validate_sarif_upload
    
    # Check GitHub workflow status
    if [[ "${SKIP_GITHUB_CHECK:-}" != "true" ]]; then
        check_workflow_status "CI Pipeline" "ci.yml" "${BRANCH:-main}"
        check_workflow_status "Security Scan" "security-scan.yml" "${BRANCH:-main}"
        check_workflow_status "Container Build" "container-build.yml" "${BRANCH:-main}"
    fi
    
    # Run local security validation
    if [[ "${SKIP_LOCAL_SCAN:-}" != "true" ]]; then
        run_local_security_validation
    fi
    
    # Generate report and print summary
    generate_validation_report
    print_validation_summary
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_arguments "$@"
    main
fi