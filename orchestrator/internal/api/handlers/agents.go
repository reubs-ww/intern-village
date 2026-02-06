// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/api/middleware"
	"github.com/intern-village/orchestrator/internal/api/response"
	"github.com/intern-village/orchestrator/internal/repository"
)

// AgentRunResponse represents an agent run in API responses.
type AgentRunResponse struct {
	ID            string  `json:"id"`
	SubtaskID     string  `json:"subtask_id"`
	AgentType     string  `json:"agent_type"`
	AttemptNumber int     `json:"attempt_number"`
	Status        string  `json:"status"`
	StartedAt     string  `json:"started_at"`
	EndedAt       *string `json:"ended_at,omitempty"`
	TokenUsage    *int    `json:"token_usage,omitempty"`
	ErrorMessage  *string `json:"error_message,omitempty"`
	LogPath       string  `json:"log_path"`
	CreatedAt     string  `json:"created_at"`
}

// AgentRunLogsResponse represents the logs for an agent run.
type AgentRunLogsResponse struct {
	RunID   string `json:"run_id"`
	LogPath string `json:"log_path"`
	Content string `json:"content"`
}

// SubtaskOwnershipChecker is an interface for checking subtask ownership.
type SubtaskOwnershipChecker interface {
	CheckSubtaskOwnership(ctx context.Context, subtaskID, userID uuid.UUID) error
}

// AgentHandler handles agent-related HTTP requests.
type AgentHandler struct {
	repo           *repository.Repository
	subtaskService SubtaskOwnershipChecker
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(repo *repository.Repository, subtaskService SubtaskOwnershipChecker) *AgentHandler {
	return &AgentHandler{
		repo:           repo,
		subtaskService: subtaskService,
	}
}

// ListRuns lists all agent runs for a subtask.
// GET /api/subtasks/{id}/runs
func (h *AgentHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse subtask ID from URL
	subtaskIDStr := chi.URLParam(r, "id")
	subtaskID, err := uuid.Parse(subtaskIDStr)
	if err != nil {
		response.BadRequest(w, "invalid subtask ID")
		return
	}

	// Verify user owns the subtask (via ownership check)
	if err := h.subtaskService.CheckSubtaskOwnership(ctx, subtaskID, userID); err != nil {
		response.ErrorFromDomain(w, err)
		return
	}

	// List agent runs
	runs, err := h.repo.ListAgentRunsBySubtask(ctx, pgtype.UUID{Bytes: subtaskID, Valid: true})
	if err != nil {
		log.Error().Err(err).
			Str("subtask_id", subtaskID.String()).
			Msg("failed to list agent runs")
		response.InternalError(w, fmt.Errorf("failed to list agent runs: %w", err))
		return
	}

	// Convert to response format
	result := make([]AgentRunResponse, len(runs))
	for i, run := range runs {
		subtaskIDStr := ""
		if run.SubtaskID.Valid {
			subtaskIDStr = uuid.UUID(run.SubtaskID.Bytes).String()
		}
		result[i] = AgentRunResponse{
			ID:            run.ID.String(),
			SubtaskID:     subtaskIDStr,
			AgentType:     run.AgentType,
			AttemptNumber: int(run.AttemptNumber),
			Status:        run.Status,
			StartedAt:     run.StartedAt.Format(time.RFC3339),
			LogPath:       run.LogPath,
			CreatedAt:     run.CreatedAt.Format(time.RFC3339),
		}

		if run.EndedAt.Valid {
			endedAt := run.EndedAt.Time.Format(time.RFC3339)
			result[i].EndedAt = &endedAt
		}

		if run.TokenUsage != nil {
			tokenUsage := int(*run.TokenUsage)
			result[i].TokenUsage = &tokenUsage
		}

		if run.ErrorMessage != nil {
			result[i].ErrorMessage = run.ErrorMessage
		}
	}

	response.OK(w, result)
}

// GetLogs gets the log content for an agent run.
// GET /api/runs/{id}/logs
func (h *AgentHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse run ID from URL
	runIDStr := chi.URLParam(r, "id")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		response.BadRequest(w, "invalid run ID")
		return
	}

	// Get the agent run
	run, err := h.repo.GetAgentRunByID(ctx, runID)
	if err != nil {
		log.Error().Err(err).
			Str("run_id", runID.String()).
			Msg("failed to get agent run")
		response.NotFound(w, "agent run not found")
		return
	}

	// Verify user owns the subtask (via ownership check)
	// Note: For Planner runs, SubtaskID may be nil - skip ownership check for those
	if run.SubtaskID.Valid {
		subtaskID := uuid.UUID(run.SubtaskID.Bytes)
		if err := h.subtaskService.CheckSubtaskOwnership(ctx, subtaskID, userID); err != nil {
			response.ErrorFromDomain(w, err)
			return
		}
	}

	// Read log file content
	content, err := os.ReadFile(run.LogPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Log file doesn't exist yet or has been cleaned up
			response.OK(w, AgentRunLogsResponse{
				RunID:   run.ID.String(),
				LogPath: run.LogPath,
				Content: "",
			})
			return
		}
		log.Error().Err(err).
			Str("run_id", runID.String()).
			Str("log_path", run.LogPath).
			Msg("failed to read log file")
		response.InternalError(w, fmt.Errorf("failed to read log file: %w", err))
		return
	}

	response.OK(w, AgentRunLogsResponse{
		RunID:   run.ID.String(),
		LogPath: run.LogPath,
		Content: string(content),
	})
}
