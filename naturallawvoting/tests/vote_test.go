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

func TestVote(t *testing.T) {
	t.Run("Vote Successfully (First Vote)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID := 1
		email := "test@example.com"
		ballotID := 1
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

		// Mock check for existing vote (none exists)
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

		var response map[string]interface{}
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.Equal(t, "Vote recorded successfully", response["message"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote Successfully (Change Vote)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		userID := 1
		email := "test@example.com"
		ballotID := 1
		oldBallotItemID := 1
		newBallotItemID := 2

		// Mock ballot exists and is active
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		// Mock ballot item belongs to ballot
		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(newBallotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		// Mock transaction begin
		testSetup.Mock.ExpectBegin()

		// Mock existing vote found
		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_item_id"}).AddRow(1, oldBallotItemID))

		// Mock decrease vote count for old choice
		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count - 1 WHERE id = $1").
			WithArgs(oldBallotItemID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock update vote record
		testSetup.Mock.ExpectExec("UPDATE votes SET ballot_item_id = $1 WHERE id = $2").
			WithArgs(newBallotItemID, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock increase vote count for new choice
		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count + 1 WHERE id = $1").
			WithArgs(newBallotItemID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock transaction commit
		testSetup.Mock.ExpectCommit()

		reqBody := models.VoteRequest{
			BallotItemID: newBallotItemID,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote on Non-existent Ballot", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		userID := 1
		email := "test@example.com"
		ballotID := 999
		ballotItemID := 1

		// Mock ballot not found
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnError(sql.ErrNoRows)

		reqBody := models.VoteRequest{
			BallotItemID: ballotItemID,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Ballot not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote on Inactive Ballot", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		// Mock ballot exists but is inactive
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(false))

		reqBody := models.VoteRequest{
			BallotItemID: ballotItemID,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Ballot is not active")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote on Invalid Ballot Item", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 999

		// Mock ballot exists and is active
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		// Mock ballot item not found
		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnError(sql.ErrNoRows)

		reqBody := models.VoteRequest{
			BallotItemID: ballotItemID,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Ballot item not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		ballotID := 1
		reqBody := models.VoteRequest{
			BallotItemID: 1,
		}

		req, err := CreateTestRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})
}

func TestGetUserVote(t *testing.T) {
	t.Run("Get User Vote Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID := 1
		email := "test@example.com"
		ballotID := 1

		// Mock user vote found
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("SELECT id, user_id, ballot_id, ballot_item_id, created_at FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "ballot_id", "ballot_item_id", "created_at"}).
				AddRow(1, userID, ballotID, 2, createdAt))

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var vote models.Vote
		err = parseJSONResponse(recorder, &vote)
		require.NoError(t, err)

		assert.Equal(t, 1, vote.ID)
		assert.Equal(t, userID, vote.UserID)
		assert.Equal(t, ballotID, vote.BallotID)
		assert.Equal(t, 2, vote.BallotItemID)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Vote Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		userID := 1
		email := "test@example.com"
		ballotID := 1

		// Mock no vote found
		testSetup.Mock.ExpectQuery("SELECT id, user_id, ballot_id, ballot_item_id, created_at FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "No vote found for this ballot")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Vote Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		ballotID := 1

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})
}

func TestGetBallotResults(t *testing.T) {
	t.Run("Get Ballot Results Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		ballotID := 1

		// Mock ballot exists
		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock ballot results
		testSetup.Mock.ExpectQuery(`SELECT id, ballot_id, title, description, vote_count
FROM ballot_items 
WHERE ballot_id = $1 
ORDER BY vote_count DESC, id ASC`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}).
				AddRow(1, ballotID, "Option 1", "First option", 10).
				AddRow(2, ballotID, "Option 2", "Second option", 5).
				AddRow(3, ballotID, "Option 3", "Third option", 3))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var response map[string]interface{}
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(ballotID), response["ballot_id"])
		assert.Equal(t, float64(18), response["total_votes"]) // 10 + 5 + 3

		results, ok := response["results"].([]interface{})
		assert.True(t, ok)
		require.Len(t, results, 3)

		// Verify results are ordered by vote count (descending)
		firstResult := results[0].(map[string]interface{})
		assert.Equal(t, float64(10), firstResult["vote_count"])
		assert.Equal(t, "Option 1", firstResult["title"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Results Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		ballotID := 999

		// Mock ballot doesn't exist
		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "Ballot not found")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Results Empty", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		
		ballotID := 1

		// Mock ballot exists
		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock empty results
		testSetup.Mock.ExpectQuery(`SELECT id, ballot_id, title, description, vote_count
FROM ballot_items 
WHERE ballot_id = $1 
ORDER BY vote_count DESC, id ASC`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "vote_count"}))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		var response map[string]interface{}
		err = parseJSONResponse(recorder, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(ballotID), response["ballot_id"])
		assert.Equal(t, float64(0), response["total_votes"])

		results, ok := response["results"].([]interface{})
		require.True(t, ok)
		assert.Len(t, results, 0)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}
