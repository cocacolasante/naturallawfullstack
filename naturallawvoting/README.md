# Voting API

A Go-based REST API for a voting application with PostgreSQL database integration.

## Features

- User registration and authentication with JWT tokens
- Create and manage voting ballots
- Vote on ballot items
- View ballot results
- User profile management
- PostgreSQL database with automatic migrations

## Project Structure

```
voting-api/
├── main.go              # Main server file
├── go.mod               # Go module dependencies
├── .env.example         # Environment variables template
├── models/              # Data models
│   ├── user.go
│   └── ballot.go
├── handlers/            # HTTP request handlers
│   ├── auth.go
│   ├── ballot.go
│   └── vote.go
├── middleware/          # HTTP middleware
│   └── auth.go
├── routes/              # Route definitions
│   └── routes.go
├── database/            # Database connection and migrations
│   ├── database.go
│   └── schema.sql
└── utils/               # Utility functions
    └── auth.go
```

## Setup

1. **Install PostgreSQL** and create a database named `voting_db`

2. **Copy environment file**:
   ```bash
   cp .env.example .env
   ```

3. **Configure environment variables** in `.env`:
   ```
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=your_password
   DB_NAME=voting_db
   DB_SSLMODE=disable
   JWT_SECRET=your-super-secret-jwt-key-here
   PORT=8080
   ```

4. **Install dependencies**:
   ```bash
   go mod tidy
   ```

5. **Run the server**:
   ```bash
   go run main.go
   ```

## API Endpoints

### Public Endpoints

- `GET /health` - Health check
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login
- `GET /api/v1/public/ballots` - Get all active ballots
- `GET /api/v1/public/ballots/:id` - Get specific ballot with items
- `GET /api/v1/public/ballots/:ballot_id/results` - Get ballot results

### Protected Endpoints (Require Authorization Header)

- `GET /api/v1/profile` - Get user profile
- `GET /api/v1/my-ballots` - Get user's created ballots
- `POST /api/v1/ballots` - Create new ballot
- `POST /api/v1/ballots/:ballot_id/vote` - Vote on a ballot
- `GET /api/v1/ballots/:ballot_id/my-vote` - Get user's vote for a ballot

## Request Examples

### Register User
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "password": "password123"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "password123"
  }'
```

### Create Ballot
```bash
curl -X POST http://localhost:8080/api/v1/ballots \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "Best Programming Language",
    "description": "Vote for your favorite programming language",
    "items": [
      {
        "title": "Go",
        "description": "Fast and efficient"
      },
      {
        "title": "Python",
        "description": "Easy and versatile"
      },
      {
        "title": "JavaScript",
        "description": "Web development standard"
      }
    ]
  }'
```

### Vote on Ballot
```bash
curl -X POST http://localhost:8080/api/v1/ballots/1/vote \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "ballot_item_id": 1
  }'
```

### Get All Ballots
```bash
curl http://localhost:8080/api/v1/public/ballots
```

### Get Ballot Results
```bash
curl http://localhost:8080/api/v1/public/ballots/1/results
```

## Database Schema

The API automatically creates the following tables:
- `users` - User accounts and authentication
- `ballots` - Voting ballots created by users
- `ballot_items` - Individual items that can be voted on
- `votes` - User votes (one vote per user per ballot)

## Security Features

- Password hashing using bcrypt
- JWT token authentication
- CORS middleware
- SQL injection protection with prepared statements
- One vote per user per ballot constraint

## Running in Production

1. Set environment variables securely
2. Use a strong JWT secret
3. Configure PostgreSQL with SSL
4. Consider using a reverse proxy like nginx
5. Set up proper logging and monitoring

## Dependencies

- **Gin** - HTTP web framework
- **lib/pq** - PostgreSQL driver
- **golang-jwt/jwt/v5** - JWT token handling
- **golang.org/x/crypto** - Password hashing
- **godotenv** - Environment variable loading