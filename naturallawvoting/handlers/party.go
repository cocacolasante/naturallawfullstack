package handlers

import (
	"database/sql"
	"net/http"
	"voting-api/database"
	"voting-api/models"

	"github.com/gin-gonic/gin"
)

type PartyHandler struct {
	db *database.DB
}

func NewPartyHandler(db *database.DB) *PartyHandler {
	return &PartyHandler{db: db}
}

// GetParties returns all political parties (summary, no external links).
func (h *PartyHandler) GetParties(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, slug, description, intro_text, sort_order, created_at
		FROM political_parties
		ORDER BY sort_order ASC, name ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var parties []models.Party
	for rows.Next() {
		var p models.Party
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.IntroText, &p.SortOrder, &p.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning party"})
			return
		}
		parties = append(parties, p)
	}

	if parties == nil {
		parties = []models.Party{}
	}
	c.JSON(http.StatusOK, parties)
}

// GetParty returns a single party by slug, including its external links.
func (h *PartyHandler) GetParty(c *gin.Context) {
	slug := c.Param("slug")

	var party models.Party
	err := h.db.QueryRow(`
		SELECT id, name, slug, description, intro_text, sort_order, created_at
		FROM political_parties
		WHERE slug = $1
	`, slug).Scan(&party.ID, &party.Name, &party.Slug, &party.Description, &party.IntroText, &party.SortOrder, &party.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Party not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	rows, err := h.db.Query(`
		SELECT id, party_id, url, label, sort_order
		FROM party_external_links
		WHERE party_id = $1
		ORDER BY sort_order ASC
	`, party.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching external links"})
		return
	}
	defer rows.Close()

	var links []models.ExternalLink
	for rows.Next() {
		var l models.ExternalLink
		if err := rows.Scan(&l.ID, &l.PartyID, &l.URL, &l.Label, &l.SortOrder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning link"})
			return
		}
		links = append(links, l)
	}

	if links == nil {
		links = []models.ExternalLink{}
	}
	party.ExternalLinks = links
	c.JSON(http.StatusOK, party)
}

// SubmitEngagement saves or updates a user's engagement responses for a party.
func (h *PartyHandler) SubmitEngagement(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	slug := c.Param("slug")

	// Verify party exists
	var partyID int
	err := h.db.QueryRow(`SELECT id FROM political_parties WHERE slug = $1`, slug).Scan(&partyID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Party not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var req models.PartyEngagementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var engagement models.PartyEngagement
	err = h.db.QueryRow(`
		INSERT INTO party_engagement (user_id, party_slug, q1_answer, q2_answer, q3_answer, q4_answer, comments)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, party_slug) DO UPDATE SET
			q1_answer = EXCLUDED.q1_answer,
			q2_answer = EXCLUDED.q2_answer,
			q3_answer = EXCLUDED.q3_answer,
			q4_answer = EXCLUDED.q4_answer,
			comments  = EXCLUDED.comments,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, party_slug, q1_answer, q2_answer, q3_answer, q4_answer, comments, created_at, updated_at
	`, userID, slug, req.Q1, req.Q2, req.Q3, req.Q4, req.Comments).Scan(
		&engagement.ID, &engagement.UserID, &engagement.PartySlug,
		&engagement.Q1Answer, &engagement.Q2Answer, &engagement.Q3Answer, &engagement.Q4Answer,
		&engagement.Comments, &engagement.CreatedAt, &engagement.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving engagement"})
		return
	}

	c.JSON(http.StatusOK, engagement)
}

// GetMyEngagement returns the current user's saved engagement for a party.
func (h *PartyHandler) GetMyEngagement(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	slug := c.Param("slug")

	var engagement models.PartyEngagement
	err := h.db.QueryRow(`
		SELECT id, user_id, party_slug, q1_answer, q2_answer, q3_answer, q4_answer, comments, created_at, updated_at
		FROM party_engagement
		WHERE user_id = $1 AND party_slug = $2
	`, userID, slug).Scan(
		&engagement.ID, &engagement.UserID, &engagement.PartySlug,
		&engagement.Q1Answer, &engagement.Q2Answer, &engagement.Q3Answer, &engagement.Q4Answer,
		&engagement.Comments, &engagement.CreatedAt, &engagement.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No engagement found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, engagement)
}
