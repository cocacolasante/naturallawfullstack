package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"voting-api/database"
	"voting-api/models"

	"github.com/gin-gonic/gin"
)

type VoteHandler struct {
	db *database.DB
}

func NewVoteHandler(db *database.DB) *VoteHandler {
	return &VoteHandler{db: db}
}

func (h *VoteHandler) Vote(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ballotIDStr := c.Param("ballot_id")
	ballotID, err := strconv.Atoi(ballotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ballot ID"})
		return
	}

	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Support both option_id (from frontend) and ballot_item_id
	ballotItemID := req.BallotItemID
	if ballotItemID == 0 && req.OptionID != 0 {
		ballotItemID = req.OptionID
	}

	if ballotItemID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "option_id or ballot_item_id is required"})
		return
	}

	// Check if ballot exists and is active
	var ballotExists bool
	err = h.db.QueryRow("SELECT is_active FROM ballots WHERE id = $1", ballotID).Scan(&ballotExists)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ballot not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !ballotExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ballot is not active"})
		return
	}

	// Check if ballot item belongs to this ballot
	var itemBallotID int
	err = h.db.QueryRow("SELECT ballot_id FROM ballot_items WHERE id = $1", ballotItemID).Scan(&itemBallotID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ballot item not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if itemBallotID != ballotID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ballot item does not belong to this ballot"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Check if user has already voted on this ballot
	var existingVoteID int
	var existingBallotItemID int
	err = tx.QueryRow("SELECT id, ballot_item_id FROM votes WHERE user_id = $1 AND ballot_id = $2", userID, ballotID).Scan(&existingVoteID, &existingBallotItemID)
	
	if err == nil {
		// User has already voted, update their vote
		// First decrease vote count for previous choice
		_, err = tx.Exec("UPDATE ballot_items SET vote_count = vote_count - 1 WHERE id = $1", existingBallotItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating vote count"})
			return
		}

		// Update the vote record
		_, err = tx.Exec("UPDATE votes SET ballot_item_id = $1 WHERE id = $2", ballotItemID, existingVoteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating vote"})
			return
		}
	} else if err == sql.ErrNoRows {
		// User hasn't voted yet, create new vote
		_, err = tx.Exec("INSERT INTO votes (user_id, ballot_id, ballot_item_id) VALUES ($1, $2, $3)", userID, ballotID, ballotItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating vote"})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Increase vote count for chosen item
	_, err = tx.Exec("UPDATE ballot_items SET vote_count = vote_count + 1 WHERE id = $1", ballotItemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating vote count"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vote recorded successfully"})
}

func (h *VoteHandler) GetUserVote(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ballotIDStr := c.Param("ballot_id")
	ballotID, err := strconv.Atoi(ballotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ballot ID"})
		return
	}

	var vote models.Vote
	err = h.db.QueryRow(
		"SELECT id, user_id, ballot_id, ballot_item_id, created_at FROM votes WHERE user_id = $1 AND ballot_id = $2",
		userID, ballotID,
	).Scan(&vote.ID, &vote.UserID, &vote.BallotID, &vote.BallotItemID, &vote.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No vote found for this ballot"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Return response with both option_id and ballot_item_id for compatibility
	c.JSON(http.StatusOK, gin.H{
		"id":              vote.ID,
		"user_id":         vote.UserID,
		"ballot_id":       vote.BallotID,
		"ballot_item_id":  vote.BallotItemID,
		"option_id":       vote.BallotItemID, // Frontend expects option_id
		"created_at":      vote.CreatedAt,
	})
}

func (h *VoteHandler) GetBallotResults(c *gin.Context) {
	ballotIDStr := c.Param("id")
	ballotID, err := strconv.Atoi(ballotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ballot ID"})
		return
	}

	// Check if ballot exists
	var ballotExists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM ballots WHERE id = $1)", ballotID).Scan(&ballotExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !ballotExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ballot not found"})
		return
	}

	// Get ballot items with vote counts
	rows, err := h.db.Query(`
		SELECT id, ballot_id, title, description, vote_count
		FROM ballot_items 
		WHERE ballot_id = $1 
		ORDER BY vote_count DESC, id ASC
	`, ballotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching results"})
		return
	}
	defer rows.Close()

	type ResultItem struct {
		ID          int    `json:"id"`
		OptionID    int    `json:"option_id"` // Frontend expects option_id
		BallotID    int    `json:"ballot_id"`
		Title       string `json:"title"`
		OptionTitle string `json:"option_title"` // Alias for title
		Description string `json:"description"`
		VoteCount   int    `json:"vote_count"`
	}

	results := make([]ResultItem, 0)
	totalVotes := 0
	for rows.Next() {
		var item models.BallotItem
		err := rows.Scan(&item.ID, &item.BallotID, &item.Title, &item.Description, &item.VoteCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning result"})
			return
		}
		results = append(results, ResultItem{
			ID:          item.ID,
			OptionID:    item.ID,
			BallotID:    item.BallotID,
			Title:       item.Title,
			OptionTitle: item.Title,
			Description: item.Description,
			VoteCount:   item.VoteCount,
		})
		totalVotes += item.VoteCount
	}

	c.JSON(http.StatusOK, gin.H{
		"ballot_id":   ballotID,
		"results":     results,
		"total_votes": totalVotes,
	})
}