# Prerequisites for Intern Village

This document describes the prerequisites for running Intern Village locally or in production.

## Required Software

### 1. Go 1.22+

Intern Village's orchestrator is written in Go.

**macOS (Homebrew):**
```bash
brew install go
```

**Linux:**
```bash
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

**Verify installation:**
```bash
go version
# Should output: go version go1.24.x ...
```

### 2. PostgreSQL 14+

Used for storing user data, projects, tasks, and agent run history.

**macOS (Homebrew):**
```bash
brew install postgresql@16
brew services start postgresql@16
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql
```

**Docker (recommended for development):**
```bash
docker run -d \
  --name intern-village-postgres \
  -e POSTGRES_USER=intern \
  -e POSTGRES_PASSWORD=intern \
  -e POSTGRES_DB=intern_village \
  -p 5432:5432 \
  postgres:16-alpine
```

**Create database (if not using Docker):**
```bash
createdb intern_village
```

### 3. Claude CLI

The orchestrator spawns Claude CLI processes to execute Planner and Worker agents.

**Installation:**
```bash
npm install -g @anthropic-ai/claude-code
```

**Configuration:**
```bash
# Set your Anthropic API key
export ANTHROPIC_API_KEY=sk-ant-your-api-key

# Verify installation
claude --version
```

**Note:** Get an API key at https://console.anthropic.com/

### 4. Beads CLI

Required for task tracking, dependency management, and agent state coordination.

The orchestrator uses Beads to:
- Track subtask dependencies (`bd dep add`)
- Manage git worktrees for agents (`bd worktree create`)
- Detect when agents complete their work (`bd close`)

**Installation:**
```bash
npm install -g @beads/bd
```

**Verify installation:**
```bash
bd --version
# Should output: bd version 0.49.x ...
```

**Note:** Beads requires a daemon process. It starts automatically when needed, but you can manually start it with:
```bash
bd daemon start
```

### 5. Git

Required for cloning repositories and managing worktrees.

**macOS:**
```bash
# Git comes with Xcode Command Line Tools
xcode-select --install
```

**Linux:**
```bash
sudo apt-get install git
```

## GitHub OAuth Application

You need to create a GitHub OAuth application for user authentication.

### Steps:

1. Go to https://github.com/settings/developers
2. Click "New OAuth App"
3. Fill in the form:
   - **Application name:** Intern Village (Development)
   - **Homepage URL:** http://localhost:8080
   - **Authorization callback URL:** http://localhost:8080/api/auth/github/callback
4. Click "Register application"
5. Copy the **Client ID**
6. Generate a new **Client Secret** and copy it
7. Add both to your `.env` file:
   ```
   GITHUB_CLIENT_ID=your-client-id
   GITHUB_CLIENT_SECRET=your-client-secret
   ```

**Note:** For production, update the URLs to your production domain.

## Environment Setup

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Generate secure secrets:**
   ```bash
   # JWT secret (at least 32 characters)
   openssl rand -base64 32

   # Encryption key (exactly 32 hex characters = 16 bytes)
   openssl rand -hex 16
   ```

3. **Fill in all required values in `.env`**

## Running the Orchestrator

### Local Development

```bash
# From project root

# Install dependencies (first time only)
make setup

# Start all services (PostgreSQL, API, Frontend)
make dev

# Run migrations (first time only)
make migrate

# Check everything is running
make status

# View logs
make logs
```

**Common commands:**
```bash
make restart    # Stop and restart all services
make stop       # Stop API and Frontend (keeps PostgreSQL)
make stop-db    # Stop PostgreSQL container
```

### Docker Compose

```bash
# From project root
docker-compose up -d

# View logs
docker-compose logs -f orchestrator

# Stop
docker-compose down
```

## Verifying Installation

1. **Check the health endpoint:**
   ```bash
   curl http://localhost:8080/health
   # Should return: {"status":"ok"}
   ```

2. **Test GitHub OAuth:**
   - Open http://localhost:8080/api/auth/github in your browser
   - You should be redirected to GitHub for authorization

## Troubleshooting

### "Claude CLI not found"
Ensure npm global packages are in your PATH:
```bash
export PATH="$PATH:$(npm config get prefix)/bin"
```

### "Database connection refused"
1. Check PostgreSQL is running:
   ```bash
   pg_isready
   ```
2. Verify your DATABASE_URL in `.env`

### "GitHub OAuth error"
1. Verify Client ID and Secret are correct
2. Ensure callback URL matches exactly (including trailing slash)
3. Check that the OAuth app is not suspended

### "Beads CLI not found"
Beads is required for the system to function. Agents use Beads to signal task completion,
and the orchestrator checks Beads state to know when to create PRs. Install it with:
```bash
npm install -g @beads/bd
```

### "Beads daemon not running"
Beads uses a background daemon for coordination. Start it manually if needed:
```bash
bd daemon start
```
