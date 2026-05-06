package handlers

import (
	"net/http"
	"strconv"
	"voting-api/database"
	"voting-api/models"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type VoteHandler struct {
	db *database.DB
}

func NewVoteHandler(db *database.DB) *VoteHandler {
	return &VoteHandler{db: db}
}

// Vote upserts a user's score (0-100) for one or more options on a ballot.
// Each ScoreEntry is independent — a user can score every option on a ballot.
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

	// Normalize entries: resolve option_id alias, validate score range, collect IDs.
	type normalized struct {
		ItemID int
		Score  int
	}
	entries := make([]normalized, 0, len(req.Scores))
	itemIDs := make([]int, 0, len(req.Scores))
	for _, s := range req.Scores {
		itemID := s.BallotItemID
		if itemID == 0 {
			itemID = s.OptionID
		}
		if itemID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "option_id or ballot_item_id is required for each score"})
			return
		}
		if s.Score == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "score is required for each entry"})
			return
		}
		if *s.Score < 0 || *s.Score > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "score must be between 0 and 100"})
			return
		}
		entries = append(entries, normalized{ItemID: itemID, Score: *s.Score})
		itemIDs = append(itemIDs, itemID)
	}

	// Verify the ballot exists and is active.
	var isActive bool
	err = h.db.QueryRow("SELECT is_active FROM ballots WHERE id = $1", ballotID).Scan(&isActive)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ballot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !isActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ballot is not active"})
		return
	}

	// Verify every submitted ballot item belongs to this ballot.
	rows, err := h.db.Query("SELECT id FROM ballot_items WHERE ballot_id = $1 AND id = ANY($2)", ballotID, pq.Array(itemIDs))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	valid := make(map[int]bool, len(itemIDs))
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		valid[id] = true
	}
	rows.Close()
	for _, id := range itemIDs {
		if !valid[id] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ballot item does not belong to this ballot"})
			return
		}
	}

	// Upsert each (user_id, ballot_item_id) score in a single transaction.
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO votes (user_id, ballot_id, ballot_item_id, score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, ballot_item_id)
		DO UPDATE SET score = EXCLUDED.score, updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(userID, ballotID, e.ItemID, e.Score); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording vote"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Vote recorded successfully",
		"ballot_id":   ballotID,
		"score_count": len(entries),
	})
}

// GetUserVote returns every score the user has cast on this ballot.
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

	rows, err := h.db.Query(
		`SELECT id, user_id, ballot_id, ballot_item_id, score, created_at, updated_at
		 FROM votes WHERE user_id = $1 AND ballot_id = $2 ORDER BY ballot_item_id ASC`,
		userID, ballotID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type entry struct {
		ID           int    `json:"id"`
		UserID       int    `json:"user_id"`
		BallotID     int    `json:"ballot_id"`
		BallotItemID int    `json:"ballot_item_id"`
		OptionID     int    `json:"option_id"` // Frontend alias
		Score        int    `json:"score"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}

	scores := make([]entry, 0)
	for rows.Next() {
		var v models.Vote
		if err := rows.Scan(&v.ID, &v.UserID, &v.BallotID, &v.BallotItemID, &v.Score, &v.CreatedAt, &v.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		scores = append(scores, entry{
			ID:           v.ID,
			UserID:       v.UserID,
			BallotID:     v.BallotID,
			BallotItemID: v.BallotItemID,
			OptionID:     v.BallotItemID,
			Score:        v.Score,
			CreatedAt:    v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:    v.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	if len(scores) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No vote found for this ballot"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ballot_id": ballotID,
		"scores":    scores,
	})
}

// GetBallotResults returns each option's average score (0-100) and voter count.
// Sorted by average score descending so the leading option appears first.
func (h *VoteHandler) GetBallotResults(c *gin.Context) {
	ballotIDStr := c.Param("id")
	ballotID, err := strconv.Atoi(ballotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ballot ID"})
		return
	}

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

	rows, err := h.db.Query(`
		SELECT bi.id, bi.ballot_id, bi.title, bi.description,
		       COALESCE(AVG(v.score), 0)::float8 AS average_score,
		       COALESCE(SUM(v.score), 0)::bigint AS total_score,
		       COUNT(v.id)::bigint AS voter_count
		FROM ballot_items bi
		LEFT JOIN votes v ON v.ballot_item_id = bi.id
		WHERE bi.ballot_id = $1
		GROUP BY bi.id, bi.ballot_id, bi.title, bi.description
		ORDER BY average_score DESC, bi.id ASC
	`, ballotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching results"})
		return
	}
	defer rows.Close()

	type ResultItem struct {
		ID           int     `json:"id"`
		OptionID     int     `json:"option_id"`
		BallotID     int     `json:"ballot_id"`
		Title        string  `json:"title"`
		OptionTitle  string  `json:"option_title"`
		Description  string  `json:"description"`
		AverageScore float64 `json:"average_score"`
		TotalScore   int64   `json:"total_score"`
		VoterCount   int64   `json:"voter_count"`
	}

	results := make([]ResultItem, 0)
	for rows.Next() {
		var r ResultItem
		if err := rows.Scan(&r.ID, &r.BallotID, &r.Title, &r.Description, &r.AverageScore, &r.TotalScore, &r.VoterCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning result"})
			return
		}
		r.OptionID = r.ID
		r.OptionTitle = r.Title
		results = append(results, r)
	}

	// total_voters is the count of distinct users who scored any option on the ballot.
	// Computed separately so subsets of voters across options aren't conflated.
	var totalVoters int64
	if err := h.db.QueryRow(
		`SELECT COUNT(DISTINCT user_id) FROM votes WHERE ballot_id = $1`,
		ballotID,
	).Scan(&totalVoters); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ballot_id":    ballotID,
		"results":      results,
		"total_voters": totalVoters,
	})
}
