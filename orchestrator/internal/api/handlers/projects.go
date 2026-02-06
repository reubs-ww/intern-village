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

// ProjectHandler handles project-related HTTP requests.
type ProjectHandler struct {
	projectService *service.ProjectService
	authService    *service.AuthService
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(projectService *service.ProjectService, authService *service.AuthService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		authService:    authService,
	}
}

// ProjectResponse represents a project in API responses.
type ProjectResponse struct {
	ID            string `json:"id"`
	GitHubOwner   string `json:"github_owner"`
	GitHubRepo    string `json:"github_repo"`
	IsFork        bool   `json:"is_fork"`
	DefaultBranch string `json:"default_branch"`
	CreatedAt     string `json:"created_at"`
}

// CreateProjectResponse includes additional info about the creation operation.
type CreateProjectResponse struct {
	ProjectResponse
	WasForked bool `json:"was_forked"` // True if a fork was created during this operation
}

// CreateProjectRequest represents the request body for creating a project.
type CreateProjectRequest struct {
	RepoURL string `json:"repo_url"`
}

// Create creates a new project.
// POST /api/projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse request body
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// Validate input
	if req.RepoURL == "" {
		response.BadRequest(w, "repo_url is required")
		return
	}

	// Decrypt user's GitHub token
	token, err := h.authService.DecryptUserToken(user)
	if err != nil {
		log.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to decrypt user token")
		response.InternalError(w, err)
		return
	}

	// Create the project
	project, err := h.projectService.CreateProject(ctx, service.CreateProjectInput{
		UserID:      user.ID,
		RepoURL:     req.RepoURL,
		GitHubToken: token,
	})
	if err != nil {
		log.Error().Err(err).
			Str("user_id", user.ID.String()).
			Str("repo_url", req.RepoURL).
			Msg("failed to create project")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("project_id", project.ID.String()).
		Str("repo", project.GitHubOwner+"/"+project.GitHubRepo).
		Bool("is_fork", project.IsFork).
		Msg("project created")

	response.Created(w, CreateProjectResponse{
		ProjectResponse: projectToResponse(project),
		WasForked:       project.IsFork,
	})
}

// List lists all projects for the authenticated user.
// GET /api/projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	projects, err := h.projectService.ListProjects(ctx, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to list projects")
		response.InternalError(w, err)
		return
	}

	// Convert to response format
	result := make([]ProjectResponse, len(projects))
	for i, p := range projects {
		result[i] = projectToResponse(p)
	}

	response.OK(w, result)
}

// Get retrieves a project by ID.
// GET /api/projects/{id}
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse project ID from URL
	projectIDStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.BadRequest(w, "invalid project ID")
		return
	}

	project, err := h.projectService.GetProject(ctx, projectID, userID)
	if err != nil {
		response.ErrorFromDomain(w, err)
		return
	}

	response.OK(w, projectToResponse(project))
}

// Delete deletes a project.
// DELETE /api/projects/{id}
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse project ID from URL
	projectIDStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.BadRequest(w, "invalid project ID")
		return
	}

	if err := h.projectService.DeleteProject(ctx, projectID, userID); err != nil {
		log.Error().Err(err).
			Str("project_id", projectID.String()).
			Str("user_id", userID.String()).
			Msg("failed to delete project")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("project_id", projectID.String()).
		Msg("project deleted")

	response.NoContent(w)
}

// Cleanup removes the clone directory for a project without deleting the project record.
// POST /api/projects/{id}/cleanup
func (h *ProjectHandler) Cleanup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Parse project ID from URL
	projectIDStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.BadRequest(w, "invalid project ID")
		return
	}

	if err := h.projectService.CleanupProject(ctx, projectID, userID); err != nil {
		log.Error().Err(err).
			Str("project_id", projectID.String()).
			Str("user_id", userID.String()).
			Msg("failed to cleanup project")
		response.ErrorFromDomain(w, err)
		return
	}

	log.Info().
		Str("project_id", projectID.String()).
		Msg("project cleaned up")

	response.OK(w, map[string]string{"message": "project cleaned up successfully"})
}

// projectToResponse converts a domain.Project to a ProjectResponse.
func projectToResponse(p *domain.Project) ProjectResponse {
	return ProjectResponse{
		ID:            p.ID.String(),
		GitHubOwner:   p.GitHubOwner,
		GitHubRepo:    p.GitHubRepo,
		IsFork:        p.IsFork,
		DefaultBranch: p.DefaultBranch,
		CreatedAt:     p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
