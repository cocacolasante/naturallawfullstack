package tests

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
	"voting-api/database"
	"voting-api/handlers"
	"voting-api/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
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

	t.Run("Vote Invalid Ballot ID Path Param", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := models.VoteRequest{
			BallotItemID: 1,
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/notanid/vote", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid ballot ID")
	})

	t.Run("Vote No option_id or ballot_item_id", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1

		// Send empty body - no ballot_item_id or option_id
		reqBody := models.VoteRequest{
			BallotItemID: 0,
			OptionID:     0,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "option_id or ballot_item_id is required")
	})

	t.Run("Vote Using option_id Field", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		optionID := 1

		// Mock ballot exists and is active
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		// Mock ballot item belongs to ballot
		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(optionID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		// Mock transaction begin
		testSetup.Mock.ExpectBegin()

		// Mock check for existing vote (none exists)
		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(sql.ErrNoRows)

		// Mock insert new vote
		testSetup.Mock.ExpectExec("INSERT INTO votes (user_id, ballot_id, ballot_item_id) VALUES ($1, $2, $3)").
			WithArgs(userID, ballotID, optionID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock update vote count
		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count + 1 WHERE id = $1").
			WithArgs(optionID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock transaction commit
		testSetup.Mock.ExpectCommit()

		// Send using option_id field (not ballot_item_id)
		reqBody := models.VoteRequest{
			OptionID: optionID,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Ballot Item Wrong Ballot (itemBallotID != ballotID)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 5

		// Mock ballot exists and is active
		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		// Mock ballot item belongs to a DIFFERENT ballot
		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(99)) // wrong ballot

		reqBody := models.VoteRequest{
			BallotItemID: ballotItemID,
		}

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Ballot item does not belong to this ballot")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Ballot Check", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnError(errors.New("db error"))

		reqBody := models.VoteRequest{BallotItemID: ballotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Ballot Item Check", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnError(errors.New("db error"))

		reqBody := models.VoteRequest{BallotItemID: ballotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Begin TX", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		testSetup.Mock.ExpectBegin().WillReturnError(errors.New("begin error"))

		reqBody := models.VoteRequest{BallotItemID: ballotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Existing Vote Query (not ErrNoRows)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		testSetup.Mock.ExpectBegin()

		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(errors.New("db error"))

		reqBody := models.VoteRequest{BallotItemID: ballotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Decrease Vote Count", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		oldBallotItemID := 1
		newBallotItemID := 2

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(newBallotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		testSetup.Mock.ExpectBegin()

		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_item_id"}).AddRow(1, oldBallotItemID))

		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count - 1 WHERE id = $1").
			WithArgs(oldBallotItemID).
			WillReturnError(errors.New("db error"))

		reqBody := models.VoteRequest{BallotItemID: newBallotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating vote count")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Update Vote Record", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		oldBallotItemID := 1
		newBallotItemID := 2

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(newBallotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		testSetup.Mock.ExpectBegin()

		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_item_id"}).AddRow(1, oldBallotItemID))

		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count - 1 WHERE id = $1").
			WithArgs(oldBallotItemID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		testSetup.Mock.ExpectExec("UPDATE votes SET ballot_item_id = $1 WHERE id = $2").
			WithArgs(newBallotItemID, 1).
			WillReturnError(errors.New("db error"))

		reqBody := models.VoteRequest{BallotItemID: newBallotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating vote")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Insert New Vote", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		testSetup.Mock.ExpectBegin()

		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectExec("INSERT INTO votes (user_id, ballot_id, ballot_item_id) VALUES ($1, $2, $3)").
			WithArgs(userID, ballotID, ballotItemID).
			WillReturnError(errors.New("insert error"))

		reqBody := models.VoteRequest{BallotItemID: ballotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating vote")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Increase Vote Count", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		testSetup.Mock.ExpectBegin()

		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectExec("INSERT INTO votes (user_id, ballot_id, ballot_item_id) VALUES ($1, $2, $3)").
			WithArgs(userID, ballotID, ballotItemID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count + 1 WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnError(errors.New("update error"))

		reqBody := models.VoteRequest{BallotItemID: ballotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error updating vote count")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Commit", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1
		ballotItemID := 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT ballot_id FROM ballot_items WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnRows(sqlmock.NewRows([]string{"ballot_id"}).AddRow(ballotID))

		testSetup.Mock.ExpectBegin()

		testSetup.Mock.ExpectQuery("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(sql.ErrNoRows)

		testSetup.Mock.ExpectExec("INSERT INTO votes (user_id, ballot_id, ballot_item_id) VALUES ($1, $2, $3)").
			WithArgs(userID, ballotID, ballotItemID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		testSetup.Mock.ExpectExec("UPDATE ballot_items SET vote_count = vote_count + 1 WHERE id = $1").
			WithArgs(ballotItemID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		testSetup.Mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		reqBody := models.VoteRequest{BallotItemID: ballotItemID}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error committing transaction")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewVoteHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("POST", "/api/v1/ballots/1/vote", models.VoteRequest{BallotItemID: 1})
		// Set the ballot_id param
		c.Params = gin.Params{gin.Param{Key: "ballot_id", Value: "1"}}
		handler.Vote(c)
		assert.Equal(t, 401, w.Code)
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

	t.Run("Get User Vote Invalid Ballot ID", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/ballots/notanid/my-vote", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid ballot ID")
	})

	t.Run("Get User Vote DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1

		testSetup.Mock.ExpectQuery("SELECT id, user_id, ballot_id, ballot_item_id, created_at FROM votes WHERE user_id = $1 AND ballot_id = $2").
			WithArgs(userID, ballotID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Vote No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewVoteHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/ballots/1/my-vote", nil)
		c.Params = gin.Params{gin.Param{Key: "ballot_id", Value: "1"}}
		handler.GetUserVote(c)
		assert.Equal(t, 401, w.Code)
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
		ORDER BY vote_count DESC, id ASC
	`).
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
		ORDER BY vote_count DESC, id ASC
	`).
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

	t.Run("Get Ballot Results Invalid Ballot ID", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots/notanid/results", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid ballot ID")
	})

	t.Run("Get Ballot Results DB Error for EXISTS check", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 1

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Results DB Error for Results Query", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 1

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		testSetup.Mock.ExpectQuery(`SELECT id, ballot_id, title, description, vote_count
		FROM ballot_items
		WHERE ballot_id = $1
		ORDER BY vote_count DESC, id ASC
	`).
			WithArgs(ballotID).
			WillReturnError(errors.New("results error"))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error fetching results")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Results Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 1

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Return wrong columns to trigger scan error
		testSetup.Mock.ExpectQuery(`SELECT id, ballot_id, title, description, vote_count
		FROM ballot_items
		WHERE ballot_id = $1
		ORDER BY vote_count DESC, id ASC
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("not-an-int"))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error scanning result")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestVoteShouldBindJSONError(t *testing.T) {
	t.Run("Vote Invalid JSON Body", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		ballotID := 1

		req, err := CreateAuthenticatedRawBodyRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), "not-valid-json{", userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})
}
