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

// AgentSpawner is an interface for spawning agents.
// This allows us to mock agent spawning in tests.
type AgentSpawner interface {
	SpawnPlanner(ctx context.Context, task *domain.Task, project *domain.Project) error
	KillAgentsForTask(ctx context.Context, taskID uuid.UUID) error
}

// TaskService handles task management operations.
type TaskService struct {
	repo           *repository.Repository
	projectService *ProjectService
	githubService  *GitHubService
	beadsService   *BeadsService
	agentSpawner   AgentSpawner
	eventHub       EventHub
}

// NewTaskService creates a new TaskService.
func NewTaskService(
	repo *repository.Repository,
	projectService *ProjectService,
	githubService *GitHubService,
	beadsService *BeadsService,
	eventHub EventHub,
) *TaskService {
	return &TaskService{
		repo:           repo,
		projectService: projectService,
		githubService:  githubService,
		beadsService:   beadsService,
		eventHub:       eventHub,
	}
}

// SetAgentSpawner sets the agent spawner for the service.
// This is set after construction to break circular dependencies.
func (s *TaskService) SetAgentSpawner(spawner AgentSpawner) {
	s.agentSpawner = spawner
}

// CreateTaskInput contains the input for creating a task.
type CreateTaskInput struct {
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description string
}

// CreateTask creates a new task and spawns the Planner agent.
func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*domain.Task, error) {
	// Verify project exists and user has access
	project, err := s.projectService.GetProject(ctx, input.ProjectID, input.UserID)
	if err != nil {
		return nil, err
	}

	// Sync repository to latest before planning (see §9.5 Repository Sync Strategy)
	if s.githubService != nil {
		if err := s.githubService.SyncRepoWithRetry(ctx, project.ClonePath, project.DefaultBranch, project.IsFork, 3); err != nil {
			return nil, fmt.Errorf("failed to sync repository before planning: %w", err)
		}
	}

	// Create the task record
	dbTask, err := s.repo.CreateTask(ctx, db.CreateTaskParams{
		ProjectID:   input.ProjectID,
		Title:       input.Title,
		Description: input.Description,
		Status:      string(domain.TaskStatusPlanning),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	task := dbTaskToDomain(dbTask)

	// Publish task:status_changed event (nil -> PLANNING)
	if s.eventHub != nil {
		s.eventHub.PublishTaskStatusChanged(input.ProjectID, task.ID, "", string(domain.TaskStatusPlanning))
	}

	// Spawn Planner agent asynchronously if spawner is set
	if s.agentSpawner != nil {
		go func() {
			// Use background context since this is async
			bgCtx := context.Background()
			if err := s.agentSpawner.SpawnPlanner(bgCtx, task, project); err != nil {
				// Log error but don't fail the task creation
				// The task will be in PLANNING state and can be retried
				fmt.Printf("failed to spawn planner for task %s: %v\n", task.ID, err)
			}
		}()
	}

	return task, nil
}

// GetTask retrieves a task by ID with ownership verification.
func (s *TaskService) GetTask(ctx context.Context, taskID, userID uuid.UUID) (*domain.Task, error) {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("task", taskID.String())
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Verify ownership through project
	_, err = s.projectService.GetProject(ctx, task.ProjectID, userID)
	if err != nil {
		return nil, err
	}

	return dbTaskToDomain(task), nil
}

// ListTasks lists all tasks for a project.
func (s *TaskService) ListTasks(ctx context.Context, projectID, userID uuid.UUID) ([]*domain.Task, error) {
	// Verify project access
	_, err := s.projectService.GetProject(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	tasks, err := s.repo.ListTasksByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	result := make([]*domain.Task, len(tasks))
	for i, t := range tasks {
		result[i] = dbTaskToDomain(t)
	}

	return result, nil
}

// DeleteTask deletes a task after killing any running agents and cleaning up.
func (s *TaskService) DeleteTask(ctx context.Context, taskID, userID uuid.UUID) error {
	// Verify ownership
	task, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return err
	}

	// Kill any running agents for this task
	if s.agentSpawner != nil {
		if err := s.agentSpawner.KillAgentsForTask(ctx, task.ID); err != nil {
			// Log but continue - we still want to delete the task
			fmt.Printf("failed to kill agents for task %s: %v\n", taskID, err)
		}
	}

	// Delete beads epic (cascades to all subtask issues)
	if task.BeadsEpicID != nil && *task.BeadsEpicID != "" && s.beadsService != nil {
		project, err := s.projectService.GetProjectByIDInternal(ctx, task.ProjectID)
		if err == nil && project.ClonePath != "" {
			if err := s.beadsService.DeleteIssue(ctx, project.ClonePath, *task.BeadsEpicID, true); err != nil {
				// Log but continue - we still want to delete the task from DB
				fmt.Printf("failed to delete beads epic %s: %v\n", *task.BeadsEpicID, err)
			}
		}
	}

	// Delete the task (cascades to subtasks)
	if err := s.repo.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// RetryPlanning resets a failed task to PLANNING status and spawns the Planner agent.
func (s *TaskService) RetryPlanning(ctx context.Context, taskID, userID uuid.UUID) (*domain.Task, error) {
	// Get task with ownership check
	task, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Validate current status
	if task.Status != domain.TaskStatusPlanningFailed {
		return nil, domain.NewUnprocessableError("task", "can only retry planning for tasks in PLANNING_FAILED status")
	}

	// Validate transition
	if !domain.CanTransitionTask(task.Status, domain.TaskStatusPlanning) {
		return nil, domain.NewInvalidTransitionError(
			"task",
			string(task.Status),
			string(domain.TaskStatusPlanning),
			"invalid task state transition",
		)
	}

	// Get project for spawning planner
	project, err := s.projectService.GetProject(ctx, task.ProjectID, userID)
	if err != nil {
		return nil, err
	}

	// Sync repository to latest before retrying planning (see §9.5 Repository Sync Strategy)
	if s.githubService != nil {
		if err := s.githubService.SyncRepoWithRetry(ctx, project.ClonePath, project.DefaultBranch, project.IsFork, 3); err != nil {
			return nil, fmt.Errorf("failed to sync repository before planning: %w", err)
		}
	}

	oldStatus := string(task.Status)

	// Update status to PLANNING
	dbTask, err := s.repo.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		ID:     taskID,
		Status: string(domain.TaskStatusPlanning),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	updatedTask := dbTaskToDomain(dbTask)

	// Publish task:status_changed event (PLANNING_FAILED -> PLANNING)
	if s.eventHub != nil {
		s.eventHub.PublishTaskStatusChanged(task.ProjectID, taskID, oldStatus, string(domain.TaskStatusPlanning))
	}

	// Spawn Planner agent asynchronously
	if s.agentSpawner != nil {
		go func() {
			bgCtx := context.Background()
			if err := s.agentSpawner.SpawnPlanner(bgCtx, updatedTask, project); err != nil {
				fmt.Printf("failed to spawn planner for task %s: %v\n", taskID, err)
			}
		}()
	}

	return updatedTask, nil
}

// MarkPlanningFailed transitions a task to PLANNING_FAILED status.
// Called when the Planner agent exceeds max retries.
func (s *TaskService) MarkPlanningFailed(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NewNotFoundError("task", taskID.String())
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Validate transition
	if !domain.CanTransitionTask(domain.TaskStatus(task.Status), domain.TaskStatusPlanningFailed) {
		return domain.NewInvalidTransitionError(
			"task",
			task.Status,
			string(domain.TaskStatusPlanningFailed),
			"task is not in PLANNING status",
		)
	}

	oldStatus := task.Status

	_, err = s.repo.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		ID:     taskID,
		Status: string(domain.TaskStatusPlanningFailed),
	})
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Publish task:status_changed event (PLANNING -> PLANNING_FAILED)
	if s.eventHub != nil {
		s.eventHub.PublishTaskStatusChanged(task.ProjectID, taskID, oldStatus, string(domain.TaskStatusPlanningFailed))
	}

	return nil
}

// TransitionToActive transitions a task to ACTIVE status.
// Called when the Planner agent completes successfully.
func (s *TaskService) TransitionToActive(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NewNotFoundError("task", taskID.String())
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	oldStatus := task.Status

	// Validate transition
	if !domain.CanTransitionTask(domain.TaskStatus(task.Status), domain.TaskStatusActive) {
		return domain.NewInvalidTransitionError(
			"task",
			task.Status,
			string(domain.TaskStatusActive),
			"task is not in PLANNING status",
		)
	}

	_, err = s.repo.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		ID:     taskID,
		Status: string(domain.TaskStatusActive),
	})
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Publish task:status_changed event (PLANNING -> ACTIVE)
	if s.eventHub != nil {
		s.eventHub.PublishTaskStatusChanged(task.ProjectID, taskID, oldStatus, string(domain.TaskStatusActive))
	}

	return nil
}

// CheckTaskCompletion checks if all subtasks are merged and transitions task to DONE.
func (s *TaskService) CheckTaskCompletion(ctx context.Context, taskID uuid.UUID) (bool, error) {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, domain.NewNotFoundError("task", taskID.String())
		}
		return false, fmt.Errorf("failed to get task: %w", err)
	}

	// Only check tasks that are ACTIVE
	if task.Status != string(domain.TaskStatusActive) {
		return false, nil
	}

	// Get all subtasks
	subtasks, err := s.repo.ListSubtasksByTask(ctx, taskID)
	if err != nil {
		return false, fmt.Errorf("failed to list subtasks: %w", err)
	}

	// Check if all subtasks are merged
	if len(subtasks) == 0 {
		return false, nil
	}

	allMerged := true
	for _, st := range subtasks {
		if st.Status != string(domain.SubtaskStatusMerged) {
			allMerged = false
			break
		}
	}

	if allMerged {
		// Transition to DONE
		_, err = s.repo.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
			ID:     taskID,
			Status: string(domain.TaskStatusDone),
		})
		if err != nil {
			return false, fmt.Errorf("failed to update task status: %w", err)
		}

		// Publish task:status_changed event (ACTIVE -> DONE)
		if s.eventHub != nil {
			s.eventHub.PublishTaskStatusChanged(task.ProjectID, taskID, task.Status, string(domain.TaskStatusDone))
		}

		return true, nil
	}

	return false, nil
}

// UpdateBeadsEpicID sets the beads epic ID for a task.
func (s *TaskService) UpdateBeadsEpicID(ctx context.Context, taskID uuid.UUID, epicID string) error {
	_, err := s.repo.UpdateTaskBeadsEpicID(ctx, db.UpdateTaskBeadsEpicIDParams{
		ID:          taskID,
		BeadsEpicID: &epicID,
	})
	if err != nil {
		return fmt.Errorf("failed to update beads epic ID: %w", err)
	}
	return nil
}

// GetTaskByIDInternal retrieves a task by ID without ownership check.
// Only for internal use by other services.
func (s *TaskService) GetTaskByIDInternal(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("task", taskID.String())
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return dbTaskToDomain(task), nil
}

// dbTaskToDomain converts a database Task to a domain Task.
func dbTaskToDomain(t db.Task) *domain.Task {
	return &domain.Task{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      domain.TaskStatus(t.Status),
		BeadsEpicID: t.BeadsEpicID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
