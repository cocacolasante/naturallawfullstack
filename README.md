# Natural Law Fullstack Voting Application

A fullstack voting application built with Go (backend) and Node.js/Express (frontend) that enables democratic participation through ballot creation, voting, and results tracking.

## Quick Start

### Prerequisites
- Go 1.23+ installed
- Node.js 16+ and npm installed
- PostgreSQL 12+ installed and running

### Setup

1. **Create PostgreSQL database:**
   ```bash
   createdb voting_db
   ```

2. **Configure environment variables:**
   ```bash
   cd naturallawvoting
   cp .env.example .env
   # Edit .env with your database credentials and JWT secret
   ```

3. **Install dependencies:**
   ```bash
   make install
   ```

4. **Start the application:**
   ```bash
   make start
   ```

The application will be available at:
- **Frontend:** http://localhost:3000
- **Backend API:** http://localhost:8080

### Available Commands

From the root directory:

- `make help` - Show all available commands
- `make install` - Install all dependencies
- `make start` - Start both backend and frontend servers
- `make stop` - Stop all servers
- `make dev` - Start in development mode with live logs
- `make test` - Run backend tests
- `make build` - Build backend binary
- `make logs` - View server logs
- `make clean` - Clean build artifacts and stop servers

## Project Structure

```
naturallawfullstack/
├── naturallawvoting/        # Go backend API
│   ├── main.go              # Main server entry point
│   ├── handlers/            # HTTP request handlers
│   ├── models/              # Data models
│   ├── middleware/          # Authentication middleware
│   ├── routes/              # Route definitions
│   ├── database/            # Database connection & migrations
│   ├── utils/               # Utility functions
│   └── tests/               # Test files
│
├── naturallawfrontend/      # Node.js frontend
│   ├── index.js             # Express server
│   └── ui/                  # Static HTML/CSS/JS files
│
├── Makefile                 # Root-level commands
└── CLAUDE.md                # Developer documentation
```

## Features

- **User Authentication:** Register, login with JWT tokens
- **Ballot Management:** Create ballots with multiple options
- **Voting System:** One vote per user per ballot
- **Results Tracking:** Real-time vote counts and percentages
- **Category Organization:** Ballots organized by government department
- **Profile Management:** Extended user profiles with demographics
- **Responsive Design:** Works on desktop and mobile

## Documentation

For detailed documentation, see:
- [`CLAUDE.md`](./CLAUDE.md) - Comprehensive developer guide
- [`naturallawvoting/README.md`](./naturallawvoting/README.md) - Backend API documentation
- [`naturallawvoting/ENVIRONMENT_SETUP.md`](./naturallawvoting/ENVIRONMENT_SETUP.md) - Environment configuration guide

## Development

### Backend Only
```bash
cd naturallawvoting
go run main.go
```

### Frontend Only
```bash
cd naturallawfrontend
npm start
```

### Running Tests
```bash
make test
# or
cd naturallawvoting && make test
```

## License

Copyright © 2025 Common-Law Republic, U.S.A.
