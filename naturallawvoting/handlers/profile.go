package handlers

import (
	"database/sql"
	"net/http"
	"time"
	"voting-api/database"
	"voting-api/models"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type ProfileHandler struct {
	db *database.DB
}

func NewProfileHandler(db *database.DB) *ProfileHandler {
	return &ProfileHandler{db: db}
}

// User Profile Handlers

func (h *ProfileHandler) GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get user email first
	var email string
	err := h.db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var profile models.UserProfile
	err = h.db.QueryRow(`
		SELECT user_id, email, full_name, birthday, gender, mothers_maiden_name,
		       phone_number, additional_emails, created_at, updated_at
		FROM user_profiles WHERE email = $1`,
		email,
	).Scan(&profile.UserID, &profile.Email, &profile.FullName, &profile.Birthday,
		&profile.Gender, &profile.MothersMaidenName, &profile.PhoneNumber,
		&profile.AdditionalEmails, &profile.CreatedAt, &profile.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) CreateUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user email
	var email string
	err := h.db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if profile already exists
	var existingProfile models.UserProfile
	err = h.db.QueryRow("SELECT user_id FROM user_profiles WHERE email = $1", email).Scan(&existingProfile.UserID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Profile already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Parse birthday if provided
	var birthday *time.Time
	if req.Birthday != "" {
		parsedDate, err := time.Parse("2006-01-02", req.Birthday)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid birthday format. Use YYYY-MM-DD"})
			return
		}
		birthday = &parsedDate
	}

	var profile models.UserProfile
	err = h.db.QueryRow(`
		INSERT INTO user_profiles
		(user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number,
		          additional_emails, created_at, updated_at`,
		userID, email, req.FullName, birthday, req.Gender, req.MothersMaidenName,
		req.PhoneNumber, pq.Array(req.AdditionalEmails),
	).Scan(&profile.UserID, &profile.Email, &profile.FullName, &profile.Birthday,
		&profile.Gender, &profile.MothersMaidenName, &profile.PhoneNumber,
		&profile.AdditionalEmails, &profile.CreatedAt, &profile.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating profile"})
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (h *ProfileHandler) UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user email
	var email string
	err := h.db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Build dynamic update query
	query := "UPDATE user_profiles SET "
	args := []interface{}{}
	argCount := 1

	if req.FullName != nil {
		query += "full_name = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.FullName)
		argCount++
	}
	if req.Birthday != nil {
		parsedDate, err := time.Parse("2006-01-02", *req.Birthday)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid birthday format. Use YYYY-MM-DD"})
			return
		}
		query += "birthday = $" + string(rune(argCount+'0')) + ", "
		args = append(args, parsedDate)
		argCount++
	}
	if req.Gender != nil {
		query += "gender = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.Gender)
		argCount++
	}
	if req.MothersMaidenName != nil {
		query += "mothers_maiden_name = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.MothersMaidenName)
		argCount++
	}
	if req.PhoneNumber != nil {
		query += "phone_number = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.PhoneNumber)
		argCount++
	}
	if req.AdditionalEmails != nil {
		query += "additional_emails = $" + string(rune(argCount+'0')) + ", "
		args = append(args, pq.Array(req.AdditionalEmails))
		argCount++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]
	query += " WHERE email = $" + string(rune(argCount+'0')) + " RETURNING user_id, email, full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails, created_at, updated_at"
	args = append(args, email)

	var profile models.UserProfile
	err = h.db.QueryRow(query, args...).Scan(
		&profile.UserID, &profile.Email, &profile.FullName, &profile.Birthday,
		&profile.Gender, &profile.MothersMaidenName, &profile.PhoneNumber,
		&profile.AdditionalEmails, &profile.CreatedAt, &profile.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) DeleteUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get user email
	var email string
	err := h.db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	result, err := h.db.Exec("DELETE FROM user_profiles WHERE email = $1", email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting profile"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile deleted successfully"})
}

// User Address Handlers

func (h *ProfileHandler) GetUserAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var address models.UserAddress
	err := h.db.QueryRow(`
		SELECT user_id, street_number, street_name, address_line_2, city, state,
		       zip_code, created_at, updated_at
		FROM user_addresses WHERE user_id = $1`,
		userID,
	).Scan(&address.UserID, &address.StreetNumber, &address.StreetName,
		&address.AddressLine2, &address.City, &address.State, &address.ZipCode,
		&address.CreatedAt, &address.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, address)
}

func (h *ProfileHandler) CreateUserAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateUserAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if address already exists
	var existingAddress models.UserAddress
	err := h.db.QueryRow("SELECT user_id FROM user_addresses WHERE user_id = $1", userID).Scan(&existingAddress.UserID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Address already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var address models.UserAddress
	err = h.db.QueryRow(`
		INSERT INTO user_addresses
		(user_id, street_number, street_name, address_line_2, city, state, zip_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_id, street_number, street_name, address_line_2, city, state,
		          zip_code, created_at, updated_at`,
		userID, req.StreetNumber, req.StreetName, req.AddressLine2, req.City, req.State, req.ZipCode,
	).Scan(&address.UserID, &address.StreetNumber, &address.StreetName,
		&address.AddressLine2, &address.City, &address.State, &address.ZipCode,
		&address.CreatedAt, &address.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating address"})
		return
	}

	c.JSON(http.StatusCreated, address)
}

func (h *ProfileHandler) UpdateUserAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateUserAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE user_addresses SET "
	args := []interface{}{}
	argCount := 1

	if req.StreetNumber != nil {
		query += "street_number = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.StreetNumber)
		argCount++
	}
	if req.StreetName != nil {
		query += "street_name = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.StreetName)
		argCount++
	}
	if req.AddressLine2 != nil {
		query += "address_line_2 = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.AddressLine2)
		argCount++
	}
	if req.City != nil {
		query += "city = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.City)
		argCount++
	}
	if req.State != nil {
		query += "state = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.State)
		argCount++
	}
	if req.ZipCode != nil {
		query += "zip_code = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.ZipCode)
		argCount++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]
	query += " WHERE user_id = $" + string(rune(argCount+'0')) + " RETURNING user_id, street_number, street_name, address_line_2, city, state, zip_code, created_at, updated_at"
	args = append(args, userID)

	var address models.UserAddress
	err := h.db.QueryRow(query, args...).Scan(
		&address.UserID, &address.StreetNumber, &address.StreetName,
		&address.AddressLine2, &address.City, &address.State, &address.ZipCode,
		&address.CreatedAt, &address.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating address"})
		return
	}

	c.JSON(http.StatusOK, address)
}

func (h *ProfileHandler) DeleteUserAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.db.Exec("DELETE FROM user_addresses WHERE user_id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting address"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address deleted successfully"})
}

// User Political Affiliation Handlers

func (h *ProfileHandler) GetUserPoliticalAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var affiliation models.UserPoliticalAffiliation
	err := h.db.QueryRow(`
		SELECT user_id, party_affiliation, created_at, updated_at
		FROM user_political_affiliations WHERE user_id = $1`,
		userID,
	).Scan(&affiliation.UserID, &affiliation.PartyAffiliation,
		&affiliation.CreatedAt, &affiliation.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Political affiliation not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, affiliation)
}

func (h *ProfileHandler) CreateUserPoliticalAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateUserPoliticalAffiliationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if affiliation already exists
	var existingAffiliation models.UserPoliticalAffiliation
	err := h.db.QueryRow("SELECT user_id FROM user_political_affiliations WHERE user_id = $1", userID).Scan(&existingAffiliation.UserID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Political affiliation already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var affiliation models.UserPoliticalAffiliation
	err = h.db.QueryRow(`
		INSERT INTO user_political_affiliations (user_id, party_affiliation)
		VALUES ($1, $2)
		RETURNING user_id, party_affiliation, created_at, updated_at`,
		userID, req.PartyAffiliation,
	).Scan(&affiliation.UserID, &affiliation.PartyAffiliation,
		&affiliation.CreatedAt, &affiliation.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating political affiliation"})
		return
	}

	c.JSON(http.StatusCreated, affiliation)
}

func (h *ProfileHandler) UpdateUserPoliticalAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateUserPoliticalAffiliationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PartyAffiliation == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	var affiliation models.UserPoliticalAffiliation
	err := h.db.QueryRow(`
		UPDATE user_political_affiliations
		SET party_affiliation = $1
		WHERE user_id = $2
		RETURNING user_id, party_affiliation, created_at, updated_at`,
		*req.PartyAffiliation, userID,
	).Scan(&affiliation.UserID, &affiliation.PartyAffiliation,
		&affiliation.CreatedAt, &affiliation.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Political affiliation not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating political affiliation"})
		return
	}

	c.JSON(http.StatusOK, affiliation)
}

func (h *ProfileHandler) DeleteUserPoliticalAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.db.Exec("DELETE FROM user_political_affiliations WHERE user_id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting political affiliation"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Political affiliation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Political affiliation deleted successfully"})
}

// User Religious Affiliation Handlers

func (h *ProfileHandler) GetUserReligiousAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var affiliation models.UserReligiousAffiliation
	err := h.db.QueryRow(`
		SELECT user_id, religion, supporting_religion, religious_services_types,
		       created_at, updated_at
		FROM user_religious_affiliations WHERE user_id = $1`,
		userID,
	).Scan(&affiliation.UserID, &affiliation.Religion, &affiliation.SupportingReligion,
		&affiliation.ReligiousServicesTypes, &affiliation.CreatedAt, &affiliation.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Religious affiliation not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, affiliation)
}

func (h *ProfileHandler) CreateUserReligiousAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateUserReligiousAffiliationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate supporting_religion is between 0-10
	if req.SupportingReligion != nil && (*req.SupportingReligion < 0 || *req.SupportingReligion > 10) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supporting_religion must be between 0 and 10"})
		return
	}

	// Check if affiliation already exists
	var existingAffiliation models.UserReligiousAffiliation
	err := h.db.QueryRow("SELECT user_id FROM user_religious_affiliations WHERE user_id = $1", userID).Scan(&existingAffiliation.UserID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Religious affiliation already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var affiliation models.UserReligiousAffiliation
	err = h.db.QueryRow(`
		INSERT INTO user_religious_affiliations
		(user_id, religion, supporting_religion, religious_services_types)
		VALUES ($1, $2, $3, $4)
		RETURNING user_id, religion, supporting_religion, religious_services_types,
		          created_at, updated_at`,
		userID, req.Religion, req.SupportingReligion, pq.Array(req.ReligiousServicesTypes),
	).Scan(&affiliation.UserID, &affiliation.Religion, &affiliation.SupportingReligion,
		&affiliation.ReligiousServicesTypes, &affiliation.CreatedAt, &affiliation.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating religious affiliation"})
		return
	}

	c.JSON(http.StatusCreated, affiliation)
}

func (h *ProfileHandler) UpdateUserReligiousAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateUserReligiousAffiliationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate supporting_religion is between 0-10
	if req.SupportingReligion != nil && (*req.SupportingReligion < 0 || *req.SupportingReligion > 10) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supporting_religion must be between 0 and 10"})
		return
	}

	// Build dynamic update query
	query := "UPDATE user_religious_affiliations SET "
	args := []interface{}{}
	argCount := 1

	if req.Religion != nil {
		query += "religion = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.Religion)
		argCount++
	}
	if req.SupportingReligion != nil {
		query += "supporting_religion = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.SupportingReligion)
		argCount++
	}
	if req.ReligiousServicesTypes != nil {
		query += "religious_services_types = $" + string(rune(argCount+'0')) + ", "
		args = append(args, pq.Array(req.ReligiousServicesTypes))
		argCount++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]
	query += " WHERE user_id = $" + string(rune(argCount+'0')) + " RETURNING user_id, religion, supporting_religion, religious_services_types, created_at, updated_at"
	args = append(args, userID)

	var affiliation models.UserReligiousAffiliation
	err := h.db.QueryRow(query, args...).Scan(
		&affiliation.UserID, &affiliation.Religion, &affiliation.SupportingReligion,
		&affiliation.ReligiousServicesTypes, &affiliation.CreatedAt, &affiliation.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Religious affiliation not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating religious affiliation"})
		return
	}

	c.JSON(http.StatusOK, affiliation)
}

func (h *ProfileHandler) DeleteUserReligiousAffiliation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.db.Exec("DELETE FROM user_religious_affiliations WHERE user_id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting religious affiliation"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Religious affiliation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Religious affiliation deleted successfully"})
}

// User Race/Ethnicity Handlers

func (h *ProfileHandler) GetUserRaceEthnicity(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var raceEthnicity models.UserRaceEthnicity
	err := h.db.QueryRow(`
		SELECT user_id, race, created_at, updated_at
		FROM user_race_ethnicity WHERE user_id = $1`,
		userID,
	).Scan(&raceEthnicity.UserID, &raceEthnicity.Race,
		&raceEthnicity.CreatedAt, &raceEthnicity.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Race/ethnicity not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, raceEthnicity)
}

func (h *ProfileHandler) CreateUserRaceEthnicity(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateUserRaceEthnicityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if race/ethnicity already exists
	var existingRaceEthnicity models.UserRaceEthnicity
	err := h.db.QueryRow("SELECT user_id FROM user_race_ethnicity WHERE user_id = $1", userID).Scan(&existingRaceEthnicity.UserID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Race/ethnicity already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var raceEthnicity models.UserRaceEthnicity
	err = h.db.QueryRow(`
		INSERT INTO user_race_ethnicity (user_id, race)
		VALUES ($1, $2)
		RETURNING user_id, race, created_at, updated_at`,
		userID, pq.Array(req.Race),
	).Scan(&raceEthnicity.UserID, &raceEthnicity.Race,
		&raceEthnicity.CreatedAt, &raceEthnicity.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating race/ethnicity"})
		return
	}

	c.JSON(http.StatusCreated, raceEthnicity)
}

func (h *ProfileHandler) UpdateUserRaceEthnicity(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateUserRaceEthnicityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Race == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	var raceEthnicity models.UserRaceEthnicity
	err := h.db.QueryRow(`
		UPDATE user_race_ethnicity
		SET race = $1
		WHERE user_id = $2
		RETURNING user_id, race, created_at, updated_at`,
		pq.Array(req.Race), userID,
	).Scan(&raceEthnicity.UserID, &raceEthnicity.Race,
		&raceEthnicity.CreatedAt, &raceEthnicity.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Race/ethnicity not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating race/ethnicity"})
		return
	}

	c.JSON(http.StatusOK, raceEthnicity)
}

func (h *ProfileHandler) DeleteUserRaceEthnicity(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.db.Exec("DELETE FROM user_race_ethnicity WHERE user_id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting race/ethnicity"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Race/ethnicity not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Race/ethnicity deleted successfully"})
}

// Economic Info Handlers

func (h *ProfileHandler) GetEconomicInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var economicInfo models.EconomicInfo
	err := h.db.QueryRow(`
		SELECT user_id, for_current_political_structure, for_capitalism, for_laws,
		       goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		       additional_text, created_at, updated_at
		FROM economic_info WHERE user_id = $1`,
		userID,
	).Scan(&economicInfo.UserID, &economicInfo.ForCurrentPoliticalStructure,
		&economicInfo.ForCapitalism, &economicInfo.ForLaws, &economicInfo.GoodsServices,
		&economicInfo.Affiliations, &economicInfo.SupportOfAltEcon, &economicInfo.SupportAltComm,
		&economicInfo.AdditionalText, &economicInfo.CreatedAt, &economicInfo.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Economic info not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, economicInfo)
}

func (h *ProfileHandler) CreateEconomicInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateEconomicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if economic info already exists
	var existingEconomicInfo models.EconomicInfo
	err := h.db.QueryRow("SELECT user_id FROM economic_info WHERE user_id = $1", userID).Scan(&existingEconomicInfo.UserID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Economic info already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var economicInfo models.EconomicInfo
	err = h.db.QueryRow(`
		INSERT INTO economic_info
		(user_id, for_current_political_structure, for_capitalism, for_laws,
		 goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING user_id, for_current_political_structure, for_capitalism, for_laws,
		          goods_services, affiliations, support_of_alt_econ, support_alt_comm,
		          additional_text, created_at, updated_at`,
		userID, req.ForCurrentPoliticalStructure, req.ForCapitalism, req.ForLaws,
		pq.Array(req.GoodsServices), pq.Array(req.Affiliations), req.SupportOfAltEcon,
		req.SupportAltComm, req.AdditionalText,
	).Scan(&economicInfo.UserID, &economicInfo.ForCurrentPoliticalStructure,
		&economicInfo.ForCapitalism, &economicInfo.ForLaws, &economicInfo.GoodsServices,
		&economicInfo.Affiliations, &economicInfo.SupportOfAltEcon, &economicInfo.SupportAltComm,
		&economicInfo.AdditionalText, &economicInfo.CreatedAt, &economicInfo.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating economic info"})
		return
	}

	c.JSON(http.StatusCreated, economicInfo)
}

func (h *ProfileHandler) UpdateEconomicInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateEconomicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE economic_info SET "
	args := []interface{}{}
	argCount := 1

	if req.ForCurrentPoliticalStructure != nil {
		query += "for_current_political_structure = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.ForCurrentPoliticalStructure)
		argCount++
	}
	if req.ForCapitalism != nil {
		query += "for_capitalism = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.ForCapitalism)
		argCount++
	}
	if req.ForLaws != nil {
		query += "for_laws = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.ForLaws)
		argCount++
	}
	if req.GoodsServices != nil {
		query += "goods_services = $" + string(rune(argCount+'0')) + ", "
		args = append(args, pq.Array(req.GoodsServices))
		argCount++
	}
	if req.Affiliations != nil {
		query += "affiliations = $" + string(rune(argCount+'0')) + ", "
		args = append(args, pq.Array(req.Affiliations))
		argCount++
	}
	if req.SupportOfAltEcon != nil {
		query += "support_of_alt_econ = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.SupportOfAltEcon)
		argCount++
	}
	if req.SupportAltComm != nil {
		query += "support_alt_comm = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.SupportAltComm)
		argCount++
	}
	if req.AdditionalText != nil {
		query += "additional_text = $" + string(rune(argCount+'0')) + ", "
		args = append(args, *req.AdditionalText)
		argCount++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]
	query += " WHERE user_id = $" + string(rune(argCount+'0')) + " RETURNING user_id, for_current_political_structure, for_capitalism, for_laws, goods_services, affiliations, support_of_alt_econ, support_alt_comm, additional_text, created_at, updated_at"
	args = append(args, userID)

	var economicInfo models.EconomicInfo
	err := h.db.QueryRow(query, args...).Scan(
		&economicInfo.UserID, &economicInfo.ForCurrentPoliticalStructure,
		&economicInfo.ForCapitalism, &economicInfo.ForLaws, &economicInfo.GoodsServices,
		&economicInfo.Affiliations, &economicInfo.SupportOfAltEcon, &economicInfo.SupportAltComm,
		&economicInfo.AdditionalText, &economicInfo.CreatedAt, &economicInfo.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Economic info not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating economic info"})
		return
	}

	c.JSON(http.StatusOK, economicInfo)
}

func (h *ProfileHandler) DeleteEconomicInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.db.Exec("DELETE FROM economic_info WHERE user_id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting economic info"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Economic info not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Economic info deleted successfully"})
}
