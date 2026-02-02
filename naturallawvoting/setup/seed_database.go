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

	// Add superstate column to ballots if it doesn't exist
	_, err = db.Exec(`
		ALTER TABLE ballots
		ADD COLUMN IF NOT EXISTS superstate VARCHAR(100)
	`)
	if err != nil {
		return fmt.Errorf("failed to add superstate column: %v", err)
	}

	// Add state column to ballots if it doesn't exist
	_, err = db.Exec(`
		ALTER TABLE ballots
		ADD COLUMN IF NOT EXISTS state VARCHAR(100)
	`)
	if err != nil {
		return fmt.Errorf("failed to add state column: %v", err)
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
		superstate  string
		state       string
		isActive    bool
	}{
		// Federal/National level ballots
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

		// NEW ENGLAND SUPER STATE (NE SS) - Ballots
		{
			creatorID:   userID1,
			title:       "Vermont State Representative Confidence Vote",
			description: "Vote of confidence for Vermont's state representatives in the Common-Law Republic.",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "vermont",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Vermont Environmental Policy Initiative",
			description: "Should Vermont prioritize renewable energy investment over the next decade?",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "vermont",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Rhode Island Coastal Protection Measures",
			description: "Vote on proposed coastal protection and climate resilience measures for Rhode Island.",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "rhode-island",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Rhode Island Education Funding Reform",
			description: "Proposed changes to education funding distribution in Rhode Island.",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "rhode-island",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Maine Fishing Rights Protection",
			description: "Should Maine strengthen protections for local fishing communities?",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "maine",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "New Hampshire Tax Policy Reform",
			description: "Vote on proposed changes to New Hampshire's state tax structure.",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "new-hampshire",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Connecticut Infrastructure Investment",
			description: "Major infrastructure investment priorities for Connecticut's transportation system.",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "connecticut",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Massachusetts Healthcare Expansion",
			description: "Should Massachusetts expand its state healthcare program?",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "massachusetts",
			isActive:    true,
		},

		// NEW YORK SUPER STATE Ballots
		{
			creatorID:   userID1,
			title:       "New York City Transit Reform",
			description: "Vote on proposed reforms to New York City's public transit system.",
			category:    "local-civil",
			superstate:  "new-york",
			state:       "new-york-city",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Upstate New York Agricultural Support",
			description: "Should New York increase support for upstate agricultural communities?",
			category:    "local-civil",
			superstate:  "new-york",
			state:       "upstate-new-york",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Long Island Environmental Initiative",
			description: "Vote on environmental protection measures for Long Island coastal areas.",
			category:    "local-civil",
			superstate:  "new-york",
			state:       "long-island",
			isActive:    true,
		},

		// JERSEY-PENN SUPER STATE Ballots
		{
			creatorID:   userID2,
			title:       "New Jersey Shore Protection Act",
			description: "Vote on enhanced protection measures for New Jersey's coastline.",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "new-jersey",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "New Jersey Transportation Funding",
			description: "Should New Jersey increase funding for public transportation infrastructure?",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "new-jersey",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Pennsylvania Energy Transition Plan",
			description: "Vote on Pennsylvania's proposed transition to renewable energy sources.",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "pennsylvania",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Pennsylvania Rural Development Initiative",
			description: "Should Pennsylvania invest more in rural community development?",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "pennsylvania",
			isActive:    true,
		},

		// GREAT LAKES SUPER STATE Ballots
		{
			creatorID:   userID1,
			title:       "Michigan Great Lakes Protection",
			description: "Vote on enhanced protection measures for Great Lakes water quality.",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "michigan",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Ohio Manufacturing Renaissance",
			description: "Should Ohio increase incentives for manufacturing sector growth?",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "ohio",
			isActive:    true,
		},

		// TEXAS SUPER STATE Ballots
		{
			creatorID:   userID1,
			title:       "Texas Energy Grid Independence",
			description: "Vote on measures to strengthen Texas energy grid resilience.",
			category:    "local-civil",
			superstate:  "texas",
			state:       "texas",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Texas Border Community Support",
			description: "Should Texas increase funding for border community infrastructure?",
			category:    "local-civil",
			superstate:  "texas",
			state:       "texas",
			isActive:    true,
		},

		// CALIFORNIA SUPER STATE Ballots
		{
			creatorID:   userID1,
			title:       "California Water Conservation Initiative",
			description: "Vote on statewide water conservation and infrastructure measures.",
			category:    "local-civil",
			superstate:  "california",
			state:       "california",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "California Wildfire Prevention Funding",
			description: "Should California increase funding for wildfire prevention and response?",
			category:    "local-civil",
			superstate:  "california",
			state:       "california",
			isActive:    true,
		},
	}

	for _, ballot := range ballots {
		query := `
			INSERT INTO ballots (creator_id, title, description, category, superstate, state, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT DO NOTHING
		`

		now := time.Now()
		_, err := db.Exec(query, ballot.creatorID, ballot.title, ballot.description, ballot.category, ballot.superstate, ballot.state, ballot.isActive, now, now)
		if err != nil {
			return fmt.Errorf("failed to insert ballot '%s': %v", ballot.title, err)
		}

		if ballot.superstate != "" {
			log.Printf("✓ Ballot created: %s (superstate: %s, state: %s)", ballot.title, ballot.superstate, ballot.state)
		} else {
			log.Printf("✓ Ballot created: %s (category: %s)", ballot.title, ballot.category)
		}
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
		// New England Super State
		"Vermont State Representative Confidence Vote",
		"Vermont Environmental Policy Initiative",
		"Rhode Island Coastal Protection Measures",
		"Rhode Island Education Funding Reform",
		"Maine Fishing Rights Protection",
		"New Hampshire Tax Policy Reform",
		"Connecticut Infrastructure Investment",
		"Massachusetts Healthcare Expansion",
		// New York Super State
		"New York City Transit Reform",
		"Upstate New York Agricultural Support",
		"Long Island Environmental Initiative",
		// Jersey-Penn Super State
		"New Jersey Shore Protection Act",
		"New Jersey Transportation Funding",
		"Pennsylvania Energy Transition Plan",
		"Pennsylvania Rural Development Initiative",
		// Great Lakes Super State
		"Michigan Great Lakes Protection",
		"Ohio Manufacturing Renaissance",
		// Texas Super State
		"Texas Energy Grid Independence",
		"Texas Border Community Support",
		// California Super State
		"California Water Conservation Initiative",
		"California Wildfire Prevention Funding",
	}

	ballotIDs := make(map[string]int)
	for _, title := range ballotTitles {
		var id int
		err := db.QueryRow("SELECT id FROM ballots WHERE title = $1", title).Scan(&id)
		if err != nil {
			log.Printf("Warning: Could not find ballot '%s', skipping items", title)
			continue
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

		// =============== LOCAL CIVIL GOVERNMENT BALLOTS ===============

		// Vermont State Representative Confidence Vote
		{"Vermont State Representative Confidence Vote", "Full Confidence", "Express full confidence in state representatives"},
		{"Vermont State Representative Confidence Vote", "Partial Confidence", "Express partial confidence with reservations"},
		{"Vermont State Representative Confidence Vote", "No Confidence", "Express no confidence in state representatives"},

		// Vermont Environmental Policy Initiative
		{"Vermont Environmental Policy Initiative", "Yes - Prioritize Renewables", "Commit Vermont to 100% renewable energy by 2035"},
		{"Vermont Environmental Policy Initiative", "Moderate Approach", "Balanced approach with gradual transition"},
		{"Vermont Environmental Policy Initiative", "No - Maintain Current Policy", "Continue current energy policies"},

		// Rhode Island Coastal Protection Measures
		{"Rhode Island Coastal Protection Measures", "Comprehensive Protection Plan", "Full implementation of coastal protection measures"},
		{"Rhode Island Coastal Protection Measures", "Targeted Protection", "Focus on highest-risk areas only"},
		{"Rhode Island Coastal Protection Measures", "Study Further", "Commission additional studies before action"},

		// Rhode Island Education Funding Reform
		{"Rhode Island Education Funding Reform", "Increase State Funding", "Increase state share of education funding"},
		{"Rhode Island Education Funding Reform", "Local Control", "Maintain local funding control"},
		{"Rhode Island Education Funding Reform", "Hybrid Model", "Balance between state and local funding"},

		// Maine Fishing Rights Protection
		{"Maine Fishing Rights Protection", "Strong Protections", "Implement strong protections for local fishing communities"},
		{"Maine Fishing Rights Protection", "Moderate Protections", "Balance fishing rights with broader interests"},
		{"Maine Fishing Rights Protection", "Current Regulations", "Maintain current regulatory framework"},

		// New Hampshire Tax Policy Reform
		{"New Hampshire Tax Policy Reform", "No Income Tax", "Maintain New Hampshire's no income tax policy"},
		{"New Hampshire Tax Policy Reform", "Modest Income Tax", "Introduce modest income tax to fund services"},
		{"New Hampshire Tax Policy Reform", "Sales Tax Instead", "Implement sales tax as alternative revenue source"},

		// Connecticut Infrastructure Investment
		{"Connecticut Infrastructure Investment", "Rail Priority", "Prioritize commuter rail improvements"},
		{"Connecticut Infrastructure Investment", "Highway Priority", "Focus on highway and bridge repairs"},
		{"Connecticut Infrastructure Investment", "Balanced Approach", "Equal investment in all transportation modes"},

		// Massachusetts Healthcare Expansion
		{"Massachusetts Healthcare Expansion", "Universal Coverage", "Expand to cover all state residents"},
		{"Massachusetts Healthcare Expansion", "Targeted Expansion", "Focus on uninsured and underinsured populations"},
		{"Massachusetts Healthcare Expansion", "Private Market", "Encourage private market solutions"},

		// New York City Transit Reform
		{"New York City Transit Reform", "Major Investment", "Significant investment in modernization"},
		{"New York City Transit Reform", "Incremental Improvements", "Gradual improvements to existing system"},
		{"New York City Transit Reform", "Private Partnership", "Public-private partnerships for transit improvement"},

		// Upstate New York Agricultural Support
		{"Upstate New York Agricultural Support", "Increase Support", "Significantly increase agricultural subsidies"},
		{"Upstate New York Agricultural Support", "Targeted Support", "Focus support on small and family farms"},
		{"Upstate New York Agricultural Support", "Market Solutions", "Reduce subsidies, focus on market access"},

		// Long Island Environmental Initiative
		{"Long Island Environmental Initiative", "Full Protection", "Comprehensive coastal and environmental protection"},
		{"Long Island Environmental Initiative", "Economic Balance", "Balance environmental and economic interests"},
		{"Long Island Environmental Initiative", "Local Control", "Let local communities decide protection levels"},

		// New Jersey Shore Protection Act
		{"New Jersey Shore Protection Act", "Maximum Protection", "Implement strongest possible coastal protections"},
		{"New Jersey Shore Protection Act", "Balanced Approach", "Balance protection with beach access and tourism"},
		{"New Jersey Shore Protection Act", "Property Rights Focus", "Prioritize private property rights"},

		// New Jersey Transportation Funding
		{"New Jersey Transportation Funding", "Increase Funding", "Significantly increase transit funding"},
		{"New Jersey Transportation Funding", "Moderate Increase", "Modest funding increase for critical projects"},
		{"New Jersey Transportation Funding", "Efficiency Focus", "Improve efficiency before adding funding"},

		// Pennsylvania Energy Transition Plan
		{"Pennsylvania Energy Transition Plan", "Rapid Transition", "Fast transition to renewable energy"},
		{"Pennsylvania Energy Transition Plan", "Gradual Transition", "Phased approach protecting energy jobs"},
		{"Pennsylvania Energy Transition Plan", "Energy Independence", "Focus on domestic energy of all types"},

		// Pennsylvania Rural Development Initiative
		{"Pennsylvania Rural Development Initiative", "Major Investment", "Significant investment in rural infrastructure"},
		{"Pennsylvania Rural Development Initiative", "Broadband Focus", "Prioritize rural broadband expansion"},
		{"Pennsylvania Rural Development Initiative", "Agricultural Focus", "Focus on supporting agricultural communities"},

		// Michigan Great Lakes Protection
		{"Michigan Great Lakes Protection", "Strongest Protections", "Implement strictest water quality standards"},
		{"Michigan Great Lakes Protection", "Balanced Protections", "Balance environmental and economic needs"},
		{"Michigan Great Lakes Protection", "Current Standards", "Maintain current protection levels"},

		// Ohio Manufacturing Renaissance
		{"Ohio Manufacturing Renaissance", "Major Incentives", "Provide significant tax incentives for manufacturers"},
		{"Ohio Manufacturing Renaissance", "Workforce Training", "Focus on workforce development and training"},
		{"Ohio Manufacturing Renaissance", "Infrastructure Investment", "Invest in infrastructure to attract manufacturing"},

		// Texas Energy Grid Independence
		{"Texas Energy Grid Independence", "Full Independence", "Maintain complete energy grid independence"},
		{"Texas Energy Grid Independence", "Limited Connections", "Allow limited connections with national grid"},
		{"Texas Energy Grid Independence", "Hybrid Approach", "Independent operation with emergency connections"},

		// Texas Border Community Support
		{"Texas Border Community Support", "Major Funding", "Significant increase in border community funding"},
		{"Texas Border Community Support", "Targeted Support", "Focus on specific infrastructure needs"},
		{"Texas Border Community Support", "Federal Partnership", "Work with federal government on shared funding"},

		// California Water Conservation Initiative
		{"California Water Conservation Initiative", "Mandatory Conservation", "Implement mandatory water conservation measures"},
		{"California Water Conservation Initiative", "Incentive-Based", "Focus on incentives for voluntary conservation"},
		{"California Water Conservation Initiative", "Infrastructure Priority", "Prioritize water infrastructure investment"},

		// California Wildfire Prevention Funding
		{"California Wildfire Prevention Funding", "Major Increase", "Significantly increase prevention and response funding"},
		{"California Wildfire Prevention Funding", "Forest Management", "Focus funding on forest management"},
		{"California Wildfire Prevention Funding", "Community Protection", "Prioritize protecting communities over wildlands"},
	}

	for _, item := range ballotItems {
		ballotID, ok := ballotIDs[item.ballotTitle]
		if !ok {
			continue // Skip if ballot wasn't found
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
