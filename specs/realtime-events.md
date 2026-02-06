<!--
 Copyright (c) 2026 Intern Village. All rights reserved.
 SPDX-License-Identifier: Proprietary
-->

# Real-Time Agent Events Specification

**Status:** Planned
**Version:** 2.0
**Last Updated:** 2026-02-05
**Architecture Reference:** [architecture.md](./architecture.md)
**Backend Reference:** [orchestrator.md](./orchestrator.md)
**Frontend Reference:** [frontend.md](./frontend.md)

---

## 1. Overview

### Purpose

When users create tasks or start subtasks, AI agents (Planner or Worker) run in the background. Currently, users rely on polling (every 5 seconds) to see status changes and have no visibility into agent logs until completion. This feature adds a **unified real-time event stream** that delivers:

1. **Log streaming**: Watch agent output as it happens
2. **Status updates**: Immediate notification when agents complete/fail
3. **Board state changes**: Tasks and subtasks update instantly

### Goals

- **Real-time visibility**: Stream agent logs and status changes as they happen
- **Unified stream**: Single SSE connection per project handles all event types
- **Eliminate polling**: Frontend no longer needs to poll for status updates
- **Universal access**: Works for both Planner (task-level) and Worker (subtask-level) agents
- **Efficient**: Multiplexed stream reduces connections and server load

### Non-Goals

- **Cross-project streaming**: Each project has its own stream (users can open multiple)
- **Historical event replay**: Only live events; historical data via existing REST APIs
- **Bi-directional communication**: SSE is one-way; actions still use REST endpoints
- **Log persistence changes**: Existing file-based logging remains unchanged

### Context from Architecture

From [architecture.md](./architecture.md):
- Previous decision: "Polling is sufficient for agent status updates"
- **This spec supersedes that decision** for projects with active agents
- Polling remains as fallback for reconnection and initial load

From [orchestrator.md §7.3](./orchestrator.md#73-agent-execution-loop):
- Logs stored at `/data/logs/{project_id}/{task_id}/{subtask_id}/run-{NNN}.log`
- Planner runs use task ID in place of subtask ID
- Logs include timestamps: `[HH:MM:SS] content`

---

## 2. Package/Module Structure

### Backend (Go)

```
orchestrator/
├── internal/
│   ├── api/
│   │   └── handlers/
│   │       └── events.go              # NEW: SSE event stream handler
│   ├── agent/
│   │   ├── executor.go                # Update: notify event hub on log writes
│   │   └── manager.go                 # Update: notify event hub on status changes
│   └── service/
│       ├── event_hub.go               # NEW: Central event distribution
│       └── log_tailer.go              # NEW: Log file tailing logic
```

### Frontend (React)

```
frontend/src/
├── api/
│   └── events.ts                      # NEW: SSE connection management
├── components/
│   ├── agents/
│   │   ├── LogViewer.tsx              # Existing: static log viewer (keep for completed runs)
│   │   └── LiveLogPanel.tsx           # NEW: Real-time log panel
│   └── tasks/
│       └── TaskCard.tsx               # Update: "View Logs" button for PLANNING
├── hooks/
│   ├── useProjectEvents.ts            # NEW: Project-wide SSE hook
│   └── useLiveLog.ts                  # NEW: Subscribe to specific agent's logs
├── contexts/
│   └── ProjectEventsContext.tsx       # NEW: Share SSE connection across components
└── pages/
    └── BoardPage.tsx                  # Update: Use events context, remove polling
```

---

## 3. Event Types

### 3.1 Event Categories

| Category | Events | Purpose |
|----------|--------|---------|
| **Agent** | `agent:started`, `agent:log`, `agent:completed`, `agent:failed` | Agent lifecycle and output |
| **Task** | `task:status_changed` | Task state transitions |
| **Subtask** | `subtask:status_changed`, `subtask:unblocked` | Subtask state transitions |
| **System** | `connected`, `heartbeat`, `error` | Connection management |

### 3.2 Event Schemas

#### agent:started

Sent when an agent begins execution.

```json
{
  "event": "agent:started",
  "data": {
    "run_id": "uuid",
    "agent_type": "PLANNER | WORKER",
    "task_id": "uuid",
    "subtask_id": "uuid | null",
    "attempt_number": 1,
    "started_at": "2026-02-05T14:32:00Z"
  }
}
```

#### agent:log

Sent for each new log line.

```json
{
  "event": "agent:log",
  "data": {
    "run_id": "uuid",
    "line": "[14:32:05] Exploring src/ directory...",
    "line_number": 42,
    "timestamp": "14:32:05"
  }
}
```

#### agent:completed

Sent when an agent finishes successfully.

```json
{
  "event": "agent:completed",
  "data": {
    "run_id": "uuid",
    "agent_type": "PLANNER | WORKER",
    "task_id": "uuid",
    "subtask_id": "uuid | null",
    "duration_ms": 154000,
    "token_usage": 12500,
    "pr_url": "https://github.com/..." // Only for WORKER
  }
}
```

#### agent:failed

Sent when an agent fails (single attempt, not max retries).

```json
{
  "event": "agent:failed",
  "data": {
    "run_id": "uuid",
    "agent_type": "PLANNER | WORKER",
    "task_id": "uuid",
    "subtask_id": "uuid | null",
    "attempt_number": 3,
    "error": "exit code: 1",
    "will_retry": true,
    "next_attempt_at": "2026-02-05T14:35:00Z"
  }
}
```

#### task:status_changed

Sent when a task transitions state.

```json
{
  "event": "task:status_changed",
  "data": {
    "task_id": "uuid",
    "old_status": "PLANNING",
    "new_status": "ACTIVE",
    "changed_at": "2026-02-05T14:32:00Z"
  }
}
```

#### subtask:status_changed

Sent when a subtask transitions state.

```json
{
  "event": "subtask:status_changed",
  "data": {
    "subtask_id": "uuid",
    "task_id": "uuid",
    "old_status": "IN_PROGRESS",
    "new_status": "COMPLETED",
    "blocked_reason": null,
    "pr_url": "https://github.com/...",
    "pr_number": 42,
    "changed_at": "2026-02-05T14:32:00Z"
  }
}
```

#### subtask:unblocked

Sent when a subtask is unblocked (dependency merged).

```json
{
  "event": "subtask:unblocked",
  "data": {
    "subtask_id": "uuid",
    "task_id": "uuid",
    "unblocked_by": "uuid",  // The subtask that was merged
    "changed_at": "2026-02-05T14:32:00Z"
  }
}
```

#### connected

Sent immediately after SSE connection established.

```json
{
  "event": "connected",
  "data": {
    "project_id": "uuid",
    "active_agents": [
      {
        "run_id": "uuid",
        "agent_type": "PLANNER",
        "task_id": "uuid",
        "subtask_id": null,
        "started_at": "2026-02-05T14:30:00Z"
      }
    ]
  }
}
```

#### heartbeat

Sent every 30 seconds to keep connection alive.

```json
{
  "event": "heartbeat",
  "data": {
    "timestamp": "2026-02-05T14:32:00Z"
  }
}
```

#### error

Sent when an error occurs.

```json
{
  "event": "error",
  "data": {
    "code": "LOG_FILE_NOT_FOUND",
    "message": "Log file for run xyz no longer exists",
    "run_id": "uuid"
  }
}
```

---

## 4. User Flows

### Flow 1: Connect to Project Events

1. User navigates to BoardPage for a project
2. Frontend establishes SSE connection to `/api/projects/{id}/events`
3. Server sends `connected` event with list of active agents
4. Frontend updates UI state, marks connection as active
5. Polling is disabled while SSE is connected

### Flow 2: View Planner Logs in Real-Time

1. User creates a task → receives `agent:started` event
2. TaskCard shows "Planning..." with "View Logs" button
3. User clicks "View Logs"
4. LiveLogPanel opens, subscribes to `agent:log` events for that run_id
5. Logs stream in real-time
6. On `agent:completed` → panel shows "Complete", task card updates
7. On `agent:failed` with `will_retry: false` → panel shows error, retry option

### Flow 3: Monitor Board While Agents Work

1. User has multiple subtasks IN_PROGRESS
2. SSE streams `agent:log` events (but panel is closed, logs are buffered/discarded)
3. When Worker completes: `agent:completed` + `subtask:status_changed` events
4. SubtaskCard immediately moves to "Completed" column
5. If dependencies unblock: `subtask:unblocked` events fire
6. Blocked subtasks move to "Ready" column instantly

### Flow 4: Reconnection After Disconnect

1. Network blip causes SSE disconnect
2. Frontend detects disconnect, shows "Reconnecting..." indicator
3. Frontend attempts reconnection with exponential backoff
4. On reconnect: `connected` event with current state
5. Frontend reconciles state (fetch latest via REST if needed)
6. Connection indicator clears

### Flow 5: View Completed Run Logs

1. User clicks "View Logs" on a completed subtask
2. System checks if run is active (no SSE connection for that run)
3. Falls back to static log fetch via `GET /api/runs/{id}/logs`
4. Existing LogViewer (dialog) displays full log content

---

## 5. API Design

### 5.1 Project Events Stream (SSE)

**Endpoint:** `GET /api/projects/{id}/events`

**Headers:**
```
Accept: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `subscribe_logs` | string | No | `none` | Comma-separated run IDs to receive log events for, or `all` |

**SSE Format:**

```
event: agent:log
data: {"run_id":"uuid","line":"[14:32:05] Starting...","line_number":1,"timestamp":"14:32:05"}

event: subtask:status_changed
data: {"subtask_id":"uuid","task_id":"uuid","old_status":"IN_PROGRESS","new_status":"COMPLETED",...}
```

**Connection Behavior:**

| Aspect | Behavior |
|--------|----------|
| Authentication | JWT required (from cookie or Authorization header) |
| Authorization | User must own the project |
| Heartbeat | Server sends `heartbeat` every 30 seconds |
| Timeout | Connection closes after 1 hour, client should reconnect |
| Max connections | 5 per user per project (prevents resource exhaustion) |

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | UNAUTHORIZED | Not authenticated |
| 403 | FORBIDDEN | User doesn't own this project |
| 404 | NOT_FOUND | Project not found |
| 429 | TOO_MANY_CONNECTIONS | Max SSE connections exceeded |

### 5.2 Subscribe to Log Stream

To receive `agent:log` events for a specific agent run, the client can:

**Option A: Query parameter on initial connection**
```
GET /api/projects/{id}/events?subscribe_logs=run-uuid-1,run-uuid-2
```

**Option B: Reconnect with updated subscription**
```
GET /api/projects/{id}/events?subscribe_logs=run-uuid-1,run-uuid-2,run-uuid-3
```

This avoids needing a separate WebSocket or bidirectional channel.

### 5.3 Get Active Runs (REST)

**Endpoint:** `GET /api/projects/{id}/active-runs`

Returns all currently running agents for a project. Used for initial state on page load and reconnection reconciliation.

**Response (200 OK):**
```json
{
  "runs": [
    {
      "id": "uuid",
      "agent_type": "PLANNER",
      "task_id": "uuid",
      "subtask_id": null,
      "attempt_number": 1,
      "status": "RUNNING",
      "started_at": "2026-02-05T14:30:00Z",
      "log_path": "/data/logs/.../run-001.log"
    },
    {
      "id": "uuid",
      "agent_type": "WORKER",
      "task_id": "uuid",
      "subtask_id": "uuid",
      "attempt_number": 2,
      "status": "RUNNING",
      "started_at": "2026-02-05T14:31:00Z",
      "log_path": "/data/logs/.../run-002.log"
    }
  ]
}
```

---

## 6. Business Logic

### 6.1 Event Hub

**Location:** `internal/service/event_hub.go`

The Event Hub is the central coordinator for real-time events. It:
- Maintains a registry of active SSE connections per project
- Receives events from agent execution, task service, subtask service
- Broadcasts events to subscribed connections
- Manages log subscriptions per connection

**Interface:**

```go
type EventHub interface {
    // Connection management
    Subscribe(ctx context.Context, projectID, userID uuid.UUID, logSubscriptions []uuid.UUID) (<-chan Event, func())
    UpdateLogSubscriptions(connID string, runIDs []uuid.UUID)

    // Event publishing (called by services)
    PublishAgentStarted(projectID uuid.UUID, run *domain.AgentRun)
    PublishAgentLog(projectID, runID uuid.UUID, line string, lineNumber int)
    PublishAgentCompleted(projectID uuid.UUID, run *domain.AgentRun, prURL string)
    PublishAgentFailed(projectID uuid.UUID, run *domain.AgentRun, err string, willRetry bool)
    PublishTaskStatusChanged(projectID, taskID uuid.UUID, oldStatus, newStatus string)
    PublishSubtaskStatusChanged(projectID uuid.UUID, subtask *domain.Subtask, oldStatus string)
    PublishSubtaskUnblocked(projectID, subtaskID, unblockedByID uuid.UUID)
}
```

**Implementation:**

```go
type eventHub struct {
    mu          sync.RWMutex
    connections map[uuid.UUID]map[string]*connection  // projectID -> connID -> connection
}

type connection struct {
    userID          uuid.UUID
    eventChan       chan Event
    logSubscriptions map[uuid.UUID]bool  // runIDs this connection wants logs for
}
```

**Behavior:**

| Scenario | Behavior |
|----------|----------|
| No subscribers for project | Events discarded (no buffering) |
| Subscriber's channel full | Event dropped for that subscriber, logged |
| Agent logs with no log subscribers | Log events not generated (saves CPU) |
| Connection closed | Removed from registry, channel closed |

### 6.2 Log Tailer

**Location:** `internal/service/log_tailer.go`

Tails log files and publishes lines to the Event Hub.

**Interface:**

```go
type LogTailer interface {
    StartTailing(ctx context.Context, runID uuid.UUID, logPath string) error
    StopTailing(runID uuid.UUID)
}
```

**Logic:**

```
StartTailing(ctx, runID, logPath):
  1. Check if already tailing this run -> return
  2. Open log file (wait up to 5s if doesn't exist yet)
  3. Start goroutine:
     LOOP every 100ms:
       - Check ctx.Done() -> cleanup, exit
       - Read new bytes from file
       - Parse into lines
       - For each line: eventHub.PublishAgentLog(projectID, runID, line, lineNum)
       - If file ends with "=== Run Complete ===" -> stop tailing
  4. Register in active tailers map

StopTailing(runID):
  1. Cancel context for that tailer
  2. Remove from active tailers map
```

### 6.3 Integration Points

**Agent Executor (`executor.go`):**

```go
// After writing each log line
if e.eventHub != nil {
    e.eventHub.PublishAgentLog(projectID, runID, line, lineNumber)
}
```

Actually, we'll use the LogTailer approach instead of modifying the executor. This keeps concerns separated and handles reconnection scenarios.

**Agent Manager (`manager.go`):**

```go
// When spawning agent goroutine
go func() {
    // Start log tailer in its own goroutine (StartTailing is blocking)
    // Must be started INSIDE the agent goroutine so the agent can create
    // the log file first. StartTailing waits up to 5s for the file to exist.
    go func() {
        logTailer.StartTailing(ctx, projectID, runID, logPath)
    }()

    // Run the actual agent loop
    loop.RunPlannerLoop(ctx, task, project, token)
}()
```

**IMPORTANT:** `StartTailing()` is a blocking call that continuously reads the log file until cancelled. It must:
1. Run in its own goroutine (not block the agent manager)
2. Be started inside the agent goroutine (so the agent creates the log file first)

**Agent Loop (`loop.go`):**

```go
// On success
eventHub.PublishAgentCompleted(projectID, agentRun, prURL)
logTailer.StopTailing(agentRun.ID)

// On failure
eventHub.PublishAgentFailed(projectID, agentRun, errorMsg, willRetry)
if !willRetry {
    logTailer.StopTailing(agentRun.ID)
}
```

**Task Service:**

```go
// On status change
eventHub.PublishTaskStatusChanged(projectID, taskID, oldStatus, newStatus)
```

**Subtask Service:**

```go
// On status change
eventHub.PublishSubtaskStatusChanged(projectID, subtask, oldStatus)

// On unblock
eventHub.PublishSubtaskUnblocked(projectID, subtaskID, unblockedByID)
```

### 6.4 SSE Handler

**Location:** `internal/api/handlers/events.go`

```go
func (h *EventHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
    // 1. Parse project ID, validate ownership
    // 2. Parse subscribe_logs query param
    // 3. Set SSE headers
    // 4. Subscribe to event hub
    // 5. Send "connected" event with active runs
    // 6. Loop:
    //    - Receive from event channel
    //    - Format as SSE
    //    - Write and flush
    //    - Send heartbeat every 30s
    // 7. On context done, cleanup
}
```

---

## 7. Frontend Architecture

### 7.1 ProjectEventsContext

**Location:** `contexts/ProjectEventsContext.tsx`

Provides a shared SSE connection for all components on BoardPage.

```typescript
interface ProjectEventsContextValue {
  isConnected: boolean;
  connectionError: string | null;

  // Subscribe to events for a specific run's logs
  subscribeToLogs: (runId: string) => void;
  unsubscribeFromLogs: (runId: string) => void;

  // Get logs for a run (returns accumulated lines)
  getLogsForRun: (runId: string) => LogLine[];

  // Current active runs
  activeRuns: ActiveRun[];
}
```

**Usage:**

```tsx
// In BoardPage
<ProjectEventsProvider projectId={projectId}>
  <Board />
  <LiveLogPanel />
</ProjectEventsProvider>

// In SubtaskCard
const { activeRuns } = useProjectEvents();
const isRunning = activeRuns.some(r => r.subtask_id === subtask.id);
```

### 7.2 useProjectEvents Hook

**Location:** `hooks/useProjectEvents.ts`

Manages the SSE connection and event handling.

```typescript
function useProjectEvents(projectId: string): {
  isConnected: boolean;
  error: string | null;
  activeRuns: ActiveRun[];
  events: ProjectEvent[];  // Recent events for debugging
  reconnect: () => void;
}
```

**Behavior:**
1. Create EventSource on mount
2. Parse events, dispatch to appropriate handlers
3. Update task/subtask state via query cache invalidation or direct updates
4. Track connection state
5. Auto-reconnect with exponential backoff (1s, 2s, 4s, max 30s)
6. Cleanup on unmount

### 7.3 useLiveLog Hook

**Location:** `hooks/useLiveLog.ts`

Subscribes to log events for a specific run.

```typescript
function useLiveLog(runId: string | null): {
  lines: LogLine[];
  isStreaming: boolean;
  error: string | null;
}
```

**Behavior:**
1. When runId provided, call `subscribeToLogs(runId)` from context
2. Return lines from context's log buffer for this run
3. On unmount or runId change, call `unsubscribeFromLogs`

### 7.4 LiveLogPanel Component

**Location:** `components/agents/LiveLogPanel.tsx`

```typescript
interface LiveLogPanelProps {
  runId: string | null;
  title: string;
  agentType: 'PLANNER' | 'WORKER';
  onClose: () => void;
}
```

**Features:**
- Sheet/panel sliding from right
- Header: title, agent type badge, status, close button
- Log display: monospace, line numbers, syntax highlighting for timestamps
- Auto-scroll with toggle
- Copy button
- Shows "Streaming..." or "Complete" indicator
- Error state with reconnect option

### 7.5 Query Cache Integration

When events arrive, update TanStack Query cache:

```typescript
// In useProjectEvents
const queryClient = useQueryClient();

function handleTaskStatusChanged(event: TaskStatusChangedEvent) {
  queryClient.setQueryData(
    ['tasks', projectId],
    (old: Task[]) => old.map(t =>
      t.id === event.task_id
        ? { ...t, status: event.new_status }
        : t
    )
  );
}

function handleSubtaskStatusChanged(event: SubtaskStatusChangedEvent) {
  queryClient.setQueryData(
    ['subtasks', event.task_id],
    (old: Subtask[]) => old.map(s =>
      s.id === event.subtask_id
        ? { ...s, status: event.new_status, pr_url: event.pr_url, ... }
        : s
    )
  );
}
```

### 7.6 Removing Polling

Update `useTasks` and `useSubtasks` hooks:

```typescript
function useTasks(projectId: string) {
  const { isConnected } = useProjectEvents();

  return useQuery({
    queryKey: ['tasks', projectId],
    queryFn: () => listTasks(projectId),
    // Only poll if SSE is disconnected
    refetchInterval: isConnected ? false : 5000,
  });
}
```

---

## 8. Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `SSE_HEARTBEAT_INTERVAL_S` | integer | No | `30` | Seconds between heartbeat events |
| `SSE_CONNECTION_TIMEOUT_M` | integer | No | `60` | Minutes before forcing reconnection |
| `SSE_MAX_CONNECTIONS_PER_USER` | integer | No | `5` | Max SSE connections per user |
| `LOG_TAIL_POLL_MS` | integer | No | `100` | Log file poll interval |
| `LOG_TAIL_MAX_LINE_BYTES` | integer | No | `1048576` | Max line length (1MB) |
| `EVENT_CHANNEL_BUFFER` | integer | No | `100` | Buffer size for event channels |

---

## 9. Security Considerations

### Authentication

- SSE endpoint requires valid JWT
- Token validated on connection establishment
- Token expiry during connection: send `error` event, close connection

### Authorization

- User can only connect to projects they own
- Events only sent for resources user has access to
- Log subscriptions validated against run ownership

### Resource Protection

- Max connections per user prevents DoS
- Event channel buffers prevent memory exhaustion
- Heartbeat timeout closes abandoned connections
- Log tailers stop when agent completes

### Data Protection

- Log content streamed over HTTPS
- No additional sensitive data beyond existing log files
- Connection state not persisted (stateless reconnection)

---

## 10. Testing Requirements

### Unit Tests

**Backend:**
- [ ] `event_hub_test.go`: Subscribe/unsubscribe connections
- [ ] `event_hub_test.go`: Publish routes to correct subscribers
- [ ] `event_hub_test.go`: Log subscriptions filter correctly
- [ ] `event_hub_test.go`: Channel full handling
- [ ] `log_tailer_test.go`: Tails new lines correctly
- [ ] `log_tailer_test.go`: Handles file not existing initially
- [ ] `log_tailer_test.go`: Stops on context cancel

**Frontend:**
- [ ] `useProjectEvents.test.ts`: Connection state transitions
- [ ] `useProjectEvents.test.ts`: Event parsing and dispatch
- [ ] `useProjectEvents.test.ts`: Reconnection logic
- [ ] `useLiveLog.test.ts`: Log accumulation
- [ ] `LiveLogPanel.test.tsx`: Renders log lines correctly

### Integration Tests

- [ ] SSE endpoint returns correct headers
- [ ] `connected` event includes active runs
- [ ] `agent:log` events stream correctly
- [ ] Status change events update board
- [ ] Reconnection receives fresh state
- [ ] 403 for unauthorized project access

### Manual Verification

- [ ] Create task, see Planner logs streaming
- [ ] Start subtask, see Worker logs streaming
- [ ] Multiple agents running, board updates in real-time
- [ ] Disconnect network, reconnection succeeds
- [ ] Task completes while viewing logs, panel updates
- [ ] Subtask completes, dependents unblock immediately

---

## 11. Implementation Phases

### Phase 1: Event Hub Infrastructure

- [ ] Create `internal/service/event_hub.go`
  - Connection registry
  - Subscribe/unsubscribe logic
  - Event publishing methods
  - Log subscription filtering
- [ ] Add unit tests for event hub
- [ ] Create `internal/service/log_tailer.go`
  - File tailing logic
  - Line parsing
  - Graceful shutdown
- [ ] Add unit tests for log tailer

### Phase 2: Backend Event Publishing

- [ ] Update `agent/manager.go` to publish `agent:started`
- [ ] Update `agent/loop.go` to publish `agent:completed`, `agent:failed`
- [ ] Update `service/task_service.go` to publish `task:status_changed`
- [ ] Update `service/subtask_service.go` to publish subtask events
- [ ] Wire event hub and log tailer into server startup
- [ ] Integration test event publishing

### Phase 3: SSE API Endpoint

- [ ] Create `internal/api/handlers/events.go`
  - SSE response handling
  - Event formatting
  - Heartbeat goroutine
  - Connection cleanup
- [ ] Add `GET /api/projects/{id}/events` route
- [ ] Add `GET /api/projects/{id}/active-runs` endpoint
- [ ] Add integration tests for SSE endpoint

### Phase 4: Frontend Event Infrastructure

- [ ] Create `contexts/ProjectEventsContext.tsx`
- [ ] Create `hooks/useProjectEvents.ts`
  - EventSource management
  - Event parsing
  - Query cache updates
  - Reconnection logic
- [ ] Create `hooks/useLiveLog.ts`
- [ ] Add unit tests for hooks

### Phase 5: Frontend UI Components

- [ ] Create `components/agents/LiveLogPanel.tsx`
  - Sheet layout
  - Log rendering
  - Auto-scroll
  - Status indicators
- [ ] Update `TaskCard.tsx` with "View Logs" for PLANNING
- [ ] Update `SubtaskCard.tsx` with "View Logs" for IN_PROGRESS
- [ ] Update `BoardPage.tsx` with events provider and log panel state

### Phase 6: Remove Polling

- [ ] Update `useTasks.ts` to disable polling when connected
- [ ] Update `useSubtasks.ts` to disable polling when connected
- [ ] Add polling fallback for disconnected state
- [ ] Test reconnection scenarios

### Phase 7: Polish

- [ ] Add copy button to log panel
- [ ] Add auto-scroll toggle
- [ ] Add connection status indicator to BoardPage header
- [ ] Handle edge cases (very long lines, rapid events)
- [ ] Final manual verification
- [ ] Update architecture.md to reflect new SSE approach

---

## 12. Dependencies

### Internal

- `internal/agent/executor.go`: Log file format (unchanged)
- `internal/agent/manager.go`: Agent lifecycle hooks
- `internal/agent/loop.go`: Completion/failure hooks
- `internal/service/*_service.go`: Status change hooks
- `internal/repository`: Agent run queries

### External (Go)

| Package | Version | Purpose |
|---------|---------|---------|
| (none new) | - | Standard library for SSE and file tailing |

### External (npm)

| Package | Version | Purpose |
|---------|---------|---------|
| (none new) | - | Native EventSource API |

---

## Appendix A: SSE Wire Format

Standard SSE format with named events:

```
event: agent:log
data: {"run_id":"abc","line":"[14:32:05] Hello","line_number":1,"timestamp":"14:32:05"}

event: subtask:status_changed
data: {"subtask_id":"xyz","old_status":"IN_PROGRESS","new_status":"COMPLETED"}

:heartbeat
```

Notes:
- Each event is `event:` line followed by `data:` line
- Multi-line data: multiple `data:` lines, joined with newlines
- Comments (`:`) used for heartbeat to prevent timeout
- Blank line terminates each event

---

## Appendix B: Reconnection Strategy

**Frontend reconnection:**

1. On disconnect, set `isConnected = false`
2. Enable polling fallback immediately
3. Attempt reconnect with backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
4. On reconnect:
   - Receive `connected` event with active runs
   - Invalidate queries to fetch fresh data
   - Disable polling
5. After 5 failed attempts, show manual reconnect button

**Backend connection lifecycle:**

1. Connection established, add to registry
2. Send `connected` event
3. Start heartbeat timer (30s)
4. On context done (client disconnect, timeout):
   - Remove from registry
   - Stop any log subscriptions for this connection

---

## Appendix C: Migration from Polling

The transition from polling to SSE is graceful:

1. **Phase 1-3**: Backend ready, frontend still polls
2. **Phase 4-5**: Frontend connects SSE, but polling remains as backup
3. **Phase 6**: Polling disabled when SSE connected
4. **Fallback**: Polling re-enables on disconnect

No breaking changes; old clients continue to work via polling.

---

## Appendix D: Future Considerations

| Feature | Why Deferred | Potential Approach |
|---------|--------------|-------------------|
| Log search | MVP focuses on streaming | Client-side Ctrl+F or server-side grep |
| Multi-project view | Scope is per-project | Aggregate SSE or multiple connections |
| Event persistence | No replay needed | Store events in DB for debugging |
| Bidirectional control | Actions use REST | Upgrade to WebSocket if needed |
| Log syntax highlighting | Nice to have | Highlight timestamps, errors, file paths |
