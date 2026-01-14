# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=voting-api
MAIN_PATH=./main.go

# Test parameters
TEST_PACKAGES=./tests/...
COVERAGE_OUT=coverage.out

.PHONY: all build clean test test-verbose test-coverage test-race run deps tidy help

# Default target
all: test build

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PATH)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_OUT)

# Run tests
test:
	$(GOTEST) -v $(TEST_PACKAGES)

# Run tests with verbose output
test-verbose:
	$(GOTEST) -v -count=1 $(TEST_PACKAGES)

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=$(COVERAGE_OUT) $(TEST_PACKAGES)
	$(GOCMD) tool cover -html=$(COVERAGE_OUT)

# Run tests with race detection
test-race:
	$(GOTEST) -v -race $(TEST_PACKAGES)

# Run specific test
test-auth:
	$(GOTEST) -v ./tests -run TestUserRegistration
	$(GOTEST) -v ./tests -run TestUserLogin
	$(GOTEST) -v ./tests -run TestGetProfile

test-ballot:
	$(GOTEST) -v ./tests -run TestCreateBallot
	$(GOTEST) -v ./tests -run TestGetAllBallots
	$(GOTEST) -v ./tests -run TestGetBallot
	$(GOTEST) -v ./tests -run TestGetUserBallots

test-vote:
	$(GOTEST) -v ./tests -run TestVote
	$(GOTEST) -v ./tests -run TestGetUserVote
	$(GOTEST) -v ./tests -run TestGetBallotResults

test-integration:
	$(GOTEST) -v ./tests -run TestFullVotingFlow
	$(GOTEST) -v ./tests -run TestHealthEndpoint
	$(GOTEST) -v ./tests -run TestJWTUtilities
	$(GOTEST) -v ./tests -run TestPasswordHashing

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PATH)
	./$(BINARY_NAME)

# Download dependencies
deps:
	$(GOGET) -d -v ./...

# Clean up go.mod and go.sum
tidy:
	$(GOMOD) tidy

# Start development server with hot reload (requires air)
dev:
	air

# Format code
fmt:
	$(GOCMD) fmt ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Security scan (requires gosec)
security:
	gosec ./...

# Generate test report
test-report: test-coverage
	@echo "Test coverage report generated: $(COVERAGE_OUT)"
	@$(GOCMD) tool cover -func=$(COVERAGE_OUT) | tail -1

# Docker build
docker-build:
	docker build -t $(BINARY_NAME) .

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run all tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-auth     - Run authentication tests only"
	@echo "  test-ballot   - Run ballot management tests only"
	@echo "  test-vote     - Run voting tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  run           - Build and run the application"
	@echo "  deps          - Download dependencies"
	@echo "  tidy          - Clean up go.mod and go.sum"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  lint          - Run linter"
	@echo "  security      - Run security scan"
	@echo "  docker-build  - Build Docker image"
	@echo "  help          - Show this help message"