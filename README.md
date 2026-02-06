# Intern Village

An AI agent orchestrator for software engineering tasks. Submit a task description, and autonomous AI agents will break it down into subtasks and implement each one as a separate pull request.

## How It Works

1. **Add a project** — Connect a GitHub repository
2. **Create a task** — Describe what you want built
3. **Planner agent runs** — Generates a spec and breaks the task into subtasks
4. **Start subtasks** — Each subtask is implemented by a Worker agent
5. **Review PRs** — Agents create pull requests for you to review and merge

## Quick Start

### Prerequisites

- **Go 1.22+** — `brew install go`
- **Node.js 18+** — `brew install node`
- **Docker** — For PostgreSQL
- **Claude CLI** — `npm install -g @anthropic-ai/claude-code`
- **Beads CLI** — `cargo install beads-cli`

See [PREREQUISITES.md](./PREREQUISITES.md) for detailed installation instructions.

### Setup

1. **Create a GitHub OAuth App:**
   - Go to https://github.com/settings/developers
   - Click "New OAuth App"
   - Set **Homepage URL** to `http://localhost:8080`
   - Set **Callback URL** to `http://localhost:8080/api/auth/github/callback`
   - Copy the Client ID and generate a Client Secret

2. **Configure environment:**
   ```bash
   cp .env.example .env
   ```

   Edit `.env` and fill in:
   ```
   GITHUB_CLIENT_ID=<your client id>
   GITHUB_CLIENT_SECRET=<your client secret>
   JWT_SECRET=<run: openssl rand -base64 32>
   ENCRYPTION_KEY=<run: openssl rand -hex 16>
   ```

3. **Install dependencies:**
   ```bash
   make setup
   ```

4. **Start all services:**
   ```bash
   make dev
   ```

5. **Open the app:**
   - Frontend: http://localhost:5173
   - API: http://localhost:8080

### Make Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start all services (PostgreSQL, API, Frontend) |
| `make stop` | Stop all services |
| `make logs` | Tail orchestrator logs |
| `make test` | Run all tests |
| `make clean` | Remove containers and build artifacts |

Run `make help` for the full list.

## Architecture

See [specs/architecture.md](./specs/architecture.md) for system design and key decisions.

## Specifications

| Spec | Description |
|------|-------------|
| [architecture.md](./specs/architecture.md) | System architecture and key decisions |
| [orchestrator.md](./specs/orchestrator.md) | Backend API, agent management, GitHub integration |
| [frontend.md](./specs/frontend.md) | React SPA, Kanban board, task management |

## License

Proprietary. Copyright (c) 2026 Intern Village.
