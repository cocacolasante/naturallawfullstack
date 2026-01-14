package tests

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
	"voting-api/models"
	"voting-api/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRegistration(t *testing.T) {
	t.Run("Successful Registration", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		// Mock that user doesn't exist
		testSetup.Mock.ExpectQuery("SELECT id FROM users WHERE email = $1 OR username = $2").
			WithArgs("test@example.com", "testuser").
			WillReturnError(sql.ErrNoRows)

		// Mock user insertion
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, username, email, created_at, updated_at").
			WithArgs("testuser", "test@example.com", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at"}).
				AddRow(1, "testuser", "test@example.com", createdAt, createdAt))

		reqBody := models.RegisterRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
		}

		req, err := CreateTestRequest("POST", "/api/v1/auth/register", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		t.Logf("Response Body: %s", recorder.Body.String())
		assert.Equal(t, 201, recorder.Code)
		
		// Verify response contains token and user data
		var response models.AuthResponse
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)
		
		assert.NotEmpty(t, response.Token)
		assert.Equal(t, "testuser", response.User.Username)
		assert.Equal(t, "test@example.com", response.User.Email)
		assert.Equal(t, 1, response.User.ID)

		// Check expectations and log any failures
		if err := testSetup.Mock.ExpectationsWereMet(); err != nil {
			t.Logf("Mock expectations not met: %v", err)
		}
	})

	t.Run("Registration with Existing User", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		// Mock that user already exists
		testSetup.Mock.ExpectQuery("SELECT id FROM users WHERE email = $1 OR username = $2").
			WithArgs("existing@example.com", "existing").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		reqBody := models.RegisterRequest{
			Username: "existing",
			Email:    "existing@example.com",
			Password: "password123",
		}

		req, err := CreateTestRequest("POST", "/api/v1/auth/register", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 409, "User already exists")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Registration with Invalid Data", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		reqBody := models.RegisterRequest{
			Username: "u", // Too short
			Email:    "invalid-email",
			Password: "123", // Too short
		}

		req, err := CreateTestRequest("POST", "/api/v1/auth/register", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})
}

func TestUserLogin(t *testing.T) {
	t.Run("Successful Login", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		password := "password123"
		hashedPassword, err := utils.HashPassword(password)
		require.NoError(t, err)

		// Mock user found in database
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = $1").
			WithArgs("test@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at"}).
				AddRow(1, "testuser", "test@example.com", hashedPassword, createdAt, createdAt))

		reqBody := models.LoginRequest{
			Email:    "test@example.com",
			Password: password,
		}

		req, err := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var response models.AuthResponse
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Token)
		assert.Equal(t, "testuser", response.User.Username)
		assert.Equal(t, "test@example.com", response.User.Email)
		assert.Empty(t, response.User.Password) // Password should not be returned

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Login with Invalid Email", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		// Mock user not found
		testSetup.Mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = $1").
			WithArgs("nonexistent@example.com").
			WillReturnError(sql.ErrNoRows)

		reqBody := models.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		req, err := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Invalid credentials")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Login with Wrong Password", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		hashedPassword, err := utils.HashPassword("correctpassword")
		require.NoError(t, err)

		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = $1").
			WithArgs("test@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at"}).
				AddRow(1, "testuser", "test@example.com", hashedPassword, createdAt, createdAt))

		reqBody := models.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		req, err := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Invalid credentials")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetProfile(t *testing.T) {
	t.Run("Get Profile Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock user query
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("SELECT id, username, email, created_at, updated_at FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at"}).
				AddRow(userID, "testuser", email, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var user models.User
		err = parseJSONResponse(recorder, &user)
		require.NoError(t, err)

		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, email, user.Email)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Profile Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/profile", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Get Profile With Invalid Token", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/profile", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-token")

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Invalid token")
	})
}

// Helper function to parse JSON response
func parseJSONResponse(recorder *httptest.ResponseRecorder, target interface{}) error {
	return parseJSONFromBytes(recorder.Body.Bytes(), target)
}

func parseJSONFromBytes(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}