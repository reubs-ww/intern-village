<!--
 Copyright (c) 2026 Intern Village. All rights reserved.
 SPDX-License-Identifier: Proprietary
-->

# Orchestrator Specification

**Status:** Complete
**Version:** 1.0
**Last Updated:** 2026-02-04
**Architecture Reference:** [architecture.md](./architecture.md)

---

## 1. Overview

### Purpose

The Orchestrator is the backend server that coordinates all operations in Intern Village. It manages the lifecycle of tasks and subtasks, spawns and monitors AI agents (Claude CLI processes), handles GitHub integration (OAuth, cloning, PRs), and provides a REST API for the Web UI.

The Orchestrator bridges user intent (task descriptions) with autonomous agent execution (Planner and Worker agents) while maintaining state consistency between Beads (source of truth for dependencies/agent state) and Postgres (source of truth for user data and UI queries).

### Goals

- **Task Management**: CRUD operations for projects, tasks, and subtasks with state machine transitions
- **Agent Orchestration**: Spawn Claude CLI processes, monitor execution, handle retries with loop-until-done pattern
- **GitHub Integration**: OAuth authentication, repo clone/fork detection, branch management, PR creation
- **State Synchronization**: Keep Beads and Postgres in sync for dependencies and status
- **API Server**: Provide REST endpoints for Web UI consumption

### Non-Goals

- **Real-time WebSockets**: Polling is sufficient for MVP (architecture decision)
- **Container-per-agent**: Git worktrees provide isolation without container overhead
- **Custom LLM hosting**: Claude CLI handles all AI execution
- **Microservices**: Single Go binary for simplicity

### Context from Architecture

Key decisions affecting this spec:

- **Beads as source of truth** for dependencies and agent state; sync to Postgres for UI queries
- **Git worktrees** for parallel agent isolation (one worktree per subtask)
- **Planner + Worker agents**: Planner generates specs/subtasks, Workers implement
- **Loop-until-done execution**: Agents retry up to 10 times until beads state is complete
- **Prompt-based agents**: No Claude CLI skills; full prompts piped via markdown files
- **Git operation split**: Claude does `add/commit`, Orchestrator does `push/PR`

---

## 2. Package Structure

```
orchestrator/
├── cmd/
│   └── orchestrator/
│       └── main.go                 # Entry point, config loading, server startup
├── internal/
│   ├── api/
│   │   ├── server.go               # HTTP server setup, middleware, routing
│   │   ├── middleware/
│   │   │   ├── auth.go             # JWT validation, session extraction
│   │   │   └── logging.go          # Request logging
│   │   └── handlers/
│   │       ├── projects.go         # Project CRUD endpoints
│   │       ├── tasks.go            # Task CRUD endpoints
│   │       ├── subtasks.go         # Subtask CRUD endpoints
│   │       ├── agents.go           # Agent status, logs endpoints
│   │       └── auth.go             # GitHub OAuth flow endpoints
│   ├── domain/
│   │   ├── models.go               # Core domain types (Project, Task, Subtask, etc.)
│   │   ├── states.go               # State machine definitions
│   │   └── errors.go               # Domain error types
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── repository.go       # Postgres repository implementation
│   │   │   ├── projects.go         # Project queries
│   │   │   ├── tasks.go            # Task queries
│   │   │   ├── subtasks.go         # Subtask queries
│   │   │   ├── users.go            # User queries
│   │   │   └── agent_runs.go       # Agent run history queries
│   │   └── repository.go           # Repository interface
│   ├── service/
│   │   ├── task_manager.go         # Task/subtask business logic
│   │   ├── agent_manager.go        # Agent spawning, monitoring, retries
│   │   ├── github_service.go       # GitHub API operations
│   │   ├── beads_service.go        # Beads CLI wrapper
│   │   └── sync_service.go         # Beads <-> Postgres sync
│   ├── agent/
│   │   ├── executor.go             # Claude CLI process management
│   │   ├── loop.go                 # Loop-until-done execution logic
│   │   └── prompts.go              # Prompt template rendering
│   └── config/
│       └── config.go               # Configuration struct and loading
├── migrations/
│   └── *.sql                       # Database migrations
├── prompts/
│   ├── planner.md                  # Planner agent prompt template
│   └── worker.md                   # Worker agent prompt template
├── generated/
│   └── db/                         # sqlc generated code
├── sqlc.yaml                       # sqlc configuration
├── go.mod
└── go.sum
```

---

## 3. User Flows

### Flow 1: Add Project

1. User clicks "Add Project" on dashboard
2. User enters GitHub repo URL (e.g., `github.com/owner/repo`)
3. Orchestrator checks user's push permissions via GitHub API
4. If push access: clone repo; else: fork first, then clone
5. Initialize Beads in cloned repo with stealth mode (`bd init --stealth --prefix iv-{id}`)
6. Create project record in Postgres
7. User sees project in dashboard, clicks to open board

### Flow 2: Create Task

1. User is on project board, clicks "New Task"
2. User enters title + description paragraph
3. Orchestrator creates task record (status: `PLANNING`)
4. **Orchestrator syncs repo to latest** (see §9.5 Repository Sync Strategy)
5. Orchestrator spawns Planner agent **in the main clone directory** (not a worktree)
6. Planner explores codebase, generates spec, creates subtasks via `bd create`
7. Orchestrator syncs Beads state to Postgres
8. Task transitions to `ACTIVE`, subtasks appear on board in "Ready" or "Blocked" columns
9. **User must manually start each subtask** - no auto-start

### Flow 3: Start Subtask

1. User views subtask board, sees subtask in "Ready" column
2. User clicks "Start" on the subtask (manual action required)
3. **Orchestrator syncs repo to latest** (see §9.5 Repository Sync Strategy)
4. Orchestrator creates worktree for subtask (`bd worktree create`)
5. Orchestrator spawns Worker agent with subtask spec/plan
6. Worker implements, tests, commits in loop-until-done pattern
7. On success: Orchestrator pushes branch, creates PR
8. Subtask moves to "Completed" with PR link

### Flow 4: Mark Merged

1. User reviews PR on GitHub, merges it
2. User returns to board, clicks "Mark Merged" on subtask
3. Orchestrator updates subtask status to `MERGED`
4. Orchestrator closes beads issue (`bd close`)
5. Dependents check: if all dependencies merged, move from `BLOCKED` to `READY`
6. If all task's subtasks merged, task transitions to `DONE`
7. Worktree cleaned up

---

## 4. Data Model

### 4.1 User

GitHub-authenticated user.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | UUID | Yes | Primary key |
| github_id | int64 | Yes | GitHub user ID |
| github_username | string | Yes | GitHub login |
| github_token | string (encrypted) | Yes | OAuth access token (AES-256-GCM) |
| created_at | timestamptz | Yes | Creation timestamp |
| updated_at | timestamptz | Yes | Last update timestamp |

### 4.2 Project

A GitHub repository the user works on.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | UUID | Yes | Primary key |
| user_id | UUID | Yes | Owner user |
| github_owner | string | Yes | Repo owner (org or user) |
| github_repo | string | Yes | Repo name |
| is_fork | boolean | Yes | Whether we forked this repo |
| upstream_owner | string | No | Original repo owner (only for forks) |
| upstream_repo | string | No | Original repo name (only for forks) |
| default_branch | string | Yes | Default branch name (e.g., "main") |
| clone_path | string | Yes | Local filesystem path to clone |
| beads_prefix | string | Yes | Beads issue prefix (e.g., "iv-") |
| created_at | timestamptz | Yes | Creation timestamp |
| updated_at | timestamptz | Yes | Last update timestamp |

**Relationships:**
- Belongs to: User
- Has many: Tasks

### 4.3 Task

User-submitted work item.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | UUID | Yes | Primary key |
| project_id | UUID | Yes | Parent project |
| title | string | Yes | Task title |
| description | text | Yes | Task description (user input) |
| status | enum | Yes | `PLANNING`, `ACTIVE`, `DONE` |
| beads_epic_id | string | No | Beads epic ID (e.g., "iv-1") |
| created_at | timestamptz | Yes | Creation timestamp |
| updated_at | timestamptz | Yes | Last update timestamp |

**Relationships:**
- Belongs to: Project
- Has many: Subtasks

### 4.4 Subtask

Broken-down work unit.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | UUID | Yes | Primary key |
| task_id | UUID | Yes | Parent task |
| title | string | Yes | Subtask title |
| spec | text | No | Specification (generated by Planner) |
| implementation_plan | text | No | Implementation plan (generated by Planner) |
| status | enum | Yes | `PENDING`, `READY`, `BLOCKED`, `IN_PROGRESS`, `COMPLETED`, `MERGED` |
| blocked_reason | enum | No | `DEPENDENCY`, `FAILURE` (only when status=BLOCKED) |
| branch_name | string | No | Git branch for this subtask |
| pr_url | string | No | GitHub PR URL |
| pr_number | int | No | GitHub PR number |
| retry_count | int | Yes | Current retry count (0-10) |
| token_usage | int | Yes | Total tokens used |
| position | int | Yes | Display order within task (for drag-and-drop) |
| beads_issue_id | string | No | Beads issue ID (e.g., "iv-2") |
| worktree_path | string | No | Path to git worktree |
| created_at | timestamptz | Yes | Creation timestamp |
| updated_at | timestamptz | Yes | Last update timestamp |

**Relationships:**
- Belongs to: Task
- Has many: SubtaskDependencies (as dependent)
- Has many: SubtaskDependencies (as dependency)
- Has many: AgentRuns

### 4.5 SubtaskDependency

Tracks which subtasks block others.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | UUID | Yes | Primary key |
| subtask_id | UUID | Yes | The subtask that is blocked |
| depends_on_id | UUID | Yes | The subtask that must complete first |
| created_at | timestamptz | Yes | Creation timestamp |

**Constraints:**
- Unique (subtask_id, depends_on_id)

### 4.6 AgentRun

History of agent executions.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | UUID | Yes | Primary key |
| subtask_id | UUID | No* | Associated subtask (for Worker runs) |
| task_id | UUID | No* | Associated task (for Planner runs) |
| agent_type | enum | Yes | `PLANNER`, `WORKER` |
| attempt_number | int | Yes | Which retry attempt (1-10) |
| status | enum | Yes | `RUNNING`, `SUCCEEDED`, `FAILED` |
| started_at | timestamptz | Yes | Start timestamp |
| ended_at | timestamptz | No | End timestamp |
| token_usage | int | No | Tokens used in this run |
| error_message | text | No | Error message if failed |
| log_path | string | Yes | Path to full log file |
| prompt_text | text | Yes | Full rendered prompt for this run |
| created_at | timestamptz | Yes | Creation timestamp |

*Either `subtask_id` or `task_id` must be set:
- **Planner runs**: `task_id` is set, `subtask_id` is null (runs before subtasks exist)
- **Worker runs**: `subtask_id` is set, `task_id` can be null (derived from subtask)

**Relationships:**
- Belongs to: Subtask (for Worker) OR Task (for Planner)

---

## 5. API Design

### Base Path

All endpoints under `/api/*`

### Authentication

- **OAuth endpoints**: No auth required
- **API endpoints**: Require valid JWT in `Authorization: Bearer {token}`

### Endpoints

#### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/auth/github` | No | Initiate GitHub OAuth flow |
| GET | `/api/auth/github/callback` | No | GitHub OAuth callback |
| POST | `/api/auth/logout` | Yes | Invalidate session |
| GET | `/api/auth/me` | Yes | Get current user info |

#### Projects

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/projects` | Yes | List user's projects |
| POST | `/api/projects` | Yes | Add new project |
| GET | `/api/projects/{id}` | Yes | Get project by ID |
| DELETE | `/api/projects/{id}` | Yes | Delete project |
| POST | `/api/projects/{id}/cleanup` | Yes | Manual cleanup (delete clone) |

#### Tasks

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/projects/{project_id}/tasks` | Yes | List tasks for project |
| POST | `/api/projects/{project_id}/tasks` | Yes | Create new task |
| GET | `/api/tasks/{id}` | Yes | Get task by ID |
| DELETE | `/api/tasks/{id}` | Yes | Delete task |

#### Subtasks

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/tasks/{task_id}/subtasks` | Yes | List subtasks for task |
| GET | `/api/subtasks/{id}` | Yes | Get subtask by ID |
| POST | `/api/subtasks/{id}/start` | Yes | Start worker agent |
| POST | `/api/subtasks/{id}/mark-merged` | Yes | Mark as merged |
| POST | `/api/subtasks/{id}/retry` | Yes | Retry failed subtask |
| PATCH | `/api/subtasks/{id}/position` | Yes | Update position (drag-and-drop) |

#### Agents

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/subtasks/{id}/runs` | Yes | List agent runs for subtask |
| GET | `/api/runs/{id}/logs` | Yes | Get agent run logs |
| GET | `/api/runs/{id}/logs/stream` | Yes | Stream logs (SSE) |

### Request/Response Examples

#### Create Project

**Request:**
```json
POST /api/projects
{
  "repo_url": "github.com/owner/repo"
}
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "github_owner": "owner",
  "github_repo": "repo",
  "is_fork": false,
  "default_branch": "main",
  "created_at": "2026-02-04T00:00:00Z"
}
```

#### Create Task

**Request:**
```json
POST /api/projects/{project_id}/tasks
{
  "title": "Add user authentication",
  "description": "Implement OAuth login with GitHub. Users should be able to sign in and see their profile."
}
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Add user authentication",
  "description": "Implement OAuth login with GitHub...",
  "status": "PLANNING",
  "created_at": "2026-02-04T00:00:00Z"
}
```

#### Start Subtask

**Request:**
```json
POST /api/subtasks/{id}/start
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440002",
  "status": "IN_PROGRESS",
  "branch_name": "iv-2-add-oauth-handler",
  "worktree_path": "/data/worktrees/iv-2",
  "current_run": {
    "id": "550e8400-e29b-41d4-a716-446655440003",
    "attempt_number": 1,
    "status": "RUNNING",
    "started_at": "2026-02-04T00:01:00Z"
  }
}
```

### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | INVALID_REQUEST | Request body validation failed |
| 401 | UNAUTHORIZED | Missing or invalid JWT |
| 403 | FORBIDDEN | User doesn't own this resource |
| 404 | NOT_FOUND | Resource not found |
| 409 | CONFLICT | Invalid state transition (e.g., starting already running subtask) |
| 422 | UNPROCESSABLE | Cannot perform action (e.g., start blocked subtask) |

---

## 6. Database Schema

### Migration: `001_initial.sql`

```sql
-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id BIGINT NOT NULL UNIQUE,
    github_username TEXT NOT NULL,
    github_token TEXT NOT NULL,  -- Encrypted with AES-256-GCM
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_github_id ON users(github_id);

-- Projects
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    github_owner TEXT NOT NULL,
    github_repo TEXT NOT NULL,
    is_fork BOOLEAN NOT NULL DEFAULT FALSE,
    upstream_owner TEXT,              -- Original repo owner (only for forks)
    upstream_repo TEXT,               -- Original repo name (only for forks)
    default_branch TEXT NOT NULL DEFAULT 'main',
    clone_path TEXT NOT NULL,
    beads_prefix TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, github_owner, github_repo)
);

CREATE INDEX idx_projects_user_id ON projects(user_id);

-- Tasks
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PLANNING',
    beads_epic_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_status ON tasks(status);

-- Subtasks
CREATE TABLE subtasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    spec TEXT,
    implementation_plan TEXT,
    status TEXT NOT NULL DEFAULT 'PENDING',
    blocked_reason TEXT,
    branch_name TEXT,
    pr_url TEXT,
    pr_number INTEGER,
    retry_count INTEGER NOT NULL DEFAULT 0,
    token_usage INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    beads_issue_id TEXT,
    worktree_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subtasks_task_id ON subtasks(task_id);
CREATE INDEX idx_subtasks_status ON subtasks(status);
CREATE INDEX idx_subtasks_beads_issue_id ON subtasks(beads_issue_id);

-- Subtask Dependencies
CREATE TABLE subtask_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subtask_id UUID NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    depends_on_id UUID NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(subtask_id, depends_on_id)
);

CREATE INDEX idx_subtask_deps_subtask_id ON subtask_dependencies(subtask_id);
CREATE INDEX idx_subtask_deps_depends_on_id ON subtask_dependencies(depends_on_id);

-- Agent Runs
-- Note: Either subtask_id (Worker) or task_id (Planner) must be set
CREATE TABLE agent_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subtask_id UUID REFERENCES subtasks(id) ON DELETE CASCADE,
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    agent_type TEXT NOT NULL,
    attempt_number INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'RUNNING',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    token_usage INTEGER,
    error_message TEXT,
    log_path TEXT NOT NULL,
    prompt_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_agent_run_parent CHECK (subtask_id IS NOT NULL OR task_id IS NOT NULL)
);

CREATE INDEX idx_agent_runs_subtask_id ON agent_runs(subtask_id);
CREATE INDEX idx_agent_runs_task_id ON agent_runs(task_id);
CREATE INDEX idx_agent_runs_status ON agent_runs(status);
```

---

## 7. Business Logic

### 7.1 Task State Machine

**States:** `PLANNING` → `ACTIVE` → `DONE`

| Current | Event | Next | Action |
|---------|-------|------|--------|
| PLANNING | Planner completes | ACTIVE | Sync subtasks from Beads |
| ACTIVE | All subtasks MERGED | DONE | (auto-transition) |

**Multiple Concurrent Tasks:**

Users can have multiple tasks active on the same project simultaneously:
- Multiple Planners can run concurrently (read-only, no conflicts)
- Each Planner creates its own epic and subtasks
- Workers are isolated via worktrees, so no conflicts
- Board shows all tasks grouped by parent task

### 7.2 Subtask State Machine

**States:** `PENDING`, `READY`, `BLOCKED`, `IN_PROGRESS`, `COMPLETED`, `MERGED`

| Current | Event | Next | Action |
|---------|-------|------|--------|
| PENDING | Planner sets deps | READY or BLOCKED | Check dependencies |
| BLOCKED (DEPENDENCY) | All deps MERGED | READY | Unblock |
| READY | User clicks Start | IN_PROGRESS | Spawn worker agent |
| IN_PROGRESS | Agent succeeds | COMPLETED | Push, create PR |
| IN_PROGRESS | Agent fails 10x | BLOCKED (FAILURE) | Needs human intervention |
| COMPLETED | User clicks Mark Merged | MERGED | Close beads issue, cleanup |
| BLOCKED (FAILURE) | User clicks Retry | IN_PROGRESS | Reset retry count, spawn agent |

**Edge Cases:**

| Scenario | Behavior |
|----------|----------|
| Start already in_progress subtask | 409 Conflict |
| Start blocked subtask | 422 Unprocessable |
| Mark merged without PR | 422 Unprocessable |
| Delete task with in_progress subtasks | Kill agents first, then delete |

### 7.3 Agent Execution Loop

**Trigger:**
- **Worker**: User clicks "Start" on subtask
- **Planner**: Auto-spawns when user creates a task

**Planner vs Worker Differences:**

| Aspect | Planner | Worker |
|--------|---------|--------|
| Working directory | Main clone | Dedicated worktree |
| Creates worktree | No | Yes |
| Modifies code | No | Yes |
| Creates beads issues | Yes (subtasks) | No |
| Closes beads issue | Yes (epic) | Yes (subtask) |

**Worker Loop Logic:**

```
1. Create worktree: `bd worktree create {subtask-id} --branch {branch-name}`
2. Render prompt template with subtask context
3. Save rendered prompt to /data/prompts/{project_id}/{task_id}/{subtask_id}.md (for audit)
4. Create AgentRun record (status: RUNNING)

LOOP (max 10 attempts):
    5. Execute from worktree directory:
       ```bash
       cd {worktree_path}
       CLAUDE_API_KEY={user_claude_key} \
         cat {prompt_path} | claude --print --dangerously-skip-permissions
       ```
    6. Capture stdout/stderr to log file
    7. Parse token usage from Claude output (if available)
    8. Check beads state: `bd show {subtask-beads-id} --json`

    IF beads status == "closed":
        9. Mark AgentRun as SUCCEEDED
        10. Git push branch
        11. Create PR via GitHub API
        12. Update subtask: status=COMPLETED, pr_url, pr_number
        EXIT LOOP

    IF attempt < max_attempts:
        13. Wait with exponential backoff (5s, 10s, 20s... cap 2min)
        14. Create new AgentRun record
        CONTINUE LOOP

    ELSE (max attempts reached):
        15. Mark AgentRun as FAILED
        16. Update subtask: status=BLOCKED, blocked_reason=FAILURE
        EXIT LOOP
```

**Exponential Backoff:**
- Base: 5 seconds
- Formula: `min(5 * 2^attempt, 120) + jitter`
- Jitter: random 0-20% of delay
- Sequence: 5s, 10s, 20s, 40s, 80s, 120s, 120s...

### 7.4 Dependency Resolution

**On subtask creation (by Planner):**
1. Planner calls `bd dep add {child} {parent}` for each dependency
2. Sync service reads `bd list --parent {epic} --json`
3. For each subtask: check `bd blocked {id}` vs `bd ready {id}`
4. Set subtask status: READY if no blockers, BLOCKED(DEPENDENCY) otherwise

**On subtask merge:**
1. Update subtask status to MERGED
2. Close beads issue: `bd close {id}`
3. Query dependents: `SELECT * FROM subtask_dependencies WHERE depends_on_id = {id}`
4. For each dependent: re-check if all its dependencies are MERGED
5. If all MERGED: update dependent status from BLOCKED to READY

**Subtask position (ordering):**

Initial `position` is assigned during sync based on:
1. Beads issue creation order (Planner creates them in logical sequence)
2. Subtasks synced in order from `bd list --parent {epic}` response

Users can later reorder via drag-and-drop (updates `position` field).

### 7.5 Beads Sync Strategy

**Event-driven sync (primary):**
- After each agent loop iteration: sync that subtask's state from Beads
- After Planner completes: full sync of all subtasks for the task
- After Mark Merged: sync dependent subtasks

**Periodic fallback (secondary):**
- Every 30 seconds: sync all `IN_PROGRESS` subtasks
- Catches any missed updates

**Sync operations:**
```bash
# Get all issues under epic
bd list --parent {epic-id} --json

# Get issue details
bd show {issue-id} --json

# Get ready issues
bd ready --parent {epic-id} --json

# Get blocked issues
bd blocked --parent {epic-id} --json
```

### 7.6 Beads Initialization Strategy

**Decision: Initialize Beads per-project (in each cloned repo), not at system level.**

**Rationale:**

| Factor | Per-Project | System-Level |
|--------|-------------|--------------|
| Git worktrees | Work naturally - worktrees share repo's `.beads` via redirect | Would need complex path mapping |
| Issue scoping | Issues belong to their repo (logical) | All projects mixed together |
| Multi-user | Each user's repos isolated | Would need namespacing |
| Beads design | This is how Beads is designed | Would fight the tool |
| Agent context | Agent works in repo, Beads is there | Would need external DB reference |

**Initialization command:**

```bash
# After cloning user's repo
cd /data/projects/{user_id}/{owner}/{repo}
bd init --stealth --prefix iv-{short_project_id}
```

**Why `--stealth` mode:**
- Uses global gitattributes/gitignore (not committed to user's repo)
- Beads files (`.beads/`) stay local to Intern Village system
- User's original repo is not polluted with task tracking artifacts
- Perfect for operating on repos you don't own

**Worktree behavior:**

When creating worktrees for subtasks, Beads automatically:
1. Creates `.beads/redirect` file in the worktree
2. Points redirect to main clone's `.beads/` directory
3. All worktrees share the same Beads database

```bash
# In main clone: /data/projects/{user_id}/{owner}/{repo}
bd worktree create subtask-iv-5 --branch iv-5-add-oauth

# Creates: /data/projects/{user_id}/{owner}/{repo}/subtask-iv-5/
# With:    /data/projects/{user_id}/{owner}/{repo}/subtask-iv-5/.beads/redirect
#          → points to main clone's .beads/
```

**Directory structure:**

```
/data/projects/
└── {user_id}/
    └── {owner}/
        └── {repo}/                    # Main clone
            ├── .beads/                # Beads database (SQLite + JSONL)
            │   ├── iv-{id}.db
            │   └── issues.jsonl
            ├── .git/
            ├── src/
            ├── subtask-iv-5/          # Worktree for subtask
            │   ├── .beads/
            │   │   └── redirect       # Points to parent .beads/
            │   └── src/
            └── subtask-iv-6/          # Another worktree
                ├── .beads/
                │   └── redirect
                └── src/
```

### 7.7 Process Management and Recovery

**Agent Process Tracking:**

Each running agent is tracked via the `agent_runs` table:
- `status = 'RUNNING'` indicates an active process
- `started_at` timestamp for detecting stale runs

**On Orchestrator startup (recovery):**

1. Query all `agent_runs` with `status = 'RUNNING'`
2. For each stale run (no associated process):
   - Mark as `FAILED` with error "Orchestrator restart"
   - Increment subtask `retry_count`
   - If under max retries: subtask stays `IN_PROGRESS` (will auto-resume)
   - If max retries reached: subtask moves to `BLOCKED (FAILURE)`
3. For subtasks still `IN_PROGRESS` with retries remaining:
   - Restart agent execution loop

**Graceful shutdown:**

1. Stop accepting new agent start requests
2. Wait for running agents to complete (with timeout)
3. If timeout: mark remaining runs as `FAILED` (will resume on restart)

**Orphan detection:**

Since agents are spawned as child processes, if Orchestrator crashes:
- Child processes may continue running or be killed by OS
- On restart, we detect orphans via the recovery logic above
- Worktrees and beads state are preserved (filesystem + beads DB)

---

## 8. Agent Prompts

### 8.1 Prompt Storage

| Location | Purpose |
|----------|---------|
| `prompts/planner.md` | Planner template (in Orchestrator codebase) |
| `prompts/worker.md` | Worker template (in Orchestrator codebase) |
| `/data/prompts/{project_id}/{task_id}/{subtask_id}.md` | Rendered prompts per run (outside repo, for audit) |

### 8.2 Planner Prompt Template

```markdown
# Planner Agent

You are a planning agent for Intern Village. Your job is to understand a task and break it down into implementable subtasks.

## Task Information

**Title:** {{.Task.Title}}

**Description:**
{{.Task.Description}}

## Repository Context

**Repo:** {{.Project.GithubOwner}}/{{.Project.GithubRepo}}
**Default Branch:** {{.Project.DefaultBranch}}
**Clone Path:** {{.Project.ClonePath}}

## Your Responsibilities

1. **Explore the codebase** to understand the architecture
2. **Generate a specification** for the overall task
3. **Break down into subtasks** (aim for small, focused PRs - each subtask = 1 PR)
4. **Create implementation plans** for each subtask
5. **Define dependencies** between subtasks (which must complete first)
6. **Create beads issues** for each subtask

## Beads Commands

Create the epic:
```bash
bd create --type epic --title "{{.Task.Title}}"
```

Create subtasks with specs in the body:
```bash
bd create --type task --parent {epic-id} --title "Subtask title" --description "Spec and implementation plan here"
```

Set dependencies:
```bash
bd dep add {child-id} {parent-id}  # child depends on parent
```

## Output Requirements

- Create one epic for the task
- Create 3-8 subtasks (adjust based on complexity)
- Each subtask should be completable in one focused session
- Each subtask body should contain:
  - **Spec**: What needs to be done
  - **Implementation Plan**: Step-by-step how to do it
  - **Acceptance Criteria**: How to verify it's done

## Completion

When you have created all subtasks with their dependencies, close the epic:
```bash
bd close {epic-id} --reason "Planning complete"
```
```

### 8.3 Worker Prompt Template

```markdown
# Worker Agent

You are an implementation agent for Intern Village. Your job is to implement a specific subtask.

## Subtask Information

**Title:** {{.Subtask.Title}}

**Beads ID:** {{.Subtask.BeadsIssueID}}

**Spec:**
{{.Subtask.Spec}}

**Implementation Plan:**
{{.Subtask.ImplementationPlan}}

## Repository Context

**Repo:** {{.Project.GithubOwner}}/{{.Project.GithubRepo}}
**Branch:** {{.Subtask.BranchName}}
**Worktree Path:** {{.Subtask.WorktreePath}}

## Your Responsibilities

1. **Study the spec and implementation plan**
2. **Implement the changes** following the plan
3. **Run tests** - they must pass
4. **Run linting** - it must pass
5. **Commit your changes** with a clear message
6. **Mark complete** when done

## Working Directory

You are working in: {{.Subtask.WorktreePath}}

This is a git worktree on branch: {{.Subtask.BranchName}}

## Completion

When implementation is complete and tests pass:
```bash
bd close {{.Subtask.BeadsIssueID}} --reason "Implementation complete"
```

## Important Notes

- Do NOT push to remote (the orchestrator handles this)
- Do NOT create the PR (the orchestrator handles this)
- Focus on implementation, testing, and committing
- If you get stuck, leave detailed notes in the beads issue
```

---

## 9. GitHub Integration

### 9.1 OAuth Flow

1. User clicks "Sign in with GitHub"
2. Redirect to: `https://github.com/login/oauth/authorize?client_id={id}&scope=read:user,repo&redirect_uri={callback}`
3. GitHub redirects to callback with `code`
4. Exchange code for access token
5. Fetch user info from GitHub API
6. Create/update User record
7. Issue JWT, set as cookie
8. Redirect to dashboard

### 9.2 Repository Operations

| Operation | When | Implementation |
|-----------|------|----------------|
| Check push access | Add project | `GET /repos/{owner}/{repo}` → `permissions.push` |
| Fork repo | No push access | `POST /repos/{owner}/{repo}/forks` |
| Clone repo | Add project | `git clone https://x-access-token:{token}@github.com/{owner}/{repo}.git` |
| Sync repo | Before Planner/Worker | See §9.5 Repository Sync Strategy |
| Create branch | Start subtask | Via `bd worktree create` |
| Push branch | Agent completes | `git push -u origin {branch}` |
| Create PR | Agent completes | `POST /repos/{owner}/{repo}/pulls` |
| Check PR status | (future) Validate merge | `GET /repos/{owner}/{repo}/pulls/{number}` |

### 9.3 Git Authentication

**Clone authentication:**

When cloning, the user's GitHub OAuth token is embedded in the remote URL:
```bash
git clone https://x-access-token:{github_token}@github.com/{owner}/{repo}.git
```

This stores the token in `.git/config` as part of the remote origin URL. The token:
- Is the user's GitHub OAuth access token (obtained during login)
- Persists in `.git/config` so subsequent `git push` commands work
- Is encrypted at rest (the clone directory is inside Orchestrator's data dir)
- Is never exposed to agents (they only do local commits)

**Push authentication:**

The Orchestrator (not the agent) performs `git push`. Since the token is in the remote URL, no additional auth is needed:
```bash
cd {worktree_path}
git push -u origin {branch_name}
```

**Alternative (future enhancement):** Use `GIT_ASKPASS` for dynamic credential injection without persisting tokens in `.git/config`.

### 9.4 PR Creation

**Request:**
```json
POST /repos/{owner}/{repo}/pulls
{
  "title": "[IV-{subtask-number}] {subtask-title}",
  "body": "## Summary\n\n{subtask-spec}\n\n## Implementation\n\n{commit-messages}\n\n---\n\n🤖 Generated by Intern Village",
  "head": "{branch-name}",
  "base": "{default-branch}"
}
```

**PR body content sources:**
- `{subtask-spec}`: From `subtasks.spec` field (Planner-generated)
- `{commit-messages}`: Extracted via `git log --oneline {branch}` on the worktree

### 9.5 Repository Sync Strategy

**Purpose:** Ensure agents always work on the latest version of the codebase before creating branches.

**When sync occurs:**
1. **Before Planner runs** (task creation) - so planning is based on current code
2. **Before Worker creates worktree** (subtask start) - so new branch is based on latest

**For direct clones (user has push access):**

```bash
cd {clone_path}
git fetch origin
git checkout {default_branch}
git reset --hard origin/{default_branch}
```

**For forks (user doesn't have push access):**

When the project is created as a fork, the Orchestrator:
1. Adds the original repo as `upstream` remote during clone setup
2. Syncs from `upstream` before agents run

```bash
# During project creation (after fork + clone):
cd {clone_path}
git remote add upstream https://github.com/{original_owner}/{original_repo}.git

# Before agent runs:
cd {clone_path}
git fetch upstream
git checkout {default_branch}
git reset --hard upstream/{default_branch}
git push origin {default_branch} --force  # Keep fork in sync
```

**Data model addition:**

The `projects` table needs to track the original repo for forks:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| upstream_owner | string | No | Original repo owner (only for forks) |
| upstream_repo | string | No | Original repo name (only for forks) |

**Sync operation summary:**

| Project Type | Remote to Fetch | Reset Target | Push to Origin |
|--------------|-----------------|--------------|----------------|
| Direct clone | origin | origin/{default_branch} | No |
| Fork | upstream | upstream/{default_branch} | Yes (force) |

**Error handling:**
- If sync fails (network error, conflicts): log error, retry up to 3 times with backoff
- If sync still fails: fail task/subtask creation with error message to user
- Never proceed with stale code if sync was attempted but failed

---

## 10. Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DATABASE_URL` | string | Yes | - | Postgres connection string |
| `GITHUB_CLIENT_ID` | string | Yes | - | OAuth app client ID |
| `GITHUB_CLIENT_SECRET` | string | Yes | - | OAuth app client secret |
| `JWT_SECRET` | string | Yes | - | JWT signing secret |
| `ENCRYPTION_KEY` | string | Yes | - | AES-256 key for token encryption |
| `CLAUDE_API_KEY` | string | Yes | - | Claude API key for agents |
| `DATA_DIR` | string | No | `/data` | Base directory for clones/worktrees |
| `PROMPTS_DIR` | string | No | `./prompts` | Prompt templates directory |
| `LOG_LEVEL` | string | No | `info` | Logging level |
| `PORT` | int | No | `8080` | HTTP server port |
| `AGENT_MAX_RETRIES` | int | No | `10` | Max retry attempts per subtask |
| `SYNC_INTERVAL_SECONDS` | int | No | `30` | Beads sync interval |

---

## 11. Security Considerations

### Authentication

- GitHub OAuth for user authentication
- JWT tokens for API authentication (24h expiry)
- Secure, HttpOnly cookies for JWT storage

### Authorization

| Resource | Access Control |
|----------|----------------|
| Project | User who created it |
| Task | Owner of parent project |
| Subtask | Owner of parent project |
| AgentRun | Owner of parent project |

### Token Protection

- GitHub OAuth tokens encrypted at rest (AES-256-GCM)
- Encryption key from environment variable
- Tokens never logged or returned in API responses

### Agent Isolation

- Each agent runs in isolated git worktree
- Agents have read-only access to Beads (`--readonly` flag where appropriate)
- Agents cannot push to remote (orchestrator handles)

### Log File Management

**Log storage location:**
```
/data/logs/{project_id}/{task_id}/{subtask_id}/
├── run-001.log          # First attempt
├── run-002.log          # Second attempt (retry)
└── run-003.log          # Third attempt
```

**Log contents:**
- Full stdout/stderr from Claude CLI
- Timestamps for each line (prefixed)
- Token usage summary (if parseable from output)

**Retention policy:**
- Logs kept for 30 days after task marked `DONE`
- Immediate cleanup available via project cleanup API
- Log files for `MERGED` subtasks cleaned up with worktree

---

## 12. Testing Requirements

### Unit Tests

- [ ] Task state machine transitions
- [ ] Subtask state machine transitions
- [ ] Dependency resolution logic
- [ ] Exponential backoff calculation
- [ ] Prompt template rendering
- [ ] JWT token validation

### Integration Tests

- [ ] GitHub OAuth flow
- [ ] Project creation with clone
- [ ] Beads sync operations
- [ ] Agent execution loop (mock Claude CLI)
- [ ] PR creation

### End-to-End Tests

- [ ] Full flow: create project → create task → Planner runs → subtasks appear
- [ ] Full flow: start subtask → Worker runs → PR created → mark merged

---

## 13. Implementation Phases

### Phase 1: Foundation

- [ ] Set up Go project structure with chi router
- [ ] Configure sqlc for database queries
- [ ] Implement database migrations
- [ ] Create domain models and error types
- [ ] Implement basic repository layer

### Phase 2: Authentication

- [ ] GitHub OAuth endpoints
- [ ] JWT generation and validation
- [ ] Auth middleware
- [ ] User CRUD operations

### Phase 3: Project Management

- [ ] Project CRUD API endpoints
- [ ] GitHub permission checking
- [ ] Clone/fork logic
- [ ] Beads initialization in cloned repos

### Phase 4: Task Management

- [ ] Task CRUD API endpoints
- [ ] Task state machine
- [ ] Subtask CRUD API endpoints
- [ ] Subtask state machine
- [ ] Dependency tracking

### Phase 5: Beads Integration

- [ ] Beads CLI wrapper service
- [ ] Sync service (Beads → Postgres)
- [ ] Worktree management
- [ ] Issue creation/closure

### Phase 6: Agent Execution

- [ ] Claude CLI executor
- [ ] Loop-until-done logic
- [ ] Prompt template rendering
- [ ] Log capture and storage
- [ ] Retry logic with backoff

### Phase 7: GitHub Operations

- [ ] Branch push
- [ ] PR creation
- [ ] PR status checking (future: merge validation)

### Phase 8: API Completion

- [ ] Agent run history endpoints
- [ ] Log streaming (SSE)
- [ ] Position update for drag-and-drop

---

## 14. Dependencies

### Internal

- Beads CLI: Task tracking, dependencies, worktrees

### External (Go)

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/go-chi/chi/v5` | latest | HTTP router |
| `github.com/jackc/pgx/v5` | latest | Postgres driver |
| `github.com/sqlc-dev/sqlc` | latest | SQL code generation |
| `github.com/golang-jwt/jwt/v5` | latest | JWT handling |
| `github.com/google/go-github/v60` | latest | GitHub API client |
| `golang.org/x/oauth2` | latest | OAuth2 client |
| `github.com/rs/zerolog` | latest | Structured logging |
| `github.com/kelseyhightower/envconfig` | latest | Environment config |

---

## Appendix A: Beads Command Reference

Commands used by Orchestrator:

```bash
# Initialize beads in repo (stealth mode - doesn't commit to user's repo)
bd init --stealth --prefix iv-{project_short_id}

# Create epic for task
bd create --type epic --title "Task title"

# Create subtask under epic
bd create --type task --parent {epic-id} --title "Subtask" --description "Spec..."

# Set dependencies
bd dep add {child-id} {parent-id}

# List issues under epic (JSON for sync)
bd list --parent {epic-id} --json

# Show issue details
bd show {issue-id} --json

# Get ready issues (no blockers)
bd ready --parent {epic-id} --json

# Get blocked issues
bd blocked --parent {epic-id} --json

# Close issue
bd close {issue-id} --reason "Complete"

# Create worktree
bd worktree create {name} --branch {branch-name}

# Remove worktree
bd worktree remove {name}

# Update issue status
bd update {issue-id} --status in_progress

# Add comment (for agent logs/notes)
bd comments add {issue-id} "Comment text"
```

---

## Appendix B: Future Considerations

| Feature | Why Deferred | Potential Approach |
|---------|--------------|-------------------|
| GitHub merge validation | MVP uses manual confirmation | Use `bd gate --type gh:pr` or GitHub webhooks |
| WebSocket for real-time | Polling sufficient for MVP | Add WebSocket gateway for status updates |
| Cost controls | Track tokens, no limits in MVP | Add budget per user/project, pause on exceed |
| Agent memory | Each agent starts fresh | Pass summary of completed subtasks as context |
| Jira integration | GitHub-first MVP | Use Beads Jira sync or build adapter |
