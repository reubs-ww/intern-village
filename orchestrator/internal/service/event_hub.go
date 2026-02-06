// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/intern-village/orchestrator/internal/domain"
)

// Event represents a real-time event that can be sent to clients.
type Event struct {
	Type string      `json:"event"`
	Data interface{} `json:"data"`
}

// MarshalData returns the event data as JSON bytes.
func (e Event) MarshalData() ([]byte, error) {
	return json.Marshal(e.Data)
}

// AgentStartedData is the data for an agent:started event.
type AgentStartedData struct {
	RunID         uuid.UUID `json:"run_id"`
	AgentType     string    `json:"agent_type"`
	TaskID        uuid.UUID `json:"task_id"`
	SubtaskID     *string   `json:"subtask_id"`
	AttemptNumber int       `json:"attempt_number"`
	StartedAt     time.Time `json:"started_at"`
}

// AgentLogData is the data for an agent:log event.
type AgentLogData struct {
	RunID      uuid.UUID `json:"run_id"`
	Line       string    `json:"line"`
	LineNumber int       `json:"line_number"`
	Timestamp  string    `json:"timestamp"`
}

// AgentCompletedData is the data for an agent:completed event.
type AgentCompletedData struct {
	RunID      uuid.UUID `json:"run_id"`
	AgentType  string    `json:"agent_type"`
	TaskID     uuid.UUID `json:"task_id"`
	SubtaskID  *string   `json:"subtask_id"`
	DurationMs int64     `json:"duration_ms"`
	TokenUsage int       `json:"token_usage"`
	PRUrl      string    `json:"pr_url,omitempty"`
}

// AgentFailedData is the data for an agent:failed event.
type AgentFailedData struct {
	RunID         uuid.UUID  `json:"run_id"`
	AgentType     string     `json:"agent_type"`
	TaskID        uuid.UUID  `json:"task_id"`
	SubtaskID     *string    `json:"subtask_id"`
	AttemptNumber int        `json:"attempt_number"`
	Error         string     `json:"error"`
	WillRetry     bool       `json:"will_retry"`
	NextAttemptAt *time.Time `json:"next_attempt_at,omitempty"`
}

// TaskStatusChangedData is the data for a task:status_changed event.
type TaskStatusChangedData struct {
	TaskID    uuid.UUID `json:"task_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	ChangedAt time.Time `json:"changed_at"`
}

// SubtaskStatusChangedData is the data for a subtask:status_changed event.
type SubtaskStatusChangedData struct {
	SubtaskID     uuid.UUID `json:"subtask_id"`
	TaskID        uuid.UUID `json:"task_id"`
	OldStatus     string    `json:"old_status"`
	NewStatus     string    `json:"new_status"`
	BlockedReason *string   `json:"blocked_reason"`
	PRUrl         *string   `json:"pr_url,omitempty"`
	PRNumber      *int      `json:"pr_number,omitempty"`
	ChangedAt     time.Time `json:"changed_at"`
}

// SubtaskUnblockedData is the data for a subtask:unblocked event.
type SubtaskUnblockedData struct {
	SubtaskID   uuid.UUID `json:"subtask_id"`
	TaskID      uuid.UUID `json:"task_id"`
	UnblockedBy uuid.UUID `json:"unblocked_by"`
	ChangedAt   time.Time `json:"changed_at"`
}

// ConnectedData is the data for a connected event.
type ConnectedData struct {
	ProjectID    uuid.UUID   `json:"project_id"`
	ActiveAgents []ActiveRun `json:"active_agents"`
}

// ActiveRun represents a currently running agent.
type ActiveRun struct {
	RunID     uuid.UUID  `json:"run_id"`
	AgentType string     `json:"agent_type"`
	TaskID    uuid.UUID  `json:"task_id"`
	SubtaskID *uuid.UUID `json:"subtask_id"`
	StartedAt time.Time  `json:"started_at"`
}

// HeartbeatData is the data for a heartbeat event.
type HeartbeatData struct {
	Timestamp time.Time `json:"timestamp"`
}

// ErrorData is the data for an error event.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	RunID   string `json:"run_id,omitempty"`
}

// Event type constants.
const (
	EventTypeAgentStarted         = "agent:started"
	EventTypeAgentLog             = "agent:log"
	EventTypeAgentCompleted       = "agent:completed"
	EventTypeAgentFailed          = "agent:failed"
	EventTypeTaskStatusChanged    = "task:status_changed"
	EventTypeSubtaskStatusChanged = "subtask:status_changed"
	EventTypeSubtaskUnblocked     = "subtask:unblocked"
	EventTypeConnected            = "connected"
	EventTypeHeartbeat            = "heartbeat"
	EventTypeError                = "error"
)

// EventHub is the central event distribution system for real-time events.
type EventHub interface {
	// Subscribe creates a new subscription for a project.
	// Returns an event channel and a cleanup function.
	// The cleanup function should be called when the client disconnects.
	Subscribe(projectID, userID uuid.UUID, logSubscriptions []uuid.UUID) (string, <-chan Event, func())

	// UpdateLogSubscriptions updates which agent runs a connection wants logs for.
	UpdateLogSubscriptions(connID string, runIDs []uuid.UUID)

	// ConnectionCount returns the number of active connections for a project.
	ConnectionCount(projectID uuid.UUID) int

	// UserConnectionCount returns the number of active connections for a user.
	UserConnectionCount(userID uuid.UUID) int

	// Publishing methods
	PublishAgentStarted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID)
	PublishAgentLog(projectID, runID uuid.UUID, line string, lineNumber int, timestamp string)
	PublishAgentCompleted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, prURL string)
	PublishAgentFailed(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, errMsg string, willRetry bool, nextAttemptAt *time.Time)
	PublishTaskStatusChanged(projectID, taskID uuid.UUID, oldStatus, newStatus string)
	PublishSubtaskStatusChanged(projectID uuid.UUID, subtask *domain.Subtask, oldStatus string)
	PublishSubtaskUnblocked(projectID, taskID, subtaskID, unblockedByID uuid.UUID)
}

// connection represents a single SSE connection.
type connection struct {
	id               string
	userID           uuid.UUID
	eventChan        chan Event
	logSubscriptions map[uuid.UUID]bool
	mu               sync.RWMutex
}

// eventHub is the default implementation of EventHub.
type eventHub struct {
	mu          sync.RWMutex
	connections map[uuid.UUID]map[string]*connection // projectID -> connID -> connection
	bufferSize  int
	logger      *slog.Logger
}

// NewEventHub creates a new EventHub.
func NewEventHub(bufferSize int, logger *slog.Logger) EventHub {
	if bufferSize <= 0 {
		bufferSize = 100 // default buffer size
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &eventHub{
		connections: make(map[uuid.UUID]map[string]*connection),
		bufferSize:  bufferSize,
		logger:      logger,
	}
}

// Subscribe creates a new subscription for a project.
func (h *eventHub) Subscribe(projectID, userID uuid.UUID, logSubscriptions []uuid.UUID) (string, <-chan Event, func()) {
	connID := uuid.New().String()

	conn := &connection{
		id:               connID,
		userID:           userID,
		eventChan:        make(chan Event, h.bufferSize),
		logSubscriptions: make(map[uuid.UUID]bool),
	}

	// Set initial log subscriptions
	for _, runID := range logSubscriptions {
		conn.logSubscriptions[runID] = true
	}

	h.mu.Lock()
	if h.connections[projectID] == nil {
		h.connections[projectID] = make(map[string]*connection)
	}
	h.connections[projectID][connID] = conn
	h.mu.Unlock()

	h.logger.Debug("client subscribed",
		"project_id", projectID,
		"user_id", userID,
		"conn_id", connID,
	)

	// Cleanup function
	cleanup := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if projectConns, ok := h.connections[projectID]; ok {
			if conn, exists := projectConns[connID]; exists {
				close(conn.eventChan)
				delete(projectConns, connID)
			}
			// Remove project entry if no more connections
			if len(projectConns) == 0 {
				delete(h.connections, projectID)
			}
		}

		h.logger.Debug("client unsubscribed",
			"project_id", projectID,
			"conn_id", connID,
		)
	}

	return connID, conn.eventChan, cleanup
}

// UpdateLogSubscriptions updates which agent runs a connection wants logs for.
func (h *eventHub) UpdateLogSubscriptions(connID string, runIDs []uuid.UUID) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Find the connection
	for _, projectConns := range h.connections {
		if conn, ok := projectConns[connID]; ok {
			conn.mu.Lock()
			conn.logSubscriptions = make(map[uuid.UUID]bool)
			for _, runID := range runIDs {
				conn.logSubscriptions[runID] = true
			}
			conn.mu.Unlock()
			return
		}
	}
}

// ConnectionCount returns the number of active connections for a project.
func (h *eventHub) ConnectionCount(projectID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if projectConns, ok := h.connections[projectID]; ok {
		return len(projectConns)
	}
	return 0
}

// UserConnectionCount returns the number of active connections for a user.
func (h *eventHub) UserConnectionCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, projectConns := range h.connections {
		for _, conn := range projectConns {
			if conn.userID == userID {
				count++
			}
		}
	}
	return count
}

// broadcast sends an event to all connections for a project.
func (h *eventHub) broadcast(projectID uuid.UUID, event Event, runID *uuid.UUID) {
	h.mu.RLock()
	projectConns, ok := h.connections[projectID]
	if !ok {
		h.mu.RUnlock()
		return
	}

	// Make a copy of connections to avoid holding lock during send
	conns := make([]*connection, 0, len(projectConns))
	for _, conn := range projectConns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		// For log events, check if the connection is subscribed
		if event.Type == EventTypeAgentLog && runID != nil {
			conn.mu.RLock()
			subscribed := conn.logSubscriptions[*runID]
			conn.mu.RUnlock()
			if !subscribed {
				continue
			}
		}

		// Non-blocking send
		select {
		case conn.eventChan <- event:
			// Event sent successfully
		default:
			// Channel full, drop event and log warning
			h.logger.Warn("event channel full, dropping event",
				"conn_id", conn.id,
				"event_type", event.Type,
			)
		}
	}
}

// PublishAgentStarted publishes an agent:started event.
func (h *eventHub) PublishAgentStarted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID) {
	var subtaskID *string
	if run.AgentType == domain.AgentTypeWorker {
		s := run.SubtaskID.String()
		subtaskID = &s
	}

	event := Event{
		Type: EventTypeAgentStarted,
		Data: AgentStartedData{
			RunID:         run.ID,
			AgentType:     string(run.AgentType),
			TaskID:        taskID,
			SubtaskID:     subtaskID,
			AttemptNumber: run.AttemptNumber,
			StartedAt:     run.StartedAt,
		},
	}

	h.broadcast(projectID, event, nil)

	h.logger.Debug("published agent:started",
		"project_id", projectID,
		"run_id", run.ID,
		"agent_type", run.AgentType,
	)
}

// PublishAgentLog publishes an agent:log event.
func (h *eventHub) PublishAgentLog(projectID, runID uuid.UUID, line string, lineNumber int, timestamp string) {
	event := Event{
		Type: EventTypeAgentLog,
		Data: AgentLogData{
			RunID:      runID,
			Line:       line,
			LineNumber: lineNumber,
			Timestamp:  timestamp,
		},
	}

	h.broadcast(projectID, event, &runID)
}

// PublishAgentCompleted publishes an agent:completed event.
func (h *eventHub) PublishAgentCompleted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, prURL string) {
	var subtaskID *string
	if run.AgentType == domain.AgentTypeWorker {
		s := run.SubtaskID.String()
		subtaskID = &s
	}

	durationMs := int64(0)
	if run.EndedAt != nil {
		durationMs = run.EndedAt.Sub(run.StartedAt).Milliseconds()
	}

	tokenUsage := 0
	if run.TokenUsage != nil {
		tokenUsage = *run.TokenUsage
	}

	event := Event{
		Type: EventTypeAgentCompleted,
		Data: AgentCompletedData{
			RunID:      run.ID,
			AgentType:  string(run.AgentType),
			TaskID:     taskID,
			SubtaskID:  subtaskID,
			DurationMs: durationMs,
			TokenUsage: tokenUsage,
			PRUrl:      prURL,
		},
	}

	h.broadcast(projectID, event, nil)

	h.logger.Debug("published agent:completed",
		"project_id", projectID,
		"run_id", run.ID,
		"duration_ms", durationMs,
	)
}

// PublishAgentFailed publishes an agent:failed event.
func (h *eventHub) PublishAgentFailed(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, errMsg string, willRetry bool, nextAttemptAt *time.Time) {
	var subtaskID *string
	if run.AgentType == domain.AgentTypeWorker {
		s := run.SubtaskID.String()
		subtaskID = &s
	}

	event := Event{
		Type: EventTypeAgentFailed,
		Data: AgentFailedData{
			RunID:         run.ID,
			AgentType:     string(run.AgentType),
			TaskID:        taskID,
			SubtaskID:     subtaskID,
			AttemptNumber: run.AttemptNumber,
			Error:         errMsg,
			WillRetry:     willRetry,
			NextAttemptAt: nextAttemptAt,
		},
	}

	h.broadcast(projectID, event, nil)

	h.logger.Debug("published agent:failed",
		"project_id", projectID,
		"run_id", run.ID,
		"will_retry", willRetry,
	)
}

// PublishTaskStatusChanged publishes a task:status_changed event.
func (h *eventHub) PublishTaskStatusChanged(projectID, taskID uuid.UUID, oldStatus, newStatus string) {
	event := Event{
		Type: EventTypeTaskStatusChanged,
		Data: TaskStatusChangedData{
			TaskID:    taskID,
			OldStatus: oldStatus,
			NewStatus: newStatus,
			ChangedAt: time.Now(),
		},
	}

	h.broadcast(projectID, event, nil)

	h.logger.Debug("published task:status_changed",
		"project_id", projectID,
		"task_id", taskID,
		"old_status", oldStatus,
		"new_status", newStatus,
	)
}

// PublishSubtaskStatusChanged publishes a subtask:status_changed event.
func (h *eventHub) PublishSubtaskStatusChanged(projectID uuid.UUID, subtask *domain.Subtask, oldStatus string) {
	var blockedReason *string
	if subtask.BlockedReason != nil {
		s := string(*subtask.BlockedReason)
		blockedReason = &s
	}

	event := Event{
		Type: EventTypeSubtaskStatusChanged,
		Data: SubtaskStatusChangedData{
			SubtaskID:     subtask.ID,
			TaskID:        subtask.TaskID,
			OldStatus:     oldStatus,
			NewStatus:     string(subtask.Status),
			BlockedReason: blockedReason,
			PRUrl:         subtask.PRUrl,
			PRNumber:      subtask.PRNumber,
			ChangedAt:     time.Now(),
		},
	}

	h.broadcast(projectID, event, nil)

	h.logger.Debug("published subtask:status_changed",
		"project_id", projectID,
		"subtask_id", subtask.ID,
		"old_status", oldStatus,
		"new_status", subtask.Status,
	)
}

// PublishSubtaskUnblocked publishes a subtask:unblocked event.
func (h *eventHub) PublishSubtaskUnblocked(projectID, taskID, subtaskID, unblockedByID uuid.UUID) {
	event := Event{
		Type: EventTypeSubtaskUnblocked,
		Data: SubtaskUnblockedData{
			SubtaskID:   subtaskID,
			TaskID:      taskID,
			UnblockedBy: unblockedByID,
			ChangedAt:   time.Now(),
		},
	}

	h.broadcast(projectID, event, nil)

	h.logger.Debug("published subtask:unblocked",
		"project_id", projectID,
		"subtask_id", subtaskID,
		"unblocked_by", unblockedByID,
	)
}
