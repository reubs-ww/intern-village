<!--
 Copyright (c) 2026 Intern Village. All rights reserved.
 SPDX-License-Identifier: Proprietary
-->

# Frontend Specification

**Status:** Draft
**Version:** 1.0
**Last Updated:** 2026-02-05
**Architecture Reference:** [architecture.md](./architecture.md)
**Backend Reference:** [orchestrator.md](./orchestrator.md)

---

## 1. Overview

### Purpose

The frontend is a single-page application (SPA) that provides a visual interface for Intern Village. Users can add projects, create tasks, monitor agent progress on a Kanban board, and manage the full lifecycle from task creation to PR merge.

### Goals

- **Task Management**: Create tasks, view subtask breakdown, monitor progress
- **Kanban Board**: Visual status tracking with drag-and-drop ordering
- **Agent Monitoring**: View agent run history and logs
- **GitHub Integration**: OAuth login, PR links, repo management

### Non-Goals

- **Mobile optimization**: Desktop-first, basic responsiveness only
- **Real-time streaming**: Polling is sufficient for MVP
- **Offline support**: Requires backend connectivity
- **Multi-language**: English only

### Context from Architecture

Key decisions affecting this spec:

- **Polling for updates**: No WebSocket/SSE required (architecture decision)
- **Single container deployment**: Frontend served as static files by Orchestrator
- **Single-user mode**: No team features, simplified auth flow
- **Manual merge confirmation**: User clicks "Mark Merged" after merging on GitHub

---

## 2. Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Framework | React 18 | User familiarity, large ecosystem |
| Build Tool | Vite | Fast dev server, optimized builds |
| Routing | React Router v6 | Standard SPA routing |
| State Management | TanStack Query | Server state caching, auto-refetch |
| Styling | Tailwind CSS | Rapid prototyping, utility-first |
| Components | shadcn/ui | Accessible, customizable, Radix-based |
| Drag & Drop | @dnd-kit/core | Modern, accessible DnD |
| HTTP Client | ky | Lightweight fetch wrapper |
| Icons | Lucide React | Consistent icon set |
| Theme | Dark mode default | User preference |

### Why This Stack

**React + Vite** over alternatives:
- Next.js: SSR not needed, adds complexity
- Vue/Svelte: User more familiar with React
- CRA: Slower, less flexible than Vite

**TanStack Query** over Redux/Zustand:
- 90% of state is server data (projects, tasks, subtasks)
- Built-in caching, refetching, loading states
- No boilerplate for async operations

**shadcn/ui** over other component libraries:
- Copy-paste components (no npm dependency lock-in)
- Built on Radix (accessible by default)
- Tailwind-native styling
- Dark mode support built-in

---

## 3. Project Structure

```
frontend/
├── index.html
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── package.json
├── src/
│   ├── main.tsx                    # Entry point
│   ├── App.tsx                     # Root component, providers, router
│   ├── api/
│   │   ├── client.ts               # HTTP client configuration
│   │   ├── auth.ts                 # Auth API calls
│   │   ├── projects.ts             # Project API calls
│   │   ├── tasks.ts                # Task API calls
│   │   ├── subtasks.ts             # Subtask API calls
│   │   └── agents.ts               # Agent run API calls
│   ├── components/
│   │   ├── ui/                     # shadcn/ui components
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── dropdown-menu.tsx
│   │   │   ├── input.tsx
│   │   │   ├── select.tsx
│   │   │   ├── badge.tsx
│   │   │   ├── skeleton.tsx
│   │   │   ├── toast.tsx
│   │   │   └── ...
│   │   ├── layout/
│   │   │   ├── Header.tsx          # Top navigation
│   │   │   ├── Sidebar.tsx         # Project list sidebar
│   │   │   └── Layout.tsx          # Page wrapper
│   │   ├── auth/
│   │   │   └── LoginButton.tsx     # GitHub OAuth trigger
│   │   ├── projects/
│   │   │   ├── ProjectCard.tsx     # Project in sidebar
│   │   │   ├── AddProjectDialog.tsx # Add project modal
│   │   │   └── ProjectSettings.tsx # Project settings/delete
│   │   ├── tasks/
│   │   │   ├── TaskCard.tsx        # Task summary card
│   │   │   ├── CreateTaskDialog.tsx # New task modal
│   │   │   └── TaskFilter.tsx      # Task dropdown filter
│   │   ├── board/
│   │   │   ├── Board.tsx           # Kanban board container
│   │   │   ├── Column.tsx          # Status column
│   │   │   ├── SubtaskCard.tsx     # Subtask card (draggable)
│   │   │   └── EmptyColumn.tsx     # Empty state
│   │   ├── subtasks/
│   │   │   ├── SubtaskDetail.tsx   # Subtask detail panel
│   │   │   ├── StartButton.tsx     # Start worker button
│   │   │   ├── MarkMergedButton.tsx # Mark merged button
│   │   │   └── RetryButton.tsx     # Retry failed subtask
│   │   └── agents/
│   │       ├── AgentRunList.tsx    # List of runs for subtask
│   │       ├── AgentRunItem.tsx    # Single run summary
│   │       └── LogViewer.tsx       # Full log display
│   ├── hooks/
│   │   ├── useAuth.ts              # Auth state hook
│   │   ├── useProjects.ts          # Projects query hook
│   │   ├── useTasks.ts             # Tasks query hook
│   │   ├── useSubtasks.ts          # Subtasks query hook
│   │   └── usePolling.ts           # Auto-refetch helper
│   ├── pages/
│   │   ├── LoginPage.tsx           # Login screen
│   │   ├── ProjectsPage.tsx        # All projects list
│   │   ├── BoardPage.tsx           # Kanban board for project
│   │   └── NotFoundPage.tsx        # 404 page
│   ├── lib/
│   │   ├── utils.ts                # Utility functions (cn, etc.)
│   │   └── constants.ts            # App constants
│   └── types/
│       └── api.ts                  # API response types
├── public/
│   └── favicon.svg
└── components.json                  # shadcn/ui config
```

---

## 4. Pages & Routes

| Route | Component | Auth Required | Description |
|-------|-----------|---------------|-------------|
| `/login` | LoginPage | No | GitHub OAuth initiation |
| `/` | ProjectsPage | Yes | All projects dashboard |
| `/projects/:id` | BoardPage | Yes | Kanban board for project |
| `*` | NotFoundPage | No | 404 fallback |

### Route Behavior

**Unauthenticated users:**
- Redirect to `/login` from protected routes
- Store intended destination, redirect after login

**After OAuth callback:**
- Backend sets JWT cookie
- Frontend redirects to stored destination or `/`

---

## 5. User Flows

### Flow 1: Login

```
1. User visits app → sees LoginPage
2. Clicks "Sign in with GitHub"
3. Redirect to GitHub OAuth (full page redirect)
4. GitHub redirects to /api/auth/github/callback
5. Backend validates, sets JWT cookie, redirects to /
6. Frontend loads ProjectsPage
```

### Flow 2: Add Project

```
1. User clicks "Add Project" button
2. AddProjectDialog opens
3. User enters GitHub repo URL (e.g., github.com/owner/repo)
4. Submit → POST /api/projects
5. On success: dialog closes, project appears in sidebar
6. On error: show error message in dialog
```

**Validation:**
- URL must match GitHub repo pattern
- Show loading state during clone (can take time)

### Flow 3: Create Task

```
1. User is on BoardPage for a project
2. Clicks "New Task" button
3. CreateTaskDialog opens
4. User enters:
   - Title (required, max 200 chars)
   - Description (required, paragraph)
5. Submit → POST /api/projects/{id}/tasks
6. On success: dialog closes, task appears with "Planning" badge
7. Planner agent runs (background)
8. Polling detects subtasks → board populates
```

### Flow 4: Monitor Planning

```
1. Task shows "Planning" status badge
2. Board is empty (no subtasks yet)
3. SSE streams real-time updates (see realtime-events.md)
4. User can click "View Logs" to see Planner output
5. When Planner completes:
   - Task status changes to "Active" (instant via SSE event)
   - Subtasks appear in Ready/Blocked columns
6. If Planner fails:
   - Task shows "Planning Failed" badge
   - "Retry Planning" button appears
```

### Flow 5: Start Subtask

```
1. User sees subtask in "Ready" column
2. Clicks on card → SubtaskDetail panel opens (or inline expand)
3. Reviews spec and implementation plan
4. Clicks "Start" button
5. POST /api/subtasks/{id}/start
6. Card moves to "In Progress" column
7. Shows spinner/working indicator
8. User can click "View Logs" to see Worker output in real-time
9. SSE event triggers completion → moves to "Completed"
```

### Flow 6: Review & Merge

```
1. Subtask is in "Completed" column
2. Card shows PR link
3. User clicks PR link → opens GitHub in new tab
4. User reviews and merges PR on GitHub
5. Returns to app, clicks "Mark Merged" on card
6. POST /api/subtasks/{id}/mark-merged
7. Card moves to "Merged" column
8. Dependent subtasks may move from "Blocked" to "Ready"
```

### Flow 7: Handle Failed Subtask

```
1. Subtask is in "Blocked" column with "failure" reason
2. User clicks card → sees "View Logs" option
3. Views logs to understand failure
4. Clicks "Retry" button
5. POST /api/subtasks/{id}/retry
6. Card moves to "In Progress"
7. Worker runs again
```

### Flow 8: View Agent Logs

```
1. User clicks subtask card
2. Detail panel shows agent run history
3. Each run shows: attempt #, status, duration, token usage
4. User clicks a run → LogViewer opens
5. Full stdout/stderr displayed (monospace, scrollable)
```

---

## 6. Component Specifications

### 6.1 Layout

**Header**
- Logo/app name (left)
- Current project name (center, if on BoardPage)
- User avatar + dropdown (right): Settings, Logout

**Sidebar** (on ProjectsPage and BoardPage)
- "Add Project" button (top)
- Project list (scrollable)
  - Each project: icon, owner/repo name
  - Active project highlighted
  - Click navigates to BoardPage

### 6.2 Board

**Board Layout**
```
┌─────────────────────────────────────────────────────────────────────────┐
│  [Task Filter ▼]  [+ New Task]                                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Ready       In Progress    Completed     Merged        Blocked         │
│  ─────────   ───────────    ──────────    ──────────    ─────────       │
│  ┌───────┐   ┌───────┐      ┌───────┐     ┌───────┐     ┌───────┐       │
│  │ Card  │   │ Card  │      │ Card  │     │ Card  │     │ Card  │       │
│  │       │   │  🔄   │      │ [PR]  │     │  ✓    │     │  ⚠    │       │
│  └───────┘   └───────┘      │[Merge]│     └───────┘     │[Retry]│       │
│  ┌───────┐                  └───────┘                   └───────┘       │
│  │ Card  │                                                              │
│  └───────┘                                                              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Columns (in order):**

| Column | Status | Cards Show |
|--------|--------|------------|
| Ready | READY | [Start] button |
| In Progress | IN_PROGRESS | Spinner, duration |
| Completed | COMPLETED | PR link, [Mark Merged] button |
| Merged | MERGED | Checkmark |
| Blocked | BLOCKED | Reason badge (dependency/failure), [Retry] if failure |

**Note:** PENDING status is omitted from display. Subtasks only appear on board after Planner assigns them a status (READY or BLOCKED).

**Task Filter Dropdown:**
- "All Tasks" (default): shows all subtasks for project
- Individual task options: filter to one task's subtasks

**Card Elements:**

| Element | Always | Conditional |
|---------|--------|-------------|
| Title | ✓ | |
| Task badge | ✓ | (parent task name, for context) |
| Status indicator | ✓ | |
| [Start] button | | READY only |
| Spinner | | IN_PROGRESS only |
| [PR →] link | | COMPLETED, MERGED |
| [Mark Merged] button | | COMPLETED only |
| Blocked reason badge | | BLOCKED only |
| [Retry] button | | BLOCKED + FAILURE reason |

**Drag & Drop:**
- Cards can be reordered within the same column
- Drag between columns is disabled (status changes via buttons only)
- Position persisted via PATCH /api/subtasks/{id}/position

### 6.3 SubtaskCard

```tsx
interface SubtaskCardProps {
  subtask: Subtask;
  onStart: () => void;
  onMarkMerged: () => void;
  onRetry: () => void;
  onClick: () => void;
}
```

**Visual States:**

| Status | Background | Icon | Actions |
|--------|------------|------|---------|
| READY | Default | — | Start button |
| IN_PROGRESS | Subtle pulse/glow | Spinner | — |
| COMPLETED | Success tint | Checkmark | PR link, Mark Merged |
| MERGED | Muted/complete | Double check | PR link |
| BLOCKED (dep) | Warning tint | Clock | — |
| BLOCKED (fail) | Error tint | X | Retry button, View Logs |

### 6.4 SubtaskDetail Panel

Opens when clicking a card. Slide-in panel from right or modal.

**Sections:**

1. **Header**: Title, status badge, parent task name
2. **Spec**: Markdown-rendered specification
3. **Implementation Plan**: Markdown-rendered plan
4. **Branch**: Branch name (if exists)
5. **PR**: Link to PR (if exists)
6. **Actions**: Start/Mark Merged/Retry buttons (context-dependent)
7. **Agent Runs**: Collapsible list of run history

### 6.5 LogViewer

Full-screen or large modal displaying agent logs.

**Features:**
- Monospace font
- Line numbers
- Auto-scroll to bottom (optional)
- Search within logs (stretch goal)
- Copy button
- Syntax highlighting for code blocks (stretch goal)

### 6.6 Dialogs

**AddProjectDialog:**
- Single input: GitHub repo URL
- Validation: must be valid GitHub URL format
- Submit button with loading state
- Error display inline

**CreateTaskDialog:**
- Title input (text, max 200 chars)
- Description textarea (multi-line, required)
- Submit button with loading state
- Error display inline

---

## 7. API Integration

### 7.1 HTTP Client

```typescript
// api/client.ts
import ky from 'ky';

export const api = ky.create({
  prefixUrl: '/api',
  credentials: 'include', // Send cookies
  hooks: {
    afterResponse: [
      async (request, options, response) => {
        if (response.status === 401) {
          window.location.href = '/login';
        }
      }
    ]
  }
});
```

### 7.2 API Functions

```typescript
// api/auth.ts
export const getMe = () => api.get('auth/me').json<User>();
export const logout = () => api.post('auth/logout');

// api/projects.ts
export const listProjects = () => api.get('projects').json<Project[]>();
export const getProject = (id: string) => api.get(`projects/${id}`).json<Project>();
export const createProject = (repoUrl: string) =>
  api.post('projects', { json: { repo_url: repoUrl } }).json<Project>();
export const deleteProject = (id: string) => api.delete(`projects/${id}`);

// api/tasks.ts
export const listTasks = (projectId: string) =>
  api.get(`projects/${projectId}/tasks`).json<Task[]>();
export const createTask = (projectId: string, title: string, description: string) =>
  api.post(`projects/${projectId}/tasks`, { json: { title, description } }).json<Task>();
export const retryPlanning = (taskId: string) =>
  api.post(`tasks/${taskId}/retry-planning`).json<Task>();
export const deleteTask = (taskId: string) => api.delete(`tasks/${taskId}`);

// api/subtasks.ts
export const listSubtasks = (taskId: string) =>
  api.get(`tasks/${taskId}/subtasks`).json<Subtask[]>();
export const startSubtask = (id: string) =>
  api.post(`subtasks/${id}/start`).json<Subtask>();
export const markMerged = (id: string) =>
  api.post(`subtasks/${id}/mark-merged`).json<Subtask>();
export const retrySubtask = (id: string) =>
  api.post(`subtasks/${id}/retry`).json<Subtask>();
export const updatePosition = (id: string, position: number) =>
  api.patch(`subtasks/${id}/position`, { json: { position } }).json<Subtask>();

// api/agents.ts
export const listAgentRuns = (subtaskId: string) =>
  api.get(`subtasks/${subtaskId}/runs`).json<AgentRun[]>();
export const getAgentLogs = (runId: string) =>
  api.get(`runs/${runId}/logs`).json<{ content: string }>();
```

### 7.3 Query Hooks (TanStack Query)

```typescript
// hooks/useProjects.ts
export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
  });
}

// hooks/useTasks.ts
export function useTasks(projectId: string) {
  const { isConnected } = useProjectEvents();
  return useQuery({
    queryKey: ['tasks', projectId],
    queryFn: () => listTasks(projectId),
    // SSE provides real-time updates; poll only as fallback when disconnected
    refetchInterval: isConnected ? false : 5000,
  });
}

// hooks/useSubtasks.ts
export function useSubtasks(taskIds: string[]) {
  const { isConnected } = useProjectEvents();
  return useQueries({
    queries: taskIds.map(taskId => ({
      queryKey: ['subtasks', taskId],
      queryFn: () => listSubtasks(taskId),
      refetchInterval: isConnected ? false : 5000,
    })),
    combine: (results) => ({
      data: results.flatMap(r => r.data ?? []),
      isLoading: results.some(r => r.isLoading),
    }),
  });
}
```

### 7.4 Real-Time Updates Strategy

**Primary: SSE (Server-Sent Events)**

The frontend establishes an SSE connection to `/api/projects/{id}/events` which provides real-time updates for all agent activity, task status changes, and subtask transitions. See [realtime-events.md](./realtime-events.md) for full specification.

| Data | Update Method | Fallback |
|------|---------------|----------|
| Projects | Refresh on focus | N/A |
| Tasks | SSE `task:status_changed` events | Poll 5s if disconnected |
| Subtasks | SSE `subtask:status_changed` events | Poll 5s if disconnected |
| Agent Logs | SSE `agent:log` events | Static fetch after completion |

**Fallback: Polling**

Polling (5 seconds) is enabled only when SSE is disconnected. This handles network interruptions gracefully.

---

## 8. Data Model

Frontend types mirror the backend API responses. See [orchestrator.md §4](./orchestrator.md#4-data-model) for the source of truth.

### 8.1 User

| Field | Type | Description |
|-------|------|-------------|
| id | string (UUID) | Unique identifier |
| github_username | string | GitHub login |
| created_at | string (ISO 8601) | Creation timestamp |

### 8.2 Project

| Field | Type | Description |
|-------|------|-------------|
| id | string (UUID) | Unique identifier |
| github_owner | string | Repo owner (org or user) |
| github_repo | string | Repo name |
| is_fork | boolean | Whether we forked this repo |
| default_branch | string | Default branch name |
| created_at | string (ISO 8601) | Creation timestamp |

### 8.3 Task

| Field | Type | Description |
|-------|------|-------------|
| id | string (UUID) | Unique identifier |
| project_id | string (UUID) | Parent project |
| title | string | Task title |
| description | string | Task description |
| status | TaskStatus | One of: PLANNING, PLANNING_FAILED, ACTIVE, DONE |
| created_at | string (ISO 8601) | Creation timestamp |

**TaskStatus Values:**

| Status | Display | UI Behavior |
|--------|---------|-------------|
| PLANNING | "Planning..." badge with spinner | Board empty, polling active |
| PLANNING_FAILED | "Planning Failed" error badge | Show "Retry Planning" button |
| ACTIVE | No badge (normal state) | Subtasks visible on board |
| DONE | "Complete" badge | All subtasks merged |

### 8.4 Subtask

| Field | Type | Description |
|-------|------|-------------|
| id | string (UUID) | Unique identifier |
| task_id | string (UUID) | Parent task |
| title | string | Subtask title |
| spec | string \| null | Specification (markdown) |
| implementation_plan | string \| null | Implementation plan (markdown) |
| status | SubtaskStatus | Current status |
| blocked_reason | BlockedReason | Why blocked (if applicable) |
| branch_name | string \| null | Git branch for this subtask |
| pr_url | string \| null | GitHub PR URL |
| pr_number | number \| null | GitHub PR number |
| retry_count | number | Current retry count (0-10) |
| token_usage | number | Total tokens used |
| position | number | Display order (for drag-and-drop) |
| created_at | string (ISO 8601) | Creation timestamp |

**SubtaskStatus Values:**

| Status | Column | Actions Available |
|--------|--------|-------------------|
| PENDING | (hidden) | None - waiting for Planner |
| READY | Ready | Start |
| IN_PROGRESS | In Progress | None - agent working |
| COMPLETED | Completed | Mark Merged, View PR |
| MERGED | Merged | View PR |
| BLOCKED | Blocked | Retry (if FAILURE), View Logs |

**BlockedReason Values:**

| Reason | Display | User Action |
|--------|---------|-------------|
| DEPENDENCY | "Waiting on dependency" | None - auto-unblocks when dep merged |
| FAILURE | "Failed after 10 attempts" | View logs, Retry |
| null | N/A | Only set when status is BLOCKED |

### 8.5 AgentRun

| Field | Type | Description |
|-------|------|-------------|
| id | string (UUID) | Unique identifier |
| subtask_id | string (UUID) | Associated subtask |
| agent_type | AgentType | PLANNER or WORKER |
| attempt_number | number | Which retry attempt (1-10) |
| status | AgentRunStatus | RUNNING, SUCCEEDED, or FAILED |
| started_at | string (ISO 8601) | Start timestamp |
| ended_at | string \| null | End timestamp |
| token_usage | number \| null | Tokens used in this run |
| error_message | string \| null | Error message if failed |

### 8.6 TypeScript Definitions

```typescript
// types/api.ts

export interface User {
  id: string;
  github_username: string;
  created_at: string;
}

export interface Project {
  id: string;
  github_owner: string;
  github_repo: string;
  is_fork: boolean;
  default_branch: string;
  created_at: string;
}

export type TaskStatus = 'PLANNING' | 'PLANNING_FAILED' | 'ACTIVE' | 'DONE';

export interface Task {
  id: string;
  project_id: string;
  title: string;
  description: string;
  status: TaskStatus;
  created_at: string;
}

export type SubtaskStatus =
  | 'PENDING'
  | 'READY'
  | 'BLOCKED'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'MERGED';

export type BlockedReason = 'DEPENDENCY' | 'FAILURE' | null;

export interface Subtask {
  id: string;
  task_id: string;
  title: string;
  spec: string | null;
  implementation_plan: string | null;
  status: SubtaskStatus;
  blocked_reason: BlockedReason;
  branch_name: string | null;
  pr_url: string | null;
  pr_number: number | null;
  retry_count: number;
  token_usage: number;
  position: number;
  created_at: string;
}

export type AgentType = 'PLANNER' | 'WORKER';
export type AgentRunStatus = 'RUNNING' | 'SUCCEEDED' | 'FAILED';

export interface AgentRun {
  id: string;
  subtask_id: string;
  agent_type: AgentType;
  attempt_number: number;
  status: AgentRunStatus;
  started_at: string;
  ended_at: string | null;
  token_usage: number | null;
  error_message: string | null;
}
```

---

## 9. Styling

### 9.1 Theme

**Dark mode default** with potential light mode toggle (post-MVP).

**Color Palette (dark mode):**

| Role | Color | Usage |
|------|-------|-------|
| Background | zinc-950 | Page background |
| Card | zinc-900 | Cards, panels |
| Border | zinc-800 | Dividers, card borders |
| Text Primary | zinc-50 | Headings, body |
| Text Secondary | zinc-400 | Labels, meta |
| Accent | blue-500 | Links, primary buttons |
| Success | green-500 | Merged, completed |
| Warning | yellow-500 | Blocked (dependency) |
| Error | red-500 | Failed, error states |

### 9.2 Typography

- **Font**: System font stack (native feel)
- **Headings**: Bold, zinc-50
- **Body**: Regular, zinc-200
- **Code/Logs**: JetBrains Mono or system monospace

### 9.3 Spacing

Using Tailwind's default spacing scale:
- Card padding: p-4
- Section gaps: gap-6
- Inline spacing: gap-2

---

## 10. Error Handling

### API Errors

| Status | User Message | Action |
|--------|--------------|--------|
| 400 | "Invalid request: {detail}" | Show in form |
| 401 | — | Redirect to login |
| 403 | "You don't have access" | Show toast |
| 404 | "Not found" | Show toast, navigate back |
| 409 | "Conflict: {detail}" | Show in form |
| 422 | "Cannot perform action: {detail}" | Show toast |
| 500 | "Something went wrong" | Show toast with retry option |

### Loading States

- **Initial load**: Skeleton cards/lists
- **Actions**: Button shows spinner, disabled state
- **Background refresh**: No visible indicator (seamless)

### Empty States

| Context | Message | Action |
|---------|---------|--------|
| No projects | "No projects yet" | "Add your first project" button |
| No tasks | "No tasks in this project" | "Create a task" button |
| Empty column | — | Show column header only, muted |

---

## 11. Business Logic

### 11.1 Authentication State Management

**Trigger:** App initialization, route navigation

**Logic:**
1. On app load, call `GET /api/auth/me`
2. If 200: user is authenticated, store user in context
3. If 401: user is not authenticated, redirect to `/login` (unless already there)
4. On logout: call `POST /api/auth/logout`, clear local state, redirect to `/login`

**Edge Cases:**

| Scenario | Behavior |
|----------|----------|
| Token expires mid-session | 401 response triggers redirect to login |
| Network error on auth check | Show error, allow retry |
| OAuth callback without code | Show error on login page |

### 11.2 Real-Time Events Logic

**Primary:** SSE connection to `/api/projects/{id}/events`

**Logic:**
1. Establish SSE connection when BoardPage mounts
2. Parse incoming events and update TanStack Query cache directly
3. If SSE disconnects, enable polling fallback (5s)
4. On reconnect, fetch fresh data to reconcile any missed events
5. On window focus, trigger immediate refetch (TanStack Query default)

**Implementation:**
```typescript
// useProjectEvents hook manages SSE connection
const { isConnected } = useProjectEvents();

// Polling only as fallback when SSE is disconnected
useQuery({
  queryKey: ['tasks', projectId],
  queryFn: () => listTasks(projectId),
  refetchInterval: isConnected ? false : 5000,
});
```

See [realtime-events.md](./realtime-events.md) for full SSE event types and frontend architecture.

### 11.3 Drag-and-Drop Reordering

**Trigger:** User drags card within a column

**Logic:**
1. On drag end: calculate new position based on drop index
2. Optimistically update local state
3. Call `PATCH /api/subtasks/{id}/position` with new position
4. On error: rollback to previous position, show toast

**Position Calculation:**
- When dropped between cards A and B: `position = (A.position + B.position) / 2`
- When dropped at start: `position = firstCard.position - 1`
- When dropped at end: `position = lastCard.position + 1`

### 11.4 Subtask Action Validation

**Before enabling action buttons, validate:**

| Action | Preconditions |
|--------|---------------|
| Start | `status === 'READY'` |
| Mark Merged | `status === 'COMPLETED'` AND `pr_url !== null` |
| Retry | `status === 'BLOCKED'` AND `blocked_reason === 'FAILURE'` |

**Edge Cases:**

| Scenario | Behavior |
|----------|----------|
| Click Start on stale data (already started) | 409 from API, show "Already in progress" toast, refetch |
| Click Mark Merged before PR exists | Button disabled, never reachable |
| Rapid double-click on action | Disable button on first click, prevent duplicate |

### 11.5 Optimistic Updates

For responsive UI, update local state before API confirms:

| Action | Optimistic Update | Rollback on Error |
|--------|-------------------|-------------------|
| Start Subtask | Move to IN_PROGRESS column | Move back to READY |
| Mark Merged | Move to MERGED column | Move back to COMPLETED |
| Retry Subtask | Move to IN_PROGRESS column | Move back to BLOCKED |
| Reorder Card | Update position immediately | Restore original position |

---

## 12. External Integrations

### 12.1 Backend API (Orchestrator)

**Purpose:** All data operations — the frontend has no direct database or GitHub access.

**Base URL:** `/api` (same origin, proxied in development)

**Authentication:** JWT in HttpOnly cookie (set by backend OAuth flow)

**Operations:**

| Domain | Operations | Reference |
|--------|------------|-----------|
| Auth | Login redirect, logout, get current user | [orchestrator.md §5](./orchestrator.md#5-api-design) |
| Projects | List, create, get, delete | [orchestrator.md §5](./orchestrator.md#5-api-design) |
| Tasks | List, create, retry planning, delete | [orchestrator.md §5](./orchestrator.md#5-api-design) |
| Subtasks | List, start, mark merged, retry, update position | [orchestrator.md §5](./orchestrator.md#5-api-design) |
| Agent Runs | List runs, get logs | [orchestrator.md §5](./orchestrator.md#5-api-design) |

**Error Handling:**

| API Status | Frontend Behavior |
|------------|-------------------|
| 200-299 | Success, update state |
| 400 | Show validation error in form |
| 401 | Redirect to login |
| 403 | Show "access denied" toast |
| 404 | Show "not found" toast, navigate back |
| 409 | Show conflict message, refetch data |
| 422 | Show "cannot perform action" toast |
| 500 | Show "something went wrong" with retry |
| Network error | Show "connection error" with retry |

### 12.2 GitHub (via PR Links)

**Purpose:** View and merge PRs (user action on GitHub, not API calls)

**Operations:**

| Operation | Trigger | Behavior |
|-----------|---------|----------|
| View PR | Click PR link on card | Open `pr_url` in new tab |
| Merge PR | User action on GitHub | Manual, outside our control |
| Return after merge | User clicks "Mark Merged" | Updates our system |

---

## 13. Configuration

### 13.1 Build-Time Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_URL` | `/api` | API base URL (only needed if API is on different origin) |

**Note:** In production, API is same-origin so no configuration needed.

### 13.2 Runtime Configuration

None required. All configuration comes from the backend API responses.

### 13.3 Development Configuration

**vite.config.ts:**
```typescript
export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
```

**Environment files:**
- `.env.development` — development overrides (optional)
- `.env.production` — production overrides (optional)

---

## 14. Security Considerations

### 14.1 Authentication

- **JWT Storage:** HttpOnly cookie set by backend (not accessible to JavaScript)
- **CSRF Protection:** SameSite cookie attribute + origin checking by backend
- **Session Duration:** 24-hour JWT expiry (backend-configured)

### 14.2 Authorization

All authorization is enforced by the backend. Frontend assumptions:

| Resource | Access Rule |
|----------|-------------|
| Project | Only visible if user created it |
| Task | Only visible if user owns parent project |
| Subtask | Only visible if user owns parent project |
| Logs | Only visible if user owns parent project |

**Frontend enforcement:** If API returns 403, show generic "access denied" — don't expose details.

### 14.3 Data Protection

| Data Type | Protection |
|-----------|------------|
| GitHub token | Never exposed to frontend (backend only) |
| User email | Not stored or displayed (GitHub username only) |
| Log contents | May contain sensitive code — no persistence in browser |

### 14.4 XSS Prevention

- **React:** Automatic escaping of rendered values
- **Markdown rendering:** Use sanitizing library (e.g., DOMPurify) if rendering user-generated markdown
- **Log display:** Render as plain text in `<pre>`, never as HTML

### 14.5 External Links

- **PR links:** Open in new tab with `rel="noopener noreferrer"`
- **No other external links** in MVP

---

## 15. Accessibility

Using shadcn/ui (Radix-based) provides:
- Keyboard navigation
- Focus management
- ARIA attributes
- Screen reader support

**Additional considerations:**
- Color contrast meets WCAG AA
- Focus visible indicators
- Semantic HTML structure
- Skip to content link

---

## 16. Build & Deployment

### Development

```bash
cd frontend
npm install
npm run dev
# Runs on http://localhost:5173
# Proxies /api to http://localhost:8080
```

### Production Build

```bash
npm run build
# Outputs to frontend/dist/
```

### Deployment

**Orchestrator serves frontend:**

The Go orchestrator serves the built frontend as static files:

```go
// In server.go
r.Handle("/*", http.FileServer(http.FS(frontendFS)))
```

**Build process:**
1. Frontend builds to `frontend/dist/`
2. Orchestrator embeds `frontend/dist/` via `go:embed`
3. Single Docker image contains both

**Dockerfile update:**
```dockerfile
# Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend (existing)
FROM golang:1.24-alpine AS builder
# ... existing Go build ...
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Final image
FROM alpine:3.19
# ... existing runtime setup ...
```

---

## 17. Testing Requirements

### Unit Tests

- [ ] `lib/utils.ts` — cn() class merging helper
- [ ] `hooks/useAuth.ts` — auth state management
- [ ] `hooks/usePolling.ts` — conditional polling logic
- [ ] Position calculation for drag-and-drop

### Component Tests

- [ ] `SubtaskCard` — renders correct actions for each status
- [ ] `SubtaskCard` — blocked reason displays correctly
- [ ] `Board` — cards appear in correct columns
- [ ] `AddProjectDialog` — validates GitHub URL format
- [ ] `CreateTaskDialog` — validates required fields
- [ ] `TaskFilter` — filtering works correctly
- [ ] `LogViewer` — displays logs with line numbers

### Integration Tests

- [ ] Login flow redirects correctly
- [ ] Protected routes redirect unauthenticated users
- [ ] Project creation shows in sidebar
- [ ] Task creation triggers polling
- [ ] Subtask start moves card to In Progress
- [ ] Mark merged updates dependents

### Manual Verification

- [ ] Full flow: login → add project → create task → planner runs → subtasks appear
- [ ] Full flow: start subtask → worker runs → PR created → mark merged
- [ ] Drag-and-drop reorders cards
- [ ] Log viewer displays correctly
- [ ] Error toasts appear on failures
- [ ] Empty states display correctly

---

## 18. Implementation Phases

See [frontend.impl.md](./frontend.impl.md) for detailed implementation checklist.

### Phase 1: Foundation
- Initialize Vite + React + TypeScript
- Configure Tailwind CSS + shadcn/ui
- Create base layout and routing

### Phase 2: Authentication
- GitHub OAuth redirect flow
- Protected route wrapper
- useAuth hook

### Phase 3: Projects
- Projects list page
- Add/delete project dialogs
- Sidebar navigation

### Phase 4: Tasks
- Task creation dialog
- Planning status indicators
- Task filter dropdown

### Phase 5: Kanban Board
- Board with status columns
- SubtaskCard component
- Drag-and-drop reordering

### Phase 6: Subtask Actions
- Start, Mark Merged, Retry buttons
- Optimistic updates
- Error handling

### Phase 7: Agent Logs
- Subtask detail panel
- Agent run history
- Log viewer

### Phase 8: Polish
- Loading skeletons
- Error toasts
- Empty states

### Phase 9: Build & Deployment
- Production build
- Orchestrator integration
- Docker setup

### Phase 10: Testing
- Unit tests
- Component tests
- Manual verification

---

## 19. Dependencies

### Internal

- **Orchestrator API**: All data operations via REST API (see [orchestrator.md](./orchestrator.md))

### External (npm)

```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.22.0",
    "@tanstack/react-query": "^5.17.0",
    "ky": "^1.2.0",
    "lucide-react": "^0.312.0",
    "@dnd-kit/core": "^6.1.0",
    "@dnd-kit/sortable": "^8.0.0",
    "@radix-ui/react-dialog": "^1.0.5",
    "@radix-ui/react-dropdown-menu": "^2.0.6",
    "@radix-ui/react-select": "^2.0.0",
    "@radix-ui/react-slot": "^1.0.2",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.0",
    "tailwind-merge": "^2.2.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.48",
    "@types/react-dom": "^18.2.18",
    "@vitejs/plugin-react": "^4.2.1",
    "autoprefixer": "^10.4.17",
    "postcss": "^8.4.33",
    "tailwindcss": "^3.4.1",
    "typescript": "^5.3.3",
    "vite": "^5.0.12",
    "vitest": "^1.2.0",
    "@testing-library/react": "^14.1.2"
  }
}
```

---

## Appendix A: API Request/Response Examples

### Create Project

**Request:**
```http
POST /api/projects
Content-Type: application/json

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
  "created_at": "2026-02-05T00:00:00Z"
}
```

### Create Task

**Request:**
```http
POST /api/projects/{project_id}/tasks
Content-Type: application/json

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
  "created_at": "2026-02-05T00:00:00Z"
}
```

### Start Subtask

**Request:**
```http
POST /api/subtasks/{id}/start
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440002",
  "task_id": "550e8400-e29b-41d4-a716-446655440001",
  "title": "Add OAuth handler",
  "status": "IN_PROGRESS",
  "branch_name": "iv-2-add-oauth-handler",
  "retry_count": 0,
  "position": 1,
  "created_at": "2026-02-05T00:00:00Z"
}
```

### Error Response

**Response (409 Conflict):**
```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Subtask is already in progress"
  }
}
```

---

## Appendix B: Keyboard Shortcuts (Stretch Goal)

| Shortcut | Action |
|----------|--------|
| `n` | New task |
| `p` | Add project |
| `?` | Show shortcuts |
| `Esc` | Close dialog/panel |
| `j/k` | Navigate cards |
| `Enter` | Open selected card |

---

## Appendix C: Future Enhancements

| Feature | Description |
|---------|-------------|
| Light mode toggle | System preference support |
| Log search | Search within agent logs |
| Keyboard shortcuts | Power user navigation |
| Notifications | Browser notifications for completed subtasks |
| Cost tracking view | Token usage summary per task/project |
| Bulk actions | Start multiple subtasks, bulk mark merged |
