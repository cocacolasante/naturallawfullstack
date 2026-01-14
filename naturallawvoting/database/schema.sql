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
    creator_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

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

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_ballots_creator_id ON ballots(creator_id);
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
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ballots_updated_at BEFORE UPDATE ON ballots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
