package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to database successfully!")

	// Seed Users
	log.Println("Seeding users...")
	if err := seedUsers(db); err != nil {
		log.Fatal("Failed to seed users:", err)
	}

	// Ensure category column exists (for databases created before this column was added)
	log.Println("Ensuring schema is up to date...")
	if err := ensureSchema(db); err != nil {
		log.Fatal("Failed to update schema:", err)
	}

	// Seed Ballots
	log.Println("Seeding ballots...")
	if err := seedBallots(db); err != nil {
		log.Fatal("Failed to seed ballots:", err)
	}

	// Seed Ballot Items
	log.Println("Seeding ballot items...")
	if err := seedBallotItems(db); err != nil {
		log.Fatal("Failed to seed ballot items:", err)
	}

	// Seed Votes
	log.Println("Seeding votes...")
	if err := seedVotes(db); err != nil {
		log.Fatal("Failed to seed votes:", err)
	}

	log.Println("Database seeded successfully!")
}

func ensureSchema(db *sql.DB) error {
	// Add category column to ballots if it doesn't exist
	_, err := db.Exec(`
		ALTER TABLE ballots
		ADD COLUMN IF NOT EXISTS category VARCHAR(100)
	`)
	if err != nil {
		return fmt.Errorf("failed to add category column: %v", err)
	}
	log.Println("✓ Schema verified/updated")
	return nil
}

func seedUsers(db *sql.DB) error {
	users := []struct {
		username string
		email    string
		password string
	}{
		{"alice_smith", "alice.smith@example.com", "password123"},
		{"bob_jones", "bob.jones@example.com", "securepass456"},
	}

	for _, user := range users {
		hashedPassword, err := HashPassword(user.password)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %v", user.username, err)
		}

		query := `
			INSERT INTO users (username, email, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (email) DO NOTHING
		`

		now := time.Now()
		_, err = db.Exec(query, user.username, user.email, hashedPassword, now, now)
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %v", user.username, err)
		}

		log.Printf("✓ User created: %s (%s)", user.username, user.email)
	}

	return nil
}

func seedBallots(db *sql.DB) error {
	// Get user IDs
	var userID1, userID2 int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", "alice.smith@example.com").Scan(&userID1)
	if err != nil {
		return fmt.Errorf("failed to get user ID for alice: %v", err)
	}

	err = db.QueryRow("SELECT id FROM users WHERE email = $1", "bob.jones@example.com").Scan(&userID2)
	if err != nil {
		return fmt.Errorf("failed to get user ID for bob: %v", err)
	}

	ballots := []struct {
		creatorID   int
		title       string
		description string
		category    string
		isActive    bool
	}{
		{
			creatorID:   userID1,
			title:       "Executive Branch Budget Priorities",
			description: "Vote on the top budget priority for the Executive Department in the upcoming fiscal year.",
			category:    "executive",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Federal Court System Reform",
			description: "Should the federal court system be expanded to address case backlogs?",
			category:    "judicial",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "House of Representatives Voting Rules",
			description: "Proposed changes to procedural rules for House votes on legislation.",
			category:    "house",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Senate Filibuster Reform",
			description: "Should the Senate modify or eliminate the filibuster rule?",
			category:    "senate",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Department of Education Focus Areas",
			description: "What should be the primary focus of the Department of Education?",
			category:    "executive",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Supreme Court Term Limits",
			description: "Should Supreme Court justices have term limits?",
			category:    "judicial",
			isActive:    true,
		},
	}

	for _, ballot := range ballots {
		query := `
			INSERT INTO ballots (creator_id, title, description, category, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT DO NOTHING
		`

		now := time.Now()
		_, err := db.Exec(query, ballot.creatorID, ballot.title, ballot.description, ballot.category, ballot.isActive, now, now)
		if err != nil {
			return fmt.Errorf("failed to insert ballot '%s': %v", ballot.title, err)
		}

		log.Printf("✓ Ballot created: %s (category: %s)", ballot.title, ballot.category)
	}

	return nil
}

func seedBallotItems(db *sql.DB) error {
	// Get ballot IDs
	ballotTitles := []string{
		"Executive Branch Budget Priorities",
		"Federal Court System Reform",
		"House of Representatives Voting Rules",
		"Senate Filibuster Reform",
		"Department of Education Focus Areas",
		"Supreme Court Term Limits",
	}

	ballotIDs := make(map[string]int)
	for _, title := range ballotTitles {
		var id int
		err := db.QueryRow("SELECT id FROM ballots WHERE title = $1", title).Scan(&id)
		if err != nil {
			return fmt.Errorf("failed to get ballot ID for '%s': %v", title, err)
		}
		ballotIDs[title] = id
	}

	ballotItems := []struct {
		ballotTitle string
		title       string
		description string
	}{
		// Executive Branch Budget Priorities
		{"Executive Branch Budget Priorities", "National Security", "Increase funding for defense and intelligence agencies"},
		{"Executive Branch Budget Priorities", "Infrastructure", "Invest in roads, bridges, and public transportation"},
		{"Executive Branch Budget Priorities", "Healthcare", "Expand healthcare programs and medical research"},
		{"Executive Branch Budget Priorities", "Education", "Increase funding for public schools and student aid"},

		// Federal Court System Reform
		{"Federal Court System Reform", "Yes - Expand Courts", "Add more federal judges to reduce case backlogs"},
		{"Federal Court System Reform", "No - Maintain Current System", "Keep the current number of federal judges"},
		{"Federal Court System Reform", "Reform Case Management", "Improve efficiency without adding judges"},

		// House of Representatives Voting Rules
		{"House of Representatives Voting Rules", "Approve Proposed Changes", "Implement the new procedural rules"},
		{"House of Representatives Voting Rules", "Reject Changes", "Keep current voting procedures"},
		{"House of Representatives Voting Rules", "Modify and Revote", "Amend the proposal and vote again"},

		// Senate Filibuster Reform
		{"Senate Filibuster Reform", "Eliminate Filibuster", "Remove the filibuster rule entirely"},
		{"Senate Filibuster Reform", "Require Talking Filibuster", "Senators must speak continuously to maintain filibuster"},
		{"Senate Filibuster Reform", "Lower Threshold to 55 Votes", "Reduce cloture threshold from 60 to 55 votes"},
		{"Senate Filibuster Reform", "Keep Current Rules", "Maintain 60-vote threshold requirement"},

		// Department of Education Focus Areas
		{"Department of Education Focus Areas", "STEM Education", "Prioritize science, technology, engineering, and math programs"},
		{"Department of Education Focus Areas", "Vocational Training", "Expand career and technical education programs"},
		{"Department of Education Focus Areas", "Student Debt Relief", "Focus on reducing student loan burden"},
		{"Department of Education Focus Areas", "Early Childhood Education", "Invest in pre-K and kindergarten programs"},

		// Supreme Court Term Limits
		{"Supreme Court Term Limits", "Yes - 18 Year Terms", "Implement 18-year term limits for justices"},
		{"Supreme Court Term Limits", "Yes - Different Term Length", "Implement term limits of a different duration"},
		{"Supreme Court Term Limits", "No - Lifetime Appointments", "Maintain current lifetime appointment system"},
	}

	for _, item := range ballotItems {
		ballotID, ok := ballotIDs[item.ballotTitle]
		if !ok {
			return fmt.Errorf("ballot ID not found for '%s'", item.ballotTitle)
		}

		query := `
			INSERT INTO ballot_items (ballot_id, title, description, vote_count)
			VALUES ($1, $2, $3, 0)
			ON CONFLICT DO NOTHING
		`

		_, err := db.Exec(query, ballotID, item.title, item.description)
		if err != nil {
			return fmt.Errorf("failed to insert ballot item '%s': %v", item.title, err)
		}

		log.Printf("✓ Ballot item created: %s", item.title)
	}

	return nil
}

func seedVotes(db *sql.DB) error {
	// Get user IDs
	var userID1, userID2 int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", "alice.smith@example.com").Scan(&userID1)
	if err != nil {
		return fmt.Errorf("failed to get user ID for alice: %v", err)
	}

	err = db.QueryRow("SELECT id FROM users WHERE email = $1", "bob.jones@example.com").Scan(&userID2)
	if err != nil {
		return fmt.Errorf("failed to get user ID for bob: %v", err)
	}

	// Get ballot IDs
	var execBallotID, judicialBallotID int
	err = db.QueryRow("SELECT id FROM ballots WHERE title = $1", "Executive Branch Budget Priorities").Scan(&execBallotID)
	if err != nil {
		return fmt.Errorf("failed to get ballot ID for executive ballot: %v", err)
	}

	err = db.QueryRow("SELECT id FROM ballots WHERE title = $1", "Federal Court System Reform").Scan(&judicialBallotID)
	if err != nil {
		return fmt.Errorf("failed to get ballot ID for judicial ballot: %v", err)
	}

	// Get ballot item IDs
	var infrastructureItemID, expandCourtsItemID int
	err = db.QueryRow("SELECT id FROM ballot_items WHERE title = $1 AND ballot_id = $2", "Infrastructure", execBallotID).Scan(&infrastructureItemID)
	if err != nil {
		return fmt.Errorf("failed to get ballot item ID for Infrastructure: %v", err)
	}

	err = db.QueryRow("SELECT id FROM ballot_items WHERE title = $1 AND ballot_id = $2", "Yes - Expand Courts", judicialBallotID).Scan(&expandCourtsItemID)
	if err != nil {
		return fmt.Errorf("failed to get ballot item ID for Expand Courts: %v", err)
	}

	votes := []struct {
		userID       int
		ballotID     int
		ballotItemID int
	}{
		// Alice votes for Infrastructure in executive ballot
		{userID1, execBallotID, infrastructureItemID},
		// Bob votes for Expand Courts in judicial ballot
		{userID2, judicialBallotID, expandCourtsItemID},
	}

	for i, vote := range votes {
		// Insert vote
		query := `
			INSERT INTO votes (user_id, ballot_id, ballot_item_id, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, ballot_id) DO NOTHING
		`

		now := time.Now()
		_, err := db.Exec(query, vote.userID, vote.ballotID, vote.ballotItemID, now)
		if err != nil {
			return fmt.Errorf("failed to insert vote #%d: %v", i+1, err)
		}

		// Update vote count
		updateQuery := `
			UPDATE ballot_items
			SET vote_count = vote_count + 1
			WHERE id = $1
		`
		_, err = db.Exec(updateQuery, vote.ballotItemID)
		if err != nil {
			return fmt.Errorf("failed to update vote count for item %d: %v", vote.ballotItemID, err)
		}

		log.Printf("✓ Vote recorded: User %d voted on ballot %d", vote.userID, vote.ballotID)
	}

	return nil
}
