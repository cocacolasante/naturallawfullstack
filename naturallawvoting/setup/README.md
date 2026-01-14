# Database Setup Scripts

This directory contains scripts to set up and seed the database for the Natural Law Voting application.

## Files

- **`seed_database.go`** - Go script to populate the database with sample data
- **`seed_database.sql`** - SQL version of the seed script (if available)

## Usage

### Running the Seed Script

**From the root directory:**
```bash
make seed
```

**From the setup directory:**
```bash
cd naturallawvoting/setup
go run seed_database.go
```

### What Gets Seeded

The seed script creates:

1. **Users:**
   - alice_smith (alice.smith@example.com) - password: `password123`
   - bob_jones (bob.jones@example.com) - password: `securepass456`

2. **Ballots (6 total - organized by category):**
   - **Executive Category (2 ballots):**
     - Executive Branch Budget Priorities
     - Department of Education Focus Areas

   - **Judicial Category (2 ballots):**
     - Federal Court System Reform
     - Supreme Court Term Limits

   - **House Category (1 ballot):**
     - House of Representatives Voting Rules

   - **Senate Category (1 ballot):**
     - Senate Filibuster Reform

3. **Ballot Items:**
   - Each ballot has 3-4 voting options with descriptions
   - Total of 24 ballot items across all ballots

4. **Sample Votes:**
   - Alice votes for "Infrastructure" in Executive Budget ballot
   - Bob votes for "Yes - Expand Courts" in Judicial Reform ballot

## Requirements

- PostgreSQL database must be running
- Database must be created (specified in `.env` file)
- `.env` file must be configured with database credentials

## Environment Setup

Make sure your `.env` file contains:
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=voting_db
DB_SSLMODE=disable
JWT_SECRET=your_secret_key
```

## Notes

- The script uses `ON CONFLICT DO NOTHING` to prevent duplicate entries
- Running the script multiple times is safe and will not create duplicates
- Vote counts are automatically updated when votes are inserted
- All timestamps are set to the current time when the script runs
