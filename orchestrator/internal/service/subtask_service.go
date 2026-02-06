// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/intern-village/orchestrator/generated/db"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
)

// WorkerSpawner is an interface for spawning worker agents.
type WorkerSpawner interface {
	SpawnWorker(ctx context.Context, subtask *domain.Subtask, project *domain.Project) error
	KillAgentsForSubtask(ctx context.Context, subtaskID uuid.UUID) error
}

// SubtaskService handles subtask management operations.
type SubtaskService struct {
	repo              *repository.Repository
	taskService       *TaskService
	dependencyService *DependencyService
	beadsService      *BeadsService
	projectService    *ProjectService
	githubService     *GitHubService
	workerSpawner     WorkerSpawner
	eventHub          EventHub
}

// NewSubtaskService creates a new SubtaskService.
func NewSubtaskService(
	repo *repository.Repository,
	taskService *TaskService,
	dependencyService *DependencyService,
	beadsService *BeadsService,
	projectService *ProjectService,
	githubService *GitHubService,
	eventHub EventHub,
) *SubtaskService {
	return &SubtaskService{
		repo:              repo,
		taskService:       taskService,
		dependencyService: dependencyService,
		beadsService:      beadsService,
		projectService:    projectService,
		githubService:     githubService,
		eventHub:          eventHub,
	}
}

// SetWorkerSpawner sets the worker spawner for the service.
func (s *SubtaskService) SetWorkerSpawner(spawner WorkerSpawner) {
	s.workerSpawner = spawner
}

// CreateSubtaskInput contains the input for creating a subtask.
type CreateSubtaskInput struct {
	TaskID             uuid.UUID
	Title              string
	Spec               *string
	ImplementationPlan *string
	BeadsIssueID       *string
}

// CreateSubtask creates a new subtask (called by sync service).
func (s *SubtaskService) CreateSubtask(ctx context.Context, input CreateSubtaskInput) (*domain.Subtask, error) {
	// Get next position
	position, err := s.repo.GetNextPosition(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next position: %w", err)
	}

	// Create the subtask record
	dbSubtask, err := s.repo.CreateSubtask(ctx, db.CreateSubtaskParams{
		TaskID:             input.TaskID,
		Title:              input.Title,
		Spec:               input.Spec,
		ImplementationPlan: input.ImplementationPlan,
		Status:             string(domain.SubtaskStatusPending),
		Position:           position,
		BeadsIssueID:       input.BeadsIssueID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subtask: %w", err)
	}

	return dbSubtaskToDomain(dbSubtask), nil
}

// GetSubtask retrieves a subtask by ID with ownership verification.
func (s *SubtaskService) GetSubtask(ctx context.Context, subtaskID, userID uuid.UUID) (*domain.Subtask, error) {
	subtask, err := s.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("subtask", subtaskID.String())
		}
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}

	// Verify ownership through task -> project chain
	_, err = s.taskService.GetTask(ctx, subtask.TaskID, userID)
	if err != nil {
		return nil, err
	}

	return dbSubtaskToDomain(subtask), nil
}

// CheckSubtaskOwnership verifies that the user owns the subtask (via task -> project chain).
// Returns nil if ownership is valid, or an error if not.
func (s *SubtaskService) CheckSubtaskOwnership(ctx context.Context, subtaskID, userID uuid.UUID) error {
	_, err := s.GetSubtask(ctx, subtaskID, userID)
	return err
}

// ListSubtasks lists all subtasks for a task.
func (s *SubtaskService) ListSubtasks(ctx context.Context, taskID, userID uuid.UUID) ([]*domain.Subtask, error) {
	// Verify task access
	_, err := s.taskService.GetTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	subtasks, err := s.repo.ListSubtasksByTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subtasks: %w", err)
	}

	result := make([]*domain.Subtask, len(subtasks))
	for i, st := range subtasks {
		result[i] = dbSubtaskToDomain(st)
	}

	return result, nil
}

// StartSubtask starts a subtask by creating a worktree and spawning the Worker agent.
func (s *SubtaskService) StartSubtask(ctx context.Context, subtaskID, userID uuid.UUID) (*domain.Subtask, error) {
	// Get subtask with ownership check
	subtask, err := s.GetSubtask(ctx, subtaskID, userID)
	if err != nil {
		return nil, err
	}

	// Validate current status
	if subtask.Status == domain.SubtaskStatusInProgress {
		return nil, domain.NewConflictError("subtask", "subtask is already in progress")
	}

	if subtask.Status == domain.SubtaskStatusBlocked {
		// Check the reason
		if subtask.BlockedReason != nil && *subtask.BlockedReason == domain.BlockedReasonDependency {
			return nil, domain.NewUnprocessableError("subtask", "subtask is blocked by dependencies")
		}
		// If blocked by FAILURE, it's a retry - allow it
	}

	if subtask.Status != domain.SubtaskStatusReady && subtask.Status != domain.SubtaskStatusBlocked {
		return nil, domain.NewUnprocessableError("subtask", fmt.Sprintf("cannot start subtask in %s status", subtask.Status))
	}

	// Validate transition
	if !domain.CanTransitionSubtask(subtask.Status, domain.SubtaskStatusInProgress) {
		return nil, domain.NewInvalidTransitionError(
			"subtask",
			string(subtask.Status),
			string(domain.SubtaskStatusInProgress),
			"invalid subtask state transition",
		)
	}

	// Get task and project for worktree creation
	task, err := s.taskService.GetTaskByIDInternal(ctx, subtask.TaskID)
	if err != nil {
		return nil, err
	}

	project, err := s.projectService.GetProject(ctx, task.ProjectID, userID)
	if err != nil {
		return nil, err
	}

	// Sync repository to latest before creating worktree (see §9.5 Repository Sync Strategy)
	if s.githubService != nil {
		if err := s.githubService.SyncRepoWithRetry(ctx, project.ClonePath, project.DefaultBranch, project.IsFork, 3); err != nil {
			return nil, fmt.Errorf("failed to sync repository before starting subtask: %w", err)
		}
	}

	// Generate branch name
	issueID := ""
	if subtask.BeadsIssueID != nil {
		issueID = *subtask.BeadsIssueID
	} else {
		// Fallback to subtask ID if no beads issue ID
		issueID = fmt.Sprintf("iv-%s", subtaskID.String()[:8])
	}
	branchName := s.beadsService.GenerateBranchName(issueID, subtask.Title)

	// Create worktree
	err = s.beadsService.CreateWorktree(ctx, project.ClonePath, subtaskID.String(), branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}
	worktreePath := fmt.Sprintf("%s/%s", project.ClonePath, subtaskID.String())

	// Update subtask with branch info and status
	_, err = s.repo.UpdateSubtaskBranch(ctx, db.UpdateSubtaskBranchParams{
		ID:           subtaskID,
		BranchName:   &branchName,
		WorktreePath: &worktreePath,
	})
	if err != nil {
		// Try to cleanup worktree on failure
		_ = s.beadsService.RemoveWorktree(ctx, project.ClonePath, subtaskID.String())
		return nil, fmt.Errorf("failed to update subtask branch: %w", err)
	}

	oldStatus := string(subtask.Status)

	// Update status to IN_PROGRESS
	dbSubtask, err := s.repo.UpdateSubtaskStatus(ctx, db.UpdateSubtaskStatusParams{
		ID:            subtaskID,
		Status:        string(domain.SubtaskStatusInProgress),
		BlockedReason: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subtask status: %w", err)
	}

	updatedSubtask := dbSubtaskToDomain(dbSubtask)

	// Publish subtask:status_changed event
	if s.eventHub != nil {
		s.eventHub.PublishSubtaskStatusChanged(project.ID, updatedSubtask, oldStatus)
	}

	// Spawn Worker agent asynchronously
	if s.workerSpawner != nil {
		go func() {
			bgCtx := context.Background()
			if err := s.workerSpawner.SpawnWorker(bgCtx, updatedSubtask, project); err != nil {
				fmt.Printf("failed to spawn worker for subtask %s: %v\n", subtaskID, err)
			}
		}()
	}

	return updatedSubtask, nil
}

// MarkMerged marks a subtask as merged after the user confirms the PR was merged.
func (s *SubtaskService) MarkMerged(ctx context.Context, subtaskID, userID uuid.UUID) (*domain.Subtask, error) {
	// Get subtask with ownership check
	subtask, err := s.GetSubtask(ctx, subtaskID, userID)
	if err != nil {
		return nil, err
	}

	// Validate current status
	if subtask.Status != domain.SubtaskStatusCompleted {
		return nil, domain.NewUnprocessableError("subtask", "can only mark COMPLETED subtasks as merged")
	}

	// Validate PR exists
	if subtask.PRUrl == nil || *subtask.PRUrl == "" {
		return nil, domain.NewUnprocessableError("subtask", "subtask has no PR URL")
	}

	// Get task and project for cleanup
	task, err := s.taskService.GetTaskByIDInternal(ctx, subtask.TaskID)
	if err != nil {
		return nil, err
	}

	project, err := s.projectService.GetProject(ctx, task.ProjectID, userID)
	if err != nil {
		return nil, err
	}

	oldStatus := string(subtask.Status)

	// Update status to MERGED
	dbSubtask, err := s.repo.UpdateSubtaskStatus(ctx, db.UpdateSubtaskStatusParams{
		ID:            subtaskID,
		Status:        string(domain.SubtaskStatusMerged),
		BlockedReason: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subtask status: %w", err)
	}

	mergedSubtask := dbSubtaskToDomain(dbSubtask)

	// Publish subtask:status_changed event
	if s.eventHub != nil {
		s.eventHub.PublishSubtaskStatusChanged(project.ID, mergedSubtask, oldStatus)
	}

	// Close beads issue
	if subtask.BeadsIssueID != nil {
		if err := s.beadsService.CloseIssue(ctx, project.ClonePath, *subtask.BeadsIssueID, "Merged"); err != nil {
			// Log but don't fail
			fmt.Printf("failed to close beads issue %s: %v\n", *subtask.BeadsIssueID, err)
		}
	}

	// Unblock dependents
	unblocked, err := s.dependencyService.UnblockDependents(ctx, subtaskID)
	if err != nil {
		// Log but don't fail
		fmt.Printf("failed to unblock dependents for subtask %s: %v\n", subtaskID, err)
	}
	if len(unblocked) > 0 {
		fmt.Printf("unblocked %d dependents for subtask %s\n", len(unblocked), subtaskID)
	}

	// Check if task is complete
	completed, err := s.taskService.CheckTaskCompletion(ctx, task.ID)
	if err != nil {
		// Log but don't fail
		fmt.Printf("failed to check task completion for %s: %v\n", task.ID, err)
	}
	if completed {
		fmt.Printf("task %s is now complete\n", task.ID)
	}

	// Cleanup worktree
	if subtask.WorktreePath != nil {
		if err := s.beadsService.RemoveWorktree(ctx, project.ClonePath, subtaskID.String()); err != nil {
			// Log but don't fail
			fmt.Printf("failed to remove worktree for subtask %s: %v\n", subtaskID, err)
		}
	}

	return mergedSubtask, nil
}

// RetrySubtask retries a failed subtask by resetting it and spawning the Worker agent.
func (s *SubtaskService) RetrySubtask(ctx context.Context, subtaskID, userID uuid.UUID) (*domain.Subtask, error) {
	// Get subtask with ownership check
	subtask, err := s.GetSubtask(ctx, subtaskID, userID)
	if err != nil {
		return nil, err
	}

	// Validate current status
	if subtask.Status != domain.SubtaskStatusBlocked {
		return nil, domain.NewUnprocessableError("subtask", "can only retry BLOCKED subtasks")
	}

	// Validate blocked reason is FAILURE
	if subtask.BlockedReason == nil || *subtask.BlockedReason != domain.BlockedReasonFailure {
		return nil, domain.NewUnprocessableError("subtask", "can only retry subtasks blocked due to failure")
	}

	// Get task and project for spawning worker
	task, err := s.taskService.GetTaskByIDInternal(ctx, subtask.TaskID)
	if err != nil {
		return nil, err
	}

	project, err := s.projectService.GetProject(ctx, task.ProjectID, userID)
	if err != nil {
		return nil, err
	}

	// Sync repository to latest before retrying (see §9.5 Repository Sync Strategy)
	if s.githubService != nil {
		if err := s.githubService.SyncRepoWithRetry(ctx, project.ClonePath, project.DefaultBranch, project.IsFork, 3); err != nil {
			return nil, fmt.Errorf("failed to sync repository before retrying subtask: %w", err)
		}
	}

	// Reset retry count
	_, err = s.repo.UpdateSubtaskRetryCount(ctx, db.UpdateSubtaskRetryCountParams{
		ID:         subtaskID,
		RetryCount: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reset retry count: %w", err)
	}

	oldStatus := string(subtask.Status)

	// Update status to IN_PROGRESS
	dbSubtask, err := s.repo.UpdateSubtaskStatus(ctx, db.UpdateSubtaskStatusParams{
		ID:            subtaskID,
		Status:        string(domain.SubtaskStatusInProgress),
		BlockedReason: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subtask status: %w", err)
	}

	updatedSubtask := dbSubtaskToDomain(dbSubtask)

	// Publish subtask:status_changed event
	if s.eventHub != nil {
		s.eventHub.PublishSubtaskStatusChanged(project.ID, updatedSubtask, oldStatus)
	}

	// Spawn Worker agent asynchronously
	if s.workerSpawner != nil {
		go func() {
			bgCtx := context.Background()
			if err := s.workerSpawner.SpawnWorker(bgCtx, updatedSubtask, project); err != nil {
				fmt.Printf("failed to spawn worker for subtask %s: %v\n", subtaskID, err)
			}
		}()
	}

	return updatedSubtask, nil
}

// UpdatePosition updates the position of a subtask (for drag-and-drop reordering).
func (s *SubtaskService) UpdatePosition(ctx context.Context, subtaskID, userID uuid.UUID, position int) (*domain.Subtask, error) {
	// Get subtask with ownership check
	_, err := s.GetSubtask(ctx, subtaskID, userID)
	if err != nil {
		return nil, err
	}

	// Update position
	//nolint:gosec // position is validated to be non-negative and bounded by UI
	dbSubtask, err := s.repo.UpdateSubtaskPosition(ctx, db.UpdateSubtaskPositionParams{
		ID:       subtaskID,
		Position: int32(position),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update position: %w", err)
	}

	return dbSubtaskToDomain(dbSubtask), nil
}

// MarkCompleted marks a subtask as completed (called by agent loop after success).
func (s *SubtaskService) MarkCompleted(ctx context.Context, subtaskID uuid.UUID, prURL string, prNumber int) error {
	// Get the subtask first to capture old status and find project ID
	oldSubtask, err := s.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	oldStatus := oldSubtask.Status

	dbSubtask, err := s.repo.UpdateSubtaskStatus(ctx, db.UpdateSubtaskStatusParams{
		ID:            subtaskID,
		Status:        string(domain.SubtaskStatusCompleted),
		BlockedReason: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to update subtask status: %w", err)
	}

	// Update PR info
	//nolint:gosec // PR numbers from GitHub are always within int32 range
	prNum := int32(prNumber)
	_, err = s.repo.UpdateSubtaskPR(ctx, db.UpdateSubtaskPRParams{
		ID:       dbSubtask.ID,
		PrUrl:    &prURL,
		PrNumber: &prNum,
	})
	if err != nil {
		return fmt.Errorf("failed to update PR info: %w", err)
	}

	// Publish subtask:status_changed event
	if s.eventHub != nil {
		// Get project ID through task
		task, err := s.taskService.GetTaskByIDInternal(ctx, dbSubtask.TaskID)
		if err == nil {
			completedSubtask := dbSubtaskToDomain(dbSubtask)
			s.eventHub.PublishSubtaskStatusChanged(task.ProjectID, completedSubtask, oldStatus)
		}
	}

	return nil
}

// MarkFailed marks a subtask as blocked due to failure (called by agent loop after max retries).
func (s *SubtaskService) MarkFailed(ctx context.Context, subtaskID uuid.UUID) error {
	// Get the subtask first to capture old status and find project ID
	oldSubtask, err := s.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	oldStatus := oldSubtask.Status

	reason := string(domain.BlockedReasonFailure)
	dbSubtask, err := s.repo.UpdateSubtaskStatus(ctx, db.UpdateSubtaskStatusParams{
		ID:            subtaskID,
		Status:        string(domain.SubtaskStatusBlocked),
		BlockedReason: &reason,
	})
	if err != nil {
		return fmt.Errorf("failed to update subtask status: %w", err)
	}

	// Publish subtask:status_changed event
	if s.eventHub != nil {
		// Get project ID through task
		task, err := s.taskService.GetTaskByIDInternal(ctx, dbSubtask.TaskID)
		if err == nil {
			failedSubtask := dbSubtaskToDomain(dbSubtask)
			s.eventHub.PublishSubtaskStatusChanged(task.ProjectID, failedSubtask, oldStatus)
		}
	}

	return nil
}

// IncrementRetryCount increments the retry count for a subtask.
func (s *SubtaskService) IncrementRetryCount(ctx context.Context, subtaskID uuid.UUID) (int, error) {
	subtask, err := s.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		return 0, fmt.Errorf("failed to get subtask: %w", err)
	}

	newCount := subtask.RetryCount + 1
	_, err = s.repo.UpdateSubtaskRetryCount(ctx, db.UpdateSubtaskRetryCountParams{
		ID:         subtaskID,
		RetryCount: newCount,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to update retry count: %w", err)
	}

	return int(newCount), nil
}

// UpdateTokenUsage adds to the token usage for a subtask.
func (s *SubtaskService) UpdateTokenUsage(ctx context.Context, subtaskID uuid.UUID, tokens int) error {
	//nolint:gosec // token counts are always positive and bounded
	_, err := s.repo.UpdateSubtaskTokenUsage(ctx, db.UpdateSubtaskTokenUsageParams{
		ID:         subtaskID,
		TokenUsage: int32(tokens),
	})
	if err != nil {
		return fmt.Errorf("failed to update token usage: %w", err)
	}
	return nil
}

// GetSubtaskByIDInternal retrieves a subtask by ID without ownership check.
// Only for internal use by other services.
func (s *SubtaskService) GetSubtaskByIDInternal(ctx context.Context, subtaskID uuid.UUID) (*domain.Subtask, error) {
	subtask, err := s.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("subtask", subtaskID.String())
		}
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}
	return dbSubtaskToDomain(subtask), nil
}

// GetSubtaskByBeadsID retrieves a subtask by its beads issue ID.
func (s *SubtaskService) GetSubtaskByBeadsID(ctx context.Context, beadsIssueID string) (*domain.Subtask, error) {
	subtask, err := s.repo.GetSubtaskByBeadsID(ctx, &beadsIssueID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("subtask", beadsIssueID)
		}
		return nil, fmt.Errorf("failed to get subtask by beads ID: %w", err)
	}
	return dbSubtaskToDomain(subtask), nil
}

// UpdateSubtaskStatus updates the status of a subtask after sync.
func (s *SubtaskService) UpdateSubtaskStatus(ctx context.Context, subtaskID uuid.UUID, status domain.SubtaskStatus, blockedReason *domain.BlockedReason) error {
	// Get the subtask first to capture old status and find project ID
	oldSubtask, err := s.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	oldStatus := oldSubtask.Status

	var reason *string
	if blockedReason != nil {
		r := string(*blockedReason)
		reason = &r
	}

	dbSubtask, err := s.repo.UpdateSubtaskStatus(ctx, db.UpdateSubtaskStatusParams{
		ID:            subtaskID,
		Status:        string(status),
		BlockedReason: reason,
	})
	if err != nil {
		return fmt.Errorf("failed to update subtask status: %w", err)
	}

	// Publish subtask:status_changed event
	if s.eventHub != nil {
		// Get project ID through task
		task, err := s.taskService.GetTaskByIDInternal(ctx, dbSubtask.TaskID)
		if err == nil {
			updatedSubtask := dbSubtaskToDomain(dbSubtask)
			s.eventHub.PublishSubtaskStatusChanged(task.ProjectID, updatedSubtask, oldStatus)
		}
	}

	return nil
}

// dbSubtaskToDomain converts a database Subtask to a domain Subtask.
func dbSubtaskToDomain(s db.Subtask) *domain.Subtask {
	var blockedReason *domain.BlockedReason
	if s.BlockedReason != nil {
		r := domain.BlockedReason(*s.BlockedReason)
		blockedReason = &r
	}

	var prNumber *int
	if s.PrNumber != nil {
		n := int(*s.PrNumber)
		prNumber = &n
	}

	return &domain.Subtask{
		ID:                 s.ID,
		TaskID:             s.TaskID,
		Title:              s.Title,
		Spec:               s.Spec,
		ImplementationPlan: s.ImplementationPlan,
		Status:             domain.SubtaskStatus(s.Status),
		BlockedReason:      blockedReason,
		BranchName:         s.BranchName,
		PRUrl:              s.PrUrl,
		PRNumber:           prNumber,
		RetryCount:         int(s.RetryCount),
		TokenUsage:         int(s.TokenUsage),
		Position:           int(s.Position),
		BeadsIssueID:       s.BeadsIssueID,
		WorktreePath:       s.WorktreePath,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}
