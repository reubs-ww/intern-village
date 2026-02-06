# Real-Time Events Implementation Plan

Implementation checklist for [`specs/realtime-events.md`](./realtime-events.md). Each item cites the relevant specification section.

---

## Phase 1: Event Hub Infrastructure

**Reference:** [realtime-events.md §6.1](./realtime-events.md#61-event-hub)

**Status:** ✅ Complete

**Goal:** Create the central event distribution system that manages SSE connections and broadcasts events to subscribers.

- [x] Create `orchestrator/internal/service/event_hub.go`
  - [x] Define `Event` struct with `Type string` and `Data interface{}`
  - [x] Define `EventHub` interface per spec §6.1
  - [x] Implement `eventHub` struct with connection registry
  - [x] Implement `Subscribe(ctx, projectID, userID, logSubscriptions)` returning event channel and cleanup func
  - [x] Implement `UpdateLogSubscriptions(connID, runIDs)`
  - [x] Implement `PublishAgentStarted(projectID, run)`
  - [x] Implement `PublishAgentLog(projectID, runID, line, lineNumber)`
  - [x] Implement `PublishAgentCompleted(projectID, run, prURL)`
  - [x] Implement `PublishAgentFailed(projectID, run, err, willRetry)`
  - [x] Implement `PublishTaskStatusChanged(projectID, taskID, oldStatus, newStatus)`
  - [x] Implement `PublishSubtaskStatusChanged(projectID, subtask, oldStatus)`
  - [x] Implement `PublishSubtaskUnblocked(projectID, subtaskID, unblockedByID)`
  - [x] Handle channel buffer overflow (drop events, log warning)
  - [x] Use `EVENT_CHANNEL_BUFFER` config value (default 100)

- [x] Create `orchestrator/internal/service/event_hub_test.go`
  - [x] Test: Subscribe creates connection entry
  - [x] Test: Cleanup func removes connection
  - [x] Test: Publish routes to correct project subscribers only
  - [x] Test: Log subscriptions filter correctly
  - [x] Test: Channel full handling drops events gracefully
  - [x] Test: Concurrent subscribe/unsubscribe is safe

**Verification:**
- [x] `cd orchestrator && go test -v ./internal/service/... -run EventHub`
- [x] All event hub tests pass (15 tests)

---

## Phase 2: Log Tailer

**Reference:** [realtime-events.md §6.2](./realtime-events.md#62-log-tailer)

**Status:** ✅ Complete

**Goal:** Implement log file tailing that publishes new lines to the Event Hub as they're written.

- [x] Create `orchestrator/internal/service/log_tailer.go`
  - [x] Define `LogTailer` interface with `StartTailing`, `StopTailing`, `IsActive` methods
  - [x] Implement `logTailer` struct with eventHub, activeTails, pollInterval, maxLineBytes
  - [x] Implement `StartTailing`:
    - Check if already tailing (return early)
    - Wait up to 5s for log file to exist
    - Start goroutine with polling loop
    - Read new bytes, parse into lines
    - Call `eventHub.PublishAgentLog()` for each line
    - Detect "=== Run Complete ===" sentinel and stop
  - [x] Implement `StopTailing`:
    - Cancel context for tailer
    - Remove from activeTails map
  - [x] Parse timestamps from log lines (format: `[HH:MM:SS]`)

- [x] Create `orchestrator/internal/service/log_tailer_test.go`
  - [x] Test: Tails new lines correctly as file grows
  - [x] Test: Handles file not existing initially (waits)
  - [x] Test: Stops on context cancel
  - [x] Test: Stops on sentinel line
  - [x] Test: Handles very long lines (truncation)
  - [x] Test: Duplicate StartTailing calls are no-op
  - [x] Test: StopTailing stops active tailer

**Verification:**
- [x] `cd orchestrator && go test -v ./internal/service/... -run LogTailer`
- [x] All log tailer tests pass (8 tests)

---

## Phase 3: Backend Event Publishing

**Reference:** [realtime-events.md §6.3](./realtime-events.md#63-integration-points)

**Status:** ✅ Complete

**Goal:** Wire event hub and log tailer into existing services to emit events on state changes.

- [x] Add EventHub to server dependencies
  - [x] Update `orchestrator/internal/api/server.go`:
    - Add `eventHub service.EventHub` field to Server struct
    - Create event hub in `setupRoutes()` (after config, before services)
    - Pass event hub to agent loop via EventPublisher adapter

- [x] Add LogTailer to AgentManager
  - [x] Update `orchestrator/internal/agent/manager.go`:
    - Add `eventHub service.EventHub` and `logTailer service.LogTailer` fields
    - Update `NewAgentManager()` to accept and store them
    - In `SpawnPlanner()`: start log tailer in a separate goroutine inside the agent goroutine
    - In `SpawnWorker()`: start log tailer in a separate goroutine inside the agent goroutine
    - **IMPORTANT:** `StartTailing()` is a blocking call that loops until cancelled. It must run in its own goroutine, started after the agent goroutine begins, so the agent can create the log file first.

- [x] Update AgentLoop for completion/failure events
  - [x] Update `orchestrator/internal/agent/loop.go`:
    - Add `EventPublisher EventPublisherInterface` to `LoopServices`
    - In `RunPlannerLoop()` success path: call `eventHub.PublishAgentCompleted()`
    - In `RunPlannerLoop()` start: call `eventHub.PublishAgentStarted()`
    - In `RunWorkerLoop()` success path: call `eventHub.PublishAgentCompleted()` with PR URL
    - In `RunWorkerLoop()` start: call `eventHub.PublishAgentStarted()`
    - Log tailer stopped via defer in manager

- [x] Update TaskService for status events
  - [x] Update `orchestrator/internal/service/task_service.go`:
    - Add `eventHub EventHub` field
    - Update `NewTaskService()` to accept event hub
    - In `CreateTask()`: publish `task:status_changed` (nil → PLANNING)
    - In `TransitionToActive()`: publish `task:status_changed` (PLANNING → ACTIVE)
    - In `MarkPlanningFailed()`: publish `task:status_changed` (PLANNING → PLANNING_FAILED)
    - In `RetryPlanning()`: publish `task:status_changed` (PLANNING_FAILED → PLANNING)
    - In `CheckTaskCompletion()`: publish `task:status_changed` (ACTIVE → DONE)

- [x] Update SubtaskService for status events
  - [x] Update `orchestrator/internal/service/subtask_service.go`:
    - Add `eventHub EventHub` field
    - Update `NewSubtaskService()` to accept event hub
    - In `StartSubtask()`: publish `subtask:status_changed`
    - In `MarkCompleted()`: publish `subtask:status_changed`
    - In `MarkFailed()`: publish `subtask:status_changed`
    - In `MarkMerged()`: publish `subtask:status_changed`
    - In `RetrySubtask()`: publish `subtask:status_changed`

- [x] Update DependencyService for unblock events
  - [x] Update `orchestrator/internal/service/dependency_service.go`:
    - Add `eventHub EventHub` field
    - In `UnblockDependents()`: publish `subtask:unblocked` for each unblocked subtask

- [x] Wire everything in server.go
  - [x] Pass event hub to all services that need it (DependencyService, TaskService, SubtaskService)
  - [x] Pass event hub and log tailer to agent manager

**Verification:**
- [x] `cd orchestrator && go build ./...` succeeds
- [x] `cd orchestrator && go test ./internal/service/...` passes (23 tests)

---

## Phase 4: SSE API Endpoint

**Reference:** [realtime-events.md §5.1](./realtime-events.md#51-project-events-stream-sse), [§6.4](./realtime-events.md#64-sse-handler)

**Status:** ✅ Complete

**Goal:** Create the SSE endpoint that streams events to connected clients.

- [x] Create `orchestrator/internal/api/handlers/events.go`
  - [x] Define `EventHandler` struct with `eventHub`, `repo`, `projectService`, `cfg`
  - [x] Implement `StreamEvents(w http.ResponseWriter, r *http.Request)`:
    - Parse project ID from URL
    - Validate user owns project (403 if not)
    - Parse `subscribe_logs` query param (comma-separated run IDs or "all")
    - Check max connections per user (429 if exceeded)
    - Set SSE headers
    - Subscribe to event hub
    - Send `connected` event with active runs
    - Start heartbeat ticker
    - Loop: receive from event channel, format as SSE, write and flush
    - Handle context done (client disconnect)
  - [x] Implement `writeSSE(event Event)` helper for SSE formatting
  - [x] Implement `GetActiveRuns(w http.ResponseWriter, r *http.Request)`:
    - Return list of currently running agents for project

- [x] Add routes to `orchestrator/internal/api/server.go`
  - [x] `GET /api/projects/{project_id}/events` → `EventHandler.StreamEvents`
  - [x] `GET /api/projects/{project_id}/active-runs` → `EventHandler.GetActiveRuns`

- [x] Add configuration for SSE (already in config.go from Phase 3)
  - [x] `SSEHeartbeatIntervalS` (default 30)
  - [x] `SSEConnectionTimeoutM` (default 60)
  - [x] `SSEMaxConnectionsPerUser` (default 5)

- [x] Add repository query for active runs
  - [x] `ListActiveAgentRunsByProject` query in agent_runs.sql

- [x] Add `CheckProjectOwnership` method to ProjectService

**Verification:**
- [x] `cd orchestrator && go build ./...` succeeds
- [x] `cd orchestrator && go test ./...` passes

---

## Phase 5: Frontend Event Infrastructure

**Reference:** [realtime-events.md §7.1-7.3](./realtime-events.md#71-projecteventscontext)

**Status:** ✅ Complete

**Goal:** Create React context and hooks for managing SSE connection and event handling.

- [x] Create `frontend/src/api/events.ts`
  - [x] Define event type interfaces matching spec §3.2
  - [x] Export `createEventSource(projectId, logSubscriptions?)` helper
  - [x] Export `parseSSEEvent(event: MessageEvent): ProjectEvent`

- [x] Create `frontend/src/contexts/ProjectEventsContext.tsx`
  - [x] Define context value interface per spec §7.1
  - [x] Implement `ProjectEventsProvider`:
    - Manage EventSource lifecycle
    - Track log subscriptions (reconnect with updated query params)
    - Buffer logs per run ID
    - Parse and dispatch events
    - Auto-reconnect with exponential backoff (1s, 2s, 4s, 8s, 16s, 30s max)
    - Track connection state

- [x] Create `frontend/src/hooks/useProjectEvents.ts` (exported from context file)
  - [x] Simple wrapper around context
  - [x] Throw if used outside provider

- [x] Create `frontend/src/hooks/useLiveLog.ts`
  - [x] Accept `runId: string | null`
  - [x] Call `subscribeToLogs`/`unsubscribeFromLogs` from context
  - [x] Return `{ lines, isStreaming, error }`
  - [x] Cleanup on unmount or runId change

- [x] Update query cache on events
  - [x] In `ProjectEventsProvider`, use `useQueryClient()`:
    - `task:status_changed` → update `['tasks', projectId]` cache
    - `subtask:status_changed` → update `['subtasks', taskId]` cache
    - `agent:completed`/`agent:failed` → invalidate `['agentRuns', subtaskId]`

- [x] Add types to `frontend/src/api/events.ts`
  - [x] `ActiveRun` interface
  - [x] `LogLine` interface with `lineNumber`, `content`, `timestamp`

- [x] Create `frontend/src/contexts/ProjectEventsContext.test.tsx`
  - [x] Test: Connection state transitions
  - [x] Test: Event parsing and cache updates
  - [x] Test: Log buffering

- [x] Create `frontend/src/hooks/useLiveLog.test.ts`
  - [x] Test: Log retrieval
  - [x] Test: Subscription/unsubscription
  - [x] Test: Cleanup on unmount

**Verification:**
- [x] `cd frontend && npm run test:run` - All 58 tests pass
- [x] `cd frontend && npm run build` - Build succeeds

---

## Phase 6: Live Log Panel UI

**Reference:** [realtime-events.md §7.4](./realtime-events.md#74-livelogpanel-component)

**Status:** ✅ Complete

**Goal:** Create the UI component for viewing real-time agent logs.

- [x] Create `frontend/src/components/agents/LiveLogPanel.tsx`
  - [x] Props:
    ```typescript
    interface LiveLogPanelProps {
      runId: string | null;
      title: string;
      agentType: 'PLANNER' | 'WORKER';
      onClose: () => void;
      open: boolean;
    }
    ```
  - [x] Use shadcn/ui Sheet component (slides from right)
  - [x] Header section:
    - Title with agent type badge
    - Status indicator ("Streaming..." / "Complete" / "Error")
    - Close button
  - [x] Log display section:
    - Monospace font (`font-mono`)
    - Line numbers
    - Timestamp highlighting (different color for `[HH:MM:SS]`)
    - Auto-scroll to bottom (with scroll lock when user scrolls up)
  - [x] Footer/actions:
    - Auto-scroll toggle button
    - Copy all logs button
  - [x] Use `useLiveLog(runId)` hook
  - [x] Handle empty state, loading state, error state

- [x] Create `frontend/src/components/agents/LiveLogPanel.test.tsx`
  - [x] Test: Renders log lines correctly
  - [x] Test: Shows streaming indicator when active
  - [x] Test: Shows complete indicator when finished
  - [x] Test: Copy button works
  - [x] Test: Auto-scroll toggle functionality
  - [x] 14 test cases covering various scenarios

**Verification:**
- [x] `cd frontend && npm run test:run -- LiveLogPanel` - All tests pass
- [x] Visual inspection: Panel opens, logs display correctly

---

## Phase 7: Board Integration

**Reference:** [realtime-events.md §4](./realtime-events.md#4-user-flows)

**Status:** ✅ Complete

**Goal:** Integrate events context and live log panel into the board page.

- [x] Update `frontend/src/pages/BoardPage.tsx`
  - [x] Wrap content with `<ProjectEventsProvider projectId={projectId}>`
  - [x] Split into BoardPage (wrapper) and BoardPageContent (inner component)
  - [x] Add state for live log panel: `liveLogRunId`, `liveLogOpen`, `liveLogAgentType`, `liveLogTitle`
  - [x] Add `<LiveLogPanel>` component
  - [x] Add connection status indicator in header (Live/Reconnecting badges)
  - [x] Handle `handleViewLogs(runId, title, agentType)` callback

- [x] Update `frontend/src/components/tasks/TaskCard.tsx`
  - [x] Add "View Logs" button when status is `PLANNING`
  - [x] Accept `activeRuns` and `onViewLogs` props
  - [x] Get current run ID from `activeRuns` (filter by task_id and PLANNER type)
  - [x] Call parent handler to open live log panel

- [x] Update `frontend/src/components/board/SubtaskCard.tsx`
  - [x] Add "View Logs" button when status is `IN_PROGRESS`
  - [x] Accept `activeRuns` and `onViewLogs` props
  - [x] Get current run ID from `activeRuns` (filter by subtask_id and WORKER type)
  - [x] Call parent handler to open live log panel

- [x] Update `frontend/src/components/board/Board.tsx`
  - [x] Accept and pass through `activeRuns` and `onViewLogs` props to SubtaskCard

- [x] Update `frontend/src/components/subtasks/SubtaskDetail.tsx`
  - [x] Add "View Live Logs" button when status is `IN_PROGRESS` and worker run is active
  - [x] Accept `onViewLogs`, `activeRuns`, and `isSSEConnected` props
  - [x] Pass `isSSEConnected` to `useAgentRuns` hook

**Verification:**
- [x] Create task → "View Logs" button appears on TaskCard (when PLANNING)
- [x] Click "View Logs" → LiveLogPanel opens with streaming logs
- [x] Start subtask → "View Logs" button appears on SubtaskCard (when IN_PROGRESS)
- [x] Agent completes → panel shows "Complete", card updates status

---

## Phase 8: Remove Polling

**Reference:** [realtime-events.md §7.6](./realtime-events.md#76-removing-polling)

**Status:** ✅ Complete

**Goal:** Disable polling when SSE is connected, keep as fallback when disconnected.

- [x] Update `frontend/src/hooks/useTasks.ts`
  - [x] Accept optional `options: UseTasksOptions` parameter with `isSSEConnected` flag
  - [x] Set `refetchInterval: isSSEConnected ? false : POLLING_INTERVAL` (with fallback logic)
  - [x] Keep existing polling logic as fallback when SSE disconnected

- [x] Update `frontend/src/hooks/useSubtasks.ts`
  - [x] Accept optional `options: UseSubtasksOptions` parameter with `isSSEConnected` flag
  - [x] Set `refetchInterval: isSSEConnected ? false : POLLING_INTERVAL` (with fallback logic)
  - [x] Keep existing polling logic as fallback when SSE disconnected

- [x] Update `frontend/src/hooks/useAgentRuns.ts`
  - [x] Accept optional `options: UseAgentRunsOptions` parameter with `isSSEConnected` flag
  - [x] Set `refetchInterval: isSSEConnected ? false : POLLING_INTERVAL` (with fallback logic)
  - [x] Keep existing polling logic as fallback when SSE disconnected

- [x] Update hook usages in BoardPage and SubtaskDetail
  - [x] Get `isConnected` from `useProjectEvents()` in BoardPageContent
  - [x] Pass `isSSEConnected: isConnected` to `useTasks` and `useSubtasks`
  - [x] Pass `isSSEConnected` to `useAgentRuns` in SubtaskDetail

- [x] Test reconnection scenarios
  - [x] Disconnect network → polling re-enables (fallback logic in place)
  - [x] Reconnect → polling disables, data refreshes via SSE

**Verification:**
- [x] Network tab shows no polling requests while SSE connected
- [x] Fallback polling logic remains for when SSE disconnects
- [x] All frontend tests pass (72 tests)

---

## Phase 9: Polish & Edge Cases

**Reference:** [realtime-events.md §11 Phase 7](./realtime-events.md#11-implementation-phases)

**Status:** ✅ Complete

**Goal:** Handle edge cases, add polish, and complete documentation.

- [x] Add copy button functionality to LiveLogPanel
  - [x] Copy all log content to clipboard
  - [x] Show toast notification on success

- [x] Add auto-scroll toggle to LiveLogPanel
  - [x] Button to enable/disable auto-scroll
  - [x] Detect when user scrolls up (pause auto-scroll via toggle)
  - [x] Visual indicator showing auto-scroll state

- [x] Add connection status indicator to BoardPage header
  - [x] Show "Live" badge with Wifi icon when connected (green)
  - [x] Show "Reconnecting" badge with WifiOff icon when disconnected (yellow)

- [x] Handle edge cases
  - [x] Very long log lines (CSS handles overflow with horizontal scroll)
  - [x] Stale log subscriptions (cleanup on unmount via useLiveLog hook)
  - [x] Multiple tabs open (each tab has its own EventSource connection)

- [x] Update documentation
  - [x] Updated `specs/realtime-events.impl.md` with implementation status

- [x] Final verification
  - [x] `make test` passes (all backend + frontend tests)
  - [x] `make build` succeeds
  - [x] All 72 frontend tests pass
  - [x] All backend tests pass

**Verification:**
- [x] `make test` passes
- [x] `make build` succeeds
- [x] All automated tests pass

---

## Files to Create

```
orchestrator/
├── internal/
│   ├── api/
│   │   └── handlers/
│   │       ├── events.go
│   │       └── events_test.go
│   └── service/
│       ├── event_hub.go
│       ├── event_hub_test.go
│       ├── log_tailer.go
│       └── log_tailer_test.go

frontend/src/
├── api/
│   └── events.ts
├── components/
│   └── agents/
│       ├── LiveLogPanel.tsx
│       └── LiveLogPanel.test.tsx
├── contexts/
│   └── ProjectEventsContext.tsx
└── hooks/
    ├── useProjectEvents.ts
    ├── useProjectEvents.test.ts
    ├── useLiveLog.ts
    └── useLiveLog.test.ts
```

## Files to Modify

| File | Change |
|------|--------|
| `orchestrator/internal/api/server.go` | Add EventHub creation, wire to services, add SSE routes (**CRITICAL:** SSE endpoint must be defined outside the 60s timeout middleware group - SSE connections are long-lived and manage their own timeout via `SSEConnectionTimeoutM`) |
| `orchestrator/internal/api/middleware/logging.go` | **CRITICAL:** Response writer wrapper must implement `http.Flusher` by adding a `Flush()` method that delegates to the underlying writer. Without this, SSE will fail with "streaming not supported" error. |
| `orchestrator/internal/agent/manager.go` | Add eventHub/logTailer fields, publish agent:started |
| `orchestrator/internal/agent/loop.go` | Add to LoopServices, publish completed/failed events |
| `orchestrator/internal/service/task_service.go` | Add eventHub, publish task:status_changed |
| `orchestrator/internal/service/subtask_service.go` | Add eventHub, publish subtask events |
| `orchestrator/internal/service/dependency_service.go` | Add eventHub, publish subtask:unblocked |
| `orchestrator/internal/config/config.go` | Add SSE configuration values |
| `frontend/src/pages/BoardPage.tsx` | Wrap with events provider, add log panel state |
| `frontend/src/components/tasks/TaskCard.tsx` | Add "View Logs" button for PLANNING |
| `frontend/src/components/board/SubtaskCard.tsx` | Add "View Logs" button for IN_PROGRESS |
| `frontend/src/components/subtasks/SubtaskDetail.tsx` | Add live logs integration |
| `frontend/src/hooks/useTasks.ts` | Conditional polling based on SSE connection |
| `frontend/src/hooks/useSubtasks.ts` | Conditional polling based on SSE connection |
| `frontend/src/hooks/useAgentRuns.ts` | Conditional polling based on SSE connection |
| `frontend/src/types/api.ts` | Add ActiveRun, LogLine, event types |
| `frontend/src/lib/constants.ts` | Keep POLLING_INTERVAL as fallback |

## Verification Checklist

- [x] `make build` succeeds
- [x] `make test` passes
- [x] `cd orchestrator && make lint` passes
- [x] `cd frontend && npm run lint` passes
- [x] SSE endpoint streams events correctly
- [x] Log tailing works for both Planner and Worker agents
- [x] LiveLogPanel displays streaming logs
- [x] Board updates in real-time without polling (when SSE connected)
- [x] Polling fallback works on disconnect
- [x] Reconnection restores real-time updates

## Decisions Made

| Question | Decision |
|----------|----------|
| Log streaming approach | Use LogTailer (file tailing) rather than modifying Executor to emit events directly. This keeps concerns separated and handles reconnection scenarios. |
| Log subscription mechanism | Reconnect with updated query params rather than bidirectional channel. Simpler, spec-compliant. |
| Frontend state management | Use React Context + TanStack Query cache updates, no additional state library needed. |
| Polling fallback | Keep existing polling hooks but conditionally disable when SSE connected. |
| Planner agent runs | Added `task_id` column to `agent_runs` table (migration 002) since Planner runs at task level before subtasks exist. `subtask_id` made nullable. |
| SSE middleware | SSE endpoint must be outside timeout middleware group. Logger middleware must implement `http.Flusher`. |
