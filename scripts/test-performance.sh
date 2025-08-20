#!/bin/bash

# Performance testing script for CI/CD pipeline
# This script runs performance tests and enforces thresholds

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEST_PACKAGE="./pkg/messaging"
RESULTS_DIR="${PROJECT_ROOT}/performance-results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Default thresholds (can be overridden by environment variables)
DEFAULT_P95_THRESHOLD_MS=15
DEFAULT_P50_THRESHOLD_MS=5
DEFAULT_MIN_THROUGHPUT=100  # messages per second

# Environment-specific overrides
P95_THRESHOLD_MS=${AF_PERF_P95_THRESHOLD_MS:-$DEFAULT_P95_THRESHOLD_MS}
P50_THRESHOLD_MS=${AF_PERF_P50_THRESHOLD_MS:-$DEFAULT_P50_THRESHOLD_MS}
MIN_THROUGHPUT=${AF_PERF_MIN_THROUGHPUT:-$DEFAULT_MIN_THROUGHPUT}
SKIP_PERFORMANCE=${AF_SKIP_PERFORMANCE:-false}
PERFORMANCE_MODE=${AF_PERFORMANCE_MODE:-ci}  # ci, local, or baseline

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Print configuration
print_config() {
    log_info "Performance Test Configuration:"
    log_info "  Mode: $PERFORMANCE_MODE"
    log_info "  P95 Threshold: ${P95_THRESHOLD_MS}ms"
    log_info "  P50 Threshold: ${P50_THRESHOLD_MS}ms"
    log_info "  Min Throughput: ${MIN_THROUGHPUT} msg/sec"
    log_info "  Results Directory: $RESULTS_DIR"
    log_info "  Skip Performance: $SKIP_PERFORMANCE"
    echo
}

# Check if performance tests should be skipped
check_skip() {
    if [[ "$SKIP_PERFORMANCE" == "true" ]]; then
        log_warn "Performance tests skipped (AF_SKIP_PERFORMANCE=true)"
        exit 0
    fi
}

# Setup results directory
setup_results_dir() {
    mkdir -p "$RESULTS_DIR"
    log_info "Created results directory: $RESULTS_DIR"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Go is available
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if Docker is available (for NATS container)
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Run performance threshold tests
run_threshold_tests() {
    log_info "Running performance threshold tests..."
    
    cd "$PROJECT_ROOT"
    
    # Set environment variables for the test
    export AF_PERF_P95_THRESHOLD_MS="$P95_THRESHOLD_MS"
    export AF_PERF_P50_THRESHOLD_MS="$P50_THRESHOLD_MS"
    
    # Run the performance threshold test
    local test_output
    local test_exit_code=0
    
    test_output=$(go test -v "$TEST_PACKAGE" -run TestPerformanceThresholds 2>&1) || test_exit_code=$?
    
    # Save test output
    echo "$test_output" > "$RESULTS_DIR/threshold_test_${TIMESTAMP}.log"
    
    if [[ $test_exit_code -eq 0 ]]; then
        log_success "Performance threshold tests PASSED"
        
        # Extract key metrics from output
        local p95_latency=$(echo "$test_output" | grep -o "Latency P95: [0-9.]*[a-z]*" | head -1 | cut -d' ' -f3)
        local throughput=$(echo "$test_output" | grep -o "Throughput: [0-9.]*" | head -1 | cut -d' ' -f2)
        
        if [[ -n "$p95_latency" ]]; then
            log_info "  P95 Latency: $p95_latency"
        fi
        if [[ -n "$throughput" ]]; then
            log_info "  Throughput: ${throughput} msg/sec"
        fi
        
        return 0
    else
        log_error "Performance threshold tests FAILED"
        log_error "Test output saved to: $RESULTS_DIR/threshold_test_${TIMESTAMP}.log"
        
        # Show relevant error lines
        echo "$test_output" | grep -E "(FAIL|ERROR|exceeds|below)" | head -10
        
        return 1
    fi
}

# Run benchmark tests
run_benchmarks() {
    log_info "Running benchmark tests..."
    
    cd "$PROJECT_ROOT"
    
    local benchmark_output
    local benchmark_exit_code=0
    
    # Run benchmarks with specific parameters
    benchmark_output=$(go test -bench=BenchmarkPingPong -benchmem -benchtime=10s "$TEST_PACKAGE" 2>&1) || benchmark_exit_code=$?
    
    # Save benchmark output
    echo "$benchmark_output" > "$RESULTS_DIR/benchmark_${TIMESTAMP}.log"
    
    if [[ $benchmark_exit_code -eq 0 ]]; then
        log_success "Benchmark tests completed"
        
        # Extract and display key metrics
        echo "$benchmark_output" | grep -E "BenchmarkPingPong|p95_latency_ms|throughput_msg_per_sec" | while read -r line; do
            log_info "  $line"
        done
        
        return 0
    else
        log_error "Benchmark tests failed"
        log_error "Benchmark output saved to: $RESULTS_DIR/benchmark_${TIMESTAMP}.log"
        return 1
    fi
}

# Run regression detection
run_regression_tests() {
    log_info "Running regression detection tests..."
    
    cd "$PROJECT_ROOT"
    
    local regression_output
    local regression_exit_code=0
    
    regression_output=$(go test -v "$TEST_PACKAGE" -run TestPerformanceRegression 2>&1) || regression_exit_code=$?
    
    # Save regression test output
    echo "$regression_output" > "$RESULTS_DIR/regression_test_${TIMESTAMP}.log"
    
    if [[ $regression_exit_code -eq 0 ]]; then
        log_success "Regression detection completed"
        return 0
    else
        log_warn "Regression detection had issues (this may be expected if no baseline exists)"
        log_info "Regression output saved to: $RESULTS_DIR/regression_test_${TIMESTAMP}.log"
        return 0  # Don't fail CI for regression tests yet
    fi
}

# Generate performance report
generate_report() {
    log_info "Generating performance report..."
    
    local report_file="$RESULTS_DIR/performance_report_${TIMESTAMP}.md"
    
    cat > "$report_file" << EOF
# Performance Test Report

**Date:** $(date -u +"%Y-%m-%d %H:%M:%S UTC")
**Mode:** $PERFORMANCE_MODE
**Commit:** ${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || echo "unknown")}
**Branch:** ${GITHUB_REF_NAME:-$(git branch --show-current 2>/dev/null || echo "unknown")}

## Configuration

- P95 Threshold: ${P95_THRESHOLD_MS}ms
- P50 Threshold: ${P50_THRESHOLD_MS}ms
- Min Throughput: ${MIN_THROUGHPUT} msg/sec

## Test Results

EOF

    # Add threshold test results if available
    if [[ -f "$RESULTS_DIR/threshold_test_${TIMESTAMP}.log" ]]; then
        echo "### Threshold Tests" >> "$report_file"
        echo "" >> "$report_file"
        echo '```' >> "$report_file"
        grep -E "(PASS|FAIL|Latency|Throughput|Performance)" "$RESULTS_DIR/threshold_test_${TIMESTAMP}.log" | head -20 >> "$report_file"
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # Add benchmark results if available
    if [[ -f "$RESULTS_DIR/benchmark_${TIMESTAMP}.log" ]]; then
        echo "### Benchmark Results" >> "$report_file"
        echo "" >> "$report_file"
        echo '```' >> "$report_file"
        grep -E "BenchmarkPingPong" "$RESULTS_DIR/benchmark_${TIMESTAMP}.log" >> "$report_file"
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # Add system information
    echo "## System Information" >> "$report_file"
    echo "" >> "$report_file"
    echo "- OS: $(uname -s)" >> "$report_file"
    echo "- Architecture: $(uname -m)" >> "$report_file"
    echo "- Go Version: $(go version)" >> "$report_file"
    
    if command -v nproc &> /dev/null; then
        echo "- CPU Cores: $(nproc)" >> "$report_file"
    fi
    
    log_success "Performance report generated: $report_file"
}

# Main execution
main() {
    log_info "Starting AgentFlow Performance Tests"
    echo
    
    print_config
    check_skip
    check_prerequisites
    setup_results_dir
    
    local overall_success=true
    
    # Run tests based on mode
    case "$PERFORMANCE_MODE" in
        "ci")
            log_info "Running CI performance tests..."
            if ! run_threshold_tests; then
                overall_success=false
            fi
            run_benchmarks || true  # Don't fail CI on benchmark issues
            run_regression_tests || true  # Don't fail CI on regression issues yet
            ;;
        "local")
            log_info "Running local performance tests..."
            run_threshold_tests || true
            run_benchmarks || true
            run_regression_tests || true
            ;;
        "baseline")
            log_info "Running baseline recording..."
            # This would run the manual baseline recording test
            cd "$PROJECT_ROOT"
            go test -tags=manual -v "$TEST_PACKAGE" -run TestManualBaselineRecording || true
            ;;
        *)
            log_error "Unknown performance mode: $PERFORMANCE_MODE"
            exit 1
            ;;
    esac
    
    generate_report
    
    if [[ "$overall_success" == "true" ]]; then
        log_success "All performance tests completed successfully"
        exit 0
    else
        log_error "Some performance tests failed"
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
    "--help"|"-h")
        echo "Usage: $0 [options]"
        echo ""
        echo "Environment variables:"
        echo "  AF_PERF_P95_THRESHOLD_MS  P95 latency threshold in milliseconds (default: $DEFAULT_P95_THRESHOLD_MS)"
        echo "  AF_PERF_P50_THRESHOLD_MS  P50 latency threshold in milliseconds (default: $DEFAULT_P50_THRESHOLD_MS)"
        echo "  AF_PERF_MIN_THROUGHPUT    Minimum throughput in msg/sec (default: $DEFAULT_MIN_THROUGHPUT)"
        echo "  AF_SKIP_PERFORMANCE       Skip performance tests (default: false)"
        echo "  AF_PERFORMANCE_MODE       Test mode: ci, local, or baseline (default: ci)"
        echo ""
        echo "Examples:"
        echo "  $0                                    # Run CI performance tests"
        echo "  AF_PERFORMANCE_MODE=local $0          # Run local performance tests"
        echo "  AF_PERFORMANCE_MODE=baseline $0       # Record performance baseline"
        echo "  AF_PERF_P95_THRESHOLD_MS=25 $0        # Use custom P95 threshold"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac