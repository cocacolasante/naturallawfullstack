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

-- Add columns if they don't exist (for existing databases)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'ballots' AND column_name = 'category') THEN
        ALTER TABLE ballots ADD COLUMN category VARCHAR(100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'ballots' AND column_name = 'superstate') THEN
        ALTER TABLE ballots ADD COLUMN superstate VARCHAR(100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'ballots' AND column_name = 'state') THEN
        ALTER TABLE ballots ADD COLUMN state VARCHAR(100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'ballots' AND column_name = 'district') THEN
        ALTER TABLE ballots ADD COLUMN district VARCHAR(100) DEFAULT '';
    END IF;
END $$;

-- Create ballot_items table
CREATE TABLE IF NOT EXISTS ballot_items (
    id SERIAL PRIMARY KEY,
    ballot_id INTEGER NOT NULL REFERENCES ballots(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT
);

-- Create votes table: one row per (user, ballot_item) with a 0-100 score
CREATE TABLE IF NOT EXISTS votes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ballot_id INTEGER NOT NULL REFERENCES ballots(id) ON DELETE CASCADE,
    ballot_item_id INTEGER NOT NULL REFERENCES ballot_items(id) ON DELETE CASCADE,
    score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, ballot_item_id)
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
CREATE INDEX IF NOT EXISTS idx_ballots_district ON ballots(district);
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

DROP TRIGGER IF EXISTS update_votes_updated_at ON votes;
CREATE TRIGGER update_votes_updated_at BEFORE UPDATE ON votes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trust votes: one score (1-100) per voter-subject pair
CREATE TABLE IF NOT EXISTS trust_votes (
    id SERIAL PRIMARY KEY,
    voter_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score INTEGER NOT NULL CHECK (score >= 1 AND score <= 100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(voter_id, subject_id)
);

-- Profile visibility thresholds per section (0 = always visible)
CREATE TABLE IF NOT EXISTS profile_visibility (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    info_threshold INTEGER NOT NULL DEFAULT 0 CHECK (info_threshold >= 0 AND info_threshold <= 100),
    address_threshold INTEGER NOT NULL DEFAULT 0 CHECK (address_threshold >= 0 AND address_threshold <= 100),
    political_threshold INTEGER NOT NULL DEFAULT 0 CHECK (political_threshold >= 0 AND political_threshold <= 100),
    religious_threshold INTEGER NOT NULL DEFAULT 0 CHECK (religious_threshold >= 0 AND religious_threshold <= 100),
    race_ethnicity_threshold INTEGER NOT NULL DEFAULT 0 CHECK (race_ethnicity_threshold >= 0 AND race_ethnicity_threshold <= 100),
    economic_threshold INTEGER NOT NULL DEFAULT 0 CHECK (economic_threshold >= 0 AND economic_threshold <= 100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trust_votes_subject ON trust_votes(subject_id);
CREATE INDEX IF NOT EXISTS idx_trust_votes_voter ON trust_votes(voter_id);

DROP TRIGGER IF EXISTS update_trust_votes_updated_at ON trust_votes;
CREATE TRIGGER update_trust_votes_updated_at BEFORE UPDATE ON trust_votes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_profile_visibility_updated_at ON profile_visibility;
CREATE TRIGGER update_profile_visibility_updated_at BEFORE UPDATE ON profile_visibility
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Political parties table
CREATE TABLE IF NOT EXISTS political_parties (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    intro_text TEXT,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Party external resource links
CREATE TABLE IF NOT EXISTS party_external_links (
    id SERIAL PRIMARY KEY,
    party_id INTEGER NOT NULL REFERENCES political_parties(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    label TEXT NOT NULL,
    sort_order INTEGER DEFAULT 0,
    UNIQUE(party_id, url)
);

-- Party engagement: user responses to the 4 preliminary questions + comments
CREATE TABLE IF NOT EXISTS party_engagement (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    party_slug VARCHAR(100) NOT NULL,
    q1_answer VARCHAR(20),
    q2_answer VARCHAR(20),
    q3_answer VARCHAR(20),
    q4_answer VARCHAR(20),
    comments TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, party_slug)
);

CREATE INDEX IF NOT EXISTS idx_party_engagement_user ON party_engagement(user_id);
CREATE INDEX IF NOT EXISTS idx_party_external_links_party ON party_external_links(party_id);

DROP TRIGGER IF EXISTS update_political_parties_updated_at ON political_parties;
CREATE TRIGGER update_political_parties_updated_at BEFORE UPDATE ON political_parties
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_party_engagement_updated_at ON party_engagement;
CREATE TRIGGER update_party_engagement_updated_at BEFORE UPDATE ON party_engagement
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Seed political parties (idempotent)
INSERT INTO political_parties (name, slug, description, intro_text, sort_order) VALUES
(
    'Republican', 'republican',
    'The Republican Party, also known as the Grand Old Party (GOP), is one of the two major political parties in the United States, founded in 1854.',
    'This page presents a framework with tools to help registered users promote a grass-roots, bottom-up consensus-building strategy within the Republican Party. Our goal is to assist activists in becoming more effective at consensus-building with Republican Party members and open-minded outsiders, in order to influence Republican Party decisions toward promoting justice, security, and peace for the People of the USA.',
    1
),
(
    'Democrat', 'democrat',
    'The Democratic Party is one of the two major contemporary political parties in the United States, tracing its origins to the Democratic-Republican Party of the early 19th century.',
    'This page presents a framework with tools to help registered users promote a grass-roots, bottom-up consensus-building strategy within the Democrat Party. Our goal is to assist activists in becoming more effective at consensus-building with Democrat Party members and open-minded outsiders, in order to influence Democrat Party decisions toward promoting justice, security, and peace for the People of the USA.',
    2
),
(
    'Socialist', 'socialist',
    'Socialist parties and movements in the United States advocate for social ownership of the means of production, greater economic equality, and democratic control of major institutions.',
    'This page presents a framework with tools to help registered users promote a grass-roots, bottom-up consensus-building strategy within Socialist communities. Our goal is to assist activists in becoming more effective at consensus-building with Socialist Party members and open-minded outsiders, in order to influence Socialist community decisions toward promoting justice, security, and peace for the People of the USA.',
    3
),
(
    'Green', 'green',
    'The Green Party of the United States is a left-wing political party focused on environmentalism, social justice, grassroots democracy, and non-violence.',
    'This page presents a framework with tools to help registered users promote a grass-roots, bottom-up consensus-building strategy within the Green Party. Our goal is to assist activists in becoming more effective at consensus-building with Green Party members and open-minded outsiders, in order to influence Green Party decisions toward promoting justice, security, and peace for the People of the USA.',
    4
),
(
    'Libertarian', 'libertarian',
    'The Libertarian Party advocates for individual freedom, voluntary association, and strict limits on government authority in both economic and personal matters.',
    'This page presents a framework with tools to help registered users promote a grass-roots, bottom-up consensus-building strategy within the Libertarian Party. Our goal is to assist activists in becoming more effective at consensus-building with Libertarian Party members and open-minded outsiders, in order to influence Libertarian Party decisions toward promoting justice, security, and peace for the People of the USA.',
    5
)
ON CONFLICT (slug) DO NOTHING;

-- Seed external links for each party (idempotent via unique constraint)
DO $$
DECLARE
    v_rep_id INTEGER;
    v_dem_id INTEGER;
    v_soc_id INTEGER;
    v_grn_id INTEGER;
    v_lib_id INTEGER;
BEGIN
    SELECT id INTO v_rep_id FROM political_parties WHERE slug = 'republican';
    SELECT id INTO v_dem_id FROM political_parties WHERE slug = 'democrat';
    SELECT id INTO v_soc_id FROM political_parties WHERE slug = 'socialist';
    SELECT id INTO v_grn_id FROM political_parties WHERE slug = 'green';
    SELECT id INTO v_lib_id FROM political_parties WHERE slug = 'libertarian';

    IF v_rep_id IS NOT NULL THEN
        INSERT INTO party_external_links (party_id, url, label, sort_order) VALUES
        (v_rep_id, 'https://www.gop.com/', 'GOP.com - Official Republican Party Website', 1),
        (v_rep_id, 'https://gop.com/about-our-party/rnc-members/', 'RNC Members List', 2),
        (v_rep_id, 'https://prod-static.gop.com/media/RNC2024-Platform.pdf', 'RNC 2024 Platform (PDF)', 3),
        (v_rep_id, 'https://ballotpedia.org/Republican_Party', 'Ballotpedia: Republican Party', 4),
        (v_rep_id, 'https://ballotpedia.org/Republican_National_Committee', 'Ballotpedia: Republican National Committee', 5),
        (v_rep_id, 'https://en.wikipedia.org/wiki/Republican_Party_(United_States)', 'Wikipedia: Republican Party', 6),
        (v_rep_id, 'https://en.wikipedia.org/wiki/Republican_National_Committee', 'Wikipedia: Republican National Committee', 7),
        (v_rep_id, 'https://www.presidency.ucsb.edu/documents/2024-republican-party-platform', 'UCSB: 2024 Republican Platform', 8),
        (v_rep_id, 'https://www.britannica.com/topic/Republican-Party', 'Britannica: Republican Party', 9)
        ON CONFLICT (party_id, url) DO NOTHING;
    END IF;

    IF v_dem_id IS NOT NULL THEN
        INSERT INTO party_external_links (party_id, url, label, sort_order) VALUES
        (v_dem_id, 'https://democrats.org/', 'Democrats.org - Official Democratic Party Website', 1),
        (v_dem_id, 'https://www.dnc.org/', 'Democratic National Committee', 2),
        (v_dem_id, 'https://ballotpedia.org/Democratic_Party', 'Ballotpedia: Democratic Party', 3),
        (v_dem_id, 'https://ballotpedia.org/Democratic_National_Committee', 'Ballotpedia: Democratic National Committee', 4),
        (v_dem_id, 'https://en.wikipedia.org/wiki/Democratic_Party_(United_States)', 'Wikipedia: Democratic Party', 5),
        (v_dem_id, 'https://en.wikipedia.org/wiki/Democratic_National_Committee', 'Wikipedia: Democratic National Committee', 6),
        (v_dem_id, 'https://www.britannica.com/topic/Democratic-Party', 'Britannica: Democratic Party', 7)
        ON CONFLICT (party_id, url) DO NOTHING;
    END IF;

    IF v_soc_id IS NOT NULL THEN
        INSERT INTO party_external_links (party_id, url, label, sort_order) VALUES
        (v_soc_id, 'https://www.sp-usa.org/', 'Socialist Party USA - Official Website', 1),
        (v_soc_id, 'https://dsausa.org/', 'Democratic Socialists of America', 2),
        (v_soc_id, 'https://ballotpedia.org/Socialist_Party_USA', 'Ballotpedia: Socialist Party USA', 3),
        (v_soc_id, 'https://en.wikipedia.org/wiki/Socialist_Party_USA', 'Wikipedia: Socialist Party USA', 4),
        (v_soc_id, 'https://en.wikipedia.org/wiki/Democratic_Socialists_of_America', 'Wikipedia: Democratic Socialists of America', 5)
        ON CONFLICT (party_id, url) DO NOTHING;
    END IF;

    IF v_grn_id IS NOT NULL THEN
        INSERT INTO party_external_links (party_id, url, label, sort_order) VALUES
        (v_grn_id, 'https://www.gp.org/', 'Green Party US - Official Website', 1),
        (v_grn_id, 'https://ballotpedia.org/Green_Party_of_the_United_States', 'Ballotpedia: Green Party of the United States', 2),
        (v_grn_id, 'https://en.wikipedia.org/wiki/Green_Party_of_the_United_States', 'Wikipedia: Green Party of the United States', 3),
        (v_grn_id, 'https://www.britannica.com/topic/Green-Party-United-States', 'Britannica: Green Party', 4)
        ON CONFLICT (party_id, url) DO NOTHING;
    END IF;

    IF v_lib_id IS NOT NULL THEN
        INSERT INTO party_external_links (party_id, url, label, sort_order) VALUES
        (v_lib_id, 'https://www.lp.org/', 'Libertarian Party - Official Website', 1),
        (v_lib_id, 'https://ballotpedia.org/Libertarian_Party', 'Ballotpedia: Libertarian Party', 2),
        (v_lib_id, 'https://en.wikipedia.org/wiki/Libertarian_Party_(United_States)', 'Wikipedia: Libertarian Party', 3),
        (v_lib_id, 'https://www.britannica.com/topic/Libertarian-Party', 'Britannica: Libertarian Party', 4)
        ON CONFLICT (party_id, url) DO NOTHING;
    END IF;
END $$;
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