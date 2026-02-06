# Intern Village Architecture

**Status:** Draft
**Last Updated:** 2026-02-04

---

## Overview

### What This System Does

Intern Village is an AI agent orchestrator for software engineering tasks. Users submit task descriptions, the system generates specifications and breaks tasks into digestible subtasks, then autonomous AI agents implement each subtask as a separate PR.

### Key Constraints

- **Scale target**: Multi-user, but not high-volume (10s of concurrent agents, not 1000s)
- **Deployment**: Docker Compose for easy self-hosting
- **Team**: Solo developer

---

## High-Level Components

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                 WEB UI                                       │
│              (Task Input, Subtask Board, Agent Status)                       │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ORCHESTRATOR                                    │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐                 │
│  │  Task Manager  │  │  Agent Manager │  │ GitHub Service │                 │
│  └────────────────┘  └────────────────┘  └────────────────┘                 │
└─────────────────────────────────────────────────────────────────────────────┘
         │                      │                      │
         ▼                      ▼                      ▼
┌─────────────┐     ┌─────────────────────┐     ┌─────────────┐
│  Postgres   │     │      AI Agents      │     │   GitHub    │
│  (State)    │     │   (Claude CLI x N)  │     │   (PRs)     │
└─────────────┘     └─────────────────────┘     └─────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │       Beads         │
                    │  (Task Tracking &   │
                    │   Dependencies)     │
                    └─────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Tech |
|-----------|---------------|------|
| Web UI | Task input form, subtask board, agent status display | React |
| Orchestrator | API server, coordinates all operations | Go |
| Task Manager | CRUD for tasks/subtasks, state machine | Go (part of Orchestrator) |
| Agent Manager | Spawns/monitors Claude CLI processes, handles retries | Go (part of Orchestrator) |
| GitHub Service | OAuth, clone/fork repos, create PRs | Go (part of Orchestrator) |
| Postgres | Users, projects, tasks, subtasks, token usage | PostgreSQL |
| Beads | Subtask dependencies, agent state, ready/blocked tracking | Beads CLI |
| AI Agents | Spec generation (Planner), code implementation (Workers) | Claude CLI |

---

## Data Flow

### Primary Flow: Task to PRs

```
User                    Orchestrator              Agents                  GitHub
  │                          │                       │                       │
  │  1. Submit task          │                       │                       │
  │─────────────────────────>│                       │                       │
  │                          │                       │                       │
  │                          │  2. Spawn Planner     │                       │
  │                          │──────────────────────>│                       │
  │                          │                       │                       │
  │                          │  3. Spec + Subtasks   │                       │
  │                          │<──────────────────────│                       │
  │                          │                       │                       │
  │  4. View subtask board   │                       │                       │
  │<─────────────────────────│                       │                       │
  │                          │                       │                       │
  │  5. Start subtask(s)     │                       │                       │
  │─────────────────────────>│                       │                       │
  │                          │                       │                       │
  │                          │  6. Spawn Worker(s)   │                       │
  │                          │──────────────────────>│                       │
  │                          │                       │                       │
  │                          │                       │  7. Create PR         │
  │                          │                       │──────────────────────>│
  │                          │                       │                       │
  │  8. Review & merge PR    │                       │                       │
  │───────────────────────────────────────────────────────────────────────────>│
```

### Subtask Lifecycle

```
┌──────────┐     Planner     ┌──────────┐   User clicks   ┌─────────────┐
│ PENDING  │ ──────────────> │  READY   │ ───────────────>│ IN_PROGRESS │
└──────────┘   generates     └──────────┘    "Start"      └─────────────┘
                  │                                              │
                  │ (has deps)                                   │
                  ▼                                              ▼
            ┌──────────┐                              ┌─────────────────────┐
            │ BLOCKED  │                              │     Agent runs      │
            │ (by dep) │                              │ (up to 10 retries)  │
            └──────────┘                              └─────────────────────┘
                  │                                       │           │
                  │ (deps merged)                  Success│           │Fail x10
                  ▼                                       ▼           ▼
            ┌──────────┐                           ┌───────────┐ ┌─────────┐
            │  READY   │                           │ COMPLETED │ │ BLOCKED │
            └──────────┘                           │ (PR open) │ │(failure)│
                                                   └───────────┘ └─────────┘
                                                         │
                                                         │ User clicks
                                                         │ "Mark Merged"
                                                         ▼
                                                   ┌───────────┐
                                                   │  MERGED   │
                                                   └───────────┘
```

**Blocked Reasons**: Subtasks in BLOCKED state have a `blocked_reason`:
- `DEPENDENCY` — waiting for another subtask to be merged
- `FAILURE` — agent failed 10 times, needs human intervention

### Task Lifecycle

```
┌──────────┐   Planner    ┌──────────┐   All subtasks   ┌──────────┐
│ PLANNING │ ───────────> │  ACTIVE  │ ───────────────> │   DONE   │
└──────────┘   completes  └──────────┘     merged       └──────────┘
```

- **PLANNING**: Planner agent is running, generating spec and subtasks
- **ACTIVE**: Subtasks exist, work in progress
- **DONE**: All subtasks have reached MERGED state (auto-transitions)

---

## Feature Sets

### MVP Features

| Feature | Description |
|---------|-------------|
| Task Input | User submits title + description paragraph |
| Planner Agent | Generates spec, breaks into subtasks, creates implementation plans |
| Subtask Board | Kanban-style view of subtasks with status columns |
| Per-Project Boards | Each project (repo) has its own board |
| All-Projects Index | Dashboard showing all projects for navigation |
| Agent Execution | One Claude CLI per subtask in isolated worktree |
| Dependency Tracking | Subtasks can block other subtasks (via Beads) |
| Manual Merge Confirmation | User clicks "Mark Merged" to unblock dependents |
| Loop-Until-Done | Agents loop until beads state is complete (max 10 attempts) |
| PR Creation | Auto-create GitHub PR when subtask completes |
| Token Tracking | Track token usage per subtask |
| GitHub OAuth | Multi-user auth via GitHub |
| Drag-and-Drop Ordering | Users can reorder subtasks within columns |

### Future Features (Post-MVP)

| Feature | Description |
|---------|-------------|
| Jira Integration | Pull tasks directly from Jira tickets |
| Notifications | Slack/Discord alerts for PR created, agent blocked |
| Cost Controls | Budget limits per user/project |
| Agent Memory | Pass completed subtask context to dependent subtasks |
| GitHub Merge Validation | Validate "Mark Merged" against GitHub API instead of trusting user |
| Undo Merged | Allow users to unmark a merged subtask (re-block dependents) |

---

## Key Decisions

### Decision 1: Beads for Task Tracking (not just Postgres)

**Context**: Need to track subtask dependencies and agent state.

**Decision**: Use Beads CLI as the source of truth for dependencies and agent state. Sync to Postgres for web UI queries.

**Trade-offs**:
- (+) Beads has first-class dependency support (`bd dep`, `bd ready`, `bd blocked`)
- (+) Beads has agent state management (`bd agent`, `bd set-state`)
- (+) Works with git worktrees out of the box
- (-) Extra sync layer between Beads and Postgres
- (-) Dependency on external CLI tool

### Decision 2: Git Worktrees for Parallel Agents

**Context**: Multiple agents work in parallel on the same repo.

**Decision**: Each agent gets its own git worktree with a dedicated branch.

**Trade-offs**:
- (+) No merge conflicts between parallel agents
- (+) Each agent has clean working directory
- (+) Beads has built-in worktree support
- (-) More disk space usage (mitigated by cleanup after PR)

### Decision 3: One Planner Agent, Many Worker Agents

**Context**: Need to generate specs and implement code.

**Decision**: Single "Planner" agent handles spec + breakdown + plans. Separate "Worker" agents handle implementation.

**Trade-offs**:
- (+) Clear separation of concerns
- (+) Planner has full context for coherent breakdown
- (+) Workers have focused, single-task context
- (-) Planner is a single point of failure for task breakdown

### Decision 4: Clone vs Fork (Auto-detect)

**Context**: Users may work on repos they own or contribute to.

**Decision**: Check user permissions. If user has push access, clone directly. If not, fork first.

**Trade-offs**:
- (+) Works for both owned repos and open source contributions
- (+) No manual user configuration needed
- (-) Slight complexity in GitHub integration

### Decision 5: System-level Dependency Caches

**Context**: Agents need to install dependencies (npm, go mod, etc.) to run tests.

**Decision**: Use system-level caches (~/.npm, ~/go/pkg/mod) shared across all agent runs.

**Trade-offs**:
- (+) Fast subsequent runs
- (+) No per-repo cache management
- (-) Cache can grow large over time
- (-) Potential version conflicts (rare)

### Decision 9: Manual Merge Confirmation (MVP)

**Context**: Need to know when a PR is merged so dependents can be unblocked.

**Decision**: User manually clicks "Mark Merged" after merging on GitHub. System trusts the user.

**Trade-offs**:
- (+) Simple implementation, no GitHub webhooks or polling
- (+) Works immediately without webhook setup
- (-) User can mark merged when PR isn't actually merged
- (-) Extra manual step for user

**Future**: Add GitHub API validation to verify PR is actually merged before accepting.

### Decision 10: Per-Project Boards with Task Grouping

**Context**: Users work on multiple repos; each repo can have multiple tasks.

**Decision**:
- All-projects index page for navigation
- Each project has its own board
- Board shows tasks as collapsible groups
- Subtasks displayed within their parent task's group

**Trade-offs**:
- (+) Clear organization by repo
- (+) Familiar Jira-like UX
- (+) Easy to focus on one project at a time
- (-) No cross-project view of all subtasks

### Decision 8: Prompt-Based Agents (No Skills)

**Context**: Claude CLI supports "skills" — reusable instruction sets. Users may have global skills (e.g., an `architect` skill for system design) that could conflict with agent behavior.

**Decision**: Agents receive instructions via detailed system prompts passed by the Orchestrator. They do not use Claude CLI skills.

**Trade-offs**:
- (+) Agent behavior is fully controlled and deterministic
- (+) No conflicts with user's personal skill configurations
- (+) Portable — works regardless of host machine's Claude setup
- (-) System prompts must be maintained in Orchestrator codebase
- (-) No reuse of community/shared skills

---

## What We're NOT Doing (and Why)

| Pattern/Feature | Why Not |
|-----------------|---------|
| Microservices | Single Go binary is simpler; solo developer can't maintain microservices |
| Kubernetes | Docker Compose is sufficient for expected scale |
| Message Queues | Direct process spawning is simpler; agent count is bounded |
| Container-per-agent | Worktrees provide enough isolation without container overhead |
| Custom LLM hosting | Claude CLI works out of the box |

---

## Technology Choices

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Frontend | React | User preference, widely supported |
| Backend | Go | User preference, good for CLI orchestration, single binary |
| Database | Postgres | Reliable, good enough for this scale |
| Task Tracking | Beads | First-class dependencies, agent state, git integration |
| AI Runtime | Claude CLI | Simple process spawning, handles context/tools |
| Auth | GitHub OAuth | Users already have GitHub accounts for repo access |
| Deployment | Docker Compose | Simple, portable, easy for solo dev |

### Decision 6: Agent Logging Strategy

**Context**: Users need visibility into what agents are doing for debugging and auditing.

**Decision**: Store full agent output (stdout/stderr) as files, one per subtask. Store summary (status, token count, error) in Postgres for quick UI display.

**Trade-offs**:
- (+) Full logs available for debugging
- (+) Quick status queries via Postgres
- (+) Logs can be streamed/tailed during execution
- (-) Log files need eventual cleanup

### Decision 7: Cleanup Policy

**Context**: Worktrees and clones consume disk space.

**Decision**:
- Delete worktree after PR is merged (human has verified)
- Keep repo clone for 7 days (speeds up future tasks on same repo)
- Provide manual "cleanup" button in UI for immediate deletion

**Trade-offs**:
- (+) Disk space reclaimed automatically
- (+) Clone cache speeds up repeat work
- (+) Manual override for immediate cleanup
- (-) Needs scheduled cleanup job for 7-day expiry

---

## Web UI Structure

### Pages

| Page | Purpose |
|------|---------|
| Login | GitHub OAuth sign-in |
| All Projects | Index of all user's projects with quick stats |
| Project Board | Kanban board for a single project |
| Task Detail | (Optional) Expanded view of a task with all subtask details |

### Project Board Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Project: owner/repo                                    [+ New Task]        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─ Task: "Add user authentication" ──────────────────────────────────────┐ │
│  │                                                                         │ │
│  │  Pending    Ready      In Progress   Completed    Merged     Blocked   │ │
│  │  ────────   ────────   ───────────   ──────────   ────────   ───────── │ │
│  │  ┌──────┐   ┌──────┐   ┌──────┐      ┌──────┐     ┌──────┐   ┌──────┐  │ │
│  │  │ ST-3 │   │ ST-1 │   │ ST-4 │      │ ST-2 │     │ ST-5 │   │ ST-6 │  │ │
│  │  │      │   │[Start│   │ 🔄   │      │[Mark │     │  ✓   │   │ ⚠    │  │ │
│  │  │      │   │  ]   │   │      │      │Merged│     │      │   │ dep  │  │ │
│  │  └──────┘   └──────┘   └──────┘      │  ]   │     └──────┘   └──────┘  │ │
│  │                                      │[PR →]│                          │ │
│  │                                      └──────┘                          │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─ Task: "Fix checkout bug" ─────────────────────────────────────────────┐ │
│  │  ...                                                                    │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Card Elements

| Element | When Shown |
|---------|------------|
| Title | Always |
| [Start] button | READY state |
| Spinner icon | IN_PROGRESS state |
| [PR →] link | COMPLETED or MERGED (links to GitHub) |
| [Mark Merged] button | COMPLETED state |
| ✓ checkmark | MERGED state |
| ⚠ + reason badge | BLOCKED state (shows "dep" or "failure") |

### Drag-and-Drop

- Users can drag subtasks to reorder within the same column
- Position is persisted to database (`position` field)
- Does not change subtask status (status changes via buttons only)

---

## User Flows

### Flow 1: Onboarding
1. User visits app → clicks "Sign in with GitHub"
2. GitHub OAuth flow → user authorizes app
3. User lands on All Projects index (empty project list)

### Flow 2: Add Project
1. User clicks "Add Project" on All Projects index
2. Enters GitHub repo URL (e.g., `github.com/owner/repo`)
3. System checks permissions, clones or forks as needed
4. Project appears in All Projects index
5. User clicks project to open its board

### Flow 3: Create Task
1. User is on a project board
2. Clicks "New Task"
3. Enters title + description paragraph
4. Submits → Task appears in PLANNING state, Planner agent spawns
5. User waits (or navigates away) while Planner works
6. When done: task transitions to ACTIVE, subtasks appear grouped under the task

### Flow 4: Work on Subtasks
1. User views subtask board (kanban columns: Pending, Ready, In Progress, Completed, Merged, Blocked)
2. Subtasks are grouped under their parent task
3. User can drag-and-drop to reorder subtasks within columns
4. User clicks "Start" on a ready subtask
5. Worker agent spawns, status updates to "In Progress"
6. User can view agent logs in real-time (optional)
7. On success: subtask moves to Completed, PR link appears
8. On failure (10 retries): subtask moves to Blocked (reason: failure), user can intervene

### Flow 5: Review & Merge
1. User clicks PR link → opens GitHub
2. Reviews code, merges PR on GitHub
3. User returns to board, clicks "Mark Merged" on the subtask
4. Subtask moves to Merged column
5. System checks dependents: if all their dependencies are now Merged, they move from Blocked to Ready
6. When all subtasks of a task are Merged, the parent task auto-transitions to DONE
7. Worktree is cleaned up

---

## Data Entities

| Entity | Purpose | Key Fields |
|--------|---------|------------|
| User | GitHub-authenticated user | github_id, github_username, github_token (encrypted) |
| Project | A GitHub repo the user works on | github_owner, github_repo, is_fork, default_branch |
| Task | User-submitted work item | title, description, status (PLANNING/ACTIVE/DONE), beads_epic_id |
| Subtask | Broken-down work unit | title, spec, implementation_plan, status (PENDING/READY/BLOCKED/IN_PROGRESS/COMPLETED/MERGED), blocked_reason (DEPENDENCY/FAILURE/null), branch_name, pr_url, retry_count, token_usage, position (for drag-and-drop ordering), beads_issue_id |
| SubtaskDependency | Tracks which subtasks block others | subtask_id, depends_on_id |
| AgentRun | History of agent executions | subtask_id, started_at, ended_at, status, token_usage, error_message |

---

## Agent Behaviors

### Design Decision: Prompt-Based Agents (No Skills)

Agents receive their instructions via **markdown prompt files** piped to the Claude CLI, not via Claude CLI skills. This ensures:

- **Isolation**: Agents don't inherit user's global skills (e.g., an `architect` skill won't conflict with the Planner agent)
- **Determinism**: Agent behavior is fully controlled by the Orchestrator
- **Portability**: No dependency on per-user skill configurations

The prompt files for each agent type are maintained in the Orchestrator codebase.

### Design Decision: Loop-Until-Done Execution

Agents run in a **loop-until-done** pattern, not a single invocation:

1. Agent Manager pipes the prompt `.md` file to Claude CLI
2. Claude runs with `--dangerously-skip-permissions` (unattended execution)
3. After each invocation, Agent Manager checks beads state
4. If subtask is not `completed`, loop continues (up to max attempts)
5. Each invocation picks up where the last left off (same worktree, same branch)

```
┌─────────────────────────────────────────────────────────────┐
│                     Agent Manager Loop                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌──────────────┐                                          │
│   │ Pipe prompt  │                                          │
│   │ to Claude    │◄─────────────────────────┐               │
│   └──────┬───────┘                          │               │
│          │                                  │               │
│          ▼                                  │               │
│   ┌──────────────┐                          │               │
│   │ Claude runs  │                          │               │
│   │ (makes       │                          │               │
│   │  progress)   │                          │               │
│   └──────┬───────┘                          │               │
│          │                                  │               │
│          ▼                                  │               │
│   ┌──────────────┐      No, attempts < max  │               │
│   │ Check beads  ├──────────────────────────┘               │
│   │ state        │                                          │
│   └──────┬───────┘                                          │
│          │                                                  │
│          │ Yes, completed                                   │
│          ▼                                                  │
│   ┌──────────────┐                                          │
│   │ Create PR,   │                                          │
│   │ cleanup      │                                          │
│   └──────────────┘                                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Why loop instead of single invocation:**
- Complex tasks may exceed single-session context limits
- Claude can make incremental progress across invocations
- Transient failures (API errors, rate limits) are automatically recovered
- Each loop iteration can incorporate fresh state (test results, linting output)

### Design Decision: Prompt Storage

Prompts have two parts:
- **Templates** (static): Base instructions for Planner/Worker agents, stored in codebase (`prompts/planner.md`, `prompts/worker.md`)
- **Rendered prompts** (per-run): Template + injected context (subtask spec, implementation plan, repo path)

**Decision**: Store full rendered prompts for each agent run.

**Rationale**:
- Debugging: See exactly what the agent was told when investigating failures
- Auditing: Full history of instructions given to agents
- Reproducibility: Can re-run with identical prompt if needed

**Storage**: `prompt_text` column in the `AgentRun` table (or as file alongside logs).

### Design Decision: Git Operation Split

| Actor | Operations | Auth Required |
|-------|------------|---------------|
| Claude Agent | `git add`, `git commit` | None (local only) |
| Orchestrator | `git push`, create PR | GitHub OAuth token |

**Rationale**:
- Claude works purely locally — no GitHub credentials needed in agent environment
- Orchestrator already holds user's GitHub token from OAuth flow
- GitHub Go SDK is more robust than shelling out to `gh` CLI
- Clear separation: Claude does code, Orchestrator does GitHub ops

---

### Planner Agent

**Trigger**: Automatically spawned when user creates a task.

**Inputs**:
- Task title and description (injected into prompt)
- Access to the cloned repo

**Responsibilities**:
1. Explore the codebase to understand architecture
2. Generate a specification for the task
3. Break the task into subtasks (aim for small, digestible PRs)
4. Create an implementation plan for each subtask
5. Define dependencies between subtasks (which must complete first)
6. Create beads issues for each subtask with specs/plans as the body

**Outputs**:
- Beads epic for the task
- Beads issues for each subtask (with dependencies set via `bd dep add`)
- Specs and implementation plans stored in subtask bodies

**Execution**: Also runs in loop-until-done pattern. Completes when all subtasks are created in beads.

### Worker Agent

**Trigger**: User clicks "Start" on a subtask.

**Inputs**:
- Subtask spec and implementation plan (injected into prompt)
- Dedicated git worktree with a fresh branch
- Access to run tests and linting

**Responsibilities**:
1. Read and understand the spec/plan
2. Implement the changes
3. Run tests → must pass
4. Run linting → must pass
5. Commit with clear message
6. Update beads state to `completed`

**Outputs**:
- Commits on the branch
- Branch pushed to remote
- PR created via GitHub API (by Orchestrator after loop completes)

**Execution**: Runs in loop-until-done pattern (see above). Each invocation works in the same worktree, allowing incremental progress.

**Termination Conditions**:
- **Success**: Beads state is `completed` → exit loop, create PR
- **Failure**: Max attempts reached (default: 10) → mark subtask as `BLOCKED` (reason: `FAILURE`), requires human intervention

---

## Beads Integration

### Commands Used

| Command | When Used | Purpose |
|---------|-----------|---------|
| `bd init` | Project setup | Initialize beads in repo clone |
| `bd create --epic "Title"` | Planner | Create epic for a task |
| `bd create --title "X" --body "spec"` | Planner | Create subtask issue |
| `bd dep add CHILD PARENT` | Planner | Set subtask dependencies |
| `bd ready` | Orchestrator | Get subtasks with no blockers |
| `bd blocked` | Orchestrator | Get blocked subtasks |
| `bd set-state ISSUE completed` | Worker | Mark subtask done |
| `bd list --json` | Orchestrator | Sync state to Postgres |
| `bd worktree` | Agent Manager | Manage worktrees for agents |

### Beads as Source of Truth

- Dependencies live in Beads (`bd dep`)
- Agent state lives in Beads (`bd set-state`)
- Orchestrator polls Beads and syncs to Postgres for UI queries

---

## GitHub Integration

### OAuth Scopes Required

| Scope | Purpose |
|-------|---------|
| `read:user` | Get user profile info |
| `repo` | Full repo access (clone, push, create PR) |

### Operations

| Operation | When | How |
|-----------|------|-----|
| Check permissions | Add project | `GET /repos/{owner}/{repo}` → check `permissions.push` |
| Fork repo | No push access | `POST /repos/{owner}/{repo}/forks` |
| Clone repo | Has push access | `git clone https://github.com/{owner}/{repo}` |
| Create branch | Start subtask | `git worktree add -b {branch}` |
| Push branch | Agent completes | `git push -u origin {branch}` |
| Create PR | Agent completes | `POST /repos/{owner}/{repo}/pulls` |
| Check PR merged | Cleanup | `GET /repos/{owner}/{repo}/pulls/{number}` → check `merged` |

---

## Configuration

Users must provide (via environment or settings UI):

| Config | Required | Purpose |
|--------|----------|---------|
| `CLAUDE_API_KEY` | Yes | User's Claude API key for agents |
| `GITHUB_CLIENT_ID` | Yes | OAuth app client ID |
| `GITHUB_CLIENT_SECRET` | Yes | OAuth app client secret |
| `DATABASE_URL` | Yes | Postgres connection string |

---

## Future Considerations

Things that might matter later but don't need solving now:

- **If agent count grows**: Consider job queue (Redis + worker pool)
- **If multi-org needed**: Add org/team model to user management

### Implemented: Real-Time Events via SSE

Real-time agent status and log streaming is now specified via Server-Sent Events (SSE). See [realtime-events.md](./realtime-events.md) for details:

- Single SSE connection per project (`GET /api/projects/{id}/events`)
- Streams agent logs, task/subtask status changes, dependency unblocking
- Replaces polling when connected; polling remains as fallback for disconnection
