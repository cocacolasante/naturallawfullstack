#!/bin/bash

# Test runner script for the voting API
# This script provides various ways to run tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
print_header() {
    echo -e "${GREEN}=== $1 ===${NC}"
}

print_warning() {
    echo -e "${YELLOW}WARNING: $1${NC}"
}

print_error() {
    echo -e "${RED}ERROR: $1${NC}"
}

print_success() {
    echo -e "${GREEN}SUCCESS: $1${NC}"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    print_success "Go is available: $(go version)"
}

# Check if required dependencies are available
check_deps() {
    print_header "Checking Dependencies"
    
    go mod download
    go mod tidy
    
    print_success "Dependencies are ready"
}

# Run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    
    # Set test environment
    export JWT_SECRET="test-secret-key"
    
    # Run tests with verbose output
    go test -v ./tests/ -run "Test.*" -skip "TestFull.*"
    
    if [ $? -eq 0 ]; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        exit 1
    fi
}

# Run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    
    # Set test environment
    export JWT_SECRET="test-secret-key"
    
    # Run integration tests
    go test -v ./tests/ -run "TestFull.*"
    
    if [ $? -eq 0 ]; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        exit 1
    fi
}

# Run all tests
run_all_tests() {
    print_header "Running All Tests"
    
    # Set test environment
    export JWT_SECRET="test-secret-key"
    
    # Run all tests
    go test -v ./tests/
    
    if [ $? -eq 0 ]; then
        print_success "All tests passed"
    else
        print_error "Some tests failed"
        exit 1
    fi
}

# Run tests with coverage
run_coverage() {
    print_header "Running Tests with Coverage"
    
    # Set test environment
    export JWT_SECRET="test-secret-key"
    
    # Create coverage directory
    mkdir -p coverage
    
    # Run tests with coverage
    go test -v -coverprofile=coverage/coverage.out ./tests/
    
    # Generate HTML coverage report
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html
    
    # Show coverage summary
    go tool cover -func=coverage/coverage.out
    
    print_success "Coverage report generated in coverage/coverage.html"
}

# Run specific test category
run_category_tests() {
    case $1 in
        auth|authentication)
            print_header "Running Authentication Tests"
            go test -v ./tests/ -run ".*Auth.*|.*Login.*|.*Register.*|.*Profile.*"
            ;;
        ballot|ballots)
            print_header "Running Ballot Tests"
            go test -v ./tests/ -run ".*Ballot.*"
            ;;
        vote|voting)
            print_header "Running Voting Tests"
            go test -v ./tests/ -run ".*Vote.*"
            ;;
        utils|utilities)
            print_header "Running Utility Tests"
            go test -v ./tests/ -run ".*JWT.*|.*Password.*"
            ;;
        *)
            print_error "Unknown test category: $1"
            echo "Available categories: auth, ballot, vote, utils"
            exit 1
            ;;
    esac
}

# Run tests with race detection
run_race_tests() {
    print_header "Running Tests with Race Detection"
    
    export JWT_SECRET="test-secret-key"
    
    go test -v -race ./tests/
    
    if [ $? -eq 0 ]; then
        print_success "Race tests passed"
    else
        print_error "Race condition detected"
        exit 1
    fi
}

# Show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  all         - Run all tests (default)"
    echo "  unit        - Run unit tests only"
    echo "  integration - Run integration tests only"
    echo "  coverage    - Run tests with coverage report"
    echo "  race        - Run tests with race detection"
    echo "  category    - Run specific category tests"
    echo "    - auth      Authentication tests"
    echo "    - ballot    Ballot management tests"
    echo "    - vote      Voting tests"
    echo "    - utils     Utility tests"
    echo "  help        - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run all tests"
    echo "  $0 unit              # Run unit tests"
    echo "  $0 coverage          # Generate coverage report"
    echo "  $0 category auth     # Run authentication tests"
}

# Main execution
main() {
    check_go
    check_deps
    
    case ${1:-all} in
        all)
            run_all_tests
            ;;
        unit)
            run_unit_tests
            ;;
        integration)
            run_integration_tests
            ;;
        coverage)
            run_coverage
            ;;
        race)
            run_race_tests
            ;;
        category)
            if [ -z "$2" ]; then
                print_error "Category not specified"
                show_usage
                exit 1
            fi
            run_category_tests $2
            ;;
        help|-h|--help)
            show_usage
            ;;
        *)
            print_error "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"