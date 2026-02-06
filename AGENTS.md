# Intern Village Agent Guidelines

## Quick Reference

| Task | Command |
|------|---------|
| Build all | `make build` |
| Test all | `make test` |
| Lint (Go) | `cd orchestrator && make lint` |
| Lint (Frontend) | `cd frontend && npm run lint` |
| Format (Go) | `cd orchestrator && make fmt` |
| Run dev (all) | `make dev` |
| Restart all | `make restart` |
| Check status | `make status` |
| Run dev (API only) | `make dev-api` |
| View logs | `make logs` |
| View frontend logs | `make logs-frontend` |
| Stop all | `make stop` |
| Run migrations | `make migrate` |

## Specifications & Documentation

- **Specs location:** `specs/`
  - `specs/architecture.md` - System architecture and key decisions
  - `specs/orchestrator.md` - Backend API, agent management, GitHub integration
  - `specs/orchestrator.impl.md` - Backend implementation checklist (all phases complete)
  - `specs/frontend.md` - React SPA specification
  - `specs/frontend.impl.md` - Frontend implementation checklist (all phases complete)
- **Prerequisites:** `PREREQUISITES.md` - Setup guide for all dependencies

**Guidance for agents:**
- Specs describe intent; code describes reality. Always verify against actual code.
- Check the codebase before concluding something is or isn't implemented.
- Follow design patterns defined in specs when implementing new features.
- The orchestrator spec (§7) contains state machine definitions for tasks and subtasks.

## Commands

### Build

```bash
# Build everything (orchestrator + frontend)
make build

# Build orchestrator only
make build-api
cd orchestrator && make build

# Build frontend only
make build-frontend
cd frontend && npm run build

# Build with embedded frontend (for Docker)
cd orchestrator && go build -tags embed_frontend -o bin/orchestrator ./cmd/orchestrator
```

### Test

```bash
# Run all tests
make test

# Orchestrator tests
cd orchestrator && make test
cd orchestrator && go test -v -race ./...

# Orchestrator tests with coverage
cd orchestrator && make test-coverage

# Frontend tests
cd frontend && npm run test:run

# Frontend tests with coverage
cd frontend && npm run test:coverage

# Run specific Go test
cd orchestrator && go test -v -run TestFunctionName ./path/to/package
```

### Lint & Format

```bash
# Go: format, vet, and lint
cd orchestrator && make check

# Go: format only
cd orchestrator && make fmt
cd orchestrator && go fmt ./...

# Go: lint only
cd orchestrator && make lint
cd orchestrator && golangci-lint run ./...

# Frontend: lint
cd frontend && npm run lint
```

### Development

```bash
# Start all services (PostgreSQL, API, Frontend)
make dev

# Check if services are running
make status

# Restart all services (stops existing, then starts fresh)
make restart

# Start PostgreSQL and API only (foreground mode)
make dev-api

# Start frontend only
make dev-frontend

# Stop all services
make stop

# View orchestrator logs
make logs

# View frontend logs
make logs-frontend
```

## Database

- **Database type:** PostgreSQL 16 (Alpine)
- **Migrations location:** `orchestrator/migrations/`
- **Migration naming:** `NNN_description.sql` (e.g., `001_initial.sql`)
- **Migration tool:** Goose-style annotations (`-- +goose Up`, `-- +goose Down`)

### Running Migrations

**IMPORTANT:** Migrations do NOT run automatically on first run. Apply them after starting the database.

```bash
# Run all migrations (recommended)
make migrate

# Or run manually for a specific migration:
cat orchestrator/migrations/001_initial.sql | \
  sed -n '/+goose Up/,/+goose Down/p' | \
  grep -v '+goose' | \
  docker exec -i intern-village-postgres psql -U intern -d intern_village

# Check if tables exist
docker exec intern-village-postgres psql -U intern -d intern_village -c "\dt"

# Connect to database interactively
docker exec -it intern-village-postgres psql -U intern -d intern_village
```

### Available Migrations

| Migration | Description |
|-----------|-------------|
| `001_initial.sql` | Initial schema: users, projects, tasks, subtasks, agent_runs |
| `002_agent_runs_task_id.sql` | Add `task_id` to agent_runs for Planner agents (task-level runs) |

### Important Notes

- **Migrations must be run manually** after starting PostgreSQL for the first time
- If you see `ERROR: relation "users" does not exist`, migrations haven't been run
- After adding new migrations, run them before testing the affected functionality
- The database container persists data unless you run `make clean`

## Local Development

### Prerequisites

See `PREREQUISITES.md` for detailed installation instructions.

Required:
- Go 1.22+
- Node.js 20+
- Docker (for PostgreSQL)
- Claude CLI (for AI agents)
- Beads CLI (for task tracking)

### Environment Variables

Copy `.env.example` to `.env` and fill in:

| Variable | Description | Required |
|----------|-------------|----------|
| `GITHUB_CLIENT_ID` | GitHub OAuth App client ID | Yes |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth App client secret | Yes |
| `JWT_SECRET` | JWT signing secret (32+ chars) | Yes |
| `ENCRYPTION_KEY` | AES-256 key (exactly 32 chars) | Yes |
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude | Yes |
| `DATABASE_URL` | PostgreSQL connection string | Auto-set by Makefile |
| `PORT` | HTTP server port | 8080 |
| `LOG_LEVEL` | Logging level | info |

### Running Locally

```bash
# First time setup
make setup

# Start all services (builds, starts DB, API, frontend)
make dev

# Run migrations (required on first run!)
make migrate

# Check everything is running
make status

# Access the app
# Frontend: http://localhost:5173
# API: http://localhost:8080
```

### Common Operations

```bash
# If services are already running and you need to restart
make restart

# If login or other features aren't working, check status and restart
make status
make restart

# View logs to debug issues
make logs            # API logs
make logs-frontend   # Frontend logs
```

## Architecture Overview

```
intern-village/
├── orchestrator/           # Go backend
│   ├── cmd/orchestrator/   # Entry point
│   ├── internal/
│   │   ├── api/            # HTTP handlers, middleware, server
│   │   ├── agent/          # Claude CLI execution, prompts, loop logic
│   │   ├── config/         # Environment configuration
│   │   ├── domain/         # Core types, state machines, errors
│   │   ├── repository/     # Database access, crypto
│   │   └── service/        # Business logic services
│   ├── generated/db/       # sqlc-generated code
│   ├── migrations/         # SQL migrations
│   └── prompts/            # Agent prompt templates
├── frontend/               # React SPA
│   └── src/
│       ├── api/            # API client functions
│       ├── components/     # React components (ui/, board/, layout/, etc.)
│       ├── hooks/          # Custom hooks (useAuth, useTasks, etc.)
│       ├── pages/          # Page components
│       ├── lib/            # Utilities
│       └── types/          # TypeScript types
├── specs/                  # Design specifications
├── docker-compose.yml      # Docker deployment
└── Makefile                # Development commands
```

**Key modules:**
- `internal/domain` - Core types: User, Project, Task, Subtask, AgentRun + state machines
- `internal/service` - Business logic: TaskService, SubtaskService, GitHubService, BeadsService
- `internal/agent` - Agent execution: Executor, AgentLoop, AgentManager, PromptRenderer
- `internal/api/handlers` - REST endpoints for auth, projects, tasks, subtasks, agents

**Key patterns:**
- Services contain business logic; handlers are thin wrappers
- State machines defined in `domain/states.go` govern Task and Subtask transitions
- Beads CLI is source of truth for dependencies; synced to Postgres for UI queries
- Agents run in loop-until-done pattern with exponential backoff

## Chi Router Patterns

The API uses [go-chi/chi](https://github.com/go-chi/chi) for routing. Important patterns to follow:

### Route Shadowing

When using `r.Route("/path", ...)`, chi creates a subrouter that handles ALL methods for that path. If you define the same path outside AND inside a group, the **subrouter shadows the outer route**.

**Wrong pattern (POST shadowed by subrouter - returns 405):**
```go
// This POST is shadowed by the r.Route("/projects") below!
r.With(authMiddleware.RequireAuth, chimw.Timeout(10*time.Minute)).Post("/projects", handler.Create)

r.Group(func(r chi.Router) {
    r.Use(authMiddleware.RequireAuth)
    r.Use(chimw.Timeout(60 * time.Second))
    r.Route("/projects", func(r chi.Router) {
        r.Get("/", handler.List)  // This subrouter takes over /projects
        // No POST handler here, so POST /projects returns 405!
    })
})
```

**Correct pattern (all methods inside the Route block):**
```go
r.Group(func(r chi.Router) {
    r.Use(authMiddleware.RequireAuth)
    r.Use(chimw.Timeout(60 * time.Second))
    r.Route("/projects", func(r chi.Router) {
        // Extended timeout for POST only, applied inside the Route block
        r.With(chimw.Timeout(10 * time.Minute)).Post("/", handler.Create)
        r.Get("/", handler.List)
    })
})
```

### Timeout Middleware

Chi's `chimw.Timeout` middleware can be applied per-route using `r.With()`. When you need a different timeout for a specific route, apply it inside the Route block.

**Note:** The group-level timeout (60s) applies to all routes, but `r.With(chimw.Timeout(...))` on a specific route overrides it for that route.

### Current Extended Timeout Routes

These routes have extended timeouts (10 minutes) for long-running operations:
- `POST /api/projects` - Cloning large repositories (defined inside `/projects` route with `r.With()`)

### Routes Without Timeout Middleware

SSE (Server-Sent Events) endpoints must NOT use timeout middleware because they are long-lived connections. These endpoints manage their own timeouts internally:
- `GET /api/projects/{project_id}/events` - SSE stream for real-time events (defined outside the 60s timeout group, uses internal `SSEConnectionTimeoutM` default 60 minutes)

### SSE and Response Writer Wrapping

SSE endpoints require the `http.ResponseWriter` to implement `http.Flusher` for streaming. Any middleware that wraps the response writer (like the Logger middleware) must also implement `Flush()`:

```go
// In middleware/logging.go - responseWriter must implement Flusher
func (rw *responseWriter) Flush() {
    if f, ok := rw.ResponseWriter.(http.Flusher); ok {
        f.Flush()
    }
}
```

**Common SSE issues:**
- `"streaming not supported"` error → Response writer wrapper doesn't implement `http.Flusher`
- Connection drops after 60s → SSE route is inside the timeout middleware group

### Log Tailer Pattern

The `LogTailer.StartTailing()` method is a **blocking call** that continuously reads a log file until cancelled. When integrating with agent spawning:

```go
// CORRECT: Start tailer in its own goroutine, inside the agent goroutine
go func() {
    go func() {
        logTailer.StartTailing(ctx, projectID, runID, logPath)
    }()
    loop.RunAgentLoop(ctx, ...)  // Agent creates the log file
}()

// WRONG: Blocks forever or times out waiting for file
logTailer.StartTailing(ctx, projectID, runID, logPath)  // Blocking!
go func() {
    loop.RunAgentLoop(ctx, ...)  // Never runs
}()
```

**Key points:**
- `StartTailing()` waits up to 5s for the log file to exist
- It must run in its own goroutine (non-blocking)
- It should start after the agent goroutine begins so the agent can create the log file first

## Code Style

### Go (Orchestrator)

- **Formatting:** `gofmt` / `goimports`
- **Linter config:** `.golangci.yml`
- **Error handling:** Return errors, don't panic. Use domain error types.
- **Naming:**
  - Files: `snake_case.go`
  - Functions/methods: `camelCase` (exported: `PascalCase`)
  - Types/interfaces: `PascalCase`

### TypeScript (Frontend)

- **Formatting:** ESLint
- **Framework:** React 19, Vite, TanStack Query, Tailwind CSS v4
- **Components:** shadcn/ui (Radix-based, manually maintained)
- **Naming:**
  - Files: `PascalCase.tsx` for components, `camelCase.ts` for utilities
  - Components: `PascalCase`
  - Hooks: `useCamelCase`

## Testing Guidelines

### Orchestrator (Go)

- **Framework:** Go testing + testify (assertions)
- **Location:** `*_test.go` files alongside source
- **Run:** `cd orchestrator && make test`

Key test files:
- `domain/states_test.go` - State machine transitions
- `agent/loop_test.go` - Backoff calculations
- `repository/crypto_test.go` - Token encryption
- `api/handlers/*_test.go` - Handler response formats

### Frontend (React)

- **Framework:** Vitest + Testing Library
- **Location:** `*.test.tsx` alongside components
- **Run:** `cd frontend && npm run test:run`

Key test files:
- `lib/utils.test.ts` - Utility functions
- `components/board/SubtaskCard.test.tsx` - Card states and actions
- `components/projects/AddProjectDialog.test.tsx` - Form validation

## Security Guidelines

- **GitHub tokens:** Encrypted at rest with AES-256-GCM (`repository/crypto.go`)
- **JWT:** HttpOnly cookies, 24h expiry, HS256 signing
- **Secrets:** Never commit `.env`, use `.env.example` as template
- **Agent isolation:** Each agent runs in isolated git worktree
- **Logging:** Never log tokens or sensitive data

## Common Gotchas

1. **"relation does not exist" errors:** Migrations haven't been run. See Database section above.

2. **OAuth callback fails:** Ensure GitHub OAuth app callback URL matches exactly: `http://localhost:8080/api/auth/github/callback`

3. **Frontend 401 errors:** JWT cookie not set or expired. Re-authenticate via GitHub OAuth.

4. **Port already in use:** Run `make stop` to kill existing processes on ports 8080/5173.

5. **sqlc changes not reflected:** After modifying SQL queries, run `cd orchestrator && make sqlc` to regenerate.

6. **Beads CLI not found:** Install Beads CLI per `PREREQUISITES.md`. Required for agent execution.

7. **Build tag for frontend embedding:** Production Docker builds use `-tags embed_frontend`. Development builds don't embed frontend.

8. **405 Method Not Allowed on valid routes:** Chi router shadowing issue. If you define a route outside a group and then use `r.Route()` for the same path inside a group, the subrouter shadows the outer route. **Solution:** Define all HTTP methods for a path inside the same `r.Route()` block. See "Chi Router Patterns" section above.

9. **Request timeout on large repo clone:** Project creation has 10-minute timeout. If cloning still times out, the repo may be too large or network is slow. Check `/tmp/orchestrator.log` for "signal: killed" errors.

## Troubleshooting

### Internal Server Error on GitHub Login

- **Symptom:** 500 error when GitHub OAuth callback completes
- **Cause:** Database tables don't exist (migrations not run)
- **Fix:** Run migrations:
  ```bash
  cat orchestrator/migrations/001_initial.sql | \
    sed -n '/+goose Up/,/+goose Down/p' | \
    grep -v '+goose' | \
    docker exec -i intern-village-postgres psql -U intern -d intern_village
  ```

### PostgreSQL Connection Refused

- **Symptom:** `connection refused` or `no such host`
- **Cause:** PostgreSQL container not running
- **Fix:** `make start-db` or `docker start intern-village-postgres`

### Frontend Can't Reach API

- **Symptom:** Network errors in browser console
- **Cause:** API not running or wrong port
- **Fix:** Ensure orchestrator is running on port 8080. Check `make logs`.

### Agent Execution Fails

- **Symptom:** Subtasks stuck in IN_PROGRESS or move to BLOCKED(FAILURE)
- **Cause:** Claude CLI not configured or Beads CLI not installed
- **Fix:** Verify `claude --version` and `bd --version` work. Set `ANTHROPIC_API_KEY`.

## External Services

| Service | Purpose | Docs |
|---------|---------|------|
| GitHub OAuth | User authentication | Create app at github.com/settings/developers |
| GitHub API | Repo ops, PR creation | Via go-github SDK |
| Anthropic API | Claude CLI for agents | console.anthropic.com |
| Beads | Task dependencies | Internal CLI tool |
