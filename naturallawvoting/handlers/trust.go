package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"voting-api/database"
	"voting-api/models"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type TrustHandler struct {
	db *database.DB
}

func NewTrustHandler(db *database.DB) *TrustHandler {
	return &TrustHandler{db: db}
}

func (h *TrustHandler) calcTrustScore(userID int) (float64, int) {
	var score float64
	var count int
	err := h.db.QueryRow(
		`SELECT COALESCE(AVG(score), 0), COUNT(*) FROM trust_votes WHERE subject_id = $1`,
		userID,
	).Scan(&score, &count)
	if err != nil {
		return 0, 0
	}
	return score, count
}

// POST /api/v1/users/:user_id/trust
func (h *TrustHandler) SubmitTrustVote(c *gin.Context) {
	voterID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	subjectID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	if voterID.(int) == subjectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot rate yourself"})
		return
	}

	var req models.TrustVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var vote models.TrustVote
	err = h.db.QueryRow(`
		INSERT INTO trust_votes (voter_id, subject_id, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (voter_id, subject_id) DO UPDATE
		  SET score = EXCLUDED.score, updated_at = NOW()
		RETURNING id, voter_id, subject_id, score, created_at, updated_at`,
		voterID, subjectID, req.Score,
	).Scan(&vote.ID, &vote.VoterID, &vote.SubjectID, &vote.Score, &vote.CreatedAt, &vote.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, vote)
}

// GET /api/v1/public/users/:user_id/trust-score
func (h *TrustHandler) GetTrustScore(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	score, count := h.calcTrustScore(userID)
	c.JSON(http.StatusOK, models.TrustScore{
		UserID:    userID,
		Score:     score,
		VoteCount: count,
	})
}

// GET /api/v1/users/:user_id/my-trust-vote
func (h *TrustHandler) GetMyTrustVote(c *gin.Context) {
	voterID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	subjectID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	var vote models.TrustVote
	err = h.db.QueryRow(`
		SELECT id, voter_id, subject_id, score, created_at, updated_at
		FROM trust_votes WHERE voter_id = $1 AND subject_id = $2`,
		voterID, subjectID,
	).Scan(&vote.ID, &vote.VoterID, &vote.SubjectID, &vote.Score, &vote.CreatedAt, &vote.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No trust vote found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, vote)
}

// GET /api/v1/my-trust-score
func (h *TrustHandler) GetMyTrustScore(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	score, count := h.calcTrustScore(userID.(int))
	c.JSON(http.StatusOK, models.TrustScore{
		UserID:    userID.(int),
		Score:     score,
		VoteCount: count,
	})
}

// GET /api/v1/profile/visibility
func (h *TrustHandler) GetProfileVisibility(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var pv models.ProfileVisibility
	err := h.db.QueryRow(`
		SELECT user_id, info_threshold, address_threshold, political_threshold,
		       religious_threshold, race_ethnicity_threshold, economic_threshold
		FROM profile_visibility WHERE user_id = $1`,
		userID,
	).Scan(&pv.UserID, &pv.InfoThreshold, &pv.AddressThreshold, &pv.PoliticalThreshold,
		&pv.ReligiousThreshold, &pv.RaceEthnicityThreshold, &pv.EconomicThreshold)

	if err == sql.ErrNoRows {
		// Return defaults (all 0)
		pv = models.ProfileVisibility{UserID: userID.(int)}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, pv)
}

// PUT /api/v1/profile/visibility
func (h *TrustHandler) UpdateProfileVisibility(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateProfileVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Upsert with defaults then update only provided fields
	_, err := h.db.Exec(`
		INSERT INTO profile_visibility (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Build dynamic update
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.InfoThreshold != nil {
		setClauses = append(setClauses, fmt.Sprintf("info_threshold = $%d", argIdx))
		args = append(args, *req.InfoThreshold)
		argIdx++
	}
	if req.AddressThreshold != nil {
		setClauses = append(setClauses, fmt.Sprintf("address_threshold = $%d", argIdx))
		args = append(args, *req.AddressThreshold)
		argIdx++
	}
	if req.PoliticalThreshold != nil {
		setClauses = append(setClauses, fmt.Sprintf("political_threshold = $%d", argIdx))
		args = append(args, *req.PoliticalThreshold)
		argIdx++
	}
	if req.ReligiousThreshold != nil {
		setClauses = append(setClauses, fmt.Sprintf("religious_threshold = $%d", argIdx))
		args = append(args, *req.ReligiousThreshold)
		argIdx++
	}
	if req.RaceEthnicityThreshold != nil {
		setClauses = append(setClauses, fmt.Sprintf("race_ethnicity_threshold = $%d", argIdx))
		args = append(args, *req.RaceEthnicityThreshold)
		argIdx++
	}
	if req.EconomicThreshold != nil {
		setClauses = append(setClauses, fmt.Sprintf("economic_threshold = $%d", argIdx))
		args = append(args, *req.EconomicThreshold)
		argIdx++
	}

	if len(setClauses) > 0 {
		query := "UPDATE profile_visibility SET "
		for i, clause := range setClauses {
			if i > 0 {
				query += ", "
			}
			query += clause
		}
		query += fmt.Sprintf(" WHERE user_id = $%d", argIdx)
		args = append(args, userID)
		if _, err := h.db.Exec(query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
	}

	var pv models.ProfileVisibility
	err = h.db.QueryRow(`
		SELECT user_id, info_threshold, address_threshold, political_threshold,
		       religious_threshold, race_ethnicity_threshold, economic_threshold
		FROM profile_visibility WHERE user_id = $1`, userID,
	).Scan(&pv.UserID, &pv.InfoThreshold, &pv.AddressThreshold, &pv.PoliticalThreshold,
		&pv.ReligiousThreshold, &pv.RaceEthnicityThreshold, &pv.EconomicThreshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, pv)
}

// GET /api/v1/public/users/search?username=X
func (h *TrustHandler) SearchUsers(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username query param required"})
		return
	}

	rows, err := h.db.Query(`
		SELECT u.id, u.username,
		       COALESCE(AVG(tv.score), 0) as trust_score,
		       COUNT(tv.id) as vote_count
		FROM users u
		LEFT JOIN trust_votes tv ON tv.subject_id = u.id
		WHERE u.username ILIKE '%' || $1 || '%'
		GROUP BY u.id
		ORDER BY u.username
		LIMIT 20`,
		username,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	users := []models.PublicUser{}
	for rows.Next() {
		var u models.PublicUser
		if err := rows.Scan(&u.ID, &u.Username, &u.TrustScore, &u.VoteCount); err != nil {
			continue
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, users)
}

// GET /api/v1/users/:user_id/profile
func (h *TrustHandler) GetUserProfileSections(c *gin.Context) {
	viewerIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	viewerID := viewerIDRaw.(int)

	subjectID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	// Get subject username
	var subjectUsername string
	err = h.db.QueryRow("SELECT username FROM users WHERE id = $1", subjectID).Scan(&subjectUsername)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	subjectTrustScore, subjectVoteCount := h.calcTrustScore(subjectID)
	isOwner := viewerID == subjectID

	// Get viewer's trust score
	var viewerScore float64
	if !isOwner {
		viewerScore, _ = h.calcTrustScore(viewerID)
	} else {
		viewerScore = 100
	}

	// Get subject's visibility thresholds
	pv := models.ProfileVisibility{UserID: subjectID}
	err = h.db.QueryRow(`
		SELECT user_id, info_threshold, address_threshold, political_threshold,
		       religious_threshold, race_ethnicity_threshold, economic_threshold
		FROM profile_visibility WHERE user_id = $1`, subjectID,
	).Scan(&pv.UserID, &pv.InfoThreshold, &pv.AddressThreshold, &pv.PoliticalThreshold,
		&pv.ReligiousThreshold, &pv.RaceEthnicityThreshold, &pv.EconomicThreshold)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	sectionResult := func(threshold int, data interface{}) gin.H {
		if isOwner || viewerScore >= float64(threshold) {
			return gin.H{"locked": false, "data": data}
		}
		return gin.H{"locked": true, "threshold": threshold, "viewer_score": viewerScore}
	}

	// Fetch each section
	// Info
	var infoData interface{}
	var subjectEmail string
	err = h.db.QueryRow("SELECT email FROM users WHERE id = $1", subjectID).Scan(&subjectEmail)
	if err == nil {
		var profile models.UserProfile
		err2 := h.db.QueryRow(`
			SELECT user_id, email, full_name, birthday, gender, mothers_maiden_name,
			       phone_number, additional_emails, created_at, updated_at
			FROM user_profiles WHERE email = $1`, subjectEmail,
		).Scan(&profile.UserID, &profile.Email, &profile.FullName, &profile.Birthday,
			&profile.Gender, &profile.MothersMaidenName, &profile.PhoneNumber,
			&profile.AdditionalEmails, &profile.CreatedAt, &profile.UpdatedAt)
		if err2 == nil {
			infoData = profile
		}
	}

	// Address
	var addressData interface{}
	var address models.UserAddress
	err = h.db.QueryRow(`
		SELECT user_id, street_number, street_name, address_line_2, city, state,
		       zip_code, created_at, updated_at
		FROM user_addresses WHERE user_id = $1`, subjectID,
	).Scan(&address.UserID, &address.StreetNumber, &address.StreetName,
		&address.AddressLine2, &address.City, &address.State, &address.ZipCode,
		&address.CreatedAt, &address.UpdatedAt)
	if err == nil {
		addressData = address
	}

	// Political
	var politicalData interface{}
	var political models.UserPoliticalAffiliation
	err = h.db.QueryRow(`
		SELECT user_id, party_affiliation, created_at, updated_at
		FROM user_political_affiliations WHERE user_id = $1`, subjectID,
	).Scan(&political.UserID, &political.PartyAffiliation, &political.CreatedAt, &political.UpdatedAt)
	if err == nil {
		politicalData = political
	}

	// Religious
	var religiousData interface{}
	var religious models.UserReligiousAffiliation
	err = h.db.QueryRow(`
		SELECT user_id, religion, supporting_religion, religious_services_types, created_at, updated_at
		FROM user_religious_affiliations WHERE user_id = $1`, subjectID,
	).Scan(&religious.UserID, &religious.Religion, &religious.SupportingReligion,
		&religious.ReligiousServicesTypes, &religious.CreatedAt, &religious.UpdatedAt)
	if err == nil {
		religiousData = religious
	}

	// Race/Ethnicity
	var raceData interface{}
	var race models.UserRaceEthnicity
	err = h.db.QueryRow(`
		SELECT user_id, race, created_at, updated_at
		FROM user_race_ethnicity WHERE user_id = $1`, subjectID,
	).Scan(&race.UserID, &race.Race, &race.CreatedAt, &race.UpdatedAt)
	if err == nil {
		raceData = race
	}

	// Economic
	var economicData interface{}
	var economic models.EconomicInfo
	err = h.db.QueryRow(`
		SELECT user_id, for_current_political_structure, for_capitalism, for_laws,
		       goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		       additional_text, created_at, updated_at
		FROM economic_info WHERE user_id = $1`, subjectID,
	).Scan(&economic.UserID, &economic.ForCurrentPoliticalStructure, &economic.ForCapitalism,
		&economic.ForLaws, pq.Array(&economic.GoodsServices), pq.Array(&economic.Affiliations),
		&economic.SupportOfAltEcon, &economic.SupportAltComm, &economic.AdditionalText,
		&economic.CreatedAt, &economic.UpdatedAt)
	if err == nil {
		economicData = economic
	}

	c.JSON(http.StatusOK, gin.H{
		"user": models.PublicUser{
			ID:         subjectID,
			Username:   subjectUsername,
			TrustScore: subjectTrustScore,
			VoteCount:  subjectVoteCount,
		},
		"info":           sectionResult(pv.InfoThreshold, infoData),
		"address":        sectionResult(pv.AddressThreshold, addressData),
		"political":      sectionResult(pv.PoliticalThreshold, politicalData),
		"religious":      sectionResult(pv.ReligiousThreshold, religiousData),
		"race_ethnicity": sectionResult(pv.RaceEthnicityThreshold, raceData),
		"economic":       sectionResult(pv.EconomicThreshold, economicData),
	})
}
