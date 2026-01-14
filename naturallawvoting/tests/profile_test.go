package tests

import (
	"database/sql"
	"net/http/httptest"
	"testing"
	"time"
	"voting-api/models"

	"github.com/DATA-DOG/go-sqlmock"
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
}
