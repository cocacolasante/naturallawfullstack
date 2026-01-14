# Environment Setup Guide

This guide explains how to configure all environment variables for the Voting API.

## Quick Start

1. **Copy the example file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit the `.env` file with your values:**
   ```bash
   nano .env
   # or
   vim .env
   # or use any text editor
   ```

3. **Set required values and start the server:**
   ```bash
   go run main.go
   ```

## Environment Variables Reference

### Database Configuration

#### Option 1: Individual Database Variables
```bash
DB_HOST=localhost          # PostgreSQL server host
DB_PORT=5432              # PostgreSQL server port
DB_USER=postgres          # Database username
DB_PASSWORD=your_password # Database user password
DB_NAME=voting_db         # Database name
DB_SSLMODE=disable        # SSL mode (disable, require, verify-full)
```

#### Option 2: Database URL (Alternative)
```bash
DATABASE_URL=postgres://username:password@localhost:5432/voting_db?sslmode=disable
```

**Note:** If `DATABASE_URL` is set, it takes precedence over individual DB_* variables.

### Security Configuration

#### JWT Secret (REQUIRED)
```bash
JWT_SECRET=your-super-secret-jwt-key-here-make-it-long-and-random
```

**⚠️ IMPORTANT:** 
- Use a strong, random secret key (minimum 32 characters)
- Never use the default value in production
- Keep this secret secure and never commit it to version control

#### JWT Secret Generation Examples:

**Option 1: OpenSSL**
```bash
openssl rand -base64 32
# Example output: 8xK9fJ2mN5qR7tW3vY6zC1eH4gL0sP9uX2bA5dF8kM7n
```

**Option 2: Python**
```bash
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
# Example output: R3mK8xP2nF7qW9tY5zH1cL4gA6dS0uX3bV8nM2jQ5rE
```

**Option 3: Node.js**
```bash
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
# Example output: 2KxR9fJ5mN8qW3tY7zC1eH4gL6sP0uX2bA8dF5kM9nQ=
```

**Option 4: Online Generator**
- Use a secure online generator like: https://generate-secret.vercel.app/32

### Server Configuration

```bash
PORT=8080                 # Server port (optional, defaults to 8080)
```

### Complete Example Configurations

#### Development Environment (.env)
```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=mypassword123
DB_NAME=voting_db
DB_SSLMODE=disable

# Security
JWT_SECRET=R3mK8xP2nF7qW9tY5zH1cL4gA6dS0uX3bV8nM2jQ5rE7x9fJ2mN5qW8t

# Server
PORT=8080
```

#### Production Environment
```bash
# Database Configuration (using URL for better security)
DATABASE_URL=postgres://voting_user:secure_password_here@db.example.com:5432/voting_production?sslmode=require

# Security
JWT_SECRET=prod-secret-key-64-chars-long-with-special-chars-and-numbers-123
```

#### Docker Environment
```bash
# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=voting_user
DB_PASSWORD=docker_password
DB_NAME=voting_db
DB_SSLMODE=disable

# Security
JWT_SECRET=docker-secret-key-for-development-only-change-in-production

# Server
PORT=8080
```

#### Testing Environment (.env.test)
```bash
# JWT Secret for testing
JWT_SECRET=test-secret-key-for-testing-only-not-secure

# Server port for tests
PORT=8081

# Test database (if using real database for integration tests)
DB_HOST=localhost
DB_PORT=5432
DB_USER=test_user
DB_PASSWORD=test_password
DB_NAME=voting_test_db
DB_SSLMODE=disable
```

## Database Setup Instructions

### PostgreSQL Setup

#### 1. Install PostgreSQL

**macOS (Homebrew):**
```bash
brew install postgresql
brew services start postgresql
```

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**CentOS/RHEL:**
```bash
sudo yum install postgresql postgresql-server
sudo postgresql-setup initdb
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### 2. Create Database and User

```bash
# Connect to PostgreSQL as superuser
sudo -u postgres psql

# Create database user
CREATE USER voting_user WITH PASSWORD 'your_secure_password';

# Create database
CREATE DATABASE voting_db OWNER voting_user;

# Grant privileges
GRANT ALL PRIVILEGES ON DATABASE voting_db TO voting_user;

# Exit PostgreSQL
\q
```

#### 3. Test Database Connection

```bash
# Test connection
psql -h localhost -U voting_user -d voting_db

# If successful, you should see:
# voting_db=>
```

### Using Docker for PostgreSQL

#### 1. Create docker-compose.yml
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: voting_user
      POSTGRES_PASSWORD: docker_password
      POSTGRES_DB: voting_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

#### 2. Start PostgreSQL
```bash
docker-compose up -d
```

#### 3. Environment Variables for Docker
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=voting_user
DB_PASSWORD=docker_password
DB_NAME=voting_db
DB_SSLMODE=disable
```

## Security Best Practices

### JWT Secret Security

1. **Length:** Minimum 32 characters, recommended 64+
2. **Randomness:** Use cryptographically secure random generation
3. **Uniqueness:** Different secret for each environment
4. **Storage:** Never commit to version control
5. **Rotation:** Regularly rotate secrets in production

### Database Security

1. **Passwords:** Use strong, unique passwords
2. **SSL:** Enable SSL in production (`DB_SSLMODE=require`)
3. **Network:** Restrict database access to application servers only
4. **Backups:** Secure database backups with encryption

### Environment File Security

1. **File Permissions:**
   ```bash
   chmod 600 .env
   ```

2. **Git Ignore:** Ensure `.env` is in `.gitignore`
   ```bash
   echo ".env" >> .gitignore
   ```

3. **Production:** Use secure secret management systems (AWS Secrets Manager, HashiCorp Vault, etc.)

## Verification Steps

### 1. Environment Variables Check
```bash
# Check if variables are loaded correctly
go run -ldflags="-X main.version=test" main.go &
curl http://localhost:8080/health
# Should return: {"status":"ok"}
```

### 2. Database Connection Check
The application will automatically:
- Test database connection on startup
- Run database migrations
- Display connection success/failure messages

### 3. JWT Token Check
```bash
# Register a test user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com", 
    "password": "password123"
  }'

# Should return a JWT token in the response
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Failed
```
Error: Failed to connect to database: dial tcp [::1]:5432: connect: connection refused
```

**Solutions:**
- Check if PostgreSQL is running: `brew services list | grep postgresql`
- Verify connection details in `.env`
- Test manual connection: `psql -h localhost -U voting_user -d voting_db`

#### 2. Database Does Not Exist
```
Error: database "voting_db" does not exist
```

**Solutions:**
- Create the database as shown in setup instructions
- Verify database name in `.env`

#### 3. Authentication Failed for User
```
Error: password authentication failed for user "voting_user"
```

**Solutions:**
- Check username and password in `.env`
- Reset user password in PostgreSQL
- Verify user exists: `sudo -u postgres psql -c "\du"`

#### 4. JWT Token Issues
```
Error: Invalid token
```

**Solutions:**
- Check JWT_SECRET is set and matches between requests
- Verify token format: "Bearer your-token-here"
- Check token expiration (tokens last 7 days by default)

#### 5. Port Already in Use
```
Error: listen tcp :8080: bind: address already in use
```

**Solutions:**
- Change PORT in `.env` to another value (e.g., 8081)
- Kill process using port: `lsof -ti:8080 | xargs kill -9`

### Environment Validation Script

Create `check_env.sh`:
```bash
#!/bin/bash

echo "=== Environment Check ==="

# Check required variables
required_vars=("JWT_SECRET" "DB_HOST" "DB_PORT" "DB_USER" "DB_PASSWORD" "DB_NAME")

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "❌ $var is not set"
    else
        if [ "$var" = "JWT_SECRET" ] || [ "$var" = "DB_PASSWORD" ]; then
            echo "✅ $var is set (hidden)"
        else
            echo "✅ $var = ${!var}"
        fi
    fi
done

echo "=== Database Connection Test ==="
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT version();" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Database connection successful"
else
    echo "❌ Database connection failed"
fi
```

Make it executable and run:
```bash
chmod +x check_env.sh
source .env && ./check_env.sh
```

## Production Deployment

### Using System Environment Variables

Instead of `.env` files, set environment variables directly:

```bash
export DATABASE_URL="postgres://user:password@db.example.com:5432/voting_db?sslmode=require"
export JWT_SECRET="your-production-secret-key"
export PORT="8080"

./voting-api
```

### Using Docker Environment

```bash
docker run -d \
  -e DATABASE_URL="postgres://user:password@db:5432/voting_db" \
  -e JWT_SECRET="production-secret" \
  -e PORT="8080" \
  -p 8080:8080 \
  voting-api
```

### Using Kubernetes ConfigMaps and Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: voting-api-secrets
type: Opaque
stringData:
  jwt-secret: "your-production-jwt-secret"
  database-url: "postgres://user:password@db:5432/voting_db"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: voting-api-config
data:
  PORT: "8080"
```

This comprehensive guide should help you properly configure all environment variables for the Voting API in any environment!