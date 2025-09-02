#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
COVERAGE_THRESHOLD=80
FAST_MODE=false
VERBOSE=false
UPDATE_GOLDEN=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --fast)
            FAST_MODE=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --update-golden)
            UPDATE_GOLDEN=true
            shift
            ;;
        --coverage-threshold)
            COVERAGE_THRESHOLD="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --fast                    Skip coverage analysis for faster execution"
            echo "  --verbose, -v             Enable verbose output"
            echo "  --update-golden          Update golden test files"
            echo "  --coverage-threshold N    Set coverage threshold (default: 80)"
            echo "  --help, -h               Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if we're in the right directory
    if [[ ! -f "$PROJECT_ROOT/Cargo.toml" ]]; then
        log_error "This script must be run from the sebuf project root or scripts directory"
        exit 1
    fi
    
    # Check if Rust is installed
    if ! command -v cargo &> /dev/null; then
        log_error "cargo is not installed. Please install Rust: https://rustup.rs/"
        exit 1
    fi
    
    # Check if protoc is installed
    if ! command -v protoc &> /dev/null; then
        log_error "protoc is not installed. Please install Protocol Buffers compiler"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Build all Rust projects
build_projects() {
    log_info "Building Rust projects..."
    
    cd "$PROJECT_ROOT"
    
    if [[ "$VERBOSE" == "true" ]]; then
        cargo build --release
    else
        cargo build --release > /dev/null 2>&1
    fi
    
    log_success "Build completed"
}

# Run unit tests
run_unit_tests() {
    log_info "Running unit tests..."
    
    cd "$PROJECT_ROOT"
    
    local test_cmd="cargo test --lib"
    if [[ "$VERBOSE" == "true" ]]; then
        test_cmd="$test_cmd --verbose"
    fi
    
    if ! eval "$test_cmd"; then
        log_error "Unit tests failed"
        return 1
    fi
    
    log_success "Unit tests passed"
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    
    cd "$PROJECT_ROOT"
    
    local test_cmd="cargo test --test integration_test"
    if [[ "$VERBOSE" == "true" ]]; then
        test_cmd="$test_cmd --verbose"
    fi
    
    # Set environment variables for golden tests
    if [[ "$UPDATE_GOLDEN" == "true" ]]; then
        export UPDATE_GOLDEN=1
        log_info "Updating golden files..."
    fi
    
    if ! eval "$test_cmd"; then
        log_error "Integration tests failed"
        return 1
    fi
    
    log_success "Integration tests passed"
}

# Run golden file tests  
run_golden_tests() {
    log_info "Running golden file tests..."
    
    cd "$PROJECT_ROOT"
    
    local test_cmd="cargo test --test golden_test"
    if [[ "$VERBOSE" == "true" ]]; then
        test_cmd="$test_cmd --verbose"
    fi
    
    # Set environment variables for golden tests
    if [[ "$UPDATE_GOLDEN" == "true" ]]; then
        export UPDATE_GOLDEN=1
        log_info "Updating golden files..."
    fi
    
    if ! eval "$test_cmd"; then
        log_error "Golden file tests failed"
        return 1
    fi
    
    log_success "Golden file tests passed"
}

# Run coverage analysis
run_coverage() {
    if [[ "$FAST_MODE" == "true" ]]; then
        log_info "Skipping coverage analysis (fast mode enabled)"
        return 0
    fi
    
    log_info "Running coverage analysis..."
    
    cd "$PROJECT_ROOT"
    
    # Check if tarpaulin is installed
    if ! command -v cargo-tarpaulin &> /dev/null; then
        log_warning "cargo-tarpaulin is not installed. Installing..."
        cargo install cargo-tarpaulin
    fi
    
    local coverage_cmd="cargo tarpaulin --out Xml --out Html --output-dir rust/coverage"
    
    if [[ "$VERBOSE" == "true" ]]; then
        coverage_cmd="$coverage_cmd --verbose"
    else
        coverage_cmd="$coverage_cmd --quiet"
    fi
    
    if ! eval "$coverage_cmd"; then
        log_error "Coverage analysis failed"
        return 1
    fi
    
    # Parse coverage percentage from tarpaulin output
    if [[ -f "rust/coverage/cobertura.xml" ]]; then
        local coverage_percent
        coverage_percent=$(grep -o 'line-rate="[0-9.]*"' rust/coverage/cobertura.xml | head -1 | grep -o '[0-9.]*')
        if [[ -n "$coverage_percent" ]]; then
            coverage_percent=$(echo "$coverage_percent * 100" | bc -l)
            coverage_percent=${coverage_percent%.*}
            
            if (( coverage_percent >= COVERAGE_THRESHOLD )); then
                log_success "Coverage: ${coverage_percent}% (threshold: ${COVERAGE_THRESHOLD}%)"
            else
                log_warning "Coverage: ${coverage_percent}% (below threshold: ${COVERAGE_THRESHOLD}%)"
                return 1
            fi
        fi
    fi
}

# Run linting
run_lint() {
    log_info "Running Rust linting..."
    
    cd "$PROJECT_ROOT"
    
    # Check formatting
    if ! cargo fmt --all -- --check; then
        log_error "Code formatting check failed. Run 'cargo fmt --all' to fix."
        return 1
    fi
    
    # Run clippy
    local clippy_cmd="cargo clippy --all -- -D warnings"
    if [[ "$VERBOSE" == "true" ]]; then
        clippy_cmd="$clippy_cmd --verbose"
    fi
    
    if ! eval "$clippy_cmd"; then
        log_error "Clippy linting failed"
        return 1
    fi
    
    log_success "Linting passed"
}

# Generate test report
generate_report() {
    log_info "Generating test report..."
    
    local report_file="$PROJECT_ROOT/rust/test_report.txt"
    
    {
        echo "Sebuf Rust Test Report"
        echo "======================"
        echo "Generated: $(date)"
        echo "Fast Mode: $FAST_MODE"
        echo "Verbose: $VERBOSE"
        echo "Coverage Threshold: ${COVERAGE_THRESHOLD}%"
        echo ""
        
        if [[ "$FAST_MODE" == "false" && -f "$PROJECT_ROOT/rust/coverage/cobertura.xml" ]]; then
            echo "Coverage Report: rust/coverage/tarpaulin-report.html"
        fi
        
        echo ""
        echo "Test Results:"
        echo "✅ Unit tests"
        echo "✅ Integration tests" 
        echo "✅ Golden file tests"
        if [[ "$FAST_MODE" == "false" ]]; then
            echo "✅ Coverage analysis"
        fi
        echo "✅ Code linting"
        
    } > "$report_file"
    
    log_success "Test report generated: rust/test_report.txt"
}

# Main execution
main() {
    echo -e "${BLUE}Sebuf Rust Test Runner${NC}"
    echo "======================"
    echo ""
    
    check_prerequisites
    build_projects
    run_unit_tests
    run_integration_tests
    run_golden_tests
    run_coverage
    run_lint
    generate_report
    
    log_success "All tests completed successfully!"
    
    if [[ "$UPDATE_GOLDEN" == "true" ]]; then
        log_info "Golden files were updated. Please review and commit the changes."
    fi
}

# Run main function
main "$@"