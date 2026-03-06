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
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Helpers
// ============================================================================

var trustScoreQuery = `SELECT COALESCE(AVG(score), 0), COUNT(*) FROM trust_votes WHERE subject_id = $1`

var visibilitySelectQuery = `
		SELECT user_id, info_threshold, address_threshold, political_threshold,
		       religious_threshold, race_ethnicity_threshold, economic_threshold
		FROM profile_visibility WHERE user_id = $1`

var visibilityInsertQuery = `
		INSERT INTO profile_visibility (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`

var submitTrustVoteQuery = `
		INSERT INTO trust_votes (voter_id, subject_id, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (voter_id, subject_id) DO UPDATE
		  SET score = EXCLUDED.score, updated_at = NOW()
		RETURNING id, voter_id, subject_id, score, created_at, updated_at`

var getMyTrustVoteQuery = `
		SELECT id, voter_id, subject_id, score, created_at, updated_at
		FROM trust_votes WHERE voter_id = $1 AND subject_id = $2`

var searchUsersQuery = `
		SELECT u.id, u.username,
		       COALESCE(AVG(tv.score), 0) as trust_score,
		       COUNT(tv.id) as vote_count
		FROM users u
		LEFT JOIN trust_votes tv ON tv.subject_id = u.id
		WHERE u.username ILIKE '%' || $1 || '%'
		GROUP BY u.id
		ORDER BY u.username
		LIMIT 20`

// Section queries used in GetUserProfileSections
var sectionInfoQuery = `
			SELECT user_id, email, full_name, birthday, gender, mothers_maiden_name,
			       phone_number, additional_emails, created_at, updated_at
			FROM user_profiles WHERE email = $1`

var sectionAddressQuery = `
		SELECT user_id, street_number, street_name, address_line_2, city, state,
		       zip_code, created_at, updated_at
		FROM user_addresses WHERE user_id = $1`

var sectionPoliticalQuery = `
		SELECT user_id, party_affiliation, created_at, updated_at
		FROM user_political_affiliations WHERE user_id = $1`

var sectionReligiousQuery = `
		SELECT user_id, religion, supporting_religion, religious_services_types, created_at, updated_at
		FROM user_religious_affiliations WHERE user_id = $1`

var sectionRaceQuery = `
		SELECT user_id, race, created_at, updated_at
		FROM user_race_ethnicity WHERE user_id = $1`

var sectionEconomicQuery = `
		SELECT user_id, for_current_political_structure, for_capitalism, for_laws,
		       goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		       additional_text, created_at, updated_at
		FROM economic_info WHERE user_id = $1`

func visibilityColumns() []string {
	return []string{"user_id", "info_threshold", "address_threshold", "political_threshold", "religious_threshold", "race_ethnicity_threshold", "economic_threshold"}
}

func trustVoteColumns() []string {
	return []string{"id", "voter_id", "subject_id", "score", "created_at", "updated_at"}
}

// mockAllSectionQueriesNoData mocks all 7 section-fetch queries (email + 6 sections) to return ErrNoRows.
func mockAllSectionQueriesNoData(ts *TestSetup, subjectID int) {
	ts.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
		WithArgs(subjectID).
		WillReturnError(sql.ErrNoRows)
	ts.Mock.ExpectQuery(sectionAddressQuery).
		WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
	ts.Mock.ExpectQuery(sectionPoliticalQuery).
		WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
	ts.Mock.ExpectQuery(sectionReligiousQuery).
		WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
	ts.Mock.ExpectQuery(sectionRaceQuery).
		WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
	ts.Mock.ExpectQuery(sectionEconomicQuery).
		WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
}

// mockAllSectionQueriesWithData mocks all 7 section-fetch queries to return real data.
func mockAllSectionQueriesWithData(ts *TestSetup, subjectID int, email string, now time.Time) {
	birthday := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)

	ts.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
		WithArgs(subjectID).
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow(email))

	ts.Mock.ExpectQuery(sectionInfoQuery).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "email", "full_name", "birthday", "gender",
			"mothers_maiden_name", "phone_number", "additional_emails", "created_at", "updated_at",
		}).AddRow(subjectID, email, "Bob Jones", birthday, "Male", "Smith", "555-9999",
			pq.Array([]string{}), now, now))

	ts.Mock.ExpectQuery(sectionAddressQuery).
		WithArgs(subjectID).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "street_number", "street_name", "address_line_2",
			"city", "state", "zip_code", "created_at", "updated_at",
		}).AddRow(subjectID, "123", "Main St", "", "Springfield", "IL", "62701", now, now))

	ts.Mock.ExpectQuery(sectionPoliticalQuery).
		WithArgs(subjectID).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "party_affiliation", "created_at", "updated_at",
		}).AddRow(subjectID, "Independent", now, now))

	ts.Mock.ExpectQuery(sectionReligiousQuery).
		WithArgs(subjectID).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "religion", "supporting_religion", "religious_services_types", "created_at", "updated_at",
		}).AddRow(subjectID, "Christianity", 7, pq.Array([]string{"Sunday Service"}), now, now))

	ts.Mock.ExpectQuery(sectionRaceQuery).
		WithArgs(subjectID).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "race", "created_at", "updated_at",
		}).AddRow(subjectID, pq.Array([]string{"White"}), now, now))

	ts.Mock.ExpectQuery(sectionEconomicQuery).
		WithArgs(subjectID).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "for_current_political_structure", "for_capitalism", "for_laws",
			"goods_services", "affiliations", "support_of_alt_econ", "support_alt_comm",
			"additional_text", "created_at", "updated_at",
		}).AddRow(subjectID, "yes", "yes", "yes",
			pq.Array([]string{}), pq.Array([]string{}), "no", "no", "", now, now))
}

// ============================================================================
// TestSubmitTrustVote
// ============================================================================

func TestSubmitTrustVote(t *testing.T) {
	t.Run("Submit Trust Vote Successfully", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		voterID := 1
		subjectID := 2
		email := "voter@example.com"
		now := time.Now()

		ts.Mock.ExpectQuery(submitTrustVoteQuery).
			WithArgs(voterID, subjectID, 80).
			WillReturnRows(sqlmock.NewRows(trustVoteColumns()).
				AddRow(10, voterID, subjectID, 80, now, now))

		body := models.TrustVoteRequest{Score: 80}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/users/%d/trust", subjectID), body, voterID, email)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)

		assert.Equal(t, 200, rec.Code)
		var vote models.TrustVote
		require.NoError(t, parseJSONResponse(rec, &vote))
		assert.Equal(t, voterID, vote.VoterID)
		assert.Equal(t, subjectID, vote.SubjectID)
		assert.Equal(t, 80, vote.Score)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Cannot Rate Yourself", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		body := models.TrustVoteRequest{Score: 50}
		req, err := CreateAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/users/%d/trust", userID), body, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 400, "Cannot rate yourself")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		body := models.TrustVoteRequest{Score: 50}
		req, err := CreateTestRequest("POST", "/api/v1/users/2/trust", body)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 401, "Authorization header required")
	})

	t.Run("Invalid User ID In Path", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		body := models.TrustVoteRequest{Score: 50}
		req, err := CreateAuthenticatedRequest("POST", "/api/v1/users/abc/trust", body, 1, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 400, "Invalid user_id")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Invalid Score Too Low", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		body := map[string]interface{}{"score": 0}
		req, err := CreateAuthenticatedRequest("POST", "/api/v1/users/2/trust", body, 1, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 400, rec.Code)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Invalid Score Too High", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		body := map[string]interface{}{"score": 101}
		req, err := CreateAuthenticatedRequest("POST", "/api/v1/users/2/trust", body, 1, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 400, rec.Code)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Database Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(submitTrustVoteQuery).
			WithArgs(1, 2, 70).
			WillReturnError(errors.New("db error"))

		body := models.TrustVoteRequest{Score: 70}
		req, err := CreateAuthenticatedRequest("POST", "/api/v1/users/2/trust", body, 1, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// TestGetTrustScore
// ============================================================================

func TestGetTrustScore(t *testing.T) {
	t.Run("Returns Score With Votes", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(72.5, 8))

		req, err := CreateTestRequest("GET", "/api/v1/public/users/2/trust-score", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var ts2 models.TrustScore
		require.NoError(t, parseJSONResponse(rec, &ts2))
		assert.Equal(t, 2, ts2.UserID)
		assert.Equal(t, 72.5, ts2.Score)
		assert.Equal(t, 8, ts2.VoteCount)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Returns Zero When No Votes", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(3).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(0, 0))

		req, err := CreateTestRequest("GET", "/api/v1/public/users/3/trust-score", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var score models.TrustScore
		require.NoError(t, parseJSONResponse(rec, &score))
		assert.Equal(t, 0.0, score.Score)
		assert.Equal(t, 0, score.VoteCount)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Invalid User ID", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/public/users/notanid/trust-score", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 400, "Invalid user_id")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// TestGetMyTrustVote
// ============================================================================

func TestGetMyTrustVote(t *testing.T) {
	t.Run("Get My Trust Vote Successfully", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		voterID := 1
		subjectID := 2
		now := time.Now()

		ts.Mock.ExpectQuery(getMyTrustVoteQuery).
			WithArgs(voterID, subjectID).
			WillReturnRows(sqlmock.NewRows(trustVoteColumns()).
				AddRow(5, voterID, subjectID, 65, now, now))

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/users/%d/my-trust-vote", subjectID), nil, voterID, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var vote models.TrustVote
		require.NoError(t, parseJSONResponse(rec, &vote))
		assert.Equal(t, 65, vote.Score)
		assert.Equal(t, voterID, vote.VoterID)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Vote Not Found", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(getMyTrustVoteQuery).
			WithArgs(1, 2).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/users/2/my-trust-vote", nil, 1, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 404, "No trust vote found")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Database Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(getMyTrustVoteQuery).
			WithArgs(1, 2).
			WillReturnError(errors.New("connection error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/users/2/my-trust-vote", nil, 1, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/users/2/my-trust-vote", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 401, "Authorization header required")
	})

	t.Run("Invalid User ID In Path", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/users/notanid/my-trust-vote", nil, 1, "voter@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 400, "Invalid user_id")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// TestGetMyTrustScore
// ============================================================================

func TestGetMyTrustScore(t *testing.T) {
	t.Run("Returns My Trust Score", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(88.0, 4))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-trust-score", nil, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var score models.TrustScore
		require.NoError(t, parseJSONResponse(rec, &score))
		assert.Equal(t, userID, score.UserID)
		assert.Equal(t, 88.0, score.Score)
		assert.Equal(t, 4, score.VoteCount)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Returns Zero When CalcTrustScore DB Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		// calcTrustScore returns (0, 0) on any DB error
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(userID).
			WillReturnError(errors.New("connection lost"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-trust-score", nil, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var score models.TrustScore
		require.NoError(t, parseJSONResponse(rec, &score))
		assert.Equal(t, 0.0, score.Score)
		assert.Equal(t, 0, score.VoteCount)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/my-trust-score", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 401, "Authorization header required")
	})
}

// ============================================================================
// TestGetProfileVisibility
// ============================================================================

func TestGetProfileVisibility(t *testing.T) {
	t.Run("Returns Existing Row", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows(visibilityColumns()).
				AddRow(userID, 10, 20, 30, 40, 50, 60))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/visibility", nil, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var pv models.ProfileVisibility
		require.NoError(t, parseJSONResponse(rec, &pv))
		assert.Equal(t, 10, pv.InfoThreshold)
		assert.Equal(t, 20, pv.AddressThreshold)
		assert.Equal(t, 30, pv.PoliticalThreshold)
		assert.Equal(t, 40, pv.ReligiousThreshold)
		assert.Equal(t, 50, pv.RaceEthnicityThreshold)
		assert.Equal(t, 60, pv.EconomicThreshold)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Returns Defaults When No Row Exists", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/visibility", nil, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var pv models.ProfileVisibility
		require.NoError(t, parseJSONResponse(rec, &pv))
		assert.Equal(t, userID, pv.UserID)
		assert.Equal(t, 0, pv.InfoThreshold)
		assert.Equal(t, 0, pv.AddressThreshold)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Database Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).
			WillReturnError(errors.New("query failed"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/profile/visibility", nil, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/profile/visibility", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 401, "Authorization header required")
	})
}

// ============================================================================
// TestUpdateProfileVisibility
// ============================================================================

func TestUpdateProfileVisibility(t *testing.T) {
	t.Run("Update All Thresholds", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1

		ts.Mock.ExpectExec(visibilityInsertQuery).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		ts.Mock.ExpectExec("UPDATE profile_visibility SET info_threshold = $1, address_threshold = $2, political_threshold = $3, religious_threshold = $4, race_ethnicity_threshold = $5, economic_threshold = $6 WHERE user_id = $7").
			WithArgs(10, 20, 30, 40, 50, 60, userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows(visibilityColumns()).
				AddRow(userID, 10, 20, 30, 40, 50, 60))

		body := map[string]interface{}{
			"info_threshold": 10, "address_threshold": 20, "political_threshold": 30,
			"religious_threshold": 40, "race_ethnicity_threshold": 50, "economic_threshold": 60,
		}
		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/visibility", body, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var pv models.ProfileVisibility
		require.NoError(t, parseJSONResponse(rec, &pv))
		assert.Equal(t, 10, pv.InfoThreshold)
		assert.Equal(t, 60, pv.EconomicThreshold)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Update Single Threshold (Address)", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1

		ts.Mock.ExpectExec(visibilityInsertQuery).
			WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))

		ts.Mock.ExpectExec("UPDATE profile_visibility SET address_threshold = $1 WHERE user_id = $2").
			WithArgs(75, userID).WillReturnResult(sqlmock.NewResult(0, 1))

		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows(visibilityColumns()).
				AddRow(userID, 0, 75, 0, 0, 0, 0))

		body := map[string]interface{}{"address_threshold": 75}
		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/visibility", body, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var pv models.ProfileVisibility
		require.NoError(t, parseJSONResponse(rec, &pv))
		assert.Equal(t, 75, pv.AddressThreshold)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("No Fields Provided - Skip Update", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1

		ts.Mock.ExpectExec(visibilityInsertQuery).
			WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))

		// No UPDATE expected — setClauses is empty
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows(visibilityColumns()).
				AddRow(userID, 0, 0, 0, 0, 0, 0))

		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/visibility", map[string]interface{}{}, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Insert DB Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		ts.Mock.ExpectExec(visibilityInsertQuery).
			WithArgs(userID).WillReturnError(errors.New("insert failed"))

		body := map[string]interface{}{"info_threshold": 50}
		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/visibility", body, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Update DB Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		ts.Mock.ExpectExec(visibilityInsertQuery).
			WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))

		ts.Mock.ExpectExec("UPDATE profile_visibility SET info_threshold = $1 WHERE user_id = $2").
			WithArgs(50, userID).WillReturnError(errors.New("update failed"))

		body := map[string]interface{}{"info_threshold": 50}
		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/visibility", body, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Final Select DB Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		ts.Mock.ExpectExec(visibilityInsertQuery).
			WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))

		ts.Mock.ExpectExec("UPDATE profile_visibility SET info_threshold = $1 WHERE user_id = $2").
			WithArgs(50, userID).WillReturnResult(sqlmock.NewResult(0, 1))

		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).WillReturnError(errors.New("select failed"))

		body := map[string]interface{}{"info_threshold": 50}
		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/visibility", body, userID, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateTestRequest("PUT", "/api/v1/profile/visibility", map[string]interface{}{"info_threshold": 50})
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 401, "Authorization header required")
	})

	t.Run("Invalid Threshold Value Out Of Range", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		body := map[string]interface{}{"info_threshold": 101}
		req, err := CreateAuthenticatedRequest("PUT", "/api/v1/profile/visibility", body, 1, "me@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 400, rec.Code)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// TestSearchUsers
// ============================================================================

func TestSearchUsers(t *testing.T) {
	t.Run("Returns Matching Users", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(searchUsersQuery).
			WithArgs("alice").
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "trust_score", "vote_count"}).
				AddRow(1, "alice_smith", 80.0, 5).
				AddRow(2, "alice_jones", 65.5, 2))

		req, err := CreateTestRequest("GET", "/api/v1/public/users/search?username=alice", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var users []models.PublicUser
		require.NoError(t, parseJSONResponse(rec, &users))
		assert.Len(t, users, 2)
		assert.Equal(t, "alice_smith", users[0].Username)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Returns Empty Array When No Match", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(searchUsersQuery).
			WithArgs("zzz").
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "trust_score", "vote_count"}))

		req, err := CreateTestRequest("GET", "/api/v1/public/users/search?username=zzz", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var users []models.PublicUser
		require.NoError(t, parseJSONResponse(rec, &users))
		assert.Len(t, users, 0)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Missing Username Query Param", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/public/users/search", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 400, "username query param required")
	})

	t.Run("Database Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery(searchUsersQuery).
			WithArgs("bob").
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", "/api/v1/public/users/search?username=bob", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Scan Error Row Is Skipped", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		// First row: bad ID type causes scan error → continue
		// Second row: valid → included in results
		ts.Mock.ExpectQuery(searchUsersQuery).
			WithArgs("test").
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "trust_score", "vote_count"}).
				AddRow("not-an-int", "bad_user", 50.0, 1).
				AddRow(9, "good_user", 70.0, 3))

		req, err := CreateTestRequest("GET", "/api/v1/public/users/search?username=test", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var users []models.PublicUser
		require.NoError(t, parseJSONResponse(rec, &users))
		// Only the valid row should be in the result
		assert.Len(t, users, 1)
		assert.Equal(t, "good_user", users[0].Username)
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// TestGetUserProfileSections
// ============================================================================

func TestGetUserProfileSections(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/users/2/profile", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 401, "Authorization header required")
	})

	t.Run("Subject Not Found", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery("SELECT username FROM users WHERE id = $1").
			WithArgs(2).WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/users/2/profile", nil, 1, "viewer@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 404, "User not found")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Username DB Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		ts.Mock.ExpectQuery("SELECT username FROM users WHERE id = $1").
			WithArgs(2).WillReturnError(errors.New("connection lost"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/users/2/profile", nil, 1, "viewer@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Visibility DB Error", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		subjectID := 2
		viewerID := 1

		ts.Mock.ExpectQuery("SELECT username FROM users WHERE id = $1").
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("bob"))

		// subject trust score
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(70.0, 3))

		// viewer trust score (not owner)
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(viewerID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(60.0, 2))

		// visibility fails with non-NoRows error
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(subjectID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/users/2/profile", nil, viewerID, "viewer@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		AssertErrorResponse(t, rec, 500, "Database error")
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Owner Views Own Profile - All Sections With Data", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		userID := 1
		email := "owner@example.com"
		now := time.Now()

		// username lookup
		ts.Mock.ExpectQuery("SELECT username FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("alice"))

		// subject trust score (for display)
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(90.0, 6))

		// isOwner = true → no viewer trust score query

		// visibility
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows(visibilityColumns()).
				AddRow(userID, 0, 0, 0, 0, 0, 0))

		// all section data
		mockAllSectionQueriesWithData(ts, userID, email, now)

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/users/%d/profile", userID), nil, userID, email)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var body map[string]interface{}
		require.NoError(t, parseJSONResponse(rec, &body))
		assert.Equal(t, "alice", body["user"].(map[string]interface{})["username"])
		assert.Equal(t, false, body["info"].(map[string]interface{})["locked"])
		assert.Equal(t, false, body["address"].(map[string]interface{})["locked"])
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Non-Owner Sufficient Trust - All Sections Visible", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		viewerID := 1
		subjectID := 2
		email := "subject@example.com"
		now := time.Now()

		ts.Mock.ExpectQuery("SELECT username FROM users WHERE id = $1").
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("bob"))

		// subject trust score
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(75.0, 5))

		// viewer trust score (80 >= all thresholds of 0)
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(viewerID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(80.0, 3))

		// visibility returns ErrNoRows → defaults to all 0 (always visible)
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(subjectID).
			WillReturnError(sql.ErrNoRows)

		// all section data present
		mockAllSectionQueriesWithData(ts, subjectID, email, now)

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/users/%d/profile", subjectID), nil, viewerID, "viewer@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var body map[string]interface{}
		require.NoError(t, parseJSONResponse(rec, &body))
		assert.Equal(t, false, body["info"].(map[string]interface{})["locked"])
		assert.Equal(t, false, body["economic"].(map[string]interface{})["locked"])
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Non-Owner Insufficient Trust - Sections Locked", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		viewerID := 1
		subjectID := 2

		ts.Mock.ExpectQuery("SELECT username FROM users WHERE id = $1").
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("bob"))

		// subject trust score
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(60.0, 4))

		// viewer trust score = 20 (below threshold of 50)
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(viewerID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(20.0, 2))

		// all thresholds = 50
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows(visibilityColumns()).
				AddRow(subjectID, 50, 50, 50, 50, 50, 50))

		// all section queries return no data (email lookup fails first)
		mockAllSectionQueriesNoData(ts, subjectID)

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/users/%d/profile", subjectID), nil, viewerID, "viewer@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var body map[string]interface{}
		require.NoError(t, parseJSONResponse(rec, &body))

		// All sections should be locked
		for _, section := range []string{"info", "address", "political", "religious", "race_ethnicity", "economic"} {
			sec := body[section].(map[string]interface{})
			assert.Equal(t, true, sec["locked"], "section %s should be locked", section)
			assert.Equal(t, float64(50), sec["threshold"])
			assert.Equal(t, float64(20), sec["viewer_score"])
		}
		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})

	t.Run("Sections With Missing Data - Data Null", func(t *testing.T) {
		ts, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer ts.DB.Close()

		viewerID := 1
		subjectID := 2

		ts.Mock.ExpectQuery("SELECT username FROM users WHERE id = $1").
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("bob"))

		// subject trust score
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(0.0, 0))

		// viewer trust score
		ts.Mock.ExpectQuery(trustScoreQuery).
			WithArgs(viewerID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count"}).AddRow(0.0, 0))

		// all thresholds = 0 → all visible
		ts.Mock.ExpectQuery(visibilitySelectQuery).
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows(visibilityColumns()).
				AddRow(subjectID, 0, 0, 0, 0, 0, 0))

		// email found, but profile not found
		ts.Mock.ExpectQuery("SELECT email FROM users WHERE id = $1").
			WithArgs(subjectID).
			WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("bob@example.com"))

		ts.Mock.ExpectQuery(sectionInfoQuery).
			WithArgs("bob@example.com").
			WillReturnError(sql.ErrNoRows)

		// remaining sections return no data
		ts.Mock.ExpectQuery(sectionAddressQuery).
			WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
		ts.Mock.ExpectQuery(sectionPoliticalQuery).
			WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
		ts.Mock.ExpectQuery(sectionReligiousQuery).
			WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
		ts.Mock.ExpectQuery(sectionRaceQuery).
			WithArgs(subjectID).WillReturnError(sql.ErrNoRows)
		ts.Mock.ExpectQuery(sectionEconomicQuery).
			WithArgs(subjectID).WillReturnError(sql.ErrNoRows)

		req, err := CreateAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/users/%d/profile", subjectID), nil, viewerID, "viewer@example.com")
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		ts.Router.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		var body map[string]interface{}
		require.NoError(t, parseJSONResponse(rec, &body))

		// all sections visible (threshold 0) but data is null
		infoSec := body["info"].(map[string]interface{})
		assert.Equal(t, false, infoSec["locked"])
		assert.Nil(t, infoSec["data"])

		addrSec := body["address"].(map[string]interface{})
		assert.Equal(t, false, addrSec["locked"])
		assert.Nil(t, addrSec["data"])

		assert.NoError(t, ts.Mock.ExpectationsWereMet())
	})
}

// ============================================================================
// TestTrustHandlerNoUserIDInContext
// Direct handler tests to cover the defensive !exists branches that are
// unreachable through the router+middleware path.
// ============================================================================

func TestTrustHandlerNoUserIDInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer mockDB.Close()

	db := &database.DB{DB: mockDB}
	h := handlers.NewTrustHandler(db)

	handlerCases := []struct {
		name string
		fn   func(*gin.Context)
	}{
		{"SubmitTrustVote", h.SubmitTrustVote},
		{"GetMyTrustVote", h.GetMyTrustVote},
		{"GetMyTrustScore", h.GetMyTrustScore},
		{"GetProfileVisibility", h.GetProfileVisibility},
		{"UpdateProfileVisibility", h.UpdateProfileVisibility},
		{"GetUserProfileSections", h.GetUserProfileSections},
	}

	for _, tc := range handlerCases {
		tc := tc
		t.Run(tc.name+" returns 401 when user_id missing from context", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			// Deliberately do NOT set user_id in context
			tc.fn(c)
			assert.Equal(t, 401, w.Code)
		})
	}
}

// TestGetUserProfileSectionsInvalidParam covers the bad path-param branch
// that is separate from the !exists branch.
func TestGetUserProfileSectionsInvalidParam(t *testing.T) {
	ts, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer ts.DB.Close()

	req, err := CreateAuthenticatedRequest("GET", "/api/v1/users/notanid/profile", nil, 1, "viewer@example.com")
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	ts.Router.ServeHTTP(rec, req)
	AssertErrorResponse(t, rec, 400, "Invalid user_id")
	assert.NoError(t, ts.Mock.ExpectationsWereMet())
}
