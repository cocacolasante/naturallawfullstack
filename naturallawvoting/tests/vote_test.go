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

const upsertVoteSQL = `
		INSERT INTO votes (user_id, ballot_id, ballot_item_id, score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, ballot_item_id)
		DO UPDATE SET score = EXCLUDED.score, updated_at = CURRENT_TIMESTAMP
	`

const getUserVoteSQL = `SELECT id, user_id, ballot_id, ballot_item_id, score, created_at, updated_at
		 FROM votes WHERE user_id = $1 AND ballot_id = $2 ORDER BY ballot_item_id ASC`

const getBallotResultsSQL = `
		SELECT bi.id, bi.ballot_id, bi.title, bi.description,
		       COALESCE(AVG(v.score), 0)::float8 AS average_score,
		       COALESCE(SUM(v.score), 0)::bigint AS total_score,
		       COUNT(v.id)::bigint AS voter_count
		FROM ballot_items bi
		LEFT JOIN votes v ON v.ballot_item_id = bi.id
		WHERE bi.ballot_id = $1
		GROUP BY bi.id, bi.ballot_id, bi.title, bi.description
		ORDER BY average_score DESC, bi.id ASC
	`

const getBallotDistinctVotersSQL = `SELECT COUNT(DISTINCT user_id) FROM votes WHERE ballot_id = $1`

func intp(v int) *int { return &v }

// singleScoreRequest builds a VoteRequest carrying one option's score.
func singleScoreRequest(itemID, score int) models.VoteRequest {
	return models.VoteRequest{
		Scores: []models.ScoreEntry{
			{BallotItemID: itemID, Score: intp(score)},
		},
	}
}

func TestVote(t *testing.T) {
	t.Run("Vote Successfully (First Score)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID, ballotID, itemID, score := 1, 1, 1, 75

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(ballotID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(itemID))

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectPrepare(upsertVoteSQL)
		testSetup.Mock.ExpectExec(upsertVoteSQL).
			WithArgs(userID, ballotID, itemID, score).
			WillReturnResult(sqlmock.NewResult(1, 1))
		testSetup.Mock.ExpectCommit()

		reqBody := singleScoreRequest(itemID, score)
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		var response map[string]interface{}
		require.NoError(t, parseJSONResponse(recorder, &response))
		assert.Equal(t, "Vote recorded successfully", response["message"])
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote Successfully (Update Existing Score)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID, ballotID, itemID, newScore := 1, 1, 1, 42

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(ballotID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(itemID))

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectPrepare(upsertVoteSQL)
		// ON CONFLICT path: same SQL, just updates the row.
		testSetup.Mock.ExpectExec(upsertVoteSQL).
			WithArgs(userID, ballotID, itemID, newScore).
			WillReturnResult(sqlmock.NewResult(0, 1))
		testSetup.Mock.ExpectCommit()

		reqBody := singleScoreRequest(itemID, newScore)
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote Bulk: Score Every Option", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID, ballotID := 1, 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(ballotID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2).AddRow(3))

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectPrepare(upsertVoteSQL)
		testSetup.Mock.ExpectExec(upsertVoteSQL).WithArgs(userID, ballotID, 1, 90).WillReturnResult(sqlmock.NewResult(1, 1))
		testSetup.Mock.ExpectExec(upsertVoteSQL).WithArgs(userID, ballotID, 2, 50).WillReturnResult(sqlmock.NewResult(2, 1))
		testSetup.Mock.ExpectExec(upsertVoteSQL).WithArgs(userID, ballotID, 3, 0).WillReturnResult(sqlmock.NewResult(3, 1))
		testSetup.Mock.ExpectCommit()

		reqBody := models.VoteRequest{
			Scores: []models.ScoreEntry{
				{BallotItemID: 1, Score: intp(90)},
				{BallotItemID: 2, Score: intp(50)},
				{BallotItemID: 3, Score: intp(0)},
			},
		}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		var response map[string]interface{}
		require.NoError(t, parseJSONResponse(recorder, &response))
		assert.Equal(t, float64(3), response["score_count"])
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote Using option_id Field", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID, ballotID, optionID, score := 1, 1, 1, 60

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(ballotID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(optionID))

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectPrepare(upsertVoteSQL)
		testSetup.Mock.ExpectExec(upsertVoteSQL).
			WithArgs(userID, ballotID, optionID, score).
			WillReturnResult(sqlmock.NewResult(1, 1))
		testSetup.Mock.ExpectCommit()

		reqBody := models.VoteRequest{
			Scores: []models.ScoreEntry{{OptionID: optionID, Score: intp(score)}},
		}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), reqBody, userID, "test@example.com")
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
		userID, ballotID := 1, 999

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), singleScoreRequest(1, 50), userID, "test@example.com")
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
		userID, ballotID := 1, 1

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(false))

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), singleScoreRequest(1, 50), userID, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Ballot is not active")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote on Ballot Item That Doesn't Belong", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID, ballotID, itemID := 1, 1, 5

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		// No matching rows — item doesn't belong to this ballot.
		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(ballotID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/ballots/%d/vote", ballotID), singleScoreRequest(itemID, 50), userID, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Ballot item does not belong to this ballot")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 50))
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Vote Invalid Ballot ID Path Param", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/notanid/vote", singleScoreRequest(1, 50), 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid ballot ID")
	})

	t.Run("Vote Missing option_id and ballot_item_id", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		reqBody := models.VoteRequest{
			Scores: []models.ScoreEntry{{Score: intp(50)}},
		}
		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", reqBody, 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "option_id or ballot_item_id is required for each score")
	})

	t.Run("Vote Score Out Of Range (Above 100)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 150), 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "score must be between 0 and 100")
	})

	t.Run("Vote Score Out Of Range (Below 0)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, -5), 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "score must be between 0 and 100")
	})

	t.Run("Vote Empty Scores Array", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		reqBody := models.VoteRequest{Scores: []models.ScoreEntry{}}
		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", reqBody, 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Vote DB Error Ballot Check", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(1).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 50), 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Validating Ballot Items", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(1, sqlmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 50), 1, "test@example.com")
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

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(1, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		testSetup.Mock.ExpectBegin().WillReturnError(errors.New("begin error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 50), 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Upsert", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(1, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectPrepare(upsertVoteSQL)
		testSetup.Mock.ExpectExec(upsertVoteSQL).
			WithArgs(1, 1, 1, 50).
			WillReturnError(errors.New("upsert error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 50), 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error recording vote")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote DB Error Commit", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery("SELECT is_active FROM ballots WHERE id = $1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		testSetup.Mock.ExpectQuery("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)").
			WithArgs(1, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectPrepare(upsertVoteSQL)
		testSetup.Mock.ExpectExec(upsertVoteSQL).
			WithArgs(1, 1, 1, 50).
			WillReturnResult(sqlmock.NewResult(1, 1))
		testSetup.Mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 50), 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error committing transaction")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Vote No User ID In Context", func(t *testing.T) {
		mockDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()

		db := &database.DB{DB: mockDB}
		handler := handlers.NewVoteHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("POST", "/api/v1/ballots/1/vote", singleScoreRequest(1, 50))
		c.Params = gin.Params{gin.Param{Key: "ballot_id", Value: "1"}}
		handler.Vote(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestGetUserVote(t *testing.T) {
	t.Run("Get User Scores Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID, ballotID := 1, 1
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectQuery(getUserVoteSQL).
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "ballot_id", "ballot_item_id", "score", "created_at", "updated_at"}).
				AddRow(1, userID, ballotID, 1, 80, createdAt, createdAt).
				AddRow(2, userID, ballotID, 2, 25, createdAt, createdAt))

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil, userID, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		var response map[string]interface{}
		require.NoError(t, parseJSONResponse(recorder, &response))

		assert.Equal(t, float64(ballotID), response["ballot_id"])
		scores := response["scores"].([]interface{})
		require.Len(t, scores, 2)
		first := scores[0].(map[string]interface{})
		assert.Equal(t, float64(1), first["ballot_item_id"])
		assert.Equal(t, float64(1), first["option_id"])
		assert.Equal(t, float64(80), first["score"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Scores Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		userID, ballotID := 1, 1

		testSetup.Mock.ExpectQuery(getUserVoteSQL).
			WithArgs(userID, ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "ballot_id", "ballot_item_id", "score", "created_at", "updated_at"}))

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/ballots/%d/my-vote", ballotID), nil, userID, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 404, "No vote found for this ballot")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Scores Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/ballots/1/my-vote", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Get User Scores Invalid Ballot ID", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/ballots/notanid/my-vote", nil, 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid ballot ID")
	})

	t.Run("Get User Scores DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery(getUserVoteSQL).
			WithArgs(1, 1).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/ballots/1/my-vote", nil, 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Scores No User ID In Context", func(t *testing.T) {
		mockDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()

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

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		testSetup.Mock.ExpectQuery(getBallotResultsSQL).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "average_score", "total_score", "voter_count"}).
				AddRow(1, ballotID, "Option 1", "First option", 87.5, int64(175), int64(2)).
				AddRow(2, ballotID, "Option 2", "Second option", 50.0, int64(100), int64(2)).
				AddRow(3, ballotID, "Option 3", "Third option", 10.0, int64(20), int64(2)))

		testSetup.Mock.ExpectQuery(getBallotDistinctVotersSQL).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		var response map[string]interface{}
		require.NoError(t, parseJSONResponse(recorder, &response))

		assert.Equal(t, float64(ballotID), response["ballot_id"])
		assert.Equal(t, float64(2), response["total_voters"])

		results := response["results"].([]interface{})
		require.Len(t, results, 3)
		first := results[0].(map[string]interface{})
		assert.Equal(t, "Option 1", first["title"])
		assert.Equal(t, 87.5, first["average_score"])
		assert.Equal(t, float64(175), first["total_score"])
		assert.Equal(t, float64(2), first["voter_count"])

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Results Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()
		ballotID := 999

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

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		testSetup.Mock.ExpectQuery(getBallotResultsSQL).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description", "average_score", "total_score", "voter_count"}))

		testSetup.Mock.ExpectQuery(getBallotDistinctVotersSQL).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d/results", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		var response map[string]interface{}
		require.NoError(t, parseJSONResponse(recorder, &response))
		assert.Equal(t, float64(ballotID), response["ballot_id"])
		assert.Equal(t, float64(0), response["total_voters"])
		results := response["results"].([]interface{})
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

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(1).
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots/1/results", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Results DB Error for Aggregation Query", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		testSetup.Mock.ExpectQuery(getBallotResultsSQL).
			WithArgs(1).
			WillReturnError(errors.New("aggregation error"))

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots/1/results", nil)
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

		testSetup.Mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		testSetup.Mock.ExpectQuery(getBallotResultsSQL).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("not-an-int"))

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots/1/results", nil)
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

		req, err := CreateAuthenticatedRawBodyRequest("POST", "/api/v1/ballots/1/vote", "not-valid-json{", 1, "test@example.com")
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})
}
