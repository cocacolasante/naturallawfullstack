package tests

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"voting-api/database"
	"voting-api/handlers"
	"voting-api/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// User Profile Tests
// ============================================================================

func TestGetUserProfile(t *testing.T) {
	t.Run("Get Profile Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		birthday := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock profile query
		testSetup.Mock.ExpectQuery(`
		SELECT user_id, email, full_name, birthday, gender, mothers_maiden_name,
		       phone_number, additional_emails, created_at, updated_at
		FROM user_profiles WHERE email = $1`).
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, "John Doe", birthday, "Male", "Smith", "555-1234", pq.Array([]string{"john@other.com"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var profile models.UserProfile
		err = parseJSONResponse(recorder, &profile)
		require.NoError(t, err)

		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, email, profile.Email)
		assert.Equal(t, "John Doe", profile.FullName)
		assert.Equal(t, "Male", profile.Gender)
		assert.Equal(t, "Smith", profile.MothersMaidenName)
		assert.Equal(t, "555-1234", profile.PhoneNumber)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Profile Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock profile not found
		testSetup.Mock.ExpectQuery(`
		SELECT user_id, email, full_name, birthday, gender, mothers_maiden_name,
		       phone_number, additional_emails, created_at, updated_at
		FROM user_profiles WHERE email = $1`).
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Profile not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Profile Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/profile/info", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})
}

func TestCreateUserProfile(t *testing.T) {
	t.Run("Create Profile Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		birthday := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)

		reqBody := models.CreateUserProfileRequest{
			FullName:          "John Doe",
			Birthday:          "1990-05-15",
			Gender:            "Male",
			MothersMaidenName: "Smith",
			PhoneNumber:       "555-1234",
			AdditionalEmails:  []string{"john@other.com"},
		}

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock check if profile exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		// Mock profile insertion
		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_profiles
		(user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number,
		          additional_emails, created_at, updated_at`).
			WithArgs(userID, email, "John Doe", birthday, "Male", "Smith", "555-1234", pq.Array([]string{"john@other.com"})).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, "John Doe", birthday, "Male", "Smith", "555-1234", pq.Array([]string{"john@other.com"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)

		var profile models.UserProfile
		err = parseJSONResponse(recorder, &profile)
		require.NoError(t, err)

		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, "John Doe", profile.FullName)
		assert.Equal(t, "Male", profile.Gender)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Profile When Already Exists", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserProfileRequest{
			FullName: "John Doe",
		}

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock profile already exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 409, "Profile already exists")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Profile With Invalid Birthday Format", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserProfileRequest{
			FullName: "John Doe",
			Birthday: "invalid-date",
		}

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock check if profile exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid birthday format. Use YYYY-MM-DD")
	})
}

func TestUpdateUserProfile(t *testing.T) {
	t.Run("Update Profile Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		birthday := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)

		newName := "Jane Doe"
		reqBody := models.UpdateUserProfileRequest{
			FullName: &newName,
		}

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock profile update
		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET full_name = $1 WHERE email = $2 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(newName, email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, newName, birthday, "Male", "Smith", "555-1234", pq.Array([]string{"john@other.com"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var profile models.UserProfile
		err = parseJSONResponse(recorder, &profile)
		require.NoError(t, err)

		assert.Equal(t, newName, profile.FullName)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		newName := "Jane Doe"
		reqBody := models.UpdateUserProfileRequest{
			FullName: &newName,
		}

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock profile not found
		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET full_name = $1 WHERE email = $2 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(newName, email).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Profile not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile With No Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.UpdateUserProfileRequest{}

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "No fields to update")
	})
}

func TestDeleteUserProfile(t *testing.T) {
	t.Run("Delete Profile Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock profile deletion
		testSetup.Mock.ExpectExec("DELETE FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Profile Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock getting email
		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		// Mock profile not found
		testSetup.Mock.ExpectExec("DELETE FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnResult(sqlmock.NewResult(0, 0))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Profile not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// User Address Tests
// ============================================================================

func TestGetUserAddress(t *testing.T) {
	t.Run("Get Address Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		// Mock address query
		testSetup.Mock.ExpectQuery(`
		SELECT user_id, street_number, street_name, address_line_2, city, state,
		       zip_code, created_at, updated_at
		FROM user_addresses WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, "123", "Main St", "Apt 4", "Boston", "MA", "02101", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/address", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var address models.UserAddress
		err = parseJSONResponse(recorder, &address)
		require.NoError(t, err)

		assert.Equal(t, userID, address.UserID)
		assert.Equal(t, "123", address.StreetNumber)
		assert.Equal(t, "Main St", address.StreetName)
		assert.Equal(t, "Apt 4", address.AddressLine2)
		assert.Equal(t, "Boston", address.City)
		assert.Equal(t, "MA", address.State)
		assert.Equal(t, "02101", address.ZipCode)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Address Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock address not found
		testSetup.Mock.ExpectQuery(`
		SELECT user_id, street_number, street_name, address_line_2, city, state,
		       zip_code, created_at, updated_at
		FROM user_addresses WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/address", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Address not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestCreateUserAddress(t *testing.T) {
	t.Run("Create Address Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		reqBody := models.CreateUserAddressRequest{
			StreetNumber: "123",
			StreetName:   "Main St",
			AddressLine2: "Apt 4",
			City:         "Boston",
			State:        "MA",
			ZipCode:      "02101",
		}

		// Mock check if address exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_addresses WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Mock address insertion
		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_addresses
		(user_id, street_number, street_name, address_line_2, city, state, zip_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_id, street_number, street_name, address_line_2, city, state,
		          zip_code, created_at, updated_at`).
			WithArgs(userID, "123", "Main St", "Apt 4", "Boston", "MA", "02101").
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, "123", "Main St", "Apt 4", "Boston", "MA", "02101", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)

		var address models.UserAddress
		err = parseJSONResponse(recorder, &address)
		require.NoError(t, err)

		assert.Equal(t, userID, address.UserID)
		assert.Equal(t, "123", address.StreetNumber)
		assert.Equal(t, "Boston", address.City)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Address When Already Exists", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserAddressRequest{
			StreetNumber: "123",
			StreetName:   "Main St",
			City:         "Boston",
			State:        "MA",
			ZipCode:      "02101",
		}

		// Mock address already exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_addresses WHERE user_id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 409, "Address already exists")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestUpdateUserAddress(t *testing.T) {
	t.Run("Update Address Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newCity := "Cambridge"
		reqBody := models.UpdateUserAddressRequest{
			City: &newCity,
		}

		// Mock address update
		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET city = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newCity, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, "123", "Main St", "Apt 4", newCity, "MA", "02101", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var address models.UserAddress
		err = parseJSONResponse(recorder, &address)
		require.NoError(t, err)

		assert.Equal(t, newCity, address.City)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestDeleteUserAddress(t *testing.T) {
	t.Run("Delete Address Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock address deletion
		testSetup.Mock.ExpectExec("DELETE FROM user_addresses WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/address", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// User Political Affiliation Tests
// ============================================================================

func TestPoliticalAffiliation(t *testing.T) {
	t.Run("Get Political Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, party_affiliation, created_at, updated_at
		FROM user_political_affiliations WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "party_affiliation", "created_at", "updated_at"}).
				AddRow(userID, "Independent", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/political", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var affiliation models.UserPoliticalAffiliation
		err = parseJSONResponse(recorder, &affiliation)
		require.NoError(t, err)

		assert.Equal(t, "Independent", affiliation.PartyAffiliation)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Political Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		reqBody := models.CreateUserPoliticalAffiliationRequest{
			PartyAffiliation: "Independent",
		}

		// Mock check if exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_political_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Mock insertion
		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_political_affiliations (user_id, party_affiliation)
		VALUES ($1, $2)
		RETURNING user_id, party_affiliation, created_at, updated_at`).
			WithArgs(userID, "Independent").
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "party_affiliation", "created_at", "updated_at"}).
				AddRow(userID, "Independent", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// User Religious Affiliation Tests
// ============================================================================

func TestReligiousAffiliation(t *testing.T) {
	t.Run("Get Religious Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		supportingReligion := 7

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, religion, supporting_religion, religious_services_types,
		       created_at, updated_at
		FROM user_religious_affiliations WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "religion", "supporting_religion", "religious_services_types", "created_at", "updated_at"}).
				AddRow(userID, "Christian", supportingReligion, pq.Array([]string{"Sunday Service", "Bible Study"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/religious", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var affiliation models.UserReligiousAffiliation
		err = parseJSONResponse(recorder, &affiliation)
		require.NoError(t, err)

		assert.Equal(t, "Christian", affiliation.Religion)
		assert.Equal(t, supportingReligion, *affiliation.SupportingReligion)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Religious Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		supportingReligion := 7

		reqBody := models.CreateUserReligiousAffiliationRequest{
			Religion:               "Christian",
			SupportingReligion:     &supportingReligion,
			ReligiousServicesTypes: []string{"Sunday Service", "Bible Study"},
		}

		// Mock check if exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Mock insertion
		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_religious_affiliations
		(user_id, religion, supporting_religion, religious_services_types)
		VALUES ($1, $2, $3, $4)
		RETURNING user_id, religion, supporting_religion, religious_services_types,
		          created_at, updated_at`).
			WithArgs(userID, "Christian", &supportingReligion, pq.Array([]string{"Sunday Service", "Bible Study"})).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "religion", "supporting_religion", "religious_services_types", "created_at", "updated_at"}).
				AddRow(userID, "Christian", supportingReligion, pq.Array([]string{"Sunday Service", "Bible Study"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Religious Affiliation With Invalid Supporting Religion", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		invalidSupport := 15

		reqBody := models.CreateUserReligiousAffiliationRequest{
			Religion:           "Christian",
			SupportingReligion: &invalidSupport,
		}

		// Mock check if exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		// Gin validation will catch this and return 400 with validation error
		assert.Equal(t, 400, recorder.Code)
	})
}

// ============================================================================
// User Race/Ethnicity Tests
// ============================================================================

func TestRaceEthnicity(t *testing.T) {
	t.Run("Get Race/Ethnicity Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, race, created_at, updated_at
		FROM user_race_ethnicity WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "race", "created_at", "updated_at"}).
				AddRow(userID, pq.Array([]string{"Asian", "Hispanic"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/race-ethnicity", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var raceEthnicity models.UserRaceEthnicity
		err = parseJSONResponse(recorder, &raceEthnicity)
		require.NoError(t, err)

		assert.Equal(t, userID, raceEthnicity.UserID)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Race/Ethnicity Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		reqBody := models.CreateUserRaceEthnicityRequest{
			Race: []string{"Asian", "Hispanic"},
		}

		// Mock check if exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_race_ethnicity WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Mock insertion
		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_race_ethnicity (user_id, race)
		VALUES ($1, $2)
		RETURNING user_id, race, created_at, updated_at`).
			WithArgs(userID, pq.Array([]string{"Asian", "Hispanic"})).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "race", "created_at", "updated_at"}).
				AddRow(userID, pq.Array([]string{"Asian", "Hispanic"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Race/Ethnicity Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		reqBody := models.UpdateUserRaceEthnicityRequest{
			Race: []string{"Black", "White"},
		}

		// Mock update
		testSetup.Mock.ExpectQuery(`
		UPDATE user_race_ethnicity
		SET race = $1
		WHERE user_id = $2
		RETURNING user_id, race, created_at, updated_at`).
			WithArgs(pq.Array([]string{"Black", "White"}), userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "race", "created_at", "updated_at"}).
				AddRow(userID, pq.Array([]string{"Black", "White"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Race/Ethnicity Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock deletion
		testSetup.Mock.ExpectExec("DELETE FROM user_race_ethnicity WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/race-ethnicity", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// Economic Info Tests
// ============================================================================

func TestGetEconomicInfo(t *testing.T) {
	t.Run("Get Economic Info Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, for_current_political_structure, for_capitalism, for_laws,
		       goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		       additional_text, created_at, updated_at
		FROM economic_info WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "for_current_political_structure", "for_capitalism", "for_laws", "goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm", "additional_text", "created_at", "updated_at"}).
				AddRow(userID, "support", "support", "favor", pq.Array([]string{"software", "consulting"}), pq.Array([]string{"tech union", "workers coop"}), "high", "medium", "additional notes", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/economic", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var economicInfo models.EconomicInfo
		err = parseJSONResponse(recorder, &economicInfo)
		require.NoError(t, err)

		assert.Equal(t, userID, economicInfo.UserID)
		assert.Equal(t, "support", economicInfo.ForCurrentPoliticalStructure)
		assert.Equal(t, "support", economicInfo.ForCapitalism)
		assert.Equal(t, "favor", economicInfo.ForLaws)
		assert.Equal(t, "high", economicInfo.SupportOfAltEcon)
		assert.Equal(t, "medium", economicInfo.SupportAltComm)
		assert.Equal(t, "additional notes", economicInfo.AdditionalText)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Economic Info Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, for_current_political_structure, for_capitalism, for_laws,
		       goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		       additional_text, created_at, updated_at
		FROM economic_info WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/economic", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Economic info not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Economic Info Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/profile/economic", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})
}

func TestCreateEconomicInfo(t *testing.T) {
	t.Run("Create Economic Info Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		reqBody := models.CreateEconomicInfoRequest{
			ForCurrentPoliticalStructure: "support",
			ForCapitalism:                "support",
			ForLaws:                      "favor",
			GoodsServices:                []string{"software", "consulting"},
			Affiliations:                 []string{"tech union", "workers coop"},
			SupportOfAltEcon:             "high",
			SupportAltComm:               "medium",
			AdditionalText:               "additional notes",
		}

		// Mock check if economic info exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Mock economic info insertion
		testSetup.Mock.ExpectQuery(`
		INSERT INTO economic_info
		(user_id, for_current_political_structure, for_capitalism, for_laws,
		 goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING user_id, for_current_political_structure, for_capitalism, for_laws,
		          goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		          additional_text, created_at, updated_at`).
			WithArgs(userID, "support", "support", "favor", pq.Array([]string{"software", "consulting"}), pq.Array([]string{"tech union", "workers coop"}), "high", "medium", "additional notes").
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "for_current_political_structure", "for_capitalism", "for_laws", "goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm", "additional_text", "created_at", "updated_at"}).
				AddRow(userID, "support", "support", "favor", pq.Array([]string{"software", "consulting"}), pq.Array([]string{"tech union", "workers coop"}), "high", "medium", "additional notes", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)

		var economicInfo models.EconomicInfo
		err = parseJSONResponse(recorder, &economicInfo)
		require.NoError(t, err)

		assert.Equal(t, userID, economicInfo.UserID)
		assert.Equal(t, "support", economicInfo.ForCurrentPoliticalStructure)
		assert.Equal(t, "support", economicInfo.ForCapitalism)
		assert.Equal(t, "favor", economicInfo.ForLaws)
		assert.Equal(t, "high", economicInfo.SupportOfAltEcon)
		assert.Equal(t, "medium", economicInfo.SupportAltComm)
		assert.Equal(t, "additional notes", economicInfo.AdditionalText)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Economic Info When Already Exists", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateEconomicInfoRequest{
			ForCurrentPoliticalStructure: "support",
			ForCapitalism:                "support",
		}

		// Mock economic info already exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 409, "Economic info already exists")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Economic Info With Empty Arrays", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		reqBody := models.CreateEconomicInfoRequest{
			ForCurrentPoliticalStructure: "support",
			ForCapitalism:                "oppose",
			ForLaws:                      "neutral",
			GoodsServices:                []string{},
			Affiliations:                 []string{},
			SupportOfAltEcon:             "low",
			SupportAltComm:               "none",
			AdditionalText:               "",
		}

		// Mock check if economic info exists
		testSetup.Mock.ExpectQuery("SELECT user_id FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Mock economic info insertion
		testSetup.Mock.ExpectQuery(`
		INSERT INTO economic_info
		(user_id, for_current_political_structure, for_capitalism, for_laws,
		 goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING user_id, for_current_political_structure, for_capitalism, for_laws,
		          goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		          additional_text, created_at, updated_at`).
			WithArgs(userID, "support", "oppose", "neutral", pq.Array([]string{}), pq.Array([]string{}), "low", "none", "").
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "for_current_political_structure", "for_capitalism", "for_laws", "goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm", "additional_text", "created_at", "updated_at"}).
				AddRow(userID, "support", "oppose", "neutral", pq.Array([]string{}), pq.Array([]string{}), "low", "none", "", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)

		var economicInfo models.EconomicInfo
		err = parseJSONResponse(recorder, &economicInfo)
		require.NoError(t, err)

		assert.Equal(t, userID, economicInfo.UserID)
		assert.Equal(t, "support", economicInfo.ForCurrentPoliticalStructure)
		assert.Equal(t, "oppose", economicInfo.ForCapitalism)
		assert.Equal(t, "neutral", economicInfo.ForLaws)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Economic Info Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		reqBody := models.CreateEconomicInfoRequest{
			ForCurrentPoliticalStructure: "support",
		}

		req, err := CreateTestRequest("POST", "/api/v1/profile/economic", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})
}

func TestUpdateEconomicInfo(t *testing.T) {
	t.Run("Update Economic Info Successfully - Single Field", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newCapitalism := "oppose"
		reqBody := models.UpdateEconomicInfoRequest{
			ForCapitalism: &newCapitalism,
		}

		// Mock economic info update
		testSetup.Mock.ExpectQuery("UPDATE economic_info SET for_capitalism = $1 WHERE user_id = $2 RETURNING user_id, for_current_political_structure, for_capitalism, for_laws, goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text, created_at, updated_at").
			WithArgs(newCapitalism, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "for_current_political_structure", "for_capitalism", "for_laws", "goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm", "additional_text", "created_at", "updated_at"}).
				AddRow(userID, "support", newCapitalism, "favor", pq.Array([]string{"software"}), pq.Array([]string{"tech union"}), "high", "medium", "notes", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var economicInfo models.EconomicInfo
		err = parseJSONResponse(recorder, &economicInfo)
		require.NoError(t, err)

		assert.Equal(t, newCapitalism, economicInfo.ForCapitalism)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Economic Info Successfully - Multiple Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newCapitalism := "neutral"
		newLaws := "oppose"
		newAdditional := "updated notes"
		reqBody := models.UpdateEconomicInfoRequest{
			ForCapitalism:  &newCapitalism,
			ForLaws:        &newLaws,
			AdditionalText: &newAdditional,
		}

		// Mock economic info update
		testSetup.Mock.ExpectQuery("UPDATE economic_info SET for_capitalism = $1, for_laws = $2, additional_text = $3 WHERE user_id = $4 RETURNING user_id, for_current_political_structure, for_capitalism, for_laws, goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text, created_at, updated_at").
			WithArgs(newCapitalism, newLaws, newAdditional, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "for_current_political_structure", "for_capitalism", "for_laws", "goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm", "additional_text", "created_at", "updated_at"}).
				AddRow(userID, "support", newCapitalism, newLaws, pq.Array([]string{"software"}), pq.Array([]string{"tech union"}), "high", "medium", newAdditional, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var economicInfo models.EconomicInfo
		err = parseJSONResponse(recorder, &economicInfo)
		require.NoError(t, err)

		assert.Equal(t, newCapitalism, economicInfo.ForCapitalism)
		assert.Equal(t, newLaws, economicInfo.ForLaws)
		assert.Equal(t, newAdditional, economicInfo.AdditionalText)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Economic Info With Arrays", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		reqBody := models.UpdateEconomicInfoRequest{
			GoodsServices: []string{"hardware", "services", "products"},
			Affiliations:  []string{"union A", "cooperative B"},
		}

		// Mock economic info update
		testSetup.Mock.ExpectQuery("UPDATE economic_info SET goods_services = $1, affiliations = $2 WHERE user_id = $3 RETURNING user_id, for_current_political_structure, for_capitalism, for_laws, goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text, created_at, updated_at").
			WithArgs(pq.Array([]string{"hardware", "services", "products"}), pq.Array([]string{"union A", "cooperative B"}), userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "for_current_political_structure", "for_capitalism", "for_laws", "goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm", "additional_text", "created_at", "updated_at"}).
				AddRow(userID, "support", "support", "favor", pq.Array([]string{"hardware", "services", "products"}), pq.Array([]string{"union A", "cooperative B"}), "high", "medium", "notes", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var economicInfo models.EconomicInfo
		err = parseJSONResponse(recorder, &economicInfo)
		require.NoError(t, err)

		assert.Equal(t, 3, len(economicInfo.GoodsServices))
		assert.Equal(t, 2, len(economicInfo.Affiliations))

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Economic Info Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		newCapitalism := "oppose"
		reqBody := models.UpdateEconomicInfoRequest{
			ForCapitalism: &newCapitalism,
		}

		// Mock economic info not found
		testSetup.Mock.ExpectQuery("UPDATE economic_info SET for_capitalism = $1 WHERE user_id = $2 RETURNING user_id, for_current_political_structure, for_capitalism, for_laws, goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text, created_at, updated_at").
			WithArgs(newCapitalism, userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Economic info not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Economic Info With No Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.UpdateEconomicInfoRequest{}

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "No fields to update")
	})

	t.Run("Update Economic Info Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		newCapitalism := "oppose"
		reqBody := models.UpdateEconomicInfoRequest{
			ForCapitalism: &newCapitalism,
		}

		req, err := CreateTestRequest("PUT", "/api/v1/profile/economic", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})
}

func TestDeleteEconomicInfo(t *testing.T) {
	t.Run("Delete Economic Info Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock economic info deletion
		testSetup.Mock.ExpectExec("DELETE FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/economic", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var response map[string]string
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.Equal(t, "Economic info deleted successfully", response["message"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Economic Info Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock economic info not found
		testSetup.Mock.ExpectExec("DELETE FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/economic", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Economic info not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Economic Info Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("DELETE", "/api/v1/profile/economic", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Delete Economic Info DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/economic", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error deleting economic info")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Economic Info No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("DELETE", "/api/v1/profile/economic", nil)
		handler.DeleteEconomicInfo(c)
		assert.Equal(t, 401, w.Code)
	})
}

// ============================================================================
// Additional Missing Coverage Tests
// ============================================================================

func TestGetUserProfileMissingBranches(t *testing.T) {
	t.Run("Get Profile Email DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Profile Profile DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, email, full_name, birthday, gender, mothers_maiden_name,
		       phone_number, additional_emails, created_at, updated_at
		FROM user_profiles WHERE email = $1`).
			WithArgs(email).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Profile No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/profile/info", nil)
		handler.GetUserProfile(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestCreateUserProfileMissingBranches(t *testing.T) {
	t.Run("Create Profile Email DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserProfileRequest{
			FullName: "John Doe",
		}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Profile Exists Check DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserProfileRequest{
			FullName: "John Doe",
		}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Profile INSERT DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserProfileRequest{
			FullName: "John Doe",
		}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_profiles
		(user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number,
		          additional_emails, created_at, updated_at`).
			WithArgs(userID, email, "John Doe", nil, "", "", "", pq.Array([]string(nil))).
			WillReturnError(errors.New("insert error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating profile")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Profile No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		reqBody := models.CreateUserProfileRequest{FullName: "John Doe"}
		c.Request, _ = CreateTestRequest("POST", "/api/v1/profile/info", reqBody)
		handler.CreateUserProfile(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestUpdateUserProfileMissingBranches(t *testing.T) {
	t.Run("Update Profile Email DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newName := "Jane Doe"
		reqBody := models.UpdateUserProfileRequest{FullName: &newName}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile Invalid Birthday Format", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		badBirthday := "not-a-date"
		reqBody := models.UpdateUserProfileRequest{Birthday: &badBirthday}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid birthday format. Use YYYY-MM-DD")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newName := "Jane Doe"
		reqBody := models.UpdateUserProfileRequest{FullName: &newName}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET full_name = $1 WHERE email = $2 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(newName, email).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating profile")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		newName := "Jane Doe"
		reqBody := models.UpdateUserProfileRequest{FullName: &newName}
		c.Request, _ = CreateTestRequest("PUT", "/api/v1/profile/info", reqBody)
		handler.UpdateUserProfile(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestDeleteUserProfileMissingBranches(t *testing.T) {
	t.Run("Delete Profile Email DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Profile Exec DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectExec("DELETE FROM user_profiles WHERE email = $1").
			WithArgs(email).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/info", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error deleting profile")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Profile No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("DELETE", "/api/v1/profile/info", nil)
		handler.DeleteUserProfile(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestGetUserAddressMissingBranches(t *testing.T) {
	t.Run("Get Address DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, street_number, street_name, address_line_2, city, state,
		       zip_code, created_at, updated_at
		FROM user_addresses WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/address", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Address No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/profile/address", nil)
		handler.GetUserAddress(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestCreateUserAddressMissingBranches(t *testing.T) {
	t.Run("Create Address Exists Check DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserAddressRequest{
			StreetNumber: "123",
			StreetName:   "Main St",
			City:         "Boston",
			State:        "MA",
			ZipCode:      "02101",
		}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_addresses WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Address INSERT DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.CreateUserAddressRequest{
			StreetNumber: "123",
			StreetName:   "Main St",
			City:         "Boston",
			State:        "MA",
			ZipCode:      "02101",
		}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_addresses WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_addresses
		(user_id, street_number, street_name, address_line_2, city, state, zip_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_id, street_number, street_name, address_line_2, city, state,
		          zip_code, created_at, updated_at`).
			WithArgs(userID, "123", "Main St", "", "Boston", "MA", "02101").
			WillReturnError(errors.New("insert error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating address")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Address No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		reqBody := models.CreateUserAddressRequest{StreetName: "Main St"}
		c.Request, _ = CreateTestRequest("POST", "/api/v1/profile/address", reqBody)
		handler.CreateUserAddress(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestUpdateUserAddressMissingBranches(t *testing.T) {
	t.Run("Update Address No Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.UpdateUserAddressRequest{}

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "No fields to update")
	})

	t.Run("Update Address Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newCity := "Cambridge"
		reqBody := models.UpdateUserAddressRequest{City: &newCity}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET city = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newCity, userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Address not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Address DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newCity := "Cambridge"
		reqBody := models.UpdateUserAddressRequest{City: &newCity}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET city = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newCity, userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating address")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Address No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		newCity := "Cambridge"
		reqBody := models.UpdateUserAddressRequest{City: &newCity}
		c.Request, _ = CreateTestRequest("PUT", "/api/v1/profile/address", reqBody)
		handler.UpdateUserAddress(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestDeleteUserAddressMissingBranches(t *testing.T) {
	t.Run("Delete Address Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_addresses WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/address", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Address not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Address DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_addresses WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/address", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error deleting address")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Address No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("DELETE", "/api/v1/profile/address", nil)
		handler.DeleteUserAddress(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestPoliticalAffiliationMissingBranches(t *testing.T) {
	t.Run("Get Political Affiliation Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, party_affiliation, created_at, updated_at
		FROM user_political_affiliations WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/political", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Political affiliation not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Political Affiliation DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, party_affiliation, created_at, updated_at
		FROM user_political_affiliations WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/political", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Political Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/profile/political", nil)
		handler.GetUserPoliticalAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Create Political Affiliation Exists Check DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserPoliticalAffiliationRequest{PartyAffiliation: "Independent"}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_political_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Political Affiliation Already Exists", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserPoliticalAffiliationRequest{PartyAffiliation: "Independent"}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_political_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 409, "Political affiliation already exists")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Political Affiliation INSERT DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserPoliticalAffiliationRequest{PartyAffiliation: "Independent"}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_political_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_political_affiliations (user_id, party_affiliation)
		VALUES ($1, $2)
		RETURNING user_id, party_affiliation, created_at, updated_at`).
			WithArgs(userID, "Independent").
			WillReturnError(errors.New("insert error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating political affiliation")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Political Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		reqBody := models.CreateUserPoliticalAffiliationRequest{PartyAffiliation: "Independent"}
		c.Request, _ = CreateTestRequest("POST", "/api/v1/profile/political", reqBody)
		handler.CreateUserPoliticalAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})
}

// ============================================================================
// Update/Delete Political Affiliation Tests (0% coverage)
// ============================================================================

func TestUpdateUserPoliticalAffiliation(t *testing.T) {
	t.Run("Update Political Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		newParty := "Democratic"
		reqBody := models.UpdateUserPoliticalAffiliationRequest{PartyAffiliation: &newParty}

		testSetup.Mock.ExpectQuery(`
		UPDATE user_political_affiliations
		SET party_affiliation = $1
		WHERE user_id = $2
		RETURNING user_id, party_affiliation, created_at, updated_at`).
			WithArgs(newParty, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "party_affiliation", "created_at", "updated_at"}).
				AddRow(userID, newParty, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var affiliation models.UserPoliticalAffiliation
		err = parseJSONResponse(recorder, &affiliation)
		require.NoError(t, err)
		assert.Equal(t, newParty, affiliation.PartyAffiliation)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Political Affiliation No Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.UpdateUserPoliticalAffiliationRequest{}

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "No fields to update")
	})

	t.Run("Update Political Affiliation Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newParty := "Democratic"
		reqBody := models.UpdateUserPoliticalAffiliationRequest{PartyAffiliation: &newParty}

		testSetup.Mock.ExpectQuery(`
		UPDATE user_political_affiliations
		SET party_affiliation = $1
		WHERE user_id = $2
		RETURNING user_id, party_affiliation, created_at, updated_at`).
			WithArgs(newParty, userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Political affiliation not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Political Affiliation DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newParty := "Democratic"
		reqBody := models.UpdateUserPoliticalAffiliationRequest{PartyAffiliation: &newParty}

		testSetup.Mock.ExpectQuery(`
		UPDATE user_political_affiliations
		SET party_affiliation = $1
		WHERE user_id = $2
		RETURNING user_id, party_affiliation, created_at, updated_at`).
			WithArgs(newParty, userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/political", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating political affiliation")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Political Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		newParty := "Democratic"
		reqBody := models.UpdateUserPoliticalAffiliationRequest{PartyAffiliation: &newParty}
		c.Request, _ = CreateTestRequest("PUT", "/api/v1/profile/political", reqBody)
		handler.UpdateUserPoliticalAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestDeleteUserPoliticalAffiliation(t *testing.T) {
	t.Run("Delete Political Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_political_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/political", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Political Affiliation Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_political_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/political", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Political affiliation not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Political Affiliation DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_political_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/political", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error deleting political affiliation")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Political Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("DELETE", "/api/v1/profile/political", nil)
		handler.DeleteUserPoliticalAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})
}

// ============================================================================
// Religious Affiliation Missing Branch Tests
// ============================================================================

func TestReligiousAffiliationMissingBranches(t *testing.T) {
	t.Run("Get Religious Affiliation Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, religion, supporting_religion, religious_services_types,
		       created_at, updated_at
		FROM user_religious_affiliations WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/religious", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Religious affiliation not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Religious Affiliation DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, religion, supporting_religion, religious_services_types,
		       created_at, updated_at
		FROM user_religious_affiliations WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/religious", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Religious Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/profile/religious", nil)
		handler.GetUserReligiousAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Create Religious Affiliation Exists Check DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserReligiousAffiliationRequest{Religion: "Christian"}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Religious Affiliation Already Exists", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserReligiousAffiliationRequest{Religion: "Christian"}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 409, "Religious affiliation already exists")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Religious Affiliation INSERT DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserReligiousAffiliationRequest{
			Religion:               "Christian",
			ReligiousServicesTypes: []string{},
		}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_religious_affiliations
		(user_id, religion, supporting_religion, religious_services_types)
		VALUES ($1, $2, $3, $4)
		RETURNING user_id, religion, supporting_religion, religious_services_types,
		          created_at, updated_at`).
			WithArgs(userID, "Christian", (*int)(nil), pq.Array([]string{})).
			WillReturnError(errors.New("insert error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating religious affiliation")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Religious Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		reqBody := models.CreateUserReligiousAffiliationRequest{Religion: "Christian"}
		c.Request, _ = CreateTestRequest("POST", "/api/v1/profile/religious", reqBody)
		handler.CreateUserReligiousAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})
}

// ============================================================================
// Update/Delete Religious Affiliation Tests (0% coverage)
// ============================================================================

func TestUpdateUserReligiousAffiliation(t *testing.T) {
	t.Run("Update Religious Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		newReligion := "Buddhist"
		reqBody := models.UpdateUserReligiousAffiliationRequest{Religion: &newReligion}

		testSetup.Mock.ExpectQuery("UPDATE user_religious_affiliations SET religion = $1 WHERE user_id = $2 RETURNING user_id, religion, supporting_religion, religious_services_types, created_at, updated_at").
			WithArgs(newReligion, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "religion", "supporting_religion", "religious_services_types", "created_at", "updated_at"}).
				AddRow(userID, newReligion, nil, pq.Array([]string{}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var affiliation models.UserReligiousAffiliation
		err = parseJSONResponse(recorder, &affiliation)
		require.NoError(t, err)
		assert.Equal(t, newReligion, affiliation.Religion)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Religious Affiliation No Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.UpdateUserReligiousAffiliationRequest{}

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "No fields to update")
	})

	t.Run("Update Religious Affiliation Invalid Supporting Religion", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		invalidVal := 15
		reqBody := models.UpdateUserReligiousAffiliationRequest{SupportingReligion: &invalidVal}

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		// Gin binding validation catches this
		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Update Religious Affiliation Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newReligion := "Buddhist"
		reqBody := models.UpdateUserReligiousAffiliationRequest{Religion: &newReligion}

		testSetup.Mock.ExpectQuery("UPDATE user_religious_affiliations SET religion = $1 WHERE user_id = $2 RETURNING user_id, religion, supporting_religion, religious_services_types, created_at, updated_at").
			WithArgs(newReligion, userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Religious affiliation not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Religious Affiliation DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newReligion := "Buddhist"
		reqBody := models.UpdateUserReligiousAffiliationRequest{Religion: &newReligion}

		testSetup.Mock.ExpectQuery("UPDATE user_religious_affiliations SET religion = $1 WHERE user_id = $2 RETURNING user_id, religion, supporting_religion, religious_services_types, created_at, updated_at").
			WithArgs(newReligion, userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating religious affiliation")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Religious Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		newReligion := "Buddhist"
		reqBody := models.UpdateUserReligiousAffiliationRequest{Religion: &newReligion}
		c.Request, _ = CreateTestRequest("PUT", "/api/v1/profile/religious", reqBody)
		handler.UpdateUserReligiousAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Update Religious Affiliation With SupportingReligion Field", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		newSupport := 8
		reqBody := models.UpdateUserReligiousAffiliationRequest{SupportingReligion: &newSupport}

		testSetup.Mock.ExpectQuery("UPDATE user_religious_affiliations SET supporting_religion = $1 WHERE user_id = $2 RETURNING user_id, religion, supporting_religion, religious_services_types, created_at, updated_at").
			WithArgs(newSupport, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "religion", "supporting_religion", "religious_services_types", "created_at", "updated_at"}).
				AddRow(userID, "Christian", newSupport, pq.Array([]string{}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Religious Affiliation With ReligiousServicesTypes Field", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		reqBody := models.UpdateUserReligiousAffiliationRequest{
			ReligiousServicesTypes: []string{"Morning Service", "Evening Service"},
		}

		testSetup.Mock.ExpectQuery("UPDATE user_religious_affiliations SET religious_services_types = $1 WHERE user_id = $2 RETURNING user_id, religion, supporting_religion, religious_services_types, created_at, updated_at").
			WithArgs(pq.Array([]string{"Morning Service", "Evening Service"}), userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "religion", "supporting_religion", "religious_services_types", "created_at", "updated_at"}).
				AddRow(userID, "Christian", nil, pq.Array([]string{"Morning Service", "Evening Service"}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/religious", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestDeleteUserReligiousAffiliation(t *testing.T) {
	t.Run("Delete Religious Affiliation Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/religious", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Religious Affiliation Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/religious", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Religious affiliation not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Religious Affiliation DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_religious_affiliations WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/religious", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error deleting religious affiliation")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Religious Affiliation No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("DELETE", "/api/v1/profile/religious", nil)
		handler.DeleteUserReligiousAffiliation(c)
		assert.Equal(t, 401, w.Code)
	})
}

// ============================================================================
// Race/Ethnicity Missing Branch Tests
// ============================================================================

func TestRaceEthnicityMissingBranches(t *testing.T) {
	t.Run("Get Race Ethnicity Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, race, created_at, updated_at
		FROM user_race_ethnicity WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/race-ethnicity", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Race/ethnicity not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Race Ethnicity DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, race, created_at, updated_at
		FROM user_race_ethnicity WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/race-ethnicity", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Race Ethnicity No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/profile/race-ethnicity", nil)
		handler.GetUserRaceEthnicity(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Create Race Ethnicity Exists Check DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserRaceEthnicityRequest{Race: []string{"Asian"}}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_race_ethnicity WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Race Ethnicity Already Exists", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserRaceEthnicityRequest{Race: []string{"Asian"}}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_race_ethnicity WHERE user_id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 409, "Race/ethnicity already exists")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Race Ethnicity INSERT DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateUserRaceEthnicityRequest{Race: []string{"Asian"}}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM user_race_ethnicity WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectQuery(`
		INSERT INTO user_race_ethnicity (user_id, race)
		VALUES ($1, $2)
		RETURNING user_id, race, created_at, updated_at`).
			WithArgs(userID, pq.Array([]string{"Asian"})).
			WillReturnError(errors.New("insert error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating race/ethnicity")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Race Ethnicity No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		reqBody := models.CreateUserRaceEthnicityRequest{Race: []string{"Asian"}}
		c.Request, _ = CreateTestRequest("POST", "/api/v1/profile/race-ethnicity", reqBody)
		handler.CreateUserRaceEthnicity(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Update Race Ethnicity No Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.UpdateUserRaceEthnicityRequest{}

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "No fields to update")
	})

	t.Run("Update Race Ethnicity Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.UpdateUserRaceEthnicityRequest{Race: []string{"White"}}

		testSetup.Mock.ExpectQuery(`
		UPDATE user_race_ethnicity
		SET race = $1
		WHERE user_id = $2
		RETURNING user_id, race, created_at, updated_at`).
			WithArgs(pq.Array([]string{"White"}), userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Race/ethnicity not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Race Ethnicity DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.UpdateUserRaceEthnicityRequest{Race: []string{"White"}}

		testSetup.Mock.ExpectQuery(`
		UPDATE user_race_ethnicity
		SET race = $1
		WHERE user_id = $2
		RETURNING user_id, race, created_at, updated_at`).
			WithArgs(pq.Array([]string{"White"}), userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/race-ethnicity", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating race/ethnicity")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Race Ethnicity No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		reqBody := models.UpdateUserRaceEthnicityRequest{Race: []string{"White"}}
		c.Request, _ = CreateTestRequest("PUT", "/api/v1/profile/race-ethnicity", reqBody)
		handler.UpdateUserRaceEthnicity(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Delete Race Ethnicity Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_race_ethnicity WHERE user_id = $1").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/race-ethnicity", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Race/ethnicity not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Race Ethnicity DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectExec("DELETE FROM user_race_ethnicity WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("DELETE", "/api/v1/profile/race-ethnicity", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error deleting race/ethnicity")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Delete Race Ethnicity No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("DELETE", "/api/v1/profile/race-ethnicity", nil)
		handler.DeleteUserRaceEthnicity(c)
		assert.Equal(t, 401, w.Code)
	})
}

// ============================================================================
// Economic Info Missing Branch Tests
// ============================================================================

func TestEconomicInfoMissingBranches(t *testing.T) {
	t.Run("Get Economic Info DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT user_id, for_current_political_structure, for_capitalism, for_laws,
		       goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		       additional_text, created_at, updated_at
		FROM economic_info WHERE user_id = $1`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/economic", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Economic Info No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/profile/economic", nil)
		handler.GetEconomicInfo(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Create Economic Info Exists Check DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateEconomicInfoRequest{ForCapitalism: "support"}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Economic Info INSERT DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		reqBody := models.CreateEconomicInfoRequest{
			ForCurrentPoliticalStructure: "support",
			ForCapitalism:                "support",
			ForLaws:                      "favor",
			GoodsServices:                []string{},
			Affiliations:                 []string{},
		}

		testSetup.Mock.ExpectQuery("SELECT user_id FROM economic_info WHERE user_id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectQuery(`
		INSERT INTO economic_info
		(user_id, for_current_political_structure, for_capitalism, for_laws,
		 goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING user_id, for_current_political_structure, for_capitalism, for_laws,
		          goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		          additional_text, created_at, updated_at`).
			WithArgs(userID, "support", "support", "favor", pq.Array([]string{}), pq.Array([]string{}), "", "", "").
			WillReturnError(errors.New("insert error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating economic info")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Economic Info No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		reqBody := models.CreateEconomicInfoRequest{ForCapitalism: "support"}
		c.Request, _ = CreateTestRequest("POST", "/api/v1/profile/economic", reqBody)
		handler.CreateEconomicInfo(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Update Economic Info DB Error (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		newCapitalism := "oppose"
		reqBody := models.UpdateEconomicInfoRequest{ForCapitalism: &newCapitalism}

		testSetup.Mock.ExpectQuery("UPDATE economic_info SET for_capitalism = $1 WHERE user_id = $2 RETURNING user_id, for_current_political_structure, for_capitalism, for_laws, goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text, created_at, updated_at").
			WithArgs(newCapitalism, userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating economic info")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Economic Info No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		newCapitalism := "oppose"
		reqBody := models.UpdateEconomicInfoRequest{ForCapitalism: &newCapitalism}
		c.Request, _ = CreateTestRequest("PUT", "/api/v1/profile/economic", reqBody)
		handler.UpdateEconomicInfo(c)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("Update Economic Info With All Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		polStr := "support"
		capStr := "support"
		lawStr := "favor"
		altEcon := "high"
		altComm := "medium"
		addText := "notes"
		reqBody := models.UpdateEconomicInfoRequest{
			ForCurrentPoliticalStructure: &polStr,
			ForCapitalism:                &capStr,
			ForLaws:                      &lawStr,
			GoodsServices:                []string{"software"},
			Affiliations:                 []string{"union"},
			SupportOfAltEcon:             &altEcon,
			SupportAltComm:               &altComm,
			AdditionalText:               &addText,
		}

		testSetup.Mock.ExpectQuery("UPDATE economic_info SET for_current_political_structure = $1, for_capitalism = $2, for_laws = $3, goods_services = $4, affiliations = $5, support_of_alt_econ = $6, support_alt_comm = $7, additional_text = $8 WHERE user_id = $9 RETURNING user_id, for_current_political_structure, for_capitalism, for_laws, goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text, created_at, updated_at").
			WithArgs(polStr, capStr, lawStr, pq.Array([]string{"software"}), pq.Array([]string{"union"}), altEcon, altComm, addText, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "for_current_political_structure", "for_capitalism", "for_laws", "goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm", "additional_text", "created_at", "updated_at"}).
				AddRow(userID, polStr, capStr, lawStr, pq.Array([]string{"software"}), pq.Array([]string{"union"}), altEcon, altComm, addText, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/economic", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// ShouldBindJSON Error Tests and Multi-Field Update Tests
// ============================================================================

func TestShouldBindJSONErrors(t *testing.T) {
	t.Run("Create User Profile Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("POST", "/api/v1/profile/info", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Update User Profile Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("PUT", "/api/v1/profile/info", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Create User Address Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("POST", "/api/v1/profile/address", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Update User Address Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("PUT", "/api/v1/profile/address", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Create Political Affiliation Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("POST", "/api/v1/profile/political", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Update Political Affiliation Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("PUT", "/api/v1/profile/political", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Create Religious Affiliation Invalid JSON (binding validation error)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("POST", "/api/v1/profile/religious", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Update Religious Affiliation Invalid JSON (binding validation error)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("PUT", "/api/v1/profile/religious", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Create Race Ethnicity Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("POST", "/api/v1/profile/race-ethnicity", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Update Race Ethnicity Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("PUT", "/api/v1/profile/race-ethnicity", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Create Economic Info Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("POST", "/api/v1/profile/economic", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Update Economic Info Invalid JSON", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRawBodyRequest("PUT", "/api/v1/profile/economic", "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})
}

// ============================================================================
// Multi-field Update Tests to Cover Dynamic Query Builder Branches
// ============================================================================

func TestUpdateUserProfileMultipleFields(t *testing.T) {
	t.Run("Update Profile With All Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		birthday := time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC)

		newName := "Updated Name"
		newBirthday := "1985-03-20"
		newGender := "Female"
		newMaidenName := "Jones"
		newPhone := "555-9999"
		newEmails := []string{"alt@test.com"}

		reqBody := models.UpdateUserProfileRequest{
			FullName:          &newName,
			Birthday:          &newBirthday,
			Gender:            &newGender,
			MothersMaidenName: &newMaidenName,
			PhoneNumber:       &newPhone,
			AdditionalEmails:  newEmails,
		}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET full_name = $1, birthday = $2, gender = $3, mothers_maiden_name = $4, phone_number = $5, additional_emails = $6 WHERE email = $7 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(newName, birthday, newGender, newMaidenName, newPhone, pq.Array(newEmails), email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, newName, birthday, newGender, newMaidenName, newPhone, pq.Array(newEmails), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile With Gender Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newGender := "Female"
		reqBody := models.UpdateUserProfileRequest{Gender: &newGender}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET gender = $1 WHERE email = $2 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(newGender, email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, "John Doe", nil, newGender, "Smith", "555-1234", pq.Array([]string{}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile With MothersMaidenName Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newMaiden := "Williams"
		reqBody := models.UpdateUserProfileRequest{MothersMaidenName: &newMaiden}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET mothers_maiden_name = $1 WHERE email = $2 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(newMaiden, email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, "John Doe", nil, "Male", newMaiden, "555-1234", pq.Array([]string{}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile With PhoneNumber Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newPhone := "555-8888"
		reqBody := models.UpdateUserProfileRequest{PhoneNumber: &newPhone}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET phone_number = $1 WHERE email = $2 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(newPhone, email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, "John Doe", nil, "Male", "Smith", newPhone, pq.Array([]string{}), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Profile With AdditionalEmails Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newEmails := []string{"new@email.com", "another@email.com"}
		reqBody := models.UpdateUserProfileRequest{AdditionalEmails: newEmails}

		testSetup.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

		testSetup.Mock.ExpectQuery("UPDATE user_profiles SET additional_emails = $1 WHERE email = $2 RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at").
			WithArgs(pq.Array(newEmails), email).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "full_name", "birthday", "gender", "mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at"}).
				AddRow(userID, email, "John Doe", nil, "Male", "Smith", "555-1234", pq.Array(newEmails), createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/info", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestUpdateUserAddressMultipleFields(t *testing.T) {
	t.Run("Update Address With All Fields", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newStreetNum := "456"
		newStreetName := "Oak Ave"
		newAddrLine2 := "Suite 200"
		newCity := "Springfield"
		newState := "IL"
		newZip := "62701"

		reqBody := models.UpdateUserAddressRequest{
			StreetNumber: &newStreetNum,
			StreetName:   &newStreetName,
			AddressLine2: &newAddrLine2,
			City:         &newCity,
			State:        &newState,
			ZipCode:      &newZip,
		}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET street_number = $1, street_name = $2, address_line_2 = $3, city = $4, state = $5, zip_code = $6 WHERE user_id = $7 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newStreetNum, newStreetName, newAddrLine2, newCity, newState, newZip, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, newStreetNum, newStreetName, newAddrLine2, newCity, newState, newZip, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Address With StreetNumber Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newStreetNum := "789"
		reqBody := models.UpdateUserAddressRequest{StreetNumber: &newStreetNum}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET street_number = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newStreetNum, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, newStreetNum, "Main St", "Apt 4", "Boston", "MA", "02101", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Address With StreetName Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newStreetName := "Elm St"
		reqBody := models.UpdateUserAddressRequest{StreetName: &newStreetName}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET street_name = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newStreetName, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, "123", newStreetName, "Apt 4", "Boston", "MA", "02101", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Address With AddressLine2 Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newLine2 := "Unit 5"
		reqBody := models.UpdateUserAddressRequest{AddressLine2: &newLine2}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET address_line_2 = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newLine2, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, "123", "Main St", newLine2, "Boston", "MA", "02101", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Address With State Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newState := "TX"
		reqBody := models.UpdateUserAddressRequest{State: &newState}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET state = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newState, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, "123", "Main St", "Apt 4", "Austin", newState, "78701", createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Update Address With ZipCode Only", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		newZip := "94101"
		reqBody := models.UpdateUserAddressRequest{ZipCode: &newZip}

		testSetup.Mock.ExpectQuery("UPDATE user_addresses SET zip_code = $1 WHERE user_id = $2 RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at").
			WithArgs(newZip, userID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "street_number", "street_name", "address_line_2", "city", "state", "zip_code", "created_at", "updated_at"}).
				AddRow(userID, "123", "Main St", "Apt 4", "San Francisco", "CA", newZip, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/address", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// Religious Affiliation Manual Validation Tests (binding.Validator = nil bypass)
// ============================================================================

// TestCreateReligiousAffiliationOutOfRangeManualCheck tests the manual supporting_religion bounds check
// which is only reachable when Gin's binding validator is disabled (binding.Validator = nil).
func TestCreateReligiousAffiliationOutOfRangeManualCheck(t *testing.T) {
	t.Run("Create Religious Affiliation Supporting Religion Out Of Range Manual Check", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		// Disable Gin's validator so the binding tag "omitempty,min=0,max=10" is not checked,
		// allowing an out-of-range value to pass ShouldBindJSON and reach the manual check.
		origValidator := binding.Validator
		binding.Validator = nil
		defer func() { binding.Validator = origValidator }()

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", 1)

		// Send supporting_religion = -1 (out of range: must be 0-10)
		body := []byte(`{"religion": "Test", "supporting_religion": -1}`)
		c.Request, err = http.NewRequest("POST", "/api/v1/profile/religious", bytes.NewBuffer(body))
		require.NoError(t, err)
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateUserReligiousAffiliation(c)

		assert.Equal(t, 400, w.Code)
	})
}

// TestUpdateReligiousAffiliationOutOfRangeManualCheck tests the manual supporting_religion bounds check
// in the Update handler, reachable only when Gin's validator is disabled.
func TestUpdateReligiousAffiliationOutOfRangeManualCheck(t *testing.T) {
	t.Run("Update Religious Affiliation Supporting Religion Out Of Range Manual Check", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewProfileHandler(db)

		// Disable Gin's validator so the binding tag "omitempty,min=0,max=10" is not checked,
		// allowing an out-of-range value to pass ShouldBindJSON and reach the manual check.
		origValidator := binding.Validator
		binding.Validator = nil
		defer func() { binding.Validator = origValidator }()

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", 1)

		// Send supporting_religion = 11 (out of range: must be 0-10)
		body := []byte(`{"supporting_religion": 11}`)
		c.Request, err = http.NewRequest("PUT", "/api/v1/profile/religious", bytes.NewBuffer(body))
		require.NoError(t, err)
		c.Request.Header.Set("Content-Type", "application/json")

		handler.UpdateUserReligiousAffiliation(c)

		assert.Equal(t, 400, w.Code)
	})
}
