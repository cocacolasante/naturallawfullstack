package tests

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
	"voting-api/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBallot(t *testing.T) {
	testSetup, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testSetup.DB.Close()

	t.Run("Create Ballot Successfully", func(t *testing.T) {
		userID := 1
		email := "test@example.com"

		// Mock transaction begin
		testSetup.Mock.ExpectBegin()

		// Mock ballot insertion
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("INSERT INTO ballots (title, description, creator_id) VALUES ($1, $2, $3) RETURNING id, title, description, creator_id, is_active, created_at, updated_at").
			WithArgs("Best Programming Language", "Vote for your favorite", userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(1, "Best Programming Language", "Vote for your favorite", userID, true, createdAt, createdAt))

		// Mock ballot items insertion
		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description, vote_count").
			WithArgs(1, "Go", "Fast and efficient").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(1, 1, "Go", "Fast and efficient", 0))

		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description, vote_count").
			WithArgs(1, "Python", "Easy to learn").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(2, 1, "Python", "Easy to learn", 0))

		// Mock transaction commit
		testSetup.Mock.ExpectCommit()

		reqBody := models.CreateBallotRequest{
			Title:       "Best Programming Language",
			Description: "Vote for your favorite",
			Items: []models.CreateBallotItemRequest{
				{Title: "Go", Description: "Fast and efficient"},
				{Title: "Python", Description: "Easy to learn"},
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

		assert.Equal(t, 1, ballot.ID)
		assert.Equal(t, "Best Programming Language", ballot.Title)
		assert.Equal(t, "Vote for your favorite", ballot.Description)
		assert.Equal(t, userID, ballot.CreatorID)
		assert.True(t, ballot.IsActive)
		assert.Len(t, ballot.Items, 2)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Ballot Without Authentication", func(t *testing.T) {
		reqBody := models.CreateBallotRequest{
			Title:       "Test Ballot",
			Description: "Test Description",
			Items: []models.CreateBallotItemRequest{
				{Title: "Option 1", Description: "First option"},
				{Title: "Option 2", Description: "Second option"},
			},
		}

		req, err := CreateTestRequest("POST", "/api/v1/ballots", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Create Ballot With Invalid Data", func(t *testing.T) {
		userID := 1
		email := "test@example.com"

		reqBody := models.CreateBallotRequest{
			Title:       "", // Empty title
			Description: "Test Description",
			Items:       []models.CreateBallotItemRequest{}, // Empty items
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})
}

func TestGetAllBallots(t *testing.T) {
	testSetup, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testSetup.DB.Close()

	t.Run("Get All Ballots Successfully", func(t *testing.T) {
		// Mock ballots query
		createdAt1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		createdAt2 := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at", "creator_username"}).
			AddRow(1, "Ballot 1", "Description 1", 1, true, createdAt1, createdAt1, "user1").
			AddRow(2, "Ballot 2", "Description 2", 2, true, createdAt2, createdAt2, "user2")

		testSetup.Mock.ExpectQuery(`SELECT b.id, b.title, b.description, b.creator_id, b.is_active, b.created_at, b.updated_at,
       u.username as creator_username
FROM ballots b 
JOIN users u ON b.creator_id = u.id 
WHERE b.is_active = true 
ORDER BY b.created_at DESC`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballots []models.Ballot
		err = parseJSONResponse(recorder, &ballots)
		require.NoError(t, err)

		assert.Len(t, ballots, 2)
		assert.Equal(t, "Ballot 1", ballots[0].Title)
		assert.Equal(t, "Ballot 2", ballots[1].Title)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots Empty Result", func(t *testing.T) {
		// Mock empty result
		rows := sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at", "creator_username"})
		testSetup.Mock.ExpectQuery(`SELECT b.id, b.title, b.description, b.creator_id, b.is_active, b.created_at, b.updated_at,
       u.username as creator_username
FROM ballots b 
JOIN users u ON b.creator_id = u.id 
WHERE b.is_active = true 
ORDER BY b.created_at DESC`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballots []models.Ballot
		err = parseJSONResponse(recorder, &ballots)
		require.NoError(t, err)

		assert.Len(t, ballots, 0)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetBallot(t *testing.T) {
	testSetup, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testSetup.DB.Close()

	t.Run("Get Ballot Successfully", func(t *testing.T) {
		ballotID := 1

		// Mock ballot query
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`SELECT b.id, b.title, b.description, b.creator_id, b.is_active, b.created_at, b.updated_at
FROM ballots b WHERE b.id = $1`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Test Ballot", "Test Description", 1, true, createdAt, createdAt))

		// Mock ballot items query
		testSetup.Mock.ExpectQuery(`SELECT id, ballot_id, title, description, vote_count
FROM ballot_items 
WHERE ballot_id = $1 
ORDER BY id ASC`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(1, ballotID, "Option 1", "First option", 5).
				AddRow(2, ballotID, "Option 2", "Second option", 3))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballot models.Ballot
		err = parseJSONResponse(recorder, &ballot)
		require.NoError(t, err)

		assert.Equal(t, ballotID, ballot.ID)
		assert.Equal(t, "Test Ballot", ballot.Title)
		require.Len(t, ballot.Items, 2)
		assert.Equal(t, 5, ballot.Items[0].VoteCount)
		assert.Equal(t, 3, ballot.Items[1].VoteCount)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Not Found", func(t *testing.T) {
		ballotID := 999

		// Mock ballot not found
		testSetup.Mock.ExpectQuery(`SELECT b.id, b.title, b.description, b.creator_id, b.is_active, b.created_at, b.updated_at
FROM ballots b WHERE b.id = $1`).
			WithArgs(ballotID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Ballot not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Invalid ID", func(t *testing.T) {
		req, err := CreateTestRequest("GET", "/api/v1/public/ballots/invalid", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid ballot ID")
	})
}

func TestGetUserBallots(t *testing.T) {
	testSetup, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testSetup.DB.Close()

	t.Run("Get User Ballots Successfully", func(t *testing.T) {
		userID := 1
		email := "test@example.com"

		// Mock user ballots query
		createdAt1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		createdAt2 := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at"}).
			AddRow(1, "My Ballot 1", "My Description 1", userID, true, createdAt1, createdAt1).
			AddRow(2, "My Ballot 2", "My Description 2", userID, false, createdAt2, createdAt2)

		testSetup.Mock.ExpectQuery(`SELECT id, title, description, creator_id, is_active, created_at, updated_at
FROM ballots 
WHERE creator_id = $1 
ORDER BY created_at DESC`).
			WithArgs(userID).
			WillReturnRows(rows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-ballots", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballots []models.Ballot
		err = parseJSONResponse(recorder, &ballots)
		require.NoError(t, err)

		assert.Len(t, ballots, 2)
		assert.Equal(t, "My Ballot 1", ballots[0].Title)
		assert.True(t, ballots[0].IsActive)
		assert.Equal(t, "My Ballot 2", ballots[1].Title)
		assert.False(t, ballots[1].IsActive)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Ballots Without Authentication", func(t *testing.T) {
		req, err := CreateTestRequest("GET", "/api/v1/my-ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Get User Ballots Empty Result", func(t *testing.T) {
		userID := 1
		email := "test@example.com"

		// Mock empty result
		rows := sqlmock.NewRows([]string{"id", "title", "description", "creator_id", "is_active", "created_at", "updated_at"})
		testSetup.Mock.ExpectQuery(`SELECT id, title, description, creator_id, is_active, created_at, updated_at
FROM ballots 
WHERE creator_id = $1 
ORDER BY created_at DESC`).
			WithArgs(userID).
			WillReturnRows(rows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-ballots", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var ballots []models.Ballot
		err = parseJSONResponse(recorder, &ballots)
		require.NoError(t, err)

		assert.Len(t, ballots, 0)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}
