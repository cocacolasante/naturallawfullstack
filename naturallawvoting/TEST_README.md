# Test Suite Documentation

This document describes the comprehensive test suite for the Voting API.

## Overview

The test suite includes:
- **Unit tests** for individual components and handlers
- **Integration tests** for complete workflows
- **Mocked database** interactions using sqlmock
- **Authentication testing** with JWT tokens
- **Error handling** validation
- **Edge cases** coverage

## Test Structure

```
tests/
├── utils.go           # Test utilities and helpers
├── auth_test.go       # User authentication tests
├── ballot_test.go     # Ballot management tests
├── vote_test.go       # Voting functionality tests
└── integration_test.go # Full workflow integration tests
```

## Test Categories

### 1. Authentication Tests (`auth_test.go`)

**TestUserRegistration**
- ✅ Successful user registration
- ✅ Registration with existing user (conflict)
- ✅ Registration with invalid data validation

**TestUserLogin**
- ✅ Successful login with correct credentials
- ✅ Login with invalid email
- ✅ Login with wrong password

**TestGetProfile**
- ✅ Get user profile successfully
- ✅ Get profile without authentication
- ✅ Get profile with invalid token

### 2. Ballot Management Tests (`ballot_test.go`)

**TestCreateBallot**
- ✅ Create ballot successfully with items
- ✅ Create ballot without authentication
- ✅ Create ballot with invalid data

**TestGetAllBallots**
- ✅ Get all active ballots (public endpoint)
- ✅ Handle empty ballot list

**TestGetBallot**
- ✅ Get specific ballot with items and vote counts
- ✅ Get non-existent ballot (404)
- ✅ Get ballot with invalid ID

**TestGetUserBallots**
- ✅ Get user's created ballots
- ✅ Get user ballots without authentication
- ✅ Handle empty user ballot list

### 3. Voting Tests (`vote_test.go`)

**TestVote**
- ✅ First vote on ballot item
- ✅ Change existing vote
- ✅ Vote on non-existent ballot
- ✅ Vote on inactive ballot
- ✅ Vote on invalid ballot item
- ✅ Vote without authentication

**TestGetUserVote**
- ✅ Get user's vote for a ballot
- ✅ Get vote when none exists
- ✅ Get vote without authentication

**TestGetBallotResults**
- ✅ Get ballot results with vote counts
- ✅ Get results for non-existent ballot
- ✅ Get results with no votes (empty)

### 4. Integration Tests (`integration_test.go`)

**TestFullVotingFlow**
Complete end-to-end workflow testing:
1. User registration
2. Create ballot with options
3. Get all ballots (public)
4. Get specific ballot details
5. Vote on ballot
6. Get user's vote
7. Get ballot results
8. Get user's created ballots
9. Get user profile

**TestHealthEndpoint**
- ✅ Health check endpoint functionality

**TestJWTUtilities**
- ✅ JWT token generation and validation
- ✅ Invalid JWT token handling

**TestPasswordHashing**
- ✅ Password hashing and verification
- ✅ Wrong password rejection

## Running Tests

### Using Make (Recommended)
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test categories
make test-auth
make test-ballot
make test-vote
make test-integration

# Run tests with race detection
make test-race
```

### Using Test Runner Script
```bash
# Make script executable (first time only)
chmod +x test_runner.sh

# Run all tests
./test_runner.sh

# Run specific categories
./test_runner.sh category auth
./test_runner.sh category ballot
./test_runner.sh category vote

# Run with coverage
./test_runner.sh coverage

# Run with race detection
./test_runner.sh race
```

### Using Go Commands Directly
```bash
# Run all tests
go test -v ./tests/

# Run specific test
go test -v ./tests/ -run TestUserRegistration

# Run tests with coverage
go test -v -coverprofile=coverage.out ./tests/
go tool cover -html=coverage.out
```

## Test Environment Setup

### Environment Variables
```bash
# Required for JWT testing
export JWT_SECRET="test-secret-key"
```

### Database Mocking
Tests use `go-sqlmock` to mock PostgreSQL database interactions:
- No real database required
- SQL queries are mocked and expectations verified
- Transactions are properly mocked
- Database errors are simulated for error handling tests

## Test Data

### Sample Test Users
```json
{
  "username": "testuser",
  "email": "test@example.com",
  "password": "password123"
}
```

### Sample Test Ballots
```json
{
  "title": "Best Programming Language",
  "description": "Vote for your favorite",
  "items": [
    {"title": "Go", "description": "Fast and efficient"},
    {"title": "Python", "description": "Easy to learn"}
  ]
}
```

## Test Coverage

The test suite aims for comprehensive coverage:
- **Handler functions**: All HTTP endpoints tested
- **Authentication**: JWT token generation, validation, middleware
- **Database operations**: All CRUD operations mocked and tested
- **Error handling**: Invalid inputs, missing resources, unauthorized access
- **Business logic**: Voting constraints, data validation

## Test Utilities

### Helper Functions
- `SetupTestEnvironment()`: Creates test router with mocked database
- `CreateTestRequest()`: Creates HTTP requests for testing
- `CreateAuthenticatedRequest()`: Creates requests with JWT tokens
- `AssertJSONResponse()`: Validates JSON response structure
- `AssertErrorResponse()`: Validates error responses

### Mock Functions
- `MockUserExists()`: Mock user existence checks
- `MockUserInsert()`: Mock user creation
- `MockUserLogin()`: Mock user authentication

## Continuous Integration

Tests are designed to run in CI environments:
- No external dependencies (database, Redis, etc.)
- Fast execution with mocked components
- Comprehensive error reporting
- Exit codes for CI/CD pipelines

## Common Test Patterns

### Testing HTTP Endpoints
```go
func TestSomeEndpoint(t *testing.T) {
    testSetup, err := SetupTestEnvironment()
    require.NoError(t, err)
    defer testSetup.DB.Close()

    // Mock database expectations
    testSetup.Mock.ExpectQuery("SELECT...").
        WithArgs(...).
        WillReturnRows(...)

    // Create request
    req, err := CreateTestRequest("GET", "/endpoint", nil)
    require.NoError(t, err)

    // Execute request
    recorder := httptest.NewRecorder()
    testSetup.Router.ServeHTTP(recorder, req)

    // Assert response
    assert.Equal(t, 200, recorder.Code)
    
    // Verify mocks
    assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
}
```

### Testing with Authentication
```go
req, err := CreateAuthenticatedRequest("POST", "/protected", body, userID, email)
```

## Troubleshooting

### Common Issues

1. **Mock Expectations Not Met**
   - Ensure SQL query patterns match exactly
   - Check parameter order and types
   - Verify all expected calls are made

2. **JWT Token Issues**
   - Set JWT_SECRET environment variable
   - Ensure token format is correct ("Bearer token")

3. **Test Isolation**
   - Each test creates fresh mock database
   - Tests don't share state
   - Mock expectations are verified after each test

### Debug Tips
- Use `go test -v` for verbose output
- Add debug prints in test helpers
- Check mock expectations with `ExpectationsWereMet()`
- Verify request/response JSON structure

## Future Enhancements

- Performance benchmarking tests
- Load testing capabilities  
- Database integration tests (optional)
- API contract testing
- Security penetration testing