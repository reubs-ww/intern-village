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

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	taskService *service.TaskService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

// TaskResponse represents a task in API responses.
type TaskResponse struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	BeadsEpicID *string `json:"beads_epic_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CreateTaskRequest represents the request body for creating a task.
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Create creates a new task.
// POST /api/projects/{project_id}/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	// Parse request body
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// Validate input
	if req.Title == "" {
		response.BadRequest(w, "title is required")
		return
	}
	if req.Description == "" {
		response.BadRequest(w, "description is required")
		return
	}

	// Create the task
	task, err := h.taskService.CreateTask(ctx, service.CreateTaskInput{
		ProjectID:   projectID,
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID.String()).
			Str("project_id", projectID.String()).
			Msg("failed to create task")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("task_id", task.ID.String()).
		Str("project_id", projectID.String()).
		Str("title", task.Title).
		Msg("task created")

	response.Created(w, taskToResponse(task))
}

// List lists all tasks for a project.
// GET /api/projects/{project_id}/tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
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

	tasks, err := h.taskService.ListTasks(ctx, projectID, userID)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID.String()).
			Str("project_id", projectID.String()).
			Msg("failed to list tasks")
		response.ErrorFromDomain(w, err)
		return
	}

	// Convert to response format
	result := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = taskToResponse(t)
	}

	response.OK(w, result)
}

// Get retrieves a task by ID.
// GET /api/tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse task ID from URL
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.BadRequest(w, "invalid task ID")
		return
	}

	task, err := h.taskService.GetTask(ctx, taskID, userID)
	if err != nil {
		response.ErrorFromDomain(w, err)
		return
	}

	response.OK(w, taskToResponse(task))
}

// Delete deletes a task.
// DELETE /api/tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse task ID from URL
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.BadRequest(w, "invalid task ID")
		return
	}

	if err := h.taskService.DeleteTask(ctx, taskID, userID); err != nil {
		log.Error().Err(err).
			Str("task_id", taskID.String()).
			Str("user_id", userID.String()).
			Msg("failed to delete task")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("task_id", taskID.String()).
		Msg("task deleted")

	response.NoContent(w)
}

// RetryPlanning retries a failed planning task.
// POST /api/tasks/{id}/retry-planning
func (h *TaskHandler) RetryPlanning(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse task ID from URL
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.BadRequest(w, "invalid task ID")
		return
	}

	task, err := h.taskService.RetryPlanning(ctx, taskID, userID)
	if err != nil {
		log.Error().Err(err).
			Str("task_id", taskID.String()).
			Str("user_id", userID.String()).
			Msg("failed to retry planning")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("task_id", taskID.String()).
		Msg("task planning retry initiated")

	response.OK(w, taskToResponse(task))
}

// taskToResponse converts a domain.Task to a TaskResponse.
func taskToResponse(t *domain.Task) TaskResponse {
	return TaskResponse{
		ID:          t.ID.String(),
		ProjectID:   t.ProjectID.String(),
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		BeadsEpicID: t.BeadsEpicID,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
