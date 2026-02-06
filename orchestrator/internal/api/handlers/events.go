// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/api/middleware"
	"github.com/intern-village/orchestrator/internal/api/response"
	"github.com/intern-village/orchestrator/internal/config"
	"github.com/intern-village/orchestrator/internal/repository"
	"github.com/intern-village/orchestrator/internal/service"
)

// EventHandler handles SSE event streaming.
type EventHandler struct {
	eventHub       service.EventHub
	repo           *repository.Repository
	projectService ProjectOwnershipChecker
	cfg            *config.Config
}

// ProjectOwnershipChecker is an interface for checking project ownership.
type ProjectOwnershipChecker interface {
	CheckProjectOwnership(projectID, userID uuid.UUID) error
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(
	eventHub service.EventHub,
	repo *repository.Repository,
	projectService ProjectOwnershipChecker,
	cfg *config.Config,
) *EventHandler {
	return &EventHandler{
		eventHub:       eventHub,
		repo:           repo,
		projectService: projectService,
		cfg:            cfg,
	}
}

// ActiveRunResponse represents an active agent run.
type ActiveRunResponse struct {
	ID        string `json:"id"`
	SubtaskID string `json:"subtask_id"`
	TaskID    string `json:"task_id"`
	AgentType string `json:"agent_type"`
	Status    string `json:"status"`
	LogPath   string `json:"log_path"`
	StartedAt string `json:"started_at"`
}

// StreamEvents handles the SSE endpoint for project events.
// GET /api/projects/{project_id}/events
func (h *EventHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse project ID from URL
	projectIDStr := chi.URLParam(r, "project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.BadRequest(w, "invalid project ID")
		return
	}

	// Verify user owns the project
	if err := h.projectService.CheckProjectOwnership(projectID, userID); err != nil {
		response.Forbidden(w, "access denied")
		return
	}

	// Check max connections per user
	currentConnections := h.eventHub.UserConnectionCount(userID)
	if currentConnections >= h.cfg.SSEMaxConnectionsPerUser {
		response.Error(w, http.StatusTooManyRequests, "TOO_MANY_CONNECTIONS",
			fmt.Sprintf("maximum %d connections per user", h.cfg.SSEMaxConnectionsPerUser))
		return
	}

	// Parse log subscriptions from query param
	subscribeLogsStr := r.URL.Query().Get("subscribe_logs")
	var logSubscriptions []uuid.UUID
	if subscribeLogsStr != "" && subscribeLogsStr != "all" {
		parts := strings.Split(subscribeLogsStr, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if runID, err := uuid.Parse(part); err == nil {
				logSubscriptions = append(logSubscriptions, runID)
			}
		}
	}
	// Note: "all" is handled specially by the event hub

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Subscribe to event hub
	connID, eventCh, cleanup := h.eventHub.Subscribe(projectID, userID, logSubscriptions)
	defer cleanup()

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		response.InternalError(w, fmt.Errorf("streaming not supported"))
		return
	}

	// Get active runs for this project
	activeRuns, err := h.getActiveRuns(projectID)
	if err != nil {
		log.Error().Err(err).Str("project_id", projectID.String()).Msg("failed to get active runs")
		activeRuns = []ActiveRunResponse{}
	}

	// Send connected event with active runs
	connectedData := map[string]interface{}{
		"connection_id": connID,
		"active_runs":   activeRuns,
	}
	if err := h.writeSSE(w, flusher, "connected", connectedData); err != nil {
		log.Error().Err(err).Msg("failed to send connected event")
		return
	}

	// Start heartbeat ticker
	heartbeatInterval := time.Duration(h.cfg.SSEHeartbeatIntervalS) * time.Second
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Connection timeout
	connectionTimeout := time.Duration(h.cfg.SSEConnectionTimeoutM) * time.Minute
	timeoutTimer := time.NewTimer(connectionTimeout)
	defer timeoutTimer.Stop()

	log.Info().
		Str("project_id", projectID.String()).
		Str("user_id", userID.String()).
		Str("conn_id", connID).
		Msg("SSE connection established")

	// Event loop
	for {
		select {
		case <-ctx.Done():
			log.Info().
				Str("conn_id", connID).
				Msg("SSE client disconnected")
			return

		case <-timeoutTimer.C:
			log.Info().
				Str("conn_id", connID).
				Msg("SSE connection timeout")
			return

		case <-heartbeatTicker.C:
			if err := h.writeSSE(w, flusher, "heartbeat", map[string]string{"time": time.Now().Format(time.RFC3339)}); err != nil {
				log.Debug().Err(err).Str("conn_id", connID).Msg("failed to send heartbeat, client disconnected")
				return
			}

		case event, ok := <-eventCh:
			if !ok {
				log.Info().
					Str("conn_id", connID).
					Msg("event channel closed")
				return
			}
			if err := h.writeSSE(w, flusher, event.Type, event.Data); err != nil {
				log.Debug().Err(err).Str("conn_id", connID).Msg("failed to send event, client disconnected")
				return
			}
		}
	}
}

// GetActiveRuns returns active agent runs for a project.
// GET /api/projects/{project_id}/active-runs
func (h *EventHandler) GetActiveRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse project ID from URL
	projectIDStr := chi.URLParam(r, "project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.BadRequest(w, "invalid project ID")
		return
	}

	// Verify user owns the project
	if err := h.projectService.CheckProjectOwnership(projectID, userID); err != nil {
		response.Forbidden(w, "access denied")
		return
	}

	// Get active runs
	activeRuns, err := h.getActiveRuns(projectID)
	if err != nil {
		log.Error().Err(err).Str("project_id", projectID.String()).Msg("failed to get active runs")
		response.InternalError(w, err)
		return
	}

	response.OK(w, activeRuns)
}

// getActiveRuns retrieves currently running agent runs for a project.
func (h *EventHandler) getActiveRuns(projectID uuid.UUID) ([]ActiveRunResponse, error) {
	ctx := context.Background()
	runs, err := h.repo.ListActiveAgentRunsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active runs: %w", err)
	}

	result := make([]ActiveRunResponse, len(runs))
	for i, run := range runs {
		result[i] = ActiveRunResponse{
			ID:        run.ID.String(),
			SubtaskID: run.SubtaskID.String(),
			TaskID:    run.TaskID.String(),
			AgentType: run.AgentType,
			Status:    run.Status,
			LogPath:   run.LogPath,
			StartedAt: run.StartedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// writeSSE writes an SSE event to the response writer.
func (h *EventHandler) writeSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Write SSE format
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	flusher.Flush()
	return nil
}
