#!/bin/bash

# Security Scanning Script
# Runs comprehensive security scans with configurable severity thresholds
# Fails on High/Critical vulnerabilities by default

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default configuration
DEFAULT_SEVERITY_THRESHOLD="high"
DEFAULT_OUTPUT_DIR="${REPO_ROOT}/security-reports"
DEFAULT_CONFIG_FILE="${REPO_ROOT}/.security-config.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_debug() {
    if [[ "${DEBUG:-}" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# Configuration
SEVERITY_THRESHOLD="${SEVERITY_THRESHOLD:-$DEFAULT_SEVERITY_THRESHOLD}"
OUTPUT_DIR="${OUTPUT_DIR:-$DEFAULT_OUTPUT_DIR}"
CONFIG_FILE="${CONFIG_FILE:-$DEFAULT_CONFIG_FILE}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Security scan results
SCAN_RESULTS=()
TOTAL_VULNERABILITIES=0
HIGH_CRITICAL_VULNERABILITIES=0
SCAN_FAILED=false

# Parse severity threshold to numeric value for comparison
severity_to_numeric() {
    case "${1,,}" in
        "critical") echo 4 ;;
        "high") echo 3 ;;
        "medium") echo 2 ;;
        "low") echo 1 ;;
        "info") echo 0 ;;
        *) echo 2 ;; # Default to medium
    esac
}

# Check if severity meets threshold
meets_threshold() {
    local severity="$1"
    local threshold_numeric
    local severity_numeric
    
    threshold_numeric=$(severity_to_numeric "$SEVERITY_THRESHOLD")
    severity_numeric=$(severity_to_numeric "$severity")
    
    [[ $severity_numeric -ge $threshold_numeric ]]
}

# Install security tools if not available
install_security_tools() {
    log_info "Checking security tool availability..."
    
    # Check gosec
    if ! command -v gosec &> /dev/null; then
        log_warn "gosec not found, installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    # Check govulncheck
    if ! command -v govulncheck &> /dev/null; then
        log_warn "govulncheck not found, installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
    
    # Check gitleaks
    if ! command -v gitleaks &> /dev/null; then
        log_warn "gitleaks not found, attempting to install..."
        if command -v brew &> /dev/null; then
            brew install gitleaks
        elif command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y gitleaks
        else
            log_warn "Please install gitleaks manually: https://github.com/gitleaks/gitleaks#installing"
        fi
    fi
    
    # Check syft
    if ! command -v syft &> /dev/null; then
        log_warn "syft not found, installing..."
        curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
    fi
    
    # Check grype
    if ! command -v grype &> /dev/null; then
        log_warn "grype not found, installing..."
        curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin
    fi
}

# Run gosec security scanner
run_gosec() {
    log_info "Running gosec security scanner..."
    
    local output_file="$OUTPUT_DIR/gosec-report.json"
    local sarif_file="$OUTPUT_DIR/gosec-report.sarif"
    
    if gosec -fmt json -out "$output_file" -fmt sarif -out "$sarif_file" ./...; then
        log_info "✓ gosec scan completed successfully"
        
        # Parse results
        if [[ -f "$output_file" ]]; then
            local issues
            issues=$(jq -r '.Issues // [] | length' "$output_file" 2>/dev/null || echo "0")
            local high_critical
            high_critical=$(jq -r '.Issues // [] | map(select(.severity == "HIGH" or .severity == "CRITICAL")) | length' "$output_file" 2>/dev/null || echo "0")
            
            SCAN_RESULTS+=("gosec: $issues total issues, $high_critical high/critical")
            TOTAL_VULNERABILITIES=$((TOTAL_VULNERABILITIES + issues))
            HIGH_CRITICAL_VULNERABILITIES=$((HIGH_CRITICAL_VULNERABILITIES + high_critical))
            
            if [[ $high_critical -gt 0 ]] && meets_threshold "high"; then
                log_error "gosec found $high_critical high/critical security issues"
                SCAN_FAILED=true
            fi
        fi
    else
        log_error "gosec scan failed"
        SCAN_RESULTS+=("gosec: FAILED")
        SCAN_FAILED=true
    fi
}

# Run govulncheck vulnerability scanner
run_govulncheck() {
    log_info "Running govulncheck vulnerability scanner..."
    
    local output_file="$OUTPUT_DIR/govulncheck-report.json"
    
    if govulncheck -json ./... > "$output_file" 2>&1; then
        log_info "✓ govulncheck scan completed successfully"
        
        # Parse results
        if [[ -f "$output_file" ]]; then
            local vulns
            vulns=$(jq -r '[.[] | select(.finding)] | length' "$output_file" 2>/dev/null || echo "0")
            
            SCAN_RESULTS+=("govulncheck: $vulns vulnerabilities found")
            TOTAL_VULNERABILITIES=$((TOTAL_VULNERABILITIES + vulns))
            
            if [[ $vulns -gt 0 ]] && meets_threshold "high"; then
                log_error "govulncheck found $vulns vulnerabilities"
                HIGH_CRITICAL_VULNERABILITIES=$((HIGH_CRITICAL_VULNERABILITIES + vulns))
                SCAN_FAILED=true
            fi
        fi
    else
        log_error "govulncheck scan failed"
        SCAN_RESULTS+=("govulncheck: FAILED")
        SCAN_FAILED=true
    fi
}

# Run OSV Scanner (if available)
run_osv_scanner() {
    log_info "Running OSV Scanner..."
    
    local output_file="$OUTPUT_DIR/osv-scanner-report.json"
    
    if command -v osv-scanner &> /dev/null; then
        if osv-scanner --format json --output "$output_file" .; then
            log_info "✓ OSV Scanner completed successfully"
            
            # Parse results
            if [[ -f "$output_file" ]]; then
                local vulns
                vulns=$(jq -r '.results[].packages[].vulnerabilities // [] | length' "$output_file" 2>/dev/null || echo "0")
                
                SCAN_RESULTS+=("osv-scanner: $vulns vulnerabilities found")
                TOTAL_VULNERABILITIES=$((TOTAL_VULNERABILITIES + vulns))
                
                if [[ $vulns -gt 0 ]] && meets_threshold "high"; then
                    log_error "OSV Scanner found $vulns vulnerabilities"
                    HIGH_CRITICAL_VULNERABILITIES=$((HIGH_CRITICAL_VULNERABILITIES + vulns))
                    SCAN_FAILED=true
                fi
            fi
        else
            log_warn "OSV Scanner failed, continuing..."
            SCAN_RESULTS+=("osv-scanner: FAILED")
        fi
    else
        log_warn "OSV Scanner not available, skipping..."
        SCAN_RESULTS+=("osv-scanner: NOT_AVAILABLE")
    fi
}

# Run gitleaks secret scanner
run_gitleaks() {
    log_info "Running gitleaks secret scanner..."
    
    local output_file="$OUTPUT_DIR/gitleaks-report.json"
    
    if gitleaks detect --source="$REPO_ROOT" --report-format json --report-path "$output_file" --verbose; then
        log_info "✓ gitleaks scan completed - no secrets found"
        SCAN_RESULTS+=("gitleaks: no secrets detected")
    else
        local exit_code=$?
        if [[ $exit_code -eq 1 ]]; then
            # Exit code 1 means secrets were found
            local secrets_count
            secrets_count=$(jq '. | length' "$output_file" 2>/dev/null || echo "unknown")
            log_error "gitleaks found $secrets_count secrets"
            SCAN_RESULTS+=("gitleaks: $secrets_count secrets found")
            HIGH_CRITICAL_VULNERABILITIES=$((HIGH_CRITICAL_VULNERABILITIES + secrets_count))
            SCAN_FAILED=true
        else
            log_error "gitleaks scan failed with exit code $exit_code"
            SCAN_RESULTS+=("gitleaks: FAILED")
            SCAN_FAILED=true
        fi
    fi
}

# Run syft SBOM generation
run_syft() {
    log_info "Running syft SBOM generation..."
    
    local sbom_file="$OUTPUT_DIR/sbom.spdx.json"
    local cyclone_file="$OUTPUT_DIR/sbom.cyclonedx.json"
    
    if syft packages . -o spdx-json="$sbom_file" -o cyclonedx-json="$cyclone_file"; then
        log_info "✓ syft SBOM generation completed"
        
        # Count packages
        local package_count
        package_count=$(jq -r '.packages | length' "$sbom_file" 2>/dev/null || echo "unknown")
        SCAN_RESULTS+=("syft: $package_count packages cataloged")
    else
        log_error "syft SBOM generation failed"
        SCAN_RESULTS+=("syft: FAILED")
    fi
}

# Run grype vulnerability scanner
run_grype() {
    log_info "Running grype vulnerability scanner..."
    
    local output_file="$OUTPUT_DIR/grype-report.json"
    local sarif_file="$OUTPUT_DIR/grype-report.sarif"
    
    # Run grype with different output formats
    if grype . -o json --file "$output_file" -o sarif --file "$sarif_file"; then
        log_info "✓ grype scan completed successfully"
        
        # Parse results
        if [[ -f "$output_file" ]]; then
            local total_vulns
            local high_critical_vulns
            
            total_vulns=$(jq -r '.matches | length' "$output_file" 2>/dev/null || echo "0")
            high_critical_vulns=$(jq -r '.matches | map(select(.vulnerability.severity == "High" or .vulnerability.severity == "Critical")) | length' "$output_file" 2>/dev/null || echo "0")
            
            SCAN_RESULTS+=("grype: $total_vulns total vulnerabilities, $high_critical_vulns high/critical")
            TOTAL_VULNERABILITIES=$((TOTAL_VULNERABILITIES + total_vulns))
            HIGH_CRITICAL_VULNERABILITIES=$((HIGH_CRITICAL_VULNERABILITIES + high_critical_vulns))
            
            if [[ $high_critical_vulns -gt 0 ]] && meets_threshold "high"; then
                log_error "grype found $high_critical_vulns high/critical vulnerabilities"
                SCAN_FAILED=true
            fi
        fi
    else
        log_warn "grype scan failed, continuing..."
        SCAN_RESULTS+=("grype: FAILED")
    fi
}

# Generate summary report
generate_summary_report() {
    local summary_file="$OUTPUT_DIR/security-summary.json"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    cat > "$summary_file" << EOF
{
  "timestamp": "$timestamp",
  "severity_threshold": "$SEVERITY_THRESHOLD",
  "scan_status": "$(if [[ "$SCAN_FAILED" == "true" ]]; then echo "FAILED"; else echo "PASSED"; fi)",
  "total_vulnerabilities": $TOTAL_VULNERABILITIES,
  "high_critical_vulnerabilities": $HIGH_CRITICAL_VULNERABILITIES,
  "scan_results": [
$(printf '    "%s"' "${SCAN_RESULTS[0]}")
$(printf ',\n    "%s"' "${SCAN_RESULTS[@]:1}")
  ],
  "reports": {
    "gosec": "$OUTPUT_DIR/gosec-report.json",
    "govulncheck": "$OUTPUT_DIR/govulncheck-report.json",
    "osv_scanner": "$OUTPUT_DIR/osv-scanner-report.json",
    "gitleaks": "$OUTPUT_DIR/gitleaks-report.json",
    "grype": "$OUTPUT_DIR/grype-report.json",
    "sbom_spdx": "$OUTPUT_DIR/sbom.spdx.json",
    "sbom_cyclonedx": "$OUTPUT_DIR/sbom.cyclonedx.json"
  }
}
EOF
    
    log_info "Security summary report generated: $summary_file"
}

# Print scan results
print_results() {
    echo
    log_info "=== Security Scan Results ==="
    echo
    
    for result in "${SCAN_RESULTS[@]}"; do
        echo "  • $result"
    done
    
    echo
    log_info "Total vulnerabilities found: $TOTAL_VULNERABILITIES"
    log_info "High/Critical vulnerabilities: $HIGH_CRITICAL_VULNERABILITIES"
    log_info "Severity threshold: $SEVERITY_THRESHOLD"
    
    if [[ "$SCAN_FAILED" == "true" ]]; then
        echo
        log_error "❌ Security scan FAILED - vulnerabilities above threshold detected"
        log_error "Review reports in: $OUTPUT_DIR"
        return 1
    else
        echo
        log_info "✅ Security scan PASSED - no vulnerabilities above threshold"
        return 0
    fi
}

# Show usage information
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Security scanning script with configurable severity thresholds.

OPTIONS:
    -t, --threshold LEVEL    Set severity threshold (critical|high|medium|low) [default: high]
    -o, --output DIR         Set output directory for reports [default: ./security-reports]
    -c, --config FILE        Use custom configuration file
    --install-tools          Install missing security tools
    --skip-install           Skip automatic tool installation
    -v, --verbose            Enable verbose output
    -h, --help               Show this help message

EXAMPLES:
    $0                       # Run with default settings (fail on high/critical)
    $0 -t critical           # Only fail on critical vulnerabilities
    $0 -o /tmp/reports       # Use custom output directory
    $0 --install-tools       # Install missing tools before scanning
    $0 -v                    # Enable verbose output

ENVIRONMENT VARIABLES:
    SEVERITY_THRESHOLD       Override default severity threshold
    OUTPUT_DIR               Override default output directory
    DEBUG                    Enable debug output (true/false)

EXIT CODES:
    0    All scans passed or no vulnerabilities above threshold
    1    Vulnerabilities above threshold detected or scan failed
    2    Invalid arguments or configuration error
EOF
}

# Parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--threshold)
                SEVERITY_THRESHOLD="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -c|--config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --install-tools)
                INSTALL_TOOLS=true
                shift
                ;;
            --skip-install)
                SKIP_INSTALL=true
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
    
    # Validate severity threshold
    case "${SEVERITY_THRESHOLD,,}" in
        critical|high|medium|low|info) ;;
        *)
            log_error "Invalid severity threshold: $SEVERITY_THRESHOLD"
            log_error "Valid options: critical, high, medium, low, info"
            exit 2
            ;;
    esac
}

# Main execution
main() {
    log_info "Starting security scan with threshold: $SEVERITY_THRESHOLD"
    log_info "Output directory: $OUTPUT_DIR"
    
    cd "$REPO_ROOT"
    
    # Install tools if requested or needed
    if [[ "${INSTALL_TOOLS:-}" == "true" ]] || [[ "${SKIP_INSTALL:-}" != "true" ]]; then
        install_security_tools
    fi
    
    # Run security scans
    run_gosec
    run_govulncheck
    run_osv_scanner
    run_gitleaks
    run_syft
    run_grype
    
    # Generate reports
    generate_summary_report
    
    # Print results and exit
    print_results
}

# Handle script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_arguments "$@"
    main
fi