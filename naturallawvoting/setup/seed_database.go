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
		// ===============================================================
		// FEDERAL/NATIONAL LEVEL BALLOTS
		// ===============================================================
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

		// ===============================================================
		// 01 - NEW ENGLAND SUPER STATE
		// ===============================================================
		// Vermont
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
		// Rhode Island
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
		// Maine
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
			title:       "Maine Forest Conservation Act",
			description: "Vote on expanding forest conservation areas in Maine.",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "maine",
			isActive:    true,
		},
		// New Hampshire
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
			title:       "New Hampshire Tourism Development",
			description: "Should New Hampshire invest more in tourism infrastructure?",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "new-hampshire",
			isActive:    true,
		},
		// Connecticut
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
			title:       "Connecticut Small Business Support",
			description: "Should Connecticut expand tax credits for small businesses?",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "connecticut",
			isActive:    true,
		},
		// Massachusetts
		{
			creatorID:   userID2,
			title:       "Massachusetts Healthcare Expansion",
			description: "Should Massachusetts expand its state healthcare program?",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "massachusetts",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Massachusetts Tech Innovation Fund",
			description: "Vote on creating a state fund to support technology startups.",
			category:    "local-civil",
			superstate:  "new-england",
			state:       "massachusetts",
			isActive:    true,
		},

		// ===============================================================
		// 02 - NEW YORK SUPER STATE
		// ===============================================================
		// Long Island
		{
			creatorID:   userID1,
			title:       "Long Island Environmental Initiative",
			description: "Vote on environmental protection measures for Long Island coastal areas.",
			category:    "local-civil",
			superstate:  "new-york",
			state:       "long-island",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Long Island Transportation Upgrade",
			description: "Should Long Island expand commuter rail services?",
			category:    "local-civil",
			superstate:  "new-york",
			state:       "long-island",
			isActive:    true,
		},
		// New York City
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
			title:       "NYC Affordable Housing Initiative",
			description: "Should NYC expand affordable housing programs?",
			category:    "local-civil",
			superstate:  "new-york",
			state:       "new-york-city",
			isActive:    true,
		},
		// Upstate New York
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
			title:       "Upstate NY Manufacturing Revival",
			description: "Vote on incentives to revitalize upstate manufacturing.",
			category:    "local-civil",
			superstate:  "new-york",
			state:       "upstate-new-york",
			isActive:    true,
		},

		// ===============================================================
		// 03 - JERSEY-PENN SUPER STATE
		// ===============================================================
		// Washington DC
		{
			creatorID:   userID1,
			title:       "DC Statehood Initiative",
			description: "Vote on advancing DC statehood within the Common-Law Republic framework.",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "washington-dc",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "DC Public Safety Reform",
			description: "Should DC implement new community-based public safety programs?",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "washington-dc",
			isActive:    true,
		},
		// Delaware
		{
			creatorID:   userID2,
			title:       "Delaware Corporate Tax Reform",
			description: "Vote on proposed changes to Delaware's corporate tax structure.",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "delaware",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Delaware Coastal Resilience Plan",
			description: "Should Delaware invest in coastal resilience infrastructure?",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "delaware",
			isActive:    true,
		},
		// Maryland
		{
			creatorID:   userID1,
			title:       "Maryland Chesapeake Bay Protection",
			description: "Vote on enhanced protection measures for the Chesapeake Bay.",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "maryland",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Maryland Education Excellence",
			description: "Should Maryland increase funding for public education?",
			category:    "local-civil",
			superstate:  "jersey-penn",
			state:       "maryland",
			isActive:    true,
		},
		// New Jersey
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
		// Pennsylvania
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

		// ===============================================================
		// 04 - GREAT LAKES SUPER STATE
		// ===============================================================
		// Kentucky
		{
			creatorID:   userID1,
			title:       "Kentucky Coal Transition Fund",
			description: "Vote on establishing a fund to support coal community transitions.",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "kentucky",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Kentucky Rural Healthcare Access",
			description: "Should Kentucky expand rural healthcare facilities?",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "kentucky",
			isActive:    true,
		},
		// Indiana
		{
			creatorID:   userID2,
			title:       "Indiana Manufacturing Investment",
			description: "Vote on incentives to attract manufacturing jobs to Indiana.",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "indiana",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Indiana Education Reform",
			description: "Should Indiana reform its public school funding formula?",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "indiana",
			isActive:    true,
		},
		// Michigan
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
			title:       "Michigan Auto Industry Support",
			description: "Should Michigan increase support for electric vehicle manufacturing?",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "michigan",
			isActive:    true,
		},
		// Ohio
		{
			creatorID:   userID2,
			title:       "Ohio Manufacturing Renaissance",
			description: "Should Ohio increase incentives for manufacturing sector growth?",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "ohio",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Ohio Infrastructure Modernization",
			description: "Vote on a comprehensive infrastructure modernization plan for Ohio.",
			category:    "local-civil",
			superstate:  "great-lakes",
			state:       "ohio",
			isActive:    true,
		},

		// ===============================================================
		// 05 - VIRGINIA-CAROLINA SUPER STATE
		// ===============================================================
		// West Virginia
		{
			creatorID:   userID1,
			title:       "West Virginia Economic Diversification",
			description: "Vote on initiatives to diversify West Virginia's economy.",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "west-virginia",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "West Virginia Broadband Expansion",
			description: "Should West Virginia invest in statewide broadband infrastructure?",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "west-virginia",
			isActive:    true,
		},
		// Virginia
		{
			creatorID:   userID2,
			title:       "Virginia Tech Corridor Development",
			description: "Vote on expanding the Northern Virginia technology corridor.",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "virginia",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Virginia Historic Preservation",
			description: "Should Virginia increase funding for historic preservation?",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "virginia",
			isActive:    true,
		},
		// South Carolina
		{
			creatorID:   userID1,
			title:       "South Carolina Tourism Investment",
			description: "Vote on increasing investment in South Carolina tourism infrastructure.",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "south-carolina",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "South Carolina Port Expansion",
			description: "Should South Carolina expand its port facilities?",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "south-carolina",
			isActive:    true,
		},
		// North Carolina
		{
			creatorID:   userID2,
			title:       "North Carolina Research Triangle Growth",
			description: "Vote on expanding the Research Triangle innovation ecosystem.",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "north-carolina",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "North Carolina Coastal Management",
			description: "Should North Carolina strengthen coastal management policies?",
			category:    "local-civil",
			superstate:  "virginia-carolina",
			state:       "north-carolina",
			isActive:    true,
		},

		// ===============================================================
		// 06 - FLORIDA-GEORGIA SUPER STATE
		// ===============================================================
		// Georgia
		{
			creatorID:   userID1,
			title:       "Georgia Agricultural Innovation",
			description: "Vote on establishing an agricultural innovation center in Georgia.",
			category:    "local-civil",
			superstate:  "florida-georgia",
			state:       "georgia",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Georgia Film Industry Expansion",
			description: "Should Georgia expand tax incentives for film production?",
			category:    "local-civil",
			superstate:  "florida-georgia",
			state:       "georgia",
			isActive:    true,
		},
		// Florida
		{
			creatorID:   userID2,
			title:       "Florida Hurricane Resilience Fund",
			description: "Vote on creating a hurricane resilience infrastructure fund.",
			category:    "local-civil",
			superstate:  "florida-georgia",
			state:       "florida",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Florida Everglades Restoration",
			description: "Should Florida increase funding for Everglades restoration?",
			category:    "local-civil",
			superstate:  "florida-georgia",
			state:       "florida",
			isActive:    true,
		},

		// ===============================================================
		// 07 - MISSISSIPPI VALLEY SUPER STATE
		// ===============================================================
		// Mississippi
		{
			creatorID:   userID1,
			title:       "Mississippi Education Improvement",
			description: "Vote on comprehensive education improvement initiatives.",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "mississippi",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Mississippi River Protection",
			description: "Should Mississippi strengthen river protection measures?",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "mississippi",
			isActive:    true,
		},
		// Arkansas
		{
			creatorID:   userID2,
			title:       "Arkansas Small Business Growth",
			description: "Vote on small business development incentives for Arkansas.",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "arkansas",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Arkansas Natural Resources Conservation",
			description: "Should Arkansas expand natural resources conservation programs?",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "arkansas",
			isActive:    true,
		},
		// Louisiana
		{
			creatorID:   userID1,
			title:       "Louisiana Coastal Restoration",
			description: "Vote on comprehensive coastal restoration funding.",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "louisiana",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Louisiana Energy Industry Transition",
			description: "Should Louisiana support energy industry diversification?",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "louisiana",
			isActive:    true,
		},
		// Alabama
		{
			creatorID:   userID2,
			title:       "Alabama Aerospace Investment",
			description: "Vote on expanding Alabama's aerospace industry.",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "alabama",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Alabama Infrastructure Improvement",
			description: "Should Alabama prioritize rural infrastructure improvements?",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "alabama",
			isActive:    true,
		},
		// Missouri
		{
			creatorID:   userID1,
			title:       "Missouri Agricultural Support",
			description: "Vote on enhanced support for Missouri family farms.",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "missouri",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Missouri River Basin Management",
			description: "Should Missouri improve river basin flood management?",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "missouri",
			isActive:    true,
		},
		// Tennessee
		{
			creatorID:   userID2,
			title:       "Tennessee Music Industry Support",
			description: "Vote on expanding support for Tennessee's music industry.",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "tennessee",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Tennessee Healthcare Access",
			description: "Should Tennessee expand rural healthcare access programs?",
			category:    "local-civil",
			superstate:  "mississippi-valley",
			state:       "tennessee",
			isActive:    true,
		},

		// ===============================================================
		// 08 - NORTH CENTRAL PLAINS SUPER STATE
		// ===============================================================
		// North Dakota
		{
			creatorID:   userID1,
			title:       "North Dakota Energy Development",
			description: "Vote on balanced energy development policies for North Dakota.",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "north-dakota",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "North Dakota Agricultural Innovation",
			description: "Should North Dakota invest in agricultural technology research?",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "north-dakota",
			isActive:    true,
		},
		// South Dakota
		{
			creatorID:   userID2,
			title:       "South Dakota Tourism Development",
			description: "Vote on expanding tourism infrastructure in South Dakota.",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "south-dakota",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "South Dakota Tribal Relations",
			description: "Should South Dakota strengthen tribal government partnerships?",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "south-dakota",
			isActive:    true,
		},
		// Iowa
		{
			creatorID:   userID1,
			title:       "Iowa Renewable Energy Initiative",
			description: "Vote on expanding Iowa's renewable energy programs.",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "iowa",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Iowa Rural Broadband Expansion",
			description: "Should Iowa invest in comprehensive rural broadband?",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "iowa",
			isActive:    true,
		},
		// Minnesota
		{
			creatorID:   userID2,
			title:       "Minnesota Clean Water Initiative",
			description: "Vote on comprehensive water quality protection measures.",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "minnesota",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Minnesota Education Excellence",
			description: "Should Minnesota increase public education funding?",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "minnesota",
			isActive:    true,
		},
		// Wisconsin
		{
			creatorID:   userID1,
			title:       "Wisconsin Dairy Industry Support",
			description: "Vote on support measures for Wisconsin's dairy industry.",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "wisconsin",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Wisconsin Great Lakes Protection",
			description: "Should Wisconsin strengthen Great Lakes water protections?",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "wisconsin",
			isActive:    true,
		},
		// Illinois
		{
			creatorID:   userID2,
			title:       "Illinois Pension Reform",
			description: "Vote on comprehensive pension system reform for Illinois.",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "illinois",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Illinois Infrastructure Renewal",
			description: "Should Illinois prioritize infrastructure renewal projects?",
			category:    "local-civil",
			superstate:  "north-central-plains",
			state:       "illinois",
			isActive:    true,
		},

		// ===============================================================
		// 09 - TEXAS SUPER STATE
		// ===============================================================
		// Texas
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
		{
			creatorID:   userID1,
			title:       "Texas Water Resource Management",
			description: "Vote on comprehensive water resource management plan.",
			category:    "local-civil",
			superstate:  "texas",
			state:       "texas",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Texas Tech Industry Growth",
			description: "Should Texas expand incentives for technology companies?",
			category:    "local-civil",
			superstate:  "texas",
			state:       "texas",
			isActive:    true,
		},

		// ===============================================================
		// 10 - SOUTH WEST SUPER STATE
		// ===============================================================
		// Nebraska
		{
			creatorID:   userID1,
			title:       "Nebraska Agricultural Technology",
			description: "Vote on agricultural technology innovation funding.",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "nebraska",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Nebraska Water Conservation",
			description: "Should Nebraska strengthen water conservation measures?",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "nebraska",
			isActive:    true,
		},
		// New Mexico
		{
			creatorID:   userID2,
			title:       "New Mexico Renewable Energy",
			description: "Vote on expanding New Mexico's renewable energy sector.",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "new-mexico",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "New Mexico Cultural Preservation",
			description: "Should New Mexico increase funding for cultural preservation?",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "new-mexico",
			isActive:    true,
		},
		// Kansas
		{
			creatorID:   userID1,
			title:       "Kansas Wind Energy Expansion",
			description: "Vote on expanding wind energy infrastructure in Kansas.",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "kansas",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Kansas Rural Development",
			description: "Should Kansas invest more in rural community development?",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "kansas",
			isActive:    true,
		},
		// Oklahoma
		{
			creatorID:   userID2,
			title:       "Oklahoma Energy Diversification",
			description: "Vote on energy industry diversification initiatives.",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "oklahoma",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Oklahoma Tribal Partnership",
			description: "Should Oklahoma strengthen economic partnerships with tribal nations?",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "oklahoma",
			isActive:    true,
		},
		// Colorado
		{
			creatorID:   userID1,
			title:       "Colorado Water Rights Management",
			description: "Vote on comprehensive water rights management reform.",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "colorado",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Colorado Affordable Housing",
			description: "Should Colorado expand affordable housing programs?",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "colorado",
			isActive:    true,
		},
		// Arizona
		{
			creatorID:   userID2,
			title:       "Arizona Water Conservation Plan",
			description: "Vote on comprehensive water conservation measures for Arizona.",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "arizona",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Arizona Solar Energy Initiative",
			description: "Should Arizona expand its solar energy infrastructure?",
			category:    "local-civil",
			superstate:  "south-west",
			state:       "arizona",
			isActive:    true,
		},

		// ===============================================================
		// 11 - PACIFIC NW SUPER STATE
		// ===============================================================
		// Wyoming
		{
			creatorID:   userID1,
			title:       "Wyoming Energy Transition",
			description: "Vote on balanced energy transition policies for Wyoming.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "wyoming",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Wyoming Wildlife Conservation",
			description: "Should Wyoming expand wildlife conservation programs?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "wyoming",
			isActive:    true,
		},
		// Alaska
		{
			creatorID:   userID2,
			title:       "Alaska Resource Development",
			description: "Vote on balanced resource development policies for Alaska.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "alaska",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Alaska Native Community Support",
			description: "Should Alaska increase support for Native community infrastructure?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "alaska",
			isActive:    true,
		},
		// Montana
		{
			creatorID:   userID1,
			title:       "Montana Public Lands Management",
			description: "Vote on public lands management policies for Montana.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "montana",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Montana Agricultural Sustainability",
			description: "Should Montana invest in sustainable agriculture programs?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "montana",
			isActive:    true,
		},
		// Hawaii
		{
			creatorID:   userID2,
			title:       "Hawaii Renewable Energy Target",
			description: "Vote on Hawaii's path to 100% renewable energy.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "hawaii",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Hawaii Native Land Rights",
			description: "Should Hawaii strengthen protections for Native Hawaiian lands?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "hawaii",
			isActive:    true,
		},
		// Idaho
		{
			creatorID:   userID1,
			title:       "Idaho Water Rights Reform",
			description: "Vote on water rights reform measures for Idaho.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "idaho",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Idaho Tech Industry Growth",
			description: "Should Idaho expand incentives for technology companies?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "idaho",
			isActive:    true,
		},
		// Nevada
		{
			creatorID:   userID2,
			title:       "Nevada Economic Diversification",
			description: "Vote on economic diversification beyond gaming industry.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "nevada",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Nevada Renewable Energy Investment",
			description: "Should Nevada increase investment in renewable energy?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "nevada",
			isActive:    true,
		},
		// Utah
		{
			creatorID:   userID1,
			title:       "Utah Public Lands Access",
			description: "Vote on public lands access and management policies.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "utah",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Utah Air Quality Improvement",
			description: "Should Utah invest more in air quality improvement?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "utah",
			isActive:    true,
		},
		// Oregon
		{
			creatorID:   userID2,
			title:       "Oregon Forest Management",
			description: "Vote on sustainable forest management policies for Oregon.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "oregon",
			isActive:    true,
		},
		{
			creatorID:   userID1,
			title:       "Oregon Clean Energy Initiative",
			description: "Should Oregon accelerate its clean energy transition?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "oregon",
			isActive:    true,
		},
		// Washington
		{
			creatorID:   userID1,
			title:       "Washington Tech Industry Support",
			description: "Vote on support measures for Washington's technology sector.",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "washington",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "Washington Environmental Protection",
			description: "Should Washington strengthen environmental protection laws?",
			category:    "local-civil",
			superstate:  "pacific-nw",
			state:       "washington",
			isActive:    true,
		},

		// ===============================================================
		// 12 - CALIFORNIA SUPER STATE
		// ===============================================================
		// California
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
		{
			creatorID:   userID1,
			title:       "California Affordable Housing Crisis",
			description: "Vote on comprehensive affordable housing solutions.",
			category:    "local-civil",
			superstate:  "california",
			state:       "california",
			isActive:    true,
		},
		{
			creatorID:   userID2,
			title:       "California High-Speed Rail Completion",
			description: "Should California prioritize high-speed rail project completion?",
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
	// Get all ballot titles that need items
	rows, err := db.Query("SELECT id, title FROM ballots")
	if err != nil {
		return fmt.Errorf("failed to query ballots: %v", err)
	}
	defer rows.Close()

	ballotIDs := make(map[string]int)
	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return fmt.Errorf("failed to scan ballot: %v", err)
		}
		ballotIDs[title] = id
	}

	// Define ballot items for each ballot type
	ballotItems := []struct {
		ballotTitle string
		title       string
		description string
	}{
		// Federal Ballots
		{"Executive Branch Budget Priorities", "National Security", "Increase funding for defense and intelligence agencies"},
		{"Executive Branch Budget Priorities", "Infrastructure", "Invest in roads, bridges, and public transportation"},
		{"Executive Branch Budget Priorities", "Healthcare", "Expand healthcare programs and medical research"},
		{"Executive Branch Budget Priorities", "Education", "Increase funding for public schools and student aid"},

		{"Federal Court System Reform", "Yes - Expand Courts", "Add more federal judges to reduce case backlogs"},
		{"Federal Court System Reform", "No - Maintain Current System", "Keep the current number of federal judges"},
		{"Federal Court System Reform", "Reform Case Management", "Improve efficiency without adding judges"},

		{"House of Representatives Voting Rules", "Approve Proposed Changes", "Implement the new procedural rules"},
		{"House of Representatives Voting Rules", "Reject Changes", "Keep current voting procedures"},
		{"House of Representatives Voting Rules", "Modify and Revote", "Amend the proposal and vote again"},

		{"Senate Filibuster Reform", "Eliminate Filibuster", "Remove the filibuster rule entirely"},
		{"Senate Filibuster Reform", "Require Talking Filibuster", "Senators must speak continuously to maintain filibuster"},
		{"Senate Filibuster Reform", "Lower Threshold to 55 Votes", "Reduce cloture threshold from 60 to 55 votes"},
		{"Senate Filibuster Reform", "Keep Current Rules", "Maintain 60-vote threshold requirement"},

		{"Department of Education Focus Areas", "STEM Education", "Prioritize science, technology, engineering, and math programs"},
		{"Department of Education Focus Areas", "Vocational Training", "Expand career and technical education programs"},
		{"Department of Education Focus Areas", "Student Debt Relief", "Focus on reducing student loan burden"},
		{"Department of Education Focus Areas", "Early Childhood Education", "Invest in pre-K and kindergarten programs"},

		{"Supreme Court Term Limits", "Yes - 18 Year Terms", "Implement 18-year term limits for justices"},
		{"Supreme Court Term Limits", "Yes - Different Term Length", "Implement term limits of a different duration"},
		{"Supreme Court Term Limits", "No - Lifetime Appointments", "Maintain current lifetime appointment system"},
	}

	// Generic options for state-level ballots
	genericStateOptions := []struct {
		suffix      string
		options     []string
		descriptions []string
	}{
		{
			suffix:      "Confidence Vote",
			options:     []string{"Full Confidence", "Partial Confidence", "No Confidence"},
			descriptions: []string{"Express full confidence in representatives", "Express partial confidence with reservations", "Express no confidence in representatives"},
		},
		{
			suffix:      "Initiative",
			options:     []string{"Strongly Support", "Support with Modifications", "Oppose"},
			descriptions: []string{"Fully support the proposed initiative", "Support with suggested modifications", "Oppose the initiative"},
		},
		{
			suffix:      "Measures",
			options:     []string{"Comprehensive Action", "Targeted Action", "Further Study"},
			descriptions: []string{"Implement comprehensive protection measures", "Focus on highest-priority areas only", "Commission additional studies before action"},
		},
		{
			suffix:      "Reform",
			options:     []string{"Major Reform", "Moderate Reform", "Minimal Change"},
			descriptions: []string{"Implement significant structural changes", "Make moderate adjustments to current system", "Keep mostly current structure with minor tweaks"},
		},
		{
			suffix:      "Protection",
			options:     []string{"Strong Protections", "Balanced Approach", "Current Standards"},
			descriptions: []string{"Implement strongest possible protections", "Balance protection with economic interests", "Maintain current protection levels"},
		},
		{
			suffix:      "Investment",
			options:     []string{"Major Investment", "Moderate Investment", "Efficiency Focus"},
			descriptions: []string{"Significant new investment in infrastructure", "Moderate funding increase for priority projects", "Focus on efficiency before new investment"},
		},
		{
			suffix:      "Expansion",
			options:     []string{"Full Expansion", "Targeted Expansion", "Maintain Current"},
			descriptions: []string{"Expand programs to maximum coverage", "Focus expansion on underserved areas", "Maintain current program scope"},
		},
		{
			suffix:      "Support",
			options:     []string{"Increase Support", "Targeted Support", "Market Solutions"},
			descriptions: []string{"Significantly increase government support", "Focus support on specific sectors", "Rely more on market-based solutions"},
		},
		{
			suffix:      "Development",
			options:     []string{"Accelerate Development", "Balanced Growth", "Sustainable Pace"},
			descriptions: []string{"Accelerate development with major investment", "Balance growth with sustainability", "Maintain sustainable development pace"},
		},
		{
			suffix:      "Plan",
			options:     []string{"Comprehensive Plan", "Phased Approach", "Study First"},
			descriptions: []string{"Implement comprehensive statewide plan", "Roll out in phases over time", "Conduct further study before implementation"},
		},
		{
			suffix:      "Fund",
			options:     []string{"Create Fund", "Expand Existing", "Private Partnership"},
			descriptions: []string{"Create new dedicated state fund", "Expand existing funding mechanisms", "Partner with private sector"},
		},
		{
			suffix:      "Act",
			options:     []string{"Pass Act", "Amend Act", "Reject Act"},
			descriptions: []string{"Pass the proposed act as written", "Amend the act with modifications", "Reject the proposed act"},
		},
		{
			suffix:      "Transition",
			options:     []string{"Rapid Transition", "Gradual Transition", "Status Quo"},
			descriptions: []string{"Fast transition to new approach", "Phased transition over time", "Maintain current approach"},
		},
		{
			suffix:      "Management",
			options:     []string{"Enhanced Management", "Balanced Management", "Current Approach"},
			descriptions: []string{"Implement enhanced management policies", "Balance multiple stakeholder interests", "Continue current management approach"},
		},
		{
			suffix:      "Conservation",
			options:     []string{"Strong Conservation", "Balanced Use", "Economic Priority"},
			descriptions: []string{"Prioritize conservation over development", "Balance conservation and economic use", "Prioritize economic development"},
		},
		{
			suffix:      "Access",
			options:     []string{"Expand Access", "Targeted Access", "Current Access"},
			descriptions: []string{"Significantly expand access statewide", "Target access expansion to underserved", "Maintain current access levels"},
		},
		{
			suffix:      "Excellence",
			options:     []string{"Major Investment", "Targeted Improvements", "Efficiency Focus"},
			descriptions: []string{"Major investment in excellence programs", "Targeted improvements in key areas", "Focus on efficiency and outcomes"},
		},
		{
			suffix:      "Partnership",
			options:     []string{"Strong Partnership", "Enhanced Cooperation", "Current Relations"},
			descriptions: []string{"Strengthen partnerships significantly", "Enhance cooperation in key areas", "Maintain current relationship levels"},
		},
		{
			suffix:      "Improvement",
			options:     []string{"Comprehensive Improvement", "Priority Focus", "Incremental Change"},
			descriptions: []string{"Comprehensive improvement across all areas", "Focus on highest priority improvements", "Make incremental changes over time"},
		},
		{
			suffix:      "Rights",
			options:     []string{"Strengthen Rights", "Balanced Approach", "Current Framework"},
			descriptions: []string{"Significantly strengthen protections", "Balance rights with other interests", "Maintain current framework"},
		},
		{
			suffix:      "Target",
			options:     []string{"Aggressive Target", "Moderate Target", "Flexible Approach"},
			descriptions: []string{"Set aggressive targets with deadlines", "Set moderate achievable targets", "Allow flexible approach based on conditions"},
		},
		{
			suffix:      "Crisis",
			options:     []string{"Emergency Action", "Urgent Response", "Measured Response"},
			descriptions: []string{"Declare emergency and take immediate action", "Urgent response with prioritized measures", "Measured response with careful planning"},
		},
		{
			suffix:      "Completion",
			options:     []string{"Prioritize Completion", "Phased Completion", "Reassess Project"},
			descriptions: []string{"Make completion a top priority", "Complete in phases as funding allows", "Reassess project scope and timeline"},
		},
		{
			suffix:      "Restoration",
			options:     []string{"Full Restoration", "Targeted Restoration", "Gradual Restoration"},
			descriptions: []string{"Comprehensive restoration program", "Focus on critical areas first", "Gradual restoration over extended period"},
		},
		{
			suffix:      "Diversification",
			options:     []string{"Active Diversification", "Supported Transition", "Market-Led Change"},
			descriptions: []string{"Active government-led diversification", "Support private sector transition", "Allow market forces to drive change"},
		},
		{
			suffix:      "Technology",
			options:     []string{"Major Investment", "Strategic Investment", "Private Sector Focus"},
			descriptions: []string{"Major public investment in technology", "Strategic investments in key areas", "Focus on private sector innovation"},
		},
		{
			suffix:      "Preservation",
			options:     []string{"Enhanced Preservation", "Targeted Preservation", "Current Levels"},
			descriptions: []string{"Significantly enhance preservation efforts", "Target most endangered resources", "Maintain current preservation levels"},
		},
		{
			suffix:      "Industry",
			options:     []string{"Strong Support", "Balanced Support", "Market Approach"},
			descriptions: []string{"Provide strong industry support", "Balance support with other priorities", "Rely on market-based approaches"},
		},
		{
			suffix:      "Growth",
			options:     []string{"Accelerated Growth", "Sustainable Growth", "Managed Growth"},
			descriptions: []string{"Accelerate growth through incentives", "Focus on sustainable growth", "Carefully manage growth rate"},
		},
		{
			suffix:      "Relations",
			options:     []string{"Strengthen Relations", "Enhanced Cooperation", "Status Quo"},
			descriptions: []string{"Significantly strengthen relationships", "Enhance cooperation in specific areas", "Maintain current relationship"},
		},
		{
			suffix:      "Sustainability",
			options:     []string{"Full Sustainability", "Transition Plan", "Current Practices"},
			descriptions: []string{"Commit to full sustainability practices", "Develop transition plan to sustainability", "Maintain current practices"},
		},
		{
			suffix:      "Housing",
			options:     []string{"Major Program", "Targeted Assistance", "Market Solutions"},
			descriptions: []string{"Create major housing program", "Provide targeted assistance to most in need", "Rely on market-based solutions"},
		},
		{
			suffix:      "Resilience",
			options:     []string{"Comprehensive Resilience", "Priority Investments", "Current Approach"},
			descriptions: []string{"Build comprehensive resilience infrastructure", "Invest in highest priority areas", "Continue current resilience approach"},
		},
		{
			suffix:      "Corridor",
			options:     []string{"Accelerate Development", "Planned Growth", "Organic Growth"},
			descriptions: []string{"Accelerate corridor development", "Follow planned development approach", "Allow organic growth patterns"},
		},
		{
			suffix:      "Modernization",
			options:     []string{"Full Modernization", "Phased Modernization", "Targeted Updates"},
			descriptions: []string{"Comprehensive modernization program", "Modernize in phases over time", "Focus on most critical updates"},
		},
		{
			suffix:      "Renaissance",
			options:     []string{"Major Initiative", "Strategic Focus", "Market-Driven"},
			descriptions: []string{"Launch major renaissance initiative", "Focus on strategic opportunities", "Support market-driven revival"},
		},
		{
			suffix:      "Revival",
			options:     []string{"Active Revival", "Supported Revival", "Natural Recovery"},
			descriptions: []string{"Actively pursue revival through incentives", "Support community-led revival efforts", "Allow natural economic recovery"},
		},
		{
			suffix:      "Quality",
			options:     []string{"Strict Standards", "Balanced Standards", "Current Standards"},
			descriptions: []string{"Implement strictest quality standards", "Balance quality with practicality", "Maintain current quality standards"},
		},
	}

	// Insert the specific federal ballot items
	for _, item := range ballotItems {
		ballotID, ok := ballotIDs[item.ballotTitle]
		if !ok {
			continue
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

	// For state-level ballots, generate generic options based on title pattern
	for title, ballotID := range ballotIDs {
		// Skip if this ballot already has items (federal ballots)
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM ballot_items WHERE ballot_id = $1", ballotID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to count items for ballot %d: %v", ballotID, err)
		}
		if count > 0 {
			continue
		}

		// Find matching option pattern based on title suffix
		var options []string
		var descriptions []string
		foundMatch := false

		for _, pattern := range genericStateOptions {
			if len(title) > len(pattern.suffix) && title[len(title)-len(pattern.suffix):] == pattern.suffix {
				options = pattern.options
				descriptions = pattern.descriptions
				foundMatch = true
				break
			}
		}

		// Default options if no pattern matched
		if !foundMatch {
			options = []string{"Yes - Support", "Yes with Modifications", "No - Oppose"}
			descriptions = []string{
				"Support this measure as proposed",
				"Support with suggested modifications",
				"Oppose this measure",
			}
		}

		// Insert the options
		for i, opt := range options {
			query := `
				INSERT INTO ballot_items (ballot_id, title, description, vote_count)
				VALUES ($1, $2, $3, 0)
				ON CONFLICT DO NOTHING
			`

			_, err := db.Exec(query, ballotID, opt, descriptions[i])
			if err != nil {
				return fmt.Errorf("failed to insert ballot item '%s': %v", opt, err)
			}
			log.Printf("✓ Ballot item created: %s (for %s)", opt, title)
		}
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
