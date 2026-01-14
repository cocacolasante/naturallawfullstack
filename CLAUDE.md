# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a fullstack voting application with a Go backend API and a Node.js/Express frontend serving static HTML/CSS/JS. The application supports user registration, authentication, ballot creation, voting, and comprehensive user profile management including political affiliations, religious affiliations, race/ethnicity, and economic information.

## Repository Structure

The repository contains two main applications in separate directories:

- **`naturallawvoting/`** - Go-based REST API backend with PostgreSQL
- **`naturallawfrontend/`** - Node.js Express server serving static UI files

## Quick Start Commands

From the root `naturallawfullstack/` directory:

**Start both backend and frontend:**
```bash
make start
```

**Start in development mode (with live logs):**
```bash
make dev
```

**Stop all servers:**
```bash
make stop
```

**Install all dependencies:**
```bash
make install
```

**View logs:**
```bash
make logs
```

**Seed database with sample data:**
```bash
make seed
```

**See all available commands:**
```bash
make help
```

## Backend (naturallawvoting/)

### Development Commands

**Build and run:**
```bash
cd naturallawvoting
go run main.go
```

**Build binary:**
```bash
make build
```

**Run all tests:**
```bash
make test
```

**Run specific test suites:**
```bash
make test-auth          # Authentication tests
make test-ballot        # Ballot management tests
make test-vote          # Voting tests
make test-integration   # Integration tests
```

**Run tests with coverage:**
```bash
make test-coverage
```

**Run tests with race detection:**
```bash
make test-race
```

**Code quality:**
```bash
make fmt       # Format code
make vet       # Vet code
make lint      # Run linter (requires golangci-lint)
make security  # Security scan (requires gosec)
```

**Development with hot reload:**
```bash
make dev  # Requires air to be installed
```

**Clean build artifacts:**
```bash
make clean
```

### Backend Architecture

**Main entry point:** `main.go`
- Loads environment variables from `.env` file
- Establishes database connection
- Runs database migrations automatically on startup
- Sets up routes via `routes/SetupRoutes()`
- Starts server on port 8080 (or PORT from env)

**Database layer:** `database/database.go`
- Handles PostgreSQL connection using `lib/pq` driver
- Supports both `DATABASE_URL` and individual `DB_*` environment variables
- Contains all schema migrations inline (CREATE TABLE statements, indexes, triggers)
- Automatic migrations run on every server startup

**Route organization:** `routes/routes.go`
- Uses Gin framework for HTTP routing
- CORS middleware configured for all origins
- Three route groups:
  - `/api/v1/auth/*` - Public authentication endpoints
  - `/api/v1/public/*` - Public read-only ballot endpoints
  - `/api/v1/*` (protected) - Authenticated endpoints using JWT middleware
- All protected routes require `Authorization: Bearer <token>` header

**Handlers:**
- `handlers/auth.go` - User registration, login, profile retrieval
- `handlers/ballot.go` - Ballot creation, retrieval, listing
- `handlers/vote.go` - Voting, vote retrieval, results calculation
- `handlers/profile.go` - Extended profile management (address, political/religious affiliations, race/ethnicity, economic info)

**Models:**
- `models/user.go` - User model
- `models/ballot.go` - Ballot and BallotItem models
- `models/profile.go` - Extended profile models (UserProfile, UserAddress, UserPoliticalAffiliation, UserReligiousAffiliation, UserRaceEthnicity, EconomicInfo)

**Middleware:**
- `middleware/auth.go` - JWT authentication middleware that validates tokens and extracts user information

**Utils:**
- `utils/auth.go` - JWT token generation/validation, password hashing with bcrypt

**Tests:**
- Located in `tests/` directory
- Uses `testify` for assertions and `go-sqlmock` for database mocking
- `tests/utils.go` contains shared test utilities
- Test files: `auth_test.go`, `ballot_test.go`, `vote_test.go`, `profile_test.go`, `integration_test.go`

### Seeding the Database

To populate the database with sample data for testing:

```bash
cd naturallawvoting/setup
go run seed_database.go
```

Or from the root directory:
```bash
make seed
```

This creates:
- 2 test users (alice_smith, bob_jones) with password "password123" and "securepass456"
- 6 sample ballots (2 for each of executive, judicial, house, senate categories)
- Multiple ballot items for each ballot with realistic options
- Sample votes to demonstrate the voting system

### Database Schema

The application uses PostgreSQL with the following tables:

**Core tables:**
- `users` - User accounts with username, email, password_hash
- `ballots` - Voting ballots with title, description, category (for filtering by department), creator reference
- `ballot_items` - Individual voting options linked to ballots
- `votes` - User votes with UNIQUE constraint (user_id, ballot_id) to enforce one vote per ballot

**Profile tables:**
- `user_profiles` - Personal information (full_name, birthday, gender, mothers_maiden_name, phone_number, additional_emails)
- `user_addresses` - Street address information
- `user_political_affiliations` - Political party affiliation
- `user_religious_affiliations` - Religion, support level (0-10 scale), religious service types
- `user_race_ethnicity` - Race/ethnicity array
- `economic_info` - Economic and political stance information

All tables with `updated_at` fields use PostgreSQL triggers to automatically update timestamps on modification.

### Environment Variables

The backend requires these environment variables (can be set via `.env` file):

**Database (Option 1 - individual variables):**
- `DB_HOST` - PostgreSQL host (default: localhost)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - Database username (default: postgres)
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name (default: voting_db)
- `DB_SSLMODE` - SSL mode (default: disable)

**Database (Option 2 - connection URL, takes precedence):**
- `DATABASE_URL` - Full PostgreSQL connection string

**Required:**
- `JWT_SECRET` - Secret key for JWT signing (must be strong in production)

**Optional:**
- `PORT` - Server port (default: 8080)

See `ENVIRONMENT_SETUP.md` for detailed setup instructions including database creation and JWT secret generation.

### API Endpoints

**Public endpoints:**
- `GET /health` - Health check
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login (returns JWT token)
- `GET /api/v1/public/ballots` - List all active ballots (supports optional `?category=<category>` query parameter for filtering)
- `GET /api/v1/public/ballots/:id` - Get specific ballot with items
- `GET /api/v1/public/ballots/:id/results` - Get ballot results

**Protected endpoints (require JWT):**
- `GET /api/v1/profile` - Get basic user profile
- `GET /api/v1/my-ballots` - Get user's created ballots
- `POST /api/v1/ballots` - Create new ballot
- `POST /api/v1/ballots/:ballot_id/vote` - Submit vote (accepts `option_id` or `ballot_item_id`)
- `GET /api/v1/ballots/:ballot_id/my-vote` - Get user's vote for specific ballot (returns `option_id`)

**Profile management (all require JWT):**
- User Info: GET/POST/PUT/DELETE `/api/v1/profile/info`
- Address: GET/POST/PUT/DELETE `/api/v1/profile/address`
- Political: GET/POST/PUT/DELETE `/api/v1/profile/political`
- Religious: GET/POST/PUT/DELETE `/api/v1/profile/religious`
- Race/Ethnicity: GET/POST/PUT/DELETE `/api/v1/profile/race-ethnicity`
- Economic: GET/POST/PUT/DELETE `/api/v1/profile/economic`

### API Request/Response Format

**Frontend-Backend Field Mapping:**

The API uses specific field names to maintain compatibility with the frontend:

- Ballot responses use `"options"` array (JSON field name for `Items` in Go struct)
- Vote submission accepts `"option_id"` field which maps internally to `ballot_item_id`
- Vote retrieval returns both `"option_id"` and `"ballot_item_id"` for compatibility
- Results endpoint includes both `"option_id"` and `"option_title"` fields

**Example Vote Request:**
```json
POST /api/v1/ballots/3/vote
{
  "option_id": 5
}
```

**Example Vote Response:**
```json
{
  "id": 1,
  "user_id": 2,
  "ballot_id": 3,
  "option_id": 5,
  "ballot_item_id": 5,
  "created_at": "2025-01-14T12:00:00Z"
}
```

### Ballot Categories

Ballots support categorization to organize them by government department:
- **executive** - Executive Department ballots
- **judicial** - Judicial Department ballots
- **house** - Legislative House of Representatives ballots
- **senate** - Legislative Senate ballots

When creating a ballot, optionally specify a `category` field. Frontend ballot index pages filter by category using the query parameter: `/api/v1/public/ballots?category=executive`

Category-specific ballot index pages:
- `USA-GoverningBodies/Executive-Department-SubDepartments-Agencies/Ballot-List-&-Index.html`
- `USA-GoverningBodies/JudicialDepartment/Ballot-List-&-Index.html`
- `USA-GoverningBodies/Legislature-HouseReps/Ballot-List-&-Index.html`
- `USA-GoverningBodies/Legislature-Senate/Ballot-List-&-Index.html`

## Frontend (naturallawfrontend/)

### Development Commands

**Install dependencies:**
```bash
cd naturallawfrontend
npm install
```

**Run development server:**
```bash
npm start
```

Server runs on port 3000 by default.

### Frontend Architecture

**Entry point:** `index.js`
- Simple Express server serving static files from `ui/` directory
- No build process or transpilation

**Static files:** `ui/` directory
- `index.html` - Main entry point
- Organized subdirectories:
  - `Accounts-Profiles-Registering/` - User account UI
  - `USA-GoverningBodies/` - Government structure UI
  - `USA-LocalJurisdictions/` - Local jurisdiction UI
  - `EveryThing-Else/` - Miscellaneous pages
  - `Support/` - Support pages
  - `Z-ReferenceDocuments/` - Reference materials
  - `WebPageEditings/` - Additional editing resources

The frontend is vanilla HTML/CSS/JavaScript with no framework.

## Key Development Patterns

### Testing Approach
- Tests use in-memory mock database (`go-sqlmock`) for unit tests
- Integration tests verify full request/response flows
- Each handler function has corresponding tests
- Test setup uses shared utilities from `tests/utils.go`

### Authentication Flow
1. User registers via POST `/api/v1/auth/register` with username, email, password
2. Password is hashed using bcrypt before storage
3. User logs in via POST `/api/v1/auth/login`
4. JWT token is generated and returned (expires in 7 days by default)
5. Client includes token in `Authorization: Bearer <token>` header for protected routes
6. Middleware validates token and extracts user_id for handler use

### Database Migration Strategy
- All migrations are embedded in `database/database.go` as a single SQL string
- Migrations use `CREATE TABLE IF NOT EXISTS` to avoid errors on restart
- Triggers and functions use `CREATE OR REPLACE` and `DROP TRIGGER IF EXISTS`
- Migrations run automatically on every server startup
- Schema changes should be added to the migration SQL string

### Vote Constraint Enforcement
- One vote per user per ballot enforced by UNIQUE(user_id, ballot_id) constraint
- When user changes vote, the application deletes old vote and creates new one
- `ballot_items.vote_count` is maintained via application logic, not database triggers

## Important Notes

- The backend uses Go 1.23.0+ with Gin framework v1.10.1
- Database connection pooling is handled by `database/sql` package
- JWT tokens include user_id in claims for authentication
- CORS is wide open (`Access-Control-Allow-Origin: *`) - restrict in production
- Database schema auto-migrates on every startup - be cautious with destructive changes
- Profile tables use different primary keys (some use user_id, some use email) - refer to schema when writing queries
