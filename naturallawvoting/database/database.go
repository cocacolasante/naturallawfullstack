package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewConnection() (*DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		host := getEnvWithDefault("DB_HOST", "localhost")
		port := getEnvWithDefault("DB_PORT", "5432")
		user := getEnvWithDefault("DB_USER", "postgres")
		password := getEnvWithDefault("DB_PASSWORD", "password")
		dbname := getEnvWithDefault("DB_NAME", "voting_db")
		sslmode := getEnvWithDefault("DB_SSLMODE", "disable")

		dbURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("Successfully connected to database")
	return &DB{db}, nil
}

func (db *DB) RunMigrations() error {
	schemaSQL := `
-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create ballots table
CREATE TABLE IF NOT EXISTS ballots (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    superstate VARCHAR(100),
    state VARCHAR(100),
    creator_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add superstate and state columns if they don't exist (for existing databases)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'ballots' AND column_name = 'superstate') THEN
        ALTER TABLE ballots ADD COLUMN superstate VARCHAR(100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'ballots' AND column_name = 'state') THEN
        ALTER TABLE ballots ADD COLUMN state VARCHAR(100);
    END IF;
END $$;

-- Create ballot_items table
CREATE TABLE IF NOT EXISTS ballot_items (
    id SERIAL PRIMARY KEY,
    ballot_id INTEGER NOT NULL REFERENCES ballots(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    vote_count INTEGER DEFAULT 0
);

-- Create votes table
CREATE TABLE IF NOT EXISTS votes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ballot_id INTEGER NOT NULL REFERENCES ballots(id) ON DELETE CASCADE,
    ballot_item_id INTEGER NOT NULL REFERENCES ballot_items(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, ballot_id)
);

-- Create user_profiles table
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) PRIMARY KEY REFERENCES users(email) ON DELETE CASCADE,
    full_name VARCHAR(255),
    birthday DATE,
    gender VARCHAR(50),
    mothers_maiden_name VARCHAR(100),
    phone_number VARCHAR(20),
    additional_emails TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create user_addresses table
CREATE TABLE IF NOT EXISTS user_addresses (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    street_number VARCHAR(20),
    street_name VARCHAR(255),
    address_line_2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(50),
    zip_code VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create user_political_affiliations table
CREATE TABLE IF NOT EXISTS user_political_affiliations (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    party_affiliation VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create user_religious_affiliations table
CREATE TABLE IF NOT EXISTS user_religious_affiliations (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    religion VARCHAR(100),
    supporting_religion INTEGER CHECK (supporting_religion >= 0 AND supporting_religion <= 10),
    religious_services_types TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create user_race_ethnicity table
CREATE TABLE IF NOT EXISTS user_race_ethnicity (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    race TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create economic_info table
CREATE TABLE IF NOT EXISTS economic_info (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    for_current_political_structure VARCHAR(255),
    for_capitalism VARCHAR(255),
    for_laws VARCHAR(255),
    goods_services TEXT[],
    affiliations TEXT[],
    support_of_alt_econ VARCHAR(255),
    support_alt_comm VARCHAR(255),
    additional_text VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_ballots_creator_id ON ballots(creator_id);
CREATE INDEX IF NOT EXISTS idx_ballots_superstate ON ballots(superstate);
CREATE INDEX IF NOT EXISTS idx_ballots_state ON ballots(state);
CREATE INDEX IF NOT EXISTS idx_ballots_category ON ballots(category);
CREATE INDEX IF NOT EXISTS idx_ballot_items_ballot_id ON ballot_items(ballot_id);
CREATE INDEX IF NOT EXISTS idx_votes_user_id ON votes(user_id);
CREATE INDEX IF NOT EXISTS idx_votes_ballot_id ON votes(ballot_id);
CREATE INDEX IF NOT EXISTS idx_votes_ballot_item_id ON votes(ballot_item_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_ballots_updated_at ON ballots;
CREATE TRIGGER update_ballots_updated_at BEFORE UPDATE ON ballots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_profiles_updated_at ON user_profiles;
CREATE TRIGGER update_user_profiles_updated_at BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_addresses_updated_at ON user_addresses;
CREATE TRIGGER update_user_addresses_updated_at BEFORE UPDATE ON user_addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_political_affiliations_updated_at ON user_political_affiliations;
CREATE TRIGGER update_user_political_affiliations_updated_at BEFORE UPDATE ON user_political_affiliations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_religious_affiliations_updated_at ON user_religious_affiliations;
CREATE TRIGGER update_user_religious_affiliations_updated_at BEFORE UPDATE ON user_religious_affiliations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_race_ethnicity_updated_at ON user_race_ethnicity;
CREATE TRIGGER update_user_race_ethnicity_updated_at BEFORE UPDATE ON user_race_ethnicity
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_economic_info_updated_at ON economic_info;
CREATE TRIGGER update_economic_info_updated_at BEFORE UPDATE ON economic_info
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
`

	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}