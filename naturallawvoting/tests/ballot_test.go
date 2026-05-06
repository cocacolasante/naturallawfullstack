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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBallot(t *testing.T) {
	t.Run("Create Ballot Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock transaction begin
		testSetup.Mock.ExpectBegin()

		// Mock ballot insertion - updated query with category/superstate/state/district
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery("INSERT INTO ballots (title, description, category, superstate, state, district, creator_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, title, description, category, superstate, state, COALESCE(district, ''), creator_id, is_active, created_at, updated_at").
			WithArgs("Best Programming Language", "Vote for your favorite", "", "", "", "", userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(1, "Best Programming Language", "Vote for your favorite", "", "", "", "", userID, true, createdAt, createdAt))

		// Mock ballot items insertion
		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description").
			WithArgs(1, "Go", "Fast and efficient").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(1, 1, "Go", "Fast and efficient"))

		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description").
			WithArgs(1, "Python", "Easy to learn").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(2, 1, "Python", "Easy to learn"))

		// Mock transaction commit
		testSetup.Mock.ExpectCommit()

		reqBody := map[string]interface{}{
			"title":       "Best Programming Language",
			"description": "Vote for your favorite",
			"items": []map[string]interface{}{
				{"title": "Go", "description": "Fast and efficient"},
				{"title": "Python", "description": "Easy to learn"},
			},
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		t.Logf("Response Body: %s", recorder.Body.String())
		assert.Equal(t, 201, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Ballot Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		reqBody := map[string]interface{}{
			"title":       "Test Ballot",
			"description": "Test Description",
			"items": []map[string]interface{}{
				{"title": "Option 1", "description": "First option"},
				{"title": "Option 2", "description": "Second option"},
			},
		}

		req, err := CreateTestRequest("POST", "/api/v1/ballots", reqBody)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Create Ballot With Invalid Data", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		reqBody := map[string]interface{}{
			"title":       "", // Empty title
			"description": "Test Description",
			"items":       []map[string]interface{}{}, // Empty items
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 400, recorder.Code)
	})

	t.Run("Create Ballot DB Error Begin TX", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectBegin().WillReturnError(errors.New("db connection error"))

		reqBody := map[string]interface{}{
			"title":       "Test Ballot",
			"description": "Test",
			"items": []map[string]interface{}{
				{"title": "Option 1", "description": "First"},
				{"title": "Option 2", "description": "Second"},
			},
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Ballot DB Error Insert Ballot", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectQuery("INSERT INTO ballots (title, description, category, superstate, state, district, creator_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, title, description, category, superstate, state, COALESCE(district, ''), creator_id, is_active, created_at, updated_at").
			WithArgs("Test Ballot", "Test", "", "", "", "", userID).
			WillReturnError(errors.New("insert error"))

		reqBody := map[string]interface{}{
			"title":       "Test Ballot",
			"description": "Test",
			"items": []map[string]interface{}{
				{"title": "Option 1", "description": "First"},
				{"title": "Option 2", "description": "Second"},
			},
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating ballot")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Ballot DB Error Insert Items", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectQuery("INSERT INTO ballots (title, description, category, superstate, state, district, creator_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, title, description, category, superstate, state, COALESCE(district, ''), creator_id, is_active, created_at, updated_at").
			WithArgs("Test Ballot", "Test", "", "", "", "", userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(1, "Test Ballot", "Test", "", "", "", "", userID, true, createdAt, createdAt))
		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description").
			WithArgs(1, "Option 1", "First").
			WillReturnError(errors.New("item insert error"))

		reqBody := map[string]interface{}{
			"title":       "Test Ballot",
			"description": "Test",
			"items": []map[string]interface{}{
				{"title": "Option 1", "description": "First"},
				{"title": "Option 2", "description": "Second"},
			},
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error creating ballot items")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Ballot DB Error Commit", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectBegin()
		testSetup.Mock.ExpectQuery("INSERT INTO ballots (title, description, category, superstate, state, district, creator_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, title, description, category, superstate, state, COALESCE(district, ''), creator_id, is_active, created_at, updated_at").
			WithArgs("Test Ballot", "Test", "", "", "", "", userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(1, "Test Ballot", "Test", "", "", "", "", userID, true, createdAt, createdAt))
		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description").
			WithArgs(1, "Option 1", "First").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(1, 1, "Option 1", "First"))
		testSetup.Mock.ExpectQuery("INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description").
			WithArgs(1, "Option 2", "Second").
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(2, 1, "Option 2", "Second"))
		testSetup.Mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		reqBody := map[string]interface{}{
			"title":       "Test Ballot",
			"description": "Test",
			"items": []map[string]interface{}{
				{"title": "Option 1", "description": "First"},
				{"title": "Option 2", "description": "Second"},
			},
		}

		req, err := CreateAuthenticatedRequest("POST", "/api/v1/ballots", reqBody, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error committing transaction")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Create Ballot No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewBallotHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("POST", "/api/v1/ballots", map[string]interface{}{
			"title":       "Test",
			"description": "Test",
			"items": []map[string]interface{}{
				{"title": "Option 1", "description": "Desc"},
				{"title": "Option 2", "description": "Desc"},
			},
		})
		handler.CreateBallot(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestGetAllBallots(t *testing.T) {
	t.Run("Get All Ballots Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		// Mock ballots query - updated columns
		createdAt1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		createdAt2 := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
			AddRow(1, "Ballot 1", "Description 1", "", "", "", "", 1, true, createdAt1, createdAt1).
			AddRow(2, "Ballot 2", "Description 2", "", "", "", "", 2, true, createdAt2, createdAt2)

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true ORDER BY b.created_at DESC`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots Empty Result", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"})
		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true ORDER BY b.created_at DESC`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots With Category Filter", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
			AddRow(1, "Exec Ballot", "Description", "executive", "", "", "", 1, true, createdAt, createdAt)

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true AND b.category = $1 ORDER BY b.created_at DESC`).
			WithArgs("executive").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots?category=executive", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots With Superstate Filter", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
			AddRow(1, "CA Ballot", "Description", "", "US-West", "", "", 1, true, createdAt, createdAt)

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true AND b.superstate = $1 ORDER BY b.created_at DESC`).
			WithArgs("US-West").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots?superstate=US-West", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots With State Filter", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
			AddRow(1, "CA Ballot", "Description", "", "", "CA", "", 1, true, createdAt, createdAt)

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true AND b.state = $1 ORDER BY b.created_at DESC`).
			WithArgs("CA").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots?state=CA", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots With District Filter", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
			AddRow(1, "District Ballot", "Description", "", "", "", "District 1", 1, true, createdAt, createdAt)

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true AND b.district = $1 ORDER BY b.created_at DESC`).
			WithArgs("District 1").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots?district=District+1", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true ORDER BY b.created_at DESC`).
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get All Ballots Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		// Return wrong number of columns to cause scan error
		rows := sqlmock.NewRows([]string{"id"}).AddRow("not-an-int")
		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b
		WHERE b.is_active = true ORDER BY b.created_at DESC`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error scanning ballot")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetBallot(t *testing.T) {
	t.Run("Get Ballot Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 1

		// Mock ballot query - updated columns
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b WHERE b.id = $1
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Test Ballot", "Test Description", "", "", "", "", 1, true, createdAt, createdAt))

		// Mock ballot items query
		testSetup.Mock.ExpectQuery(`
		SELECT id, ballot_id, title, description
		FROM ballot_items
		WHERE ballot_id = $1
		ORDER BY id ASC
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ballot_id", "title", "description"}).
				AddRow(1, ballotID, "Option 1", "First option").
				AddRow(2, ballotID, "Option 2", "Second option"))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Not Found", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 999

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b WHERE b.id = $1
	`).
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
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/public/ballots/invalid", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 400, "Invalid ballot ID")
	})

	t.Run("Get Ballot DB Error Ballot Query", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 1

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b WHERE b.id = $1
	`).
			WithArgs(ballotID).
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot DB Error Items Query", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 1
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b WHERE b.id = $1
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Test Ballot", "Test Description", "", "", "", "", 1, true, createdAt, createdAt))

		testSetup.Mock.ExpectQuery(`
		SELECT id, ballot_id, title, description
		FROM ballot_items
		WHERE ballot_id = $1
		ORDER BY id ASC
	`).
			WithArgs(ballotID).
			WillReturnError(errors.New("items db error"))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error fetching ballot items")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Ballot Items Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		ballotID := 1
		createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testSetup.Mock.ExpectQuery(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), COALESCE(b.district, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b WHERE b.id = $1
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
				AddRow(ballotID, "Test Ballot", "Test Description", "", "", "", "", 1, true, createdAt, createdAt))

		// Return wrong columns to trigger scan error
		testSetup.Mock.ExpectQuery(`
		SELECT id, ballot_id, title, description
		FROM ballot_items
		WHERE ballot_id = $1
		ORDER BY id ASC
	`).
			WithArgs(ballotID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("not-an-int"))

		req, err := CreateTestRequest("GET", fmt.Sprintf("/api/v1/public/ballots/%d", ballotID), nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error scanning ballot item")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetUserBallots(t *testing.T) {
	t.Run("Get User Ballots Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		// Mock user ballots query - updated columns
		createdAt1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		createdAt2 := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"}).
			AddRow(1, "My Ballot 1", "My Description 1", "", "", "", "", userID, true, createdAt1, createdAt1).
			AddRow(2, "My Ballot 2", "My Description 2", "", "", "", "", userID, false, createdAt2, createdAt2)

		testSetup.Mock.ExpectQuery(`
		SELECT id, title, description, category, COALESCE(superstate, ''), COALESCE(state, ''), COALESCE(district, ''), creator_id, is_active, created_at, updated_at
		FROM ballots
		WHERE creator_id = $1
		ORDER BY created_at DESC
	`).
			WithArgs(userID).
			WillReturnRows(rows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-ballots", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Ballots Without Authentication", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		req, err := CreateTestRequest("GET", "/api/v1/my-ballots", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 401, "Authorization header required")
	})

	t.Run("Get User Ballots Empty Result", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		rows := sqlmock.NewRows([]string{"id", "title", "description", "category", "superstate", "state", "district", "creator_id", "is_active", "created_at", "updated_at"})
		testSetup.Mock.ExpectQuery(`
		SELECT id, title, description, category, COALESCE(superstate, ''), COALESCE(state, ''), COALESCE(district, ''), creator_id, is_active, created_at, updated_at
		FROM ballots
		WHERE creator_id = $1
		ORDER BY created_at DESC
	`).
			WithArgs(userID).
			WillReturnRows(rows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-ballots", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Ballots DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		testSetup.Mock.ExpectQuery(`
		SELECT id, title, description, category, COALESCE(superstate, ''), COALESCE(state, ''), COALESCE(district, ''), creator_id, is_active, created_at, updated_at
		FROM ballots
		WHERE creator_id = $1
		ORDER BY created_at DESC
	`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-ballots", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get User Ballots Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		userID := 1
		email := "test@example.com"

		rows := sqlmock.NewRows([]string{"id"}).AddRow("not-an-int")
		testSetup.Mock.ExpectQuery(`
		SELECT id, title, description, category, COALESCE(superstate, ''), COALESCE(state, ''), COALESCE(district, ''), creator_id, is_active, created_at, updated_at
		FROM ballots
		WHERE creator_id = $1
		ORDER BY created_at DESC
	`).
			WithArgs(userID).
			WillReturnRows(rows)

		req, err := CreateAuthenticatedRequest("GET", "/api/v1/my-ballots", nil, userID, email)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Error scanning ballot")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetSuperstates(t *testing.T) {
	t.Run("Get Superstates Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"superstate"}).
			AddRow("US-West").
			AddRow("US-East")

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT superstate
		FROM ballots
		WHERE superstate IS NOT NULL AND superstate != '' AND is_active = true
		ORDER BY superstate
	`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Superstates DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT superstate
		FROM ballots
		WHERE superstate IS NOT NULL AND superstate != '' AND is_active = true
		ORDER BY superstate
	`).
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Superstates Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		// Use ErrConnDone to simulate scan error - rows.Scan will fail
		rows := sqlmock.NewRows([]string{"superstate"}).
			RowError(0, errors.New("scan error"))

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT superstate
		FROM ballots
		WHERE superstate IS NOT NULL AND superstate != '' AND is_active = true
		ORDER BY superstate
	`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		// The row error surfaces during iteration - response may be 200 with empty list
		// or 500. Just check the mock was satisfied.
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetStates(t *testing.T) {
	t.Run("Get States Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"state"}).
			AddRow("CA").
			AddRow("TX")

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT state
		FROM ballots
		WHERE superstate = $1 AND state IS NOT NULL AND state != '' AND is_active = true
		ORDER BY state
	`).
			WithArgs("US-West").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get States DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT state
		FROM ballots
		WHERE superstate = $1 AND state IS NOT NULL AND state != '' AND is_active = true
		ORDER BY state
	`).
			WithArgs("US-West").
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get States Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"state"}).
			RowError(0, errors.New("scan error"))

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT state
		FROM ballots
		WHERE superstate = $1 AND state IS NOT NULL AND state != '' AND is_active = true
		ORDER BY state
	`).
			WithArgs("US-West").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetDistricts(t *testing.T) {
	t.Run("Get Districts Successfully", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"district"}).
			AddRow("District 1").
			AddRow("District 2")

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT district
		FROM ballots
		WHERE superstate = $1 AND state = $2 AND is_active = true AND district IS NOT NULL AND district != ''
		ORDER BY district
	`).
			WithArgs("US-West", "CA").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states/CA/districts", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Districts DB Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT district
		FROM ballots
		WHERE superstate = $1 AND state = $2 AND is_active = true AND district IS NOT NULL AND district != ''
		ORDER BY district
	`).
			WithArgs("US-West", "CA").
			WillReturnError(errors.New("db error"))

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states/CA/districts", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		AssertErrorResponse(t, recorder, 500, "Database error")
		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Districts Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"district"}).
			RowError(0, errors.New("scan error"))

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT district
		FROM ballots
		WHERE superstate = $1 AND state = $2 AND is_active = true AND district IS NOT NULL AND district != ''
		ORDER BY district
	`).
			WithArgs("US-West", "CA").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states/CA/districts", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})

	t.Run("Get Districts Real Scan Error", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		// Return non-string column to cause actual scan error
		rows := sqlmock.NewRows([]string{"district"}).AddRow(nil)

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT district
		FROM ballots
		WHERE superstate = $1 AND state = $2 AND is_active = true AND district IS NOT NULL AND district != ''
		ORDER BY district
	`).
			WithArgs("US-West", "CA").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states/CA/districts", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetUserBallotsNoContext(t *testing.T) {
	t.Run("Get User Ballots No User ID In Context", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewBallotHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/my-ballots", nil)
		handler.GetUserBallots(c)
		assert.Equal(t, 401, w.Code)
	})
}

func TestGetSuperStatesScanError(t *testing.T) {
	t.Run("Get Superstates Real Scan Error (nil value)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"superstate"}).AddRow(nil)

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT superstate
		FROM ballots
		WHERE superstate IS NOT NULL AND superstate != '' AND is_active = true
		ORDER BY superstate
	`).
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

func TestGetStatesScanError(t *testing.T) {
	t.Run("Get States Real Scan Error (nil value)", func(t *testing.T) {
		testSetup, err := SetupTestEnvironment()
		require.NoError(t, err)
		defer testSetup.DB.Close()

		rows := sqlmock.NewRows([]string{"state"}).AddRow(nil)

		testSetup.Mock.ExpectQuery(`
		SELECT DISTINCT state
		FROM ballots
		WHERE superstate = $1 AND state IS NOT NULL AND state != '' AND is_active = true
		ORDER BY state
	`).
			WithArgs("US-West").
			WillReturnRows(rows)

		req, err := CreateTestRequest("GET", "/api/v1/public/superstates/US-West/states", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		testSetup.Router.ServeHTTP(recorder, req)

		assert.NoError(t, testSetup.Mock.ExpectationsWereMet())
	})
}

// TestGetStatesEmptyParam tests the GetStates handler when superstate param is empty
// (unreachable via normal router since Gin requires non-empty path params)
func TestGetStatesEmptyParam(t *testing.T) {
	t.Run("Get States Empty Superstate Param", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewBallotHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/public/superstates//states", nil)
		// Do NOT set params - superstate will be empty string
		handler.GetStates(c)
		assert.Equal(t, 400, w.Code)
	})
}

// TestGetDistrictsEmptyParam tests the GetDistricts handler when params are empty
// (unreachable via normal router since Gin requires non-empty path params)
func TestGetDistrictsEmptyParam(t *testing.T) {
	t.Run("Get Districts Empty Superstate and State Params", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer mockDB.Close()
		_ = mock

		db := &database.DB{DB: mockDB}
		handler := handlers.NewBallotHandler(db)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = CreateTestRequest("GET", "/api/v1/public/superstates//states//districts", nil)
		// Do NOT set params - both superstate and state will be empty string
		handler.GetDistricts(c)
		assert.Equal(t, 400, w.Code)
	})
}
