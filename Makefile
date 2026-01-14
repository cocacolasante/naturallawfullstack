# Natural Law Fullstack Makefile
# This Makefile provides commands to manage both backend (Go) and frontend (Node.js) services

.PHONY: help install start stop dev backend frontend clean test build logs seed

# Default target
help:
	@echo "Natural Law Fullstack - Available Commands:"
	@echo ""
	@echo "  make install       - Install all dependencies (backend + frontend)"
	@echo "  make start         - Start both backend and frontend servers"
	@echo "  make stop          - Stop both backend and frontend servers"
	@echo "  make dev           - Start both servers in development mode (with logs)"
	@echo ""
	@echo "  make backend       - Start only the backend server (port 8080)"
	@echo "  make frontend      - Start only the frontend server (port 3000)"
	@echo ""
	@echo "  make build         - Build the backend binary"
	@echo "  make test          - Run backend tests"
	@echo "  make seed          - Seed database with sample data"
	@echo "  make clean         - Clean build artifacts and stop all servers"
	@echo "  make logs          - Show logs from running servers"
	@echo ""

# Install all dependencies
install:
	@echo "📦 Installing backend dependencies..."
	cd naturallawvoting && go mod tidy
	@echo "📦 Installing frontend dependencies..."
	cd naturallawfrontend && npm install
	@echo "✅ All dependencies installed!"

# Start both servers in background
start:
	@mkdir -p logs
	@echo "🚀 Starting backend server..."
	@cd naturallawvoting && go run main.go
	@sleep 2
	@echo "🚀 Starting frontend server..."
	@cd naturallawfrontend && npm start
	@sleep 2
	@echo "✅ Both servers started!"
	@echo "   Backend:  http://localhost:8080"
	@echo "   Frontend: http://localhost:3000"
	@echo ""
	@echo "   Run 'make logs' to view logs"
	@echo "   Run 'make stop' to stop servers"

# Stop both servers
stop:
	@echo "🛑 Stopping servers..."
	@if [ -f logs/backend.pid ]; then \
		kill -9 $$(cat logs/backend.pid) 2>/dev/null || true; \
		rm logs/backend.pid; \
		echo "   Backend stopped"; \
	fi
	@if [ -f logs/frontend.pid ]; then \
		kill -9 $$(cat logs/frontend.pid) 2>/dev/null || true; \
		rm logs/frontend.pid; \
		echo "   Frontend stopped"; \
	fi
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@lsof -ti:3000 | xargs kill -9 2>/dev/null || true
	@echo "✅ All servers stopped"

# Start both servers in foreground with live logs (for development)
dev:
	@echo "🔧 Starting development mode..."
	@echo "   Backend will run on http://localhost:8080"
	@echo "   Frontend will run on http://localhost:3000"
	@echo ""
	@mkdir -p logs
	@trap 'make stop' INT; \
	cd naturallawvoting && go run main.go & \
	BACKEND_PID=$$!; \
	cd ../naturallawfrontend && npm start & \
	FRONTEND_PID=$$!; \
	wait

# Start only backend server
backend:
	@echo "🚀 Starting backend server on http://localhost:8080..."
	@cd naturallawvoting && go run main.go

# Start only frontend server
frontend:
	@echo "🚀 Starting frontend server on http://localhost:3000..."
	@cd naturallawfrontend && npm start

# Build backend binary
build:
	@echo "🔨 Building backend..."
	@cd naturallawvoting && make build
	@echo "✅ Backend built successfully!"

# Run backend tests
test:
	@echo "🧪 Running backend tests..."
	@cd naturallawvoting && make test

# Seed database with sample data
seed:
	@echo "🌱 Seeding database with sample data..."
	@cd naturallawvoting/setup && go run seed_database.go
	@echo "✅ Database seeded successfully!"

# Clean everything
clean: stop
	@echo "🧹 Cleaning build artifacts..."
	@cd naturallawvoting && make clean
	@rm -rf logs/*.log logs/*.pid
	@echo "✅ Clean complete"

# Show logs from running servers
logs:
	@if [ -f logs/backend.pid ] || [ -f logs/frontend.pid ]; then \
		echo "📋 Server Logs:"; \
		echo ""; \
		if [ -f logs/backend.log ]; then \
			echo "=== Backend Logs ==="; \
			tail -20 logs/backend.log; \
			echo ""; \
		fi; \
		if [ -f logs/frontend.log ]; then \
			echo "=== Frontend Logs ==="; \
			tail -20 logs/frontend.log; \
		fi; \
	else \
		echo "ℹ️  No servers are currently running"; \
		echo "   Run 'make start' to start servers"; \
	fi

# Initialize logs directory
logs-init:
	@mkdir -p logs
	@touch logs/.gitkeep
