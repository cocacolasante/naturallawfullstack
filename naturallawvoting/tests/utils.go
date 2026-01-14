package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"voting-api/database"
	"voting-api/routes"
	"voting-api/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestSetup contains the test environment setup
type TestSetup struct {
	Router *gin.Engine
	DB     *database.DB
	Mock   sqlmock.Sqlmock
}

// SetupTestEnvironment creates a test environment with mocked database
func SetupTestEnvironment() (*TestSetup, error) {
	gin.SetMode(gin.TestMode)
	
	// Create mock database with exact query matching
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		return nil, err
	}

	db := &database.DB{DB: mockDB}
	router := routes.SetupRoutes(db)

	return &TestSetup{
		Router: router,
		DB:     db,
		Mock:   mock,
	}, nil
}

// CreateTestRequest creates an HTTP request for testing
func CreateTestRequest(method, url string, body interface{}) (*http.Request, error) {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

// CreateAuthenticatedRequest creates an HTTP request with JWT token
func CreateAuthenticatedRequest(method, url string, body interface{}, userID int, email string) (*http.Request, error) {
	req, err := CreateTestRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	token, err := utils.GenerateJWT(userID, email)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

// AssertJSONResponse asserts that the response matches expected JSON
func AssertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
	assert.Equal(t, expectedStatus, recorder.Code)
	
	if expectedBody != nil {
		var actualBody map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &actualBody)
		assert.NoError(t, err)

		expectedJSON, _ := json.Marshal(expectedBody)
		var expectedBodyMap map[string]interface{}
		json.Unmarshal(expectedJSON, &expectedBodyMap)

		// Compare relevant fields (excluding timestamps which may vary)
		for key, expectedValue := range expectedBodyMap {
			if key != "created_at" && key != "updated_at" {
				assert.Equal(t, expectedValue, actualBody[key], fmt.Sprintf("Field %s doesn't match", key))
			}
		}
	}
}

// AssertErrorResponse asserts that the response contains an error message
func AssertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	assert.Equal(t, expectedStatus, recorder.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	errorMsg, exists := response["error"]
	assert.True(t, exists, "Expected error field in response")
	assert.Equal(t, expectedError, errorMsg)
}

// MockUserExists mocks a database query to check if user exists
func (ts *TestSetup) MockUserExists(email, username string, exists bool) {
	if exists {
		ts.Mock.ExpectQuery("SELECT id FROM users WHERE email = \\$1 OR username = \\$2").
			WithArgs(email, username).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	} else {
		ts.Mock.ExpectQuery("SELECT id FROM users WHERE email = \\$1 OR username = \\$2").
			WithArgs(email, username).
			WillReturnError(sql.ErrNoRows)
	}
}

// MockUserInsert mocks user insertion into database
func (ts *TestSetup) MockUserInsert(userID int, username, email string) {
	ts.Mock.ExpectQuery("INSERT INTO users").
		WithArgs(username, email, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at"}).
			AddRow(userID, username, email, "2023-01-01T00:00:00Z", "2023-01-01T00:00:00Z"))
}

// MockUserLogin mocks user login query
func (ts *TestSetup) MockUserLogin(email, hashedPassword string, userID int, username string, found bool) {
	if found {
		ts.Mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at"}).
				AddRow(userID, username, email, hashedPassword, "2023-01-01T00:00:00Z", "2023-01-01T00:00:00Z"))
	} else {
		ts.Mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)
	}
}