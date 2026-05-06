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

		// Mock ballot insertion - updated query with category/superstate/state/district
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("INSERT INTO ballots (title, description, category, superstate, state, district, creator_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, title, description, category, superstate, state, COALESCE(district, ''), creator_id, is_active, created_at, updated_at").
			WithArgs("Integration Test Ballot", "Testing the full workflow", "", "", "", "", userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", "", "", "", "", userID, true, createdAt, createdAt))

		// Mock ballot items insertion
		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description").
			WithArgs(ballotID, "Option A", "First choice").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(1, ballotID, "Option A", "First choice"))

		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description").
			WithArgs(ballotID, "Option B", "Second choice").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(2, ballotID, "Option B", "Second choice"))

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
		// Mock ballots query - updated columns
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true ORDER BY b.created_at DESC`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", "", "", "", "", userID, true, createdAt, createdAt))

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
		// Mock ballot query - updated columns
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b WHERE b.id = $1
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", "", "", "", "", userID, true, createdAt, createdAt))

		// Mock ballot items query
		testSetup.Mock.ExpectQuery(`
		SELECT id, ballot_id, title, description
		FROM ballot_items
		WHERE ballot_id = $1
		ORDER BY id ASC
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(1, ballotID, "Option A", "First choice").
				AddRow(2, ballotID, "Option B", "Second choice"))

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
		assert.Equal(t, "Option A", ballot.Items[0].Title)
		assert.Equal(t, "Option B", ballot.Items[1].Title)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("5. Vote on Ballot", func(t *testing.T) {
		itemA := 1
		itemB := 2

		// Mock ballot exists and is active
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		// Mock all submitted ballot items belong to this ballot
		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(ballotID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(itemA).AddRow(itemB))

		// Transaction with prepared upsert per score
		testSetup.Mock.ExpectBegin()
		upsertSQL := `
		INSERT INTO votes (user_id, ballot_id, ballot_item_id, score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, ballot_item_id)
		DO UPDATE SET score = EXCLUDED.score, updated_at = CURRENT_TIMESTAMP
	`
		testSetup.Mock.ExpectPrepare(upsertSQL)
		testSetup.Mock.ExpectExec(upsertSQL).
			WithArgs(userID, ballotID, itemA, 80).
			WillReturnResult(sqlmock.NewResult(1, 1))
		testSetup.Mock.ExpectExec(upsertSQL).
			WithArgs(userID, ballotID, itemB, 30).
			WillReturnResult(sqlmock.NewResult(2, 1))
		testSetup.Mock.ExpectCommit()

		score80, score30 := 80, 30
		reqBody := models.VoteRequest{
			Scores: []models.ScoreEntry{
				{BallotItemID: itemA, Score: &score80},
				{BallotItemID: itemB, Score: &score30},
			},
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("6. Get User's Vote", func(t *testing.T) {
		itemA := 1
		itemB := 2

		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`SELECT id, user_id, ballot_id, ballot_item_id, score, created_at, updated_at
		 FROM votes WHERE user_id = $1 AND ballot_id = $2 ORDER BY ballot_item_id ASC`).
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "ballot_id", "ballot_item_id", "score", "created_at", "updated_at"}).
				AddRow(1, userID, ballotID, itemA, 80, createdAt, createdAt).
				AddRow(2, userID, ballotID, itemB, 30, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var response map[string]interface{}
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(ballotID), response["ballot_id"])
		scores, ok := response["scores"].([]interface{})
		require.True(t, ok)
		require.Len(t, scores, 2)

		first := scores[0].(map[string]interface{})
		assert.Equal(t, float64(itemA), first["ballot_item_id"])
		assert.Equal(t, float64(80), first["score"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("7. Get Ballot Results", func(t *testing.T) {
		// Mock ballot exists
		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock weighted-score aggregation
		testSetup.Mock.ExpectQuery(`
		SELECT bi.id, bi.ballot_id, bi.title, bi.description,
		       COALESCE(AVG(v.score), 0)::float8 AS average_score,
		       COALESCE(SUM(v.score), 0)::bigint AS total_score,
		       COUNT(v.id)::bigint AS voter_count
		FROM ballot_items bi
		LEFT JOIN votes v ON v.ballot_item_id = bi.id
		WHERE bi.ballot_id = $1
		GROUP BY bi.id, bi.ballot_id, bi.title, bi.description
		ORDER BY average_score DESC, bi.id ASC
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "average_score", "total_score", "voter_count"}).
				AddRow(1, ballotID, "Option A", "First choice", 80.0, int64(80), int64(1)).
				AddRow(2, ballotID, "Option B", "Second choice", 30.0, int64(30), int64(1)))

		testSetup.Mock.ExpectQuery(`SELECT COUNT(DISTINCT user_id) FROM votes WHERE ballot_id = $1`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var response map[string]interface{}
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(ballotID), response["ballot_id"])
		assert.Equal(t, float64(1), response["total_voters"])

		results, ok := response["results"].([]interface{})
		require.True(t, ok)
		require.Len(t, results, 2)

		firstResult := results[0].(map[string]interface{})
		assert.Equal(t, "Option A", firstResult["title"])
		assert.Equal(t, float64(80), firstResult["average_score"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("8. Get User's Ballots", func(t *testing.T) {
		// Mock user ballots query - updated columns
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`
		SELECT id, title, description, category, COALESCE(superstate, ''), COALESCE(state, ''), COALESCE(district, ''), creator_id, is_active, created_at, updated_at
		FROM ballots
		WHERE creator_id = $1
		ORDER BY created_at DESC
	`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Integration Test Ballot", "Testing the full workflow", "", "", "", "", userID, true, createdAt, createdAt))

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
