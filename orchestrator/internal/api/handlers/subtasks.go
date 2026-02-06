// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/api/middleware"
	"github.com/intern-village/orchestrator/internal/api/response"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/service"
)

// SubtaskHandler handles subtask-related HTTP requests.
type SubtaskHandler struct {
	subtaskService *service.SubtaskService
}

// NewSubtaskHandler creates a new SubtaskHandler.
func NewSubtaskHandler(subtaskService *service.SubtaskService) *SubtaskHandler {
	return &SubtaskHandler{
		subtaskService: subtaskService,
	}
}

// SubtaskResponse represents a subtask in API responses.
type SubtaskResponse struct {
	ID                 string  `json:"id"`
	TaskID             string  `json:"task_id"`
	Title              string  `json:"title"`
	Spec               *string `json:"spec,omitempty"`
	ImplementationPlan *string `json:"implementation_plan,omitempty"`
	Status             string  `json:"status"`
	BlockedReason      *string `json:"blocked_reason,omitempty"`
	BranchName         *string `json:"branch_name,omitempty"`
	PRUrl              *string `json:"pr_url,omitempty"`
	PRNumber           *int    `json:"pr_number,omitempty"`
	RetryCount         int     `json:"retry_count"`
	TokenUsage         int     `json:"token_usage"`
	Position           int     `json:"position"`
	BeadsIssueID       *string `json:"beads_issue_id,omitempty"`
	WorktreePath       *string `json:"worktree_path,omitempty"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// UpdatePositionRequest represents the request body for updating subtask position.
type UpdatePositionRequest struct {
	Position int `json:"position"`
}

// List lists all subtasks for a task.
// GET /api/tasks/{task_id}/subtasks
func (h *SubtaskHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse task ID from URL
	taskIDStr := chi.URLParam(r, "task_id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.BadRequest(w, "invalid task ID")
		return
	}

	subtasks, err := h.subtaskService.ListSubtasks(ctx, taskID, userID)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID.String()).
			Str("task_id", taskID.String()).
			Msg("failed to list subtasks")
		response.ErrorFromDomain(w, err)
		return
	}

	// Convert to response format
	result := make([]SubtaskResponse, len(subtasks))
	for i, s := range subtasks {
		result[i] = subtaskToResponse(s)
	}

	response.OK(w, result)
}

// Get retrieves a subtask by ID.
// GET /api/subtasks/{id}
func (h *SubtaskHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	subtask, err := h.subtaskService.GetSubtask(ctx, subtaskID, userID)
	if err != nil {
		response.ErrorFromDomain(w, err)
		return
	}

	response.OK(w, subtaskToResponse(subtask))
}

// Start starts a subtask by spawning the Worker agent.
// POST /api/subtasks/{id}/start
func (h *SubtaskHandler) Start(w http.ResponseWriter, r *http.Request) {
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

	subtask, err := h.subtaskService.StartSubtask(ctx, subtaskID, userID)
	if err != nil {
		log.Error().Err(err).
			Str("subtask_id", subtaskID.String()).
			Str("user_id", userID.String()).
			Msg("failed to start subtask")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("subtask_id", subtaskID.String()).
		Str("branch_name", ptrStr(subtask.BranchName)).
		Msg("subtask started")

	response.OK(w, subtaskToResponse(subtask))
}

// MarkMerged marks a subtask as merged.
// POST /api/subtasks/{id}/mark-merged
func (h *SubtaskHandler) MarkMerged(w http.ResponseWriter, r *http.Request) {
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

	subtask, err := h.subtaskService.MarkMerged(ctx, subtaskID, userID)
	if err != nil {
		log.Error().Err(err).
			Str("subtask_id", subtaskID.String()).
			Str("user_id", userID.String()).
			Msg("failed to mark subtask as merged")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("subtask_id", subtaskID.String()).
		Msg("subtask marked as merged")

	response.OK(w, subtaskToResponse(subtask))
}

// Retry retries a failed subtask.
// POST /api/subtasks/{id}/retry
func (h *SubtaskHandler) Retry(w http.ResponseWriter, r *http.Request) {
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

	subtask, err := h.subtaskService.RetrySubtask(ctx, subtaskID, userID)
	if err != nil {
		log.Error().Err(err).
			Str("subtask_id", subtaskID.String()).
			Str("user_id", userID.String()).
			Msg("failed to retry subtask")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("subtask_id", subtaskID.String()).
		Msg("subtask retry initiated")

	response.OK(w, subtaskToResponse(subtask))
}

// UpdatePosition updates the position of a subtask.
// PATCH /api/subtasks/{id}/position
func (h *SubtaskHandler) UpdatePosition(w http.ResponseWriter, r *http.Request) {
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

	// Parse request body
	var req UpdatePositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// Validate position
	if req.Position < 0 {
		response.BadRequest(w, "position must be non-negative")
		return
	}

	subtask, err := h.subtaskService.UpdatePosition(ctx, subtaskID, userID, req.Position)
	if err != nil {
		log.Error().Err(err).
			Str("subtask_id", subtaskID.String()).
			Int("position", req.Position).
			Msg("failed to update subtask position")
		response.ErrorFromDomain(w, err)
		return
	}

	response.OK(w, subtaskToResponse(subtask))
}

// subtaskToResponse converts a domain.Subtask to a SubtaskResponse.
func subtaskToResponse(s *domain.Subtask) SubtaskResponse {
	var blockedReason *string
	if s.BlockedReason != nil {
		r := string(*s.BlockedReason)
		blockedReason = &r
	}

	return SubtaskResponse{
		ID:                 s.ID.String(),
		TaskID:             s.TaskID.String(),
		Title:              s.Title,
		Spec:               s.Spec,
		ImplementationPlan: s.ImplementationPlan,
		Status:             string(s.Status),
		BlockedReason:      blockedReason,
		BranchName:         s.BranchName,
		PRUrl:              s.PRUrl,
		PRNumber:           s.PRNumber,
		RetryCount:         s.RetryCount,
		TokenUsage:         s.TokenUsage,
		Position:           s.Position,
		BeadsIssueID:       s.BeadsIssueID,
		WorktreePath:       s.WorktreePath,
		CreatedAt:          s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ptrStr safely returns the value of a string pointer or empty string.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
