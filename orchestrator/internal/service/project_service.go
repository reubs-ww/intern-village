// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/intern-village/orchestrator/generated/db"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
)

// Project service errors.
var (
	ErrProjectAlreadyExists = errors.New("project already exists")
	ErrProjectNotFound      = errors.New("project not found")
	ErrProjectAccessDenied  = errors.New("access denied to project")
	ErrClonePathExists      = errors.New("clone path already exists")
)

// ProjectService handles project management operations.
type ProjectService struct {
	repo          *repository.Repository
	crypto        *repository.Crypto
	githubService *GitHubService
	beadsService  *BeadsService
	dataDir       string
}

// NewProjectService creates a new ProjectService.
func NewProjectService(
	repo *repository.Repository,
	crypto *repository.Crypto,
	githubService *GitHubService,
	beadsService *BeadsService,
	dataDir string,
) *ProjectService {
	return &ProjectService{
		repo:          repo,
		crypto:        crypto,
		githubService: githubService,
		beadsService:  beadsService,
		dataDir:       dataDir,
	}
}

// CreateProjectInput contains the input for creating a project.
type CreateProjectInput struct {
	UserID      uuid.UUID
	RepoURL     string
	GitHubToken string // Decrypted token
}

// CreateProject creates a new project by cloning a GitHub repository.
// If the user doesn't have push access, the repo is forked first.
func (s *ProjectService) CreateProject(ctx context.Context, input CreateProjectInput) (*domain.Project, error) {
	// Parse the repository URL
	owner, repo, err := s.githubService.ParseRepoURL(input.RepoURL)
	if err != nil {
		return nil, err
	}

	// Check if project already exists for this user
	_, err = s.repo.GetProjectByOwnerRepo(ctx, db.GetProjectByOwnerRepoParams{
		UserID:      input.UserID,
		GithubOwner: owner,
		GithubRepo:  repo,
	})
	if err == nil {
		return nil, domain.NewConflictError("project", "already exists for this repository")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing project: %w", err)
	}

	// Get repository info and check access
	repoInfo, err := s.githubService.GetRepoInfo(ctx, owner, repo, input.GitHubToken)
	if err != nil {
		return nil, err
	}

	// Determine if we need to fork
	isFork := false
	actualOwner := owner
	actualRepo := repo
	var upstreamOwner, upstreamRepo *string // Original repo info for forks

	if !repoInfo.HasPushAccess {
		// Fork the repository
		forkInfo, err := s.githubService.ForkRepo(ctx, owner, repo, input.GitHubToken)
		if err != nil {
			return nil, err
		}
		isFork = true
		actualOwner = forkInfo.Owner
		actualRepo = forkInfo.Repo
		// Store original repo info for syncing
		upstreamOwner = &owner
		upstreamRepo = &repo
	}

	// Generate paths
	clonePath := s.generateClonePath(input.UserID, actualOwner, actualRepo)
	beadsPrefix := s.generateBeadsPrefix()

	// Check if clone path already exists
	if _, err := os.Stat(clonePath); err == nil {
		return nil, domain.NewConflictError("project", "clone directory already exists")
	}

	// Clone the repository
	if err := s.githubService.CloneRepo(ctx, actualOwner, actualRepo, input.GitHubToken, clonePath); err != nil {
		return nil, err
	}

	// If this is a fork, add upstream remote for syncing
	if isFork && upstreamOwner != nil && upstreamRepo != nil {
		if err := s.githubService.AddUpstreamRemote(ctx, clonePath, *upstreamOwner, *upstreamRepo); err != nil {
			// Cleanup the clone on failure
			_ = os.RemoveAll(clonePath)
			return nil, err
		}
	}

	// Initialize beads in the cloned repo
	if err := s.beadsService.Init(ctx, clonePath, beadsPrefix); err != nil {
		// Cleanup the clone on failure
		_ = os.RemoveAll(clonePath)
		return nil, err
	}

	// Create the project record
	dbProject, err := s.repo.CreateProject(ctx, db.CreateProjectParams{
		UserID:        input.UserID,
		GithubOwner:   actualOwner,
		GithubRepo:    actualRepo,
		IsFork:        isFork,
		UpstreamOwner: upstreamOwner,
		UpstreamRepo:  upstreamRepo,
		DefaultBranch: repoInfo.DefaultBranch,
		ClonePath:     clonePath,
		BeadsPrefix:   beadsPrefix,
	})
	if err != nil {
		// Cleanup the clone on failure
		_ = os.RemoveAll(clonePath)
		return nil, fmt.Errorf("failed to create project record: %w", err)
	}

	return dbProjectToDomain(dbProject), nil
}

// GetProject retrieves a project by ID with ownership verification.
func (s *ProjectService) GetProject(ctx context.Context, projectID, userID uuid.UUID) (*domain.Project, error) {
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("project", projectID.String())
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if project.UserID != userID {
		return nil, domain.NewForbiddenError("project", "access denied")
	}

	return dbProjectToDomain(project), nil
}

// ListProjects lists all projects for a user.
func (s *ProjectService) ListProjects(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	projects, err := s.repo.ListProjectsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	result := make([]*domain.Project, len(projects))
	for i, p := range projects {
		result[i] = dbProjectToDomain(p)
	}

	return result, nil
}

// DeleteProject deletes a project and its clone.
func (s *ProjectService) DeleteProject(ctx context.Context, projectID, userID uuid.UUID) error {
	// Get project with ownership check
	project, err := s.GetProject(ctx, projectID, userID)
	if err != nil {
		return err
	}

	// Delete the clone directory (ignore errors - directory might not exist)
	if project.ClonePath != "" {
		_ = os.RemoveAll(project.ClonePath)
	}

	// Delete the project record
	if err := s.repo.DeleteProject(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// CleanupProject removes the clone directory without deleting the project record.
// This is useful for manual cleanup of disk space.
func (s *ProjectService) CleanupProject(ctx context.Context, projectID, userID uuid.UUID) error {
	// Get project with ownership check
	project, err := s.GetProject(ctx, projectID, userID)
	if err != nil {
		return err
	}

	// Delete the clone directory
	if project.ClonePath != "" {
		if err := os.RemoveAll(project.ClonePath); err != nil {
			return fmt.Errorf("failed to cleanup clone: %w", err)
		}
	}

	return nil
}

// generateClonePath generates a path for cloning a repository.
// Format: {dataDir}/projects/{userID}/{owner}/{repo}
func (s *ProjectService) generateClonePath(userID uuid.UUID, owner, repo string) string {
	return filepath.Join(s.dataDir, "projects", userID.String(), owner, repo)
}

// generateBeadsPrefix generates a unique beads prefix for a project.
// Format: iv-{short_uuid}
func (s *ProjectService) generateBeadsPrefix() string {
	id := uuid.New()
	// Use first 8 chars of UUID for brevity
	shortID := strings.ReplaceAll(id.String()[:8], "-", "")
	return fmt.Sprintf("iv-%s", shortID)
}

// GetProjectByIDInternal retrieves a project by ID without ownership check.
// Only for internal use by other services.
func (s *ProjectService) GetProjectByIDInternal(ctx context.Context, projectID uuid.UUID) (*domain.Project, error) {
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("project", projectID.String())
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return dbProjectToDomain(project), nil
}

// GetProjectInternal is an alias for GetProjectByIDInternal for backwards compatibility.
func (s *ProjectService) GetProjectInternal(ctx context.Context, projectID uuid.UUID) (*domain.Project, error) {
	return s.GetProjectByIDInternal(ctx, projectID)
}

// CheckProjectOwnership verifies that the user owns the project.
// Returns nil if ownership is valid, or an error if not.
func (s *ProjectService) CheckProjectOwnership(projectID, userID uuid.UUID) error {
	ctx := context.Background()
	_, err := s.GetProject(ctx, projectID, userID)
	return err
}

// dbProjectToDomain converts a database Project to a domain Project.
func dbProjectToDomain(p db.Project) *domain.Project {
	return &domain.Project{
		ID:            p.ID,
		UserID:        p.UserID,
		GitHubOwner:   p.GithubOwner,
		GitHubRepo:    p.GithubRepo,
		IsFork:        p.IsFork,
		UpstreamOwner: p.UpstreamOwner,
		UpstreamRepo:  p.UpstreamRepo,
		DefaultBranch: p.DefaultBranch,
		ClonePath:     p.ClonePath,
		BeadsPrefix:   p.BeadsPrefix,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}
