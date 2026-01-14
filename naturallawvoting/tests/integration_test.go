package tests

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
	"voting-api/models"
	"voting-api/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullVotingFlow tests the complete workflow from user registration to voting
func TestFullVotingFlow(t *testing.T) {
	testSetup, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testSetup.DB.Close()

	// User data
	username := "integrationuser"
	email := "integration@example.com"
	password := "password123"
	var userID = 1
	var ballotID = 1
	var token string

	t.Run("1. Register User", func(t *testing.T) {
		// Mock user doesn't exist
		testSetup.Mock.ExpectQuery("SELECT id FROM users WHERE email = $1 OR username = $2").
			WithArgs(email, username).
			WillReturnError(sql.ErrNoRows)

		// Mock user insertion
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, username, email, created_at, updated_at").
			WithArgs(username, email, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at"}).
				AddRow(userID, username, email, createdAt, createdAt))

		reqBody := models.RegisterRequest{
			Username: username,
			Email:    email,
			Password: password,
		}

		req, err := CreateTestRequest("POST", "/api/v1/auth/register", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)

		var authResponse models.AuthResponse
		err = parseJSONResponse(recorder, &authResponse)
		require.NoError(t, err)

		token = authResponse.Token
		assert.NotEmpty(t, token)
		assert.Equal(t, userID, authResponse.User.ID)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("2. Create Ballot", func(t *testing.T) {
		// Mock transaction begin
		testSetup.Mock.ExpectBegin()

		// Mock ballot insertion
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("INSERT INTO ballots (title, description, creator_id) VALUES ($1, $2, $3) RETURNING id, title, description, creator_id, is_active, created_at, updated_at").
			WithArgs("Integration Test Ballot", "Testing the full workflow", userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", userID, true, createdAt, createdAt))

		// Mock ballot items insertion
		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description, vote_count").
			WithArgs(ballotID, "Option A", "First choice").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(1, ballotID, "Option A", "First choice", 0))

		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description, vote_count").
			WithArgs(ballotID, "Option B", "Second choice").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(2, ballotID, "Option B", "Second choice", 0))

		// Mock transaction commit
		testSetup.Mock.ExpectCommit()

		reqBody := models.CreateBallotRequest{
			Title:       "Integration Test Ballot",
			Description: "Testing the full workflow",
			Items: []models.CreateBallotItemRequest{
				{Title: "Option A", Description: "First choice"},
				{Title: "Option B", Description: "Second choice"},
			},
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 201, recorder.Code)

		var ballot models.Ballot
		err = parseJSONResponse(recorder, &ballot)
		require.NoError(t, err)

		assert.Equal(t, ballotID, ballot.ID)
		assert.Len(t, ballot.Items, 2)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("3. Get All Ballots (Public)", func(t *testing.T) {
		// Mock ballots query
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`SELECT b.id, b.title, b.description, b.creator_id, b.is_active, b.created_at, b.updated_at,
       u.username as creator_username
FROM ballots b 
JOIN users u ON b.creator_id = u.id 
WHERE b.is_active = true 
ORDER BY b.created_at DESC`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at", "creator_username"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", userID, true, createdAt, createdAt, username))

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballots []models.Ballot
		err = parseJSONResponse(recorder, &ballots)
		require.NoError(t, err)

		assert.Len(t, ballots, 1)
		assert.Equal(t, "Integration Test Ballot", ballots[0].Title)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("4. Get Specific Ballot with Items", func(t *testing.T) {
		// Mock ballot query
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`SELECT b.id, b.title, b.description, b.creator_id, b.is_active, b.created_at, b.updated_at
FROM ballots b WHERE b.id = $1`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", userID, true, createdAt, createdAt))

		// Mock ballot items query
		testSetup.Mock.ExpectQuery(`SELECT id, ballot_id, title, description, vote_count
FROM ballot_items 
WHERE ballot_id = $1 
ORDER BY id ASC`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(1, ballotID, "Option A", "First choice", 0).
				AddRow(2, ballotID, "Option B", "Second choice", 0))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballot models.Ballot
		err = parseJSONResponse(recorder, &ballot)
		require.NoError(t, err)

		assert.Equal(t, ballotID, ballot.ID)
		require.Len(t, ballot.Items, 2)
		assert.Equal(t, 0, ballot.Items[0].VoteCount)
		assert.Equal(t, 0, ballot.Items[1].VoteCount)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("5. Vote on Ballot", func(t *testing.T) {
		ballotItemID := 1

		// Mock ballot exists and is active
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		// Mock ballot item belongs to ballot
		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		// Mock transaction begin
		testSetup.Mock.ExpectBegin()

		// Mock no existing vote
		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(sql.ErrNoRows)

		// Mock insert new vote
		testSetup.Mock.ExpectExec("INSERT INTO votes (user_id, ballot_id, ballot_item_id) VALUES ($1, $2, $3)").
			WithArgs(userID, ballotID, ballotItemID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock update vote count
		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count + 1 WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock transaction commit
		testSetup.Mock.ExpectCommit()

		reqBody := models.VoteRequest{
			BallotItemID: ballotItemID,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("6. Get User's Vote", func(t *testing.T) {
		ballotItemID := 1

		// Mock user vote found
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("SELECT id, user_id, ballot_id, ballot_item_id, created_at FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "ballot_id", "ballot_item_id", "created_at"}).
				AddRow(1, userID, ballotID, ballotItemID, createdAt))

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var vote models.Vote
		err = parseJSONResponse(recorder, &vote)
		require.NoError(t, err)

		assert.Equal(t, userID, vote.UserID)
		assert.Equal(t, ballotID, vote.BallotID)
		assert.Equal(t, ballotItemID, vote.BallotItemID)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("7. Get Ballot Results", func(t *testing.T) {
		// Mock ballot exists
		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock ballot results (Option A should have 1 vote now)
		testSetup.Mock.ExpectQuery(`SELECT id, ballot_id, title, description, vote_count
FROM ballot_items 
WHERE ballot_id = $1 
ORDER BY vote_count DESC, id ASC`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(1, ballotID, "Option A", "First choice", 1).
				AddRow(2, ballotID, "Option B", "Second choice", 0))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var response map[string]interface{}
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(ballotID), response["ballot_id"])
		assert.Equal(t, float64(1), response["total_votes"])

		results, ok := response["results"].([]interface{})
		require.True(t, ok)
		require.Len(t, results, 2)

		// Option A should be first (highest vote count)
		firstResult := results[0].(map[string]interface{})
		assert.Equal(t, "Option A", firstResult["title"])
		assert.Equal(t, float64(1), firstResult["vote_count"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("8. Get User's Ballots", func(t *testing.T) {
		// Mock user ballots query
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`SELECT id, title, description, creator_id, is_active, created_at, updated_at
FROM ballots 
WHERE creator_id = $1 
ORDER BY created_at DESC`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", userID, true, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-ballots", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballots []models.Ballot
		err = parseJSONResponse(recorder, &ballots)
		require.NoError(t, err)

		assert.Len(t, ballots, 1)
		assert.Equal(t, ballotID, ballots[0].ID)
		assert.Equal(t, "Integration Test Ballot", ballots[0].Title)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("9. Get User Profile", func(t *testing.T) {
		// Mock user query
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("SELECT id, username, email, created_at, updated_at FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at"}).
				AddRow(userID, username, email, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var user models.User
		err = parseJSONResponse(recorder, &user)
		require.NoError(t, err)

		assert.Equal(t, userID, user.ID)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, email, user.Email)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	testSetup, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testSetup.DB.Close()

	req, err := CreateTestRequest("GET", "/health", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	testSetup.Router.ServeHTTP(recorder, req)

	assert.Equal(t, 200, recorder.Code)

	var response map[string]interface{}
	err = parseJSONResponse(recorder, &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
}

// TestJWTUtilities tests JWT token generation and validation
func TestJWTUtilities(t *testing.T) {
	userID := 123
	email := "test@example.com"

	t.Run("Generate and Validate JWT", func(t *testing.T) {
		// Generate token
		token, err := utils.GenerateJWT(userID, email)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate token
		claims, err := utils.ValidateJWT(token)
		require.NoError(t, err)

		assert.Equal(t, float64(userID), claims["user_id"])
		assert.Equal(t, email, claims["email"])
	})

	t.Run("Validate Invalid JWT", func(t *testing.T) {
		_, err := utils.ValidateJWT("invalid.token.here")
		assert.Error(t, err)
	})
}

// TestPasswordHashing tests password hashing and verification
func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	t.Run("Hash and Check Password", func(t *testing.T) {
		// Hash password
		hashedPassword, err := utils.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hashedPassword)
		assert.NotEqual(t, password, hashedPassword)

		// Check correct password
		isValid := utils.CheckPassword(password, hashedPassword)
		assert.True(t, isValid)

		// Check incorrect password
		isInvalid := utils.CheckPassword("wrongpassword", hashedPassword)
		assert.False(t, isInvalid)
	})
}
