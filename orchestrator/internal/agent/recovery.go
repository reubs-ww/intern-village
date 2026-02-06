// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/generated/db"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
)

// Recovery handles recovery of stale agent runs on orchestrator startup.
type Recovery struct {
	repo           *repository.Repository
	manager        *AgentManager
	projectService ProjectServiceInterface
	taskService    TaskServiceInterface
	subtaskService SubtaskServiceInterface
	maxRetries     int
}

// ProjectServiceInterface defines the project service methods used for recovery.
type ProjectServiceInterface interface {
	GetProjectByIDInternal(ctx context.Context, projectID uuid.UUID) (*domain.Project, error)
}

// NewRecovery creates a new Recovery instance.
func NewRecovery(
	repo *repository.Repository,
	manager *AgentManager,
	projectService ProjectServiceInterface,
	taskService TaskServiceInterface,
	subtaskService SubtaskServiceInterface,
	maxRetries int,
) *Recovery {
	return &Recovery{
		repo:           repo,
		manager:        manager,
		projectService: projectService,
		taskService:    taskService,
		subtaskService: subtaskService,
		maxRetries:     maxRetries,
	}
}

// RecoverStaleAgents recovers any agent runs that were marked as RUNNING but have no active process.
// This should be called on orchestrator startup.
func (r *Recovery) RecoverStaleAgents(ctx context.Context) error {
	log.Info().Msg("checking for stale agent runs")

	// Get all running agent runs
	runningRuns, err := r.repo.GetRunningAgentRuns(ctx)
	if err != nil {
		return err
	}

	if len(runningRuns) == 0 {
		log.Info().Msg("no stale agent runs found")
		return nil
	}

	log.Info().Int("count", len(runningRuns)).Msg("found stale agent runs")

	// Mark stale runs as failed (anything started more than 5 minutes ago with no process)
	cutoff := time.Now().Add(-5 * time.Minute)
	err = r.repo.MarkStaleAgentRunsFailed(ctx, cutoff)
	if err != nil {
		log.Error().Err(err).Msg("failed to mark stale agent runs as failed")
	}

	// Group runs by subtask to determine if we should restart
	// Note: Planner runs have SubtaskID as nil, they are task-level runs
	subtaskRuns := make(map[uuid.UUID][]db.AgentRun)
	for _, run := range runningRuns {
		// Skip Planner runs (they have nil SubtaskID) - they don't need recovery
		// since they run synchronously during task creation
		if !run.SubtaskID.Valid {
			continue
		}
		subtaskRuns[run.SubtaskID.Bytes] = append(subtaskRuns[run.SubtaskID.Bytes], run)
	}

	// Process each subtask with stale runs
	for subtaskID, runs := range subtaskRuns {
		if err := r.recoverSubtask(ctx, subtaskID, runs); err != nil {
			log.Error().
				Err(err).
				Str("subtask_id", subtaskID.String()).
				Msg("failed to recover subtask")
		}
	}

	return nil
}

// recoverSubtask attempts to recover a subtask with stale agent runs.
func (r *Recovery) recoverSubtask(ctx context.Context, subtaskID uuid.UUID, runs []db.AgentRun) error {
	if len(runs) == 0 {
		return nil
	}

	// Get the latest run
	latestRun := runs[0]
	for _, run := range runs {
		if run.AttemptNumber > latestRun.AttemptNumber {
			latestRun = run
		}
	}

	// Check agent type
	if domain.AgentType(latestRun.AgentType) == domain.AgentTypePlanner {
		return r.recoverPlannerRun(ctx, subtaskID, latestRun)
	}
	return r.recoverWorkerRun(ctx, subtaskID, latestRun)
}

// recoverPlannerRun handles recovery of a planner agent run.
// The subtaskID is actually the taskID for planner runs.
func (r *Recovery) recoverPlannerRun(ctx context.Context, taskID uuid.UUID, run db.AgentRun) error {
	log.Info().
		Str("task_id", taskID.String()).
		Int32("attempt", run.AttemptNumber).
		Msg("recovering planner run")

	// Get task
	task, err := r.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Check if we've exceeded max retries
	if int(run.AttemptNumber) >= r.maxRetries {
		log.Warn().
			Str("task_id", taskID.String()).
			Msg("planner max retries reached, marking task as failed")

		if err := r.taskService.MarkPlanningFailed(ctx, taskID); err != nil {
			return err
		}
		return nil
	}

	// Task is still in PLANNING state, we can restart the planner
	if domain.TaskStatus(task.Status) == domain.TaskStatusPlanning {
		log.Info().
			Str("task_id", taskID.String()).
			Msg("restarting planner for task")

		// Get project
		project, err := r.projectService.GetProjectByIDInternal(ctx, task.ProjectID)
		if err != nil {
			return err
		}

		// Convert db task to domain task
		domainTask := &domain.Task{
			ID:          task.ID,
			ProjectID:   task.ProjectID,
			Title:       task.Title,
			Description: task.Description,
			Status:      domain.TaskStatus(task.Status),
			BeadsEpicID: task.BeadsEpicID,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
		}

		// Restart the planner
		if err := r.manager.SpawnPlanner(ctx, domainTask, project); err != nil {
			log.Error().Err(err).Msg("failed to restart planner")
			return err
		}
	}

	return nil
}

// recoverWorkerRun handles recovery of a worker agent run.
func (r *Recovery) recoverWorkerRun(ctx context.Context, subtaskID uuid.UUID, run db.AgentRun) error {
	log.Info().
		Str("subtask_id", subtaskID.String()).
		Int32("attempt", run.AttemptNumber).
		Msg("recovering worker run")

	// Get subtask
	subtask, err := r.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		return err
	}

	// Check if we've exceeded max retries
	if int(run.AttemptNumber) >= r.maxRetries {
		log.Warn().
			Str("subtask_id", subtaskID.String()).
			Msg("worker max retries reached, marking subtask as failed")

		if err := r.subtaskService.MarkFailed(ctx, subtaskID); err != nil {
			return err
		}
		return nil
	}

	// Subtask is still IN_PROGRESS, we can restart the worker
	if domain.SubtaskStatus(subtask.Status) == domain.SubtaskStatusInProgress {
		log.Info().
			Str("subtask_id", subtaskID.String()).
			Msg("restarting worker for subtask")

		// Get task and project
		task, err := r.repo.GetTaskByID(ctx, subtask.TaskID)
		if err != nil {
			return err
		}

		project, err := r.projectService.GetProjectByIDInternal(ctx, task.ProjectID)
		if err != nil {
			return err
		}

		// Convert db subtask to domain subtask
		var blockedReason *domain.BlockedReason
		if subtask.BlockedReason != nil {
			r := domain.BlockedReason(*subtask.BlockedReason)
			blockedReason = &r
		}

		var prNumber *int
		if subtask.PrNumber != nil {
			n := int(*subtask.PrNumber)
			prNumber = &n
		}

		domainSubtask := &domain.Subtask{
			ID:                 subtask.ID,
			TaskID:             subtask.TaskID,
			Title:              subtask.Title,
			Spec:               subtask.Spec,
			ImplementationPlan: subtask.ImplementationPlan,
			Status:             domain.SubtaskStatus(subtask.Status),
			BlockedReason:      blockedReason,
			BranchName:         subtask.BranchName,
			PRUrl:              subtask.PrUrl,
			PRNumber:           prNumber,
			RetryCount:         int(subtask.RetryCount),
			TokenUsage:         int(subtask.TokenUsage),
			Position:           int(subtask.Position),
			BeadsIssueID:       subtask.BeadsIssueID,
			WorktreePath:       subtask.WorktreePath,
			CreatedAt:          subtask.CreatedAt,
			UpdatedAt:          subtask.UpdatedAt,
		}

		// Restart the worker
		if err := r.manager.SpawnWorker(ctx, domainSubtask, project); err != nil {
			log.Error().Err(err).Msg("failed to restart worker")
			return err
		}
	}

	return nil
}
