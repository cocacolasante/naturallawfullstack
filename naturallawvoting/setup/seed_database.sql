-- Seed script for Natural Law Voting Backend
-- This script populates the database with sample data for testing

-- Note: Password hashes are for "password123" and "securepass456" respectively
-- You may need to regenerate these using bcrypt if your security requirements differ

-- Clean up existing data (optional - uncomment if you want a fresh start)
-- DELETE FROM votes;
-- DELETE FROM ballot_items;
-- DELETE FROM ballots;
-- DELETE FROM users;

-- Seed Users (2 entries)
INSERT INTO users (username, email, password_hash, created_at, updated_at)
VALUES 
    ('alice_smith', 'alice.smith@example.com', '$2a$10$YourHashedPasswordHere1', NOW(), NOW()),
    ('bob_jones', 'bob.jones@example.com', '$2a$10$YourHashedPasswordHere2', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

-- Seed Ballots (2 entries)
-- Using a subquery to get the creator_id from users
INSERT INTO ballots (creator_id, title, description, is_active, created_at, updated_at)
VALUES 
    (
        (SELECT id FROM users WHERE email = 'alice.smith@example.com'),
        'Favorite Programming Language 2024',
        'Vote for your favorite programming language to use in 2024. This will help us decide our tech stack.',
        true,
        NOW(),
        NOW()
    ),
    (
        (SELECT id FROM users WHERE email = 'bob.jones@example.com'),
        'Team Building Activity Selection',
        'Help us choose our next team building activity for the upcoming quarter.',
        true,
        NOW(),
        NOW()
    )
ON CONFLICT DO NOTHING;

-- Seed Ballot Items (4 for first ballot, 4 for second ballot = 8 total)
-- Items for Programming Language ballot
INSERT INTO ballot_items (ballot_id, title, description, created_at, updated_at)
VALUES 
    -- Programming Language options
    (
        (SELECT id FROM ballots WHERE title = 'Favorite Programming Language 2024'),
        'Go (Golang)',
        'Statically typed, compiled language with excellent concurrency support',
        NOW(),
        NOW()
    ),
    (
        (SELECT id FROM ballots WHERE title = 'Favorite Programming Language 2024'),
        'Python',
        'High-level, interpreted language perfect for rapid development and data science',
        NOW(),
        NOW()
    ),
    (
        (SELECT id FROM ballots WHERE title = 'Favorite Programming Language 2024'),
        'Rust',
        'Systems programming language focused on safety, speed, and concurrency',
        NOW(),
        NOW()
    ),
    (
        (SELECT id FROM ballots WHERE title = 'Favorite Programming Language 2024'),
        'TypeScript',
        'JavaScript with static typing for large-scale applications',
        NOW(),
        NOW()
    ),
    -- Team Building Activity options
    (
        (SELECT id FROM ballots WHERE title = 'Team Building Activity Selection'),
        'Escape Room Challenge',
        'Work together to solve puzzles and escape within 60 minutes',
        NOW(),
        NOW()
    ),
    (
        (SELECT id FROM ballots WHERE title = 'Team Building Activity Selection'),
        'Cooking Class',
        'Learn to cook a new cuisine together as a team',
        NOW(),
        NOW()
    ),
    (
        (SELECT id FROM ballots WHERE title = 'Team Building Activity Selection'),
        'Outdoor Adventure Day',
        'Hiking, zip-lining, and team challenges in nature',
        NOW(),
        NOW()
    ),
    (
        (SELECT id FROM ballots WHERE title = 'Team Building Activity Selection'),
        'Virtual Reality Experience',
        'Explore VR games and experiences as a group',
        NOW(),
        NOW()
    )
ON CONFLICT DO NOTHING;

-- Seed Votes (2 entries - one from each user)
INSERT INTO votes (user_id, ballot_id, ballot_item_id, created_at)
VALUES 
    -- Alice votes for Go in the programming language ballot
    (
        (SELECT id FROM users WHERE email = 'alice.smith@example.com'),
        (SELECT id FROM ballots WHERE title = 'Favorite Programming Language 2024'),
        (SELECT id FROM ballot_items WHERE title = 'Go (Golang)' AND ballot_id = (SELECT id FROM ballots WHERE title = 'Favorite Programming Language 2024')),
        NOW()
    ),
    -- Bob votes for Escape Room in the team building ballot
    (
        (SELECT id FROM users WHERE email = 'bob.jones@example.com'),
        (SELECT id FROM ballots WHERE title = 'Team Building Activity Selection'),
        (SELECT id FROM ballot_items WHERE title = 'Escape Room Challenge' AND ballot_id = (SELECT id FROM ballots WHERE title = 'Team Building Activity Selection')),
        NOW()
    )
ON CONFLICT (user_id, ballot_id) DO NOTHING;

-- Verify the data was inserted
SELECT 'Users inserted:' AS info, COUNT(*) AS count FROM users;
SELECT 'Ballots inserted:' AS info, COUNT(*) AS count FROM ballots;
SELECT 'Ballot items inserted:' AS info, COUNT(*) AS count FROM ballot_items;
SELECT 'Votes inserted:' AS info, COUNT(*) AS count FROM votes;
