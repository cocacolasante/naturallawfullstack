package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"voting-api/database"
	"voting-api/models"

	"github.com/gin-gonic/gin"
)

type BallotHandler struct {
	db *database.DB
}

func NewBallotHandler(db *database.DB) *BallotHandler {
	return &BallotHandler{db: db}
}

func (h *BallotHandler) CreateBallot(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateBallotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Insert ballot
	var ballot models.Ballot
	err = tx.QueryRow(
		"INSERT INTO ballots (title, description, category, superstate, state, creator_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, title, description, category, superstate, state, creator_id, is_active, created_at, updated_at",
		req.Title, req.Description, req.Category, req.Superstate, req.State, userID,
	).Scan(&ballot.ID, &ballot.Title, &ballot.Description, &ballot.Category, &ballot.Superstate, &ballot.State, &ballot.CreatorID, &ballot.IsActive, &ballot.CreatedAt, &ballot.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating ballot"})
		return
	}

	// Insert ballot items
	var items []models.BallotItem
	for _, item := range req.Items {
		var ballotItem models.BallotItem
		err = tx.QueryRow(
			"INSERT INTO ballot_items (ballot_id, title, description) VALUES ($1, $2, $3) RETURNING id, ballot_id, title, description, vote_count",
			ballot.ID, item.Title, item.Description,
		).Scan(&ballotItem.ID, &ballotItem.BallotID, &ballotItem.Title, &ballotItem.Description, &ballotItem.VoteCount)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating ballot items"})
			return
		}
		items = append(items, ballotItem)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error committing transaction"})
		return
	}

	ballot.Items = items
	c.JSON(http.StatusCreated, ballot)
}

func (h *BallotHandler) GetAllBallots(c *gin.Context) {
	category := c.Query("category")
	superstate := c.Query("superstate")
	state := c.Query("state")

	query := `
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), b.creator_id, b.is_active, b.created_at, b.updated_at,
		       u.username as creator_username
		FROM ballots b
		JOIN users u ON b.creator_id = u.id
		WHERE b.is_active = true`

	var args []interface{}
	argIndex := 1

	if category != "" {
		query += ` AND b.category = $` + strconv.Itoa(argIndex)
		args = append(args, category)
		argIndex++
	}

	if superstate != "" {
		query += ` AND b.superstate = $` + strconv.Itoa(argIndex)
		args = append(args, superstate)
		argIndex++
	}

	if state != "" {
		query += ` AND b.state = $` + strconv.Itoa(argIndex)
		args = append(args, state)
		argIndex++
	}

	query += ` ORDER BY b.created_at DESC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var ballots []models.Ballot
	for rows.Next() {
		var ballot models.Ballot
		var creatorUsername string
		err := rows.Scan(
			&ballot.ID, &ballot.Title, &ballot.Description, &ballot.Category, &ballot.Superstate, &ballot.State, &ballot.CreatorID,
			&ballot.IsActive, &ballot.CreatedAt, &ballot.UpdatedAt, &creatorUsername,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning ballot"})
			return
		}
		ballots = append(ballots, ballot)
	}

	c.JSON(http.StatusOK, ballots)
}

func (h *BallotHandler) GetBallot(c *gin.Context) {
	ballotIDStr := c.Param("id")
	ballotID, err := strconv.Atoi(ballotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ballot ID"})
		return
	}

	// Get ballot
	var ballot models.Ballot
	err = h.db.QueryRow(`
		SELECT b.id, b.title, b.description, b.category, COALESCE(b.superstate, ''), COALESCE(b.state, ''), b.creator_id, b.is_active, b.created_at, b.updated_at
		FROM ballots b WHERE b.id = $1
	`, ballotID).Scan(
		&ballot.ID, &ballot.Title, &ballot.Description, &ballot.Category, &ballot.Superstate, &ballot.State, &ballot.CreatorID,
		&ballot.IsActive, &ballot.CreatedAt, &ballot.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ballot not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get ballot items with vote counts
	rows, err := h.db.Query(`
		SELECT id, ballot_id, title, description, vote_count
		FROM ballot_items 
		WHERE ballot_id = $1 
		ORDER BY id ASC
	`, ballotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching ballot items"})
		return
	}
	defer rows.Close()

	var items []models.BallotItem
	for rows.Next() {
		var item models.BallotItem
		err := rows.Scan(&item.ID, &item.BallotID, &item.Title, &item.Description, &item.VoteCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning ballot item"})
			return
		}
		items = append(items, item)
	}

	ballot.Items = items
	c.JSON(http.StatusOK, ballot)
}

func (h *BallotHandler) GetUserBallots(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := h.db.Query(`
		SELECT id, title, description, category, COALESCE(superstate, ''), COALESCE(state, ''), creator_id, is_active, created_at, updated_at
		FROM ballots
		WHERE creator_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var ballots []models.Ballot
	for rows.Next() {
		var ballot models.Ballot
		err := rows.Scan(
			&ballot.ID, &ballot.Title, &ballot.Description, &ballot.Category, &ballot.Superstate, &ballot.State, &ballot.CreatorID,
			&ballot.IsActive, &ballot.CreatedAt, &ballot.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning ballot"})
			return
		}
		ballots = append(ballots, ballot)
	}

	c.JSON(http.StatusOK, ballots)
}

// GetSuperstates returns a list of all superstates that have ballots
func (h *BallotHandler) GetSuperstates(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT DISTINCT superstate
		FROM ballots
		WHERE superstate IS NOT NULL AND superstate != '' AND is_active = true
		ORDER BY superstate
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var superstates []string
	for rows.Next() {
		var superstate string
		if err := rows.Scan(&superstate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning superstate"})
			return
		}
		superstates = append(superstates, superstate)
	}

	c.JSON(http.StatusOK, gin.H{"superstates": superstates})
}

// GetStates returns a list of all states within a superstate that have ballots
func (h *BallotHandler) GetStates(c *gin.Context) {
	superstate := c.Param("superstate")
	if superstate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Superstate parameter required"})
		return
	}

	rows, err := h.db.Query(`
		SELECT DISTINCT state
		FROM ballots
		WHERE superstate = $1 AND state IS NOT NULL AND state != '' AND is_active = true
		ORDER BY state
	`, superstate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var states []string
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning state"})
			return
		}
		states = append(states, state)
	}

	c.JSON(http.StatusOK, gin.H{"superstate": superstate, "states": states})
}