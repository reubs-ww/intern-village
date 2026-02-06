// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/generated/db"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
)

// uuidToPgtype converts a google/uuid.UUID to pgtype.UUID
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// pgtypeToUUID converts a pgtype.UUID to google/uuid.UUID
// Returns uuid.Nil if the pgtype.UUID is not valid
func pgtypeToUUID(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}
	return p.Bytes
}

// LoopServices contains services required by the agent loop.
type LoopServices struct {
	Repo           *repository.Repository
	BeadsService   BeadsServiceInterface
	GitHubService  GitHubServiceInterface
	SyncService    SyncServiceInterface
	TaskService    TaskServiceInterface
	SubtaskService SubtaskServiceInterface
	EventPublisher EventPublisherInterface
	LogTailer      LogTailerInterface
}

// BeadsServiceInterface defines the beads service methods used by the agent loop.
type BeadsServiceInterface interface {
	ShowIssue(ctx context.Context, repoPath, issueID string) (*BeadsIssue, error)
	CloseIssue(ctx context.Context, repoPath, issueID, reason string) error
	FindEpicByTaskID(ctx context.Context, repoPath, taskIDPrefix string) (*BeadsIssue, error)
}

// BeadsDependency represents a dependency relationship from Beads.
type BeadsDependency struct {
	IssueID     string `json:"issue_id"`
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"` // "parent-child", "blocks"
}

// BeadsIssue is a local type matching the service.BeadsIssue.
type BeadsIssue struct {
	ID           string            `json:"id"`
	Type         string            `json:"issue_type"` // "epic" or "task"
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Status       string            `json:"status"`
	ParentID     string            `json:"parent,omitempty"`
	Dependencies []BeadsDependency `json:"dependencies,omitempty"`
}

// GetDependencyIDs returns just the depends_on_id values from blocking dependencies.
func (b *BeadsIssue) GetDependencyIDs() []string {
	var ids []string
	for _, dep := range b.Dependencies {
		if dep.Type == "blocks" {
			ids = append(ids, dep.DependsOnID)
		}
	}
	return ids
}

// GitHubServiceInterface defines the GitHub service methods used by the agent loop.
type GitHubServiceInterface interface {
	PushBranch(ctx context.Context, repoPath, branch string) error
	CreatePR(ctx context.Context, owner, repo, accessToken, head, base, title, body string) (*PRInfo, error)
	GetCommitMessages(ctx context.Context, repoPath, baseBranch string) ([]string, error)
}

// PRInfo contains pull request information.
type PRInfo struct {
	Number  int
	URL     string
	HTMLURL string
}

// SyncServiceInterface defines the sync service methods used by the agent loop.
type SyncServiceInterface interface {
	SyncTaskFromBeads(ctx context.Context, taskID uuid.UUID, repoPath string) error
}

// TaskServiceInterface defines the task service methods used by the agent loop.
type TaskServiceInterface interface {
	TransitionToActive(ctx context.Context, taskID uuid.UUID) error
	MarkPlanningFailed(ctx context.Context, taskID uuid.UUID) error
	UpdateBeadsEpicID(ctx context.Context, taskID uuid.UUID, epicID string) error
}

// SubtaskServiceInterface defines the subtask service methods used by the agent loop.
type SubtaskServiceInterface interface {
	MarkCompleted(ctx context.Context, subtaskID uuid.UUID, prURL string, prNumber int) error
	MarkFailed(ctx context.Context, subtaskID uuid.UUID) error
	IncrementRetryCount(ctx context.Context, subtaskID uuid.UUID) (int, error)
	UpdateTokenUsage(ctx context.Context, subtaskID uuid.UUID, tokens int) error
}

// EventPublisherInterface defines the event publishing methods used by the agent loop.
type EventPublisherInterface interface {
	PublishAgentStarted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID)
	PublishAgentCompleted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, prURL string)
	PublishAgentFailed(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, errMsg string, willRetry bool, nextAttemptAt *time.Time)
}

// LogTailerInterface defines the log tailing methods used by the agent loop.
type LogTailerInterface interface {
	StartTailing(ctx context.Context, projectID, runID uuid.UUID, logPath string) error
	StopTailing(runID uuid.UUID)
}

// AgentLoop manages the loop-until-done execution pattern for agents.
type AgentLoop struct {
	executor       *Executor
	promptRenderer *PromptRenderer
	services       LoopServices
	maxRetries     int
}

// NewAgentLoop creates a new AgentLoop.
func NewAgentLoop(
	executor *Executor,
	promptRenderer *PromptRenderer,
	services LoopServices,
	maxRetries int,
) *AgentLoop {
	return &AgentLoop{
		executor:       executor,
		promptRenderer: promptRenderer,
		services:       services,
		maxRetries:     maxRetries,
	}
}

// RunPlannerLoop runs the Planner agent once.
// The Planner runs in the main clone directory (not a worktree).
// NOTE: Simplified - no retry loop, just run once and mark complete on exit code 0.
func (l *AgentLoop) RunPlannerLoop(ctx context.Context, task *domain.Task, project *domain.Project, userToken string) error {
	log.Info().
		Str("task_id", task.ID.String()).
		Str("project_id", project.ID.String()).
		Msg("starting planner")

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	attempt := 1

	// Render prompt
	promptContent, err := l.promptRenderer.RenderPlannerPrompt(task, project)
	if err != nil {
		return fmt.Errorf("failed to render planner prompt: %w", err)
	}

	// Save prompt
	promptPath, err := l.promptRenderer.SavePrompt(
		promptContent,
		project.ID.String(),
		task.ID.String(),
		"", // No subtask ID for planner
	)
	if err != nil {
		return fmt.Errorf("failed to save planner prompt: %w", err)
	}

	// Create agent run record (task-level for Planner)
	agentRun, err := l.services.Repo.CreateAgentRunForTask(ctx, db.CreateAgentRunForTaskParams{
		TaskID:        uuidToPgtype(task.ID),
		AgentType:     string(domain.AgentTypePlanner),
		AttemptNumber: int32(attempt),
		Status:        string(domain.AgentRunStatusRunning),
		LogPath:       l.executor.GetLogPath(project.ID.String(), task.ID.String(), "", attempt),
		PromptText:    promptContent,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent run record: %w", err)
	}

	// Publish agent:started event
	if l.services.EventPublisher != nil {
		run := &domain.AgentRun{
			ID:            agentRun.ID,
			TaskID:        &task.ID,
			AgentType:     domain.AgentTypePlanner,
			AttemptNumber: int(agentRun.AttemptNumber),
			Status:        domain.AgentRunStatusRunning,
			StartedAt:     agentRun.StartedAt,
			LogPath:       agentRun.LogPath,
		}
		l.services.EventPublisher.PublishAgentStarted(project.ID, run, task.ID)
	}

	// Start Claude asynchronously (creates log file immediately)
	claudeRun, err := l.executor.ExecuteClaudeAsync(
		ctx,
		project.ClonePath,
		promptPath,
		project.ID.String(),
		task.ID.String(),
		"",
		attempt,
	)
	if err != nil {
		l.markAgentRunFailed(ctx, agentRun.ID, err.Error())
		return fmt.Errorf("planner execution failed: %w", err)
	}

	// Start log tailing now that log file exists
	if l.services.LogTailer != nil {
		go func() {
			if err := l.services.LogTailer.StartTailing(ctx, project.ID, agentRun.ID, claudeRun.LogPath); err != nil {
				log.Warn().Err(err).Str("run_id", agentRun.ID.String()).Msg("failed to start log tailing for planner")
			}
		}()
	}

	// Wait for Claude to complete
	result := claudeRun.Wait()

	// Stop log tailing
	if l.services.LogTailer != nil {
		l.services.LogTailer.StopTailing(agentRun.ID)
	}

	if result.Error != nil && ctx.Err() != nil {
		l.markAgentRunFailed(ctx, agentRun.ID, result.Error.Error())
		return ctx.Err()
	}

	// Update agent run with token usage
	if result.TokenUsage > 0 {
		//nolint:gosec // TokenUsage is always positive and bounded
		_, _ = l.services.Repo.UpdateAgentRunTokenUsage(ctx, db.UpdateAgentRunTokenUsageParams{
			ID:         agentRun.ID,
			TokenUsage: &[]int32{int32(result.TokenUsage)}[0],
		})
	}

	// Simplified: if exit code 0, consider planner successful
	if result.ExitCode == 0 {
		// Find the epic created by the Planner using the task ID prefix
		// Epic titles are formatted as "[{taskID_prefix}] {title}" by the planner
		taskIDPrefix := task.ID.String()[:8]
		epic, err := l.services.BeadsService.FindEpicByTaskID(ctx, project.ClonePath, taskIDPrefix)
		if err != nil {
			log.Error().Err(err).Msg("failed to find epic by task ID")
		} else if epic != nil {
			// Store the epic ID in the task
			if err := l.services.TaskService.UpdateBeadsEpicID(ctx, task.ID, epic.ID); err != nil {
				log.Error().Err(err).Str("epic_id", epic.ID).Msg("failed to update task with epic ID")
			} else {
				log.Info().
					Str("task_id", task.ID.String()).
					Str("epic_id", epic.ID).
					Msg("found and stored epic ID")

				// Sync subtasks from Beads to Postgres
				if err := l.services.SyncService.SyncTaskFromBeads(ctx, task.ID, project.ClonePath); err != nil {
					log.Error().Err(err).Msg("failed to sync subtasks from beads")
				} else {
					log.Info().
						Str("task_id", task.ID.String()).
						Msg("synced subtasks from beads to postgres")
				}
			}
		} else {
			log.Warn().
				Str("task_id", task.ID.String()).
				Str("task_id_prefix", taskIDPrefix).
				Msg("no epic found with task ID prefix - subtasks may not appear")
		}

		// Transition task to ACTIVE
		if err := l.services.TaskService.TransitionToActive(ctx, task.ID); err != nil {
			log.Error().Err(err).Msg("failed to transition task to ACTIVE")
		}

		l.markAgentRunSucceeded(ctx, agentRun.ID)

		// Publish agent:completed event
		if l.services.EventPublisher != nil {
			now := time.Now()
			run := &domain.AgentRun{
				ID:            agentRun.ID,
				TaskID:        &task.ID,
				AgentType:     domain.AgentTypePlanner,
				AttemptNumber: int(agentRun.AttemptNumber),
				Status:        domain.AgentRunStatusSucceeded,
				StartedAt:     agentRun.StartedAt,
				EndedAt:       &now,
				TokenUsage:    &result.TokenUsage,
			}
			l.services.EventPublisher.PublishAgentCompleted(project.ID, run, task.ID, "")
		}

		log.Info().
			Str("task_id", task.ID.String()).
			Msg("planner completed successfully")
		return nil
	}

	// Planner failed
	l.markAgentRunFailed(ctx, agentRun.ID, fmt.Sprintf("exit code: %d", result.ExitCode))
	if err := l.services.TaskService.MarkPlanningFailed(ctx, task.ID); err != nil {
		log.Error().Err(err).Msg("failed to mark task planning as failed")
	}

	return fmt.Errorf("planner failed with exit code: %d", result.ExitCode)
}

// RunWorkerLoop runs the Worker agent loop.
// The Worker runs in a dedicated worktree.
func (l *AgentLoop) RunWorkerLoop(ctx context.Context, subtask *domain.Subtask, project *domain.Project, userToken string) error {
	log.Info().
		Str("subtask_id", subtask.ID.String()).
		Str("project_id", project.ID.String()).
		Msg("starting worker loop")

	for attempt := 1; attempt <= l.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Info().
			Str("subtask_id", subtask.ID.String()).
			Int("attempt", attempt).
			Msg("worker attempt")

		// Increment retry count
		_, err := l.services.SubtaskService.IncrementRetryCount(ctx, subtask.ID)
		if err != nil {
			log.Error().Err(err).Msg("failed to increment retry count")
		}

		// Render prompt
		promptContent, err := l.promptRenderer.RenderWorkerPrompt(subtask, project)
		if err != nil {
			return fmt.Errorf("failed to render worker prompt: %w", err)
		}

		// Get task ID for prompt path
		taskID := subtask.TaskID.String()

		// Save prompt
		promptPath, err := l.promptRenderer.SavePrompt(
			promptContent,
			project.ID.String(),
			taskID,
			subtask.ID.String(),
		)
		if err != nil {
			return fmt.Errorf("failed to save worker prompt: %w", err)
		}

		// Determine work directory
		workDir := project.ClonePath
		if subtask.WorktreePath != nil && *subtask.WorktreePath != "" {
			workDir = *subtask.WorktreePath
		}

		// Create agent run record
		agentRun, err := l.services.Repo.CreateAgentRun(ctx, db.CreateAgentRunParams{
			SubtaskID:     uuidToPgtype(subtask.ID),
			AgentType:     string(domain.AgentTypeWorker),
			AttemptNumber: int32(attempt), //nolint:gosec // attempt is bounded by maxRetries
			Status:        string(domain.AgentRunStatusRunning),
			LogPath:       l.executor.GetLogPath(project.ID.String(), taskID, subtask.ID.String(), attempt),
			PromptText:    promptContent,
		})
		if err != nil {
			return fmt.Errorf("failed to create agent run record: %w", err)
		}

		// Publish agent:started event
		if l.services.EventPublisher != nil {
			subtaskIDPtr := pgtypeToUUID(agentRun.SubtaskID)
			run := &domain.AgentRun{
				ID:            agentRun.ID,
				SubtaskID:     &subtaskIDPtr,
				AgentType:     domain.AgentTypeWorker,
				AttemptNumber: int(agentRun.AttemptNumber),
				Status:        domain.AgentRunStatusRunning,
				StartedAt:     agentRun.StartedAt,
				LogPath:       agentRun.LogPath,
			}
			l.services.EventPublisher.PublishAgentStarted(project.ID, run, subtask.TaskID)
		}

		// Start Claude asynchronously (creates log file immediately)
		claudeRun, err := l.executor.ExecuteClaudeAsync(
			ctx,
			workDir,
			promptPath,
			project.ID.String(),
			taskID,
			subtask.ID.String(),
			attempt,
		)
		if err != nil {
			l.markAgentRunFailed(ctx, agentRun.ID, err.Error())
			if attempt < l.maxRetries {
				l.backoff(ctx, attempt)
				continue
			}
			l.markSubtaskFailed(ctx, subtask.ID)
			return fmt.Errorf("worker execution failed: %w", err)
		}

		// Start log tailing now that log file exists
		if l.services.LogTailer != nil {
			go func() {
				if err := l.services.LogTailer.StartTailing(ctx, project.ID, agentRun.ID, claudeRun.LogPath); err != nil {
					log.Warn().Err(err).Str("run_id", agentRun.ID.String()).Msg("failed to start log tailing for worker")
				}
			}()
		}

		// Wait for Claude to complete
		result := claudeRun.Wait()

		// Stop log tailing for this attempt
		if l.services.LogTailer != nil {
			l.services.LogTailer.StopTailing(agentRun.ID)
		}

		if result.Error != nil && ctx.Err() != nil {
			// Context was canceled
			l.markAgentRunFailed(ctx, agentRun.ID, result.Error.Error())
			l.markSubtaskFailed(ctx, subtask.ID)
			return ctx.Err()
		}

		// Update token usage
		if result.TokenUsage > 0 {
			if err := l.services.SubtaskService.UpdateTokenUsage(ctx, subtask.ID, result.TokenUsage); err != nil {
				log.Error().Err(err).Msg("failed to update subtask token usage")
			}
			//nolint:gosec // TokenUsage is always positive and bounded
			_, _ = l.services.Repo.UpdateAgentRunTokenUsage(ctx, db.UpdateAgentRunTokenUsageParams{
				ID:         agentRun.ID,
				TokenUsage: &[]int32{int32(result.TokenUsage)}[0],
			})
		}

		// Check beads issue status
		if subtask.BeadsIssueID != nil && *subtask.BeadsIssueID != "" {
			issue, err := l.services.BeadsService.ShowIssue(ctx, project.ClonePath, *subtask.BeadsIssueID)
			if err == nil && issue.Status == "closed" {
				// Worker completed successfully
				l.markAgentRunSucceeded(ctx, agentRun.ID)

				// Push branch to remote
				if subtask.BranchName != nil && *subtask.BranchName != "" {
					if err := l.services.GitHubService.PushBranch(ctx, workDir, *subtask.BranchName); err != nil {
						log.Error().Err(err).Msg("failed to push branch")
						// Continue anyway - we'll handle PR creation failure
					}

					// Create PR
					prTitle := fmt.Sprintf("[IV-%s] %s", subtask.ID.String()[:8], subtask.Title)

					// Get commit messages for PR body
					commits, _ := l.services.GitHubService.GetCommitMessages(ctx, workDir, project.DefaultBranch)
					commitList := ""
					for _, c := range commits {
						commitList += fmt.Sprintf("- %s\n", c)
					}

					spec := ""
					if subtask.Spec != nil {
						spec = *subtask.Spec
					}

					prBody := fmt.Sprintf("## Summary\n\n%s\n\n## Commits\n\n%s\n---\n\n:robot: Generated by Intern Village", spec, commitList)

					prInfo, err := l.services.GitHubService.CreatePR(
						ctx,
						project.GitHubOwner,
						project.GitHubRepo,
						userToken,
						*subtask.BranchName,
						project.DefaultBranch,
						prTitle,
						prBody,
					)
					if err != nil {
						log.Error().Err(err).Msg("failed to create PR")
						// Mark as completed without PR
						if err := l.services.SubtaskService.MarkCompleted(ctx, subtask.ID, "", 0); err != nil {
							log.Error().Err(err).Msg("failed to mark subtask as completed")
						}
					} else {
						// Mark as completed with PR info
						if err := l.services.SubtaskService.MarkCompleted(ctx, subtask.ID, prInfo.HTMLURL, prInfo.Number); err != nil {
							log.Error().Err(err).Msg("failed to mark subtask as completed")
						}
					}
				}

				// Publish agent:completed event
				if l.services.EventPublisher != nil {
					now := time.Now()
					prURL := ""
					if subtask.PRUrl != nil {
						prURL = *subtask.PRUrl
					}
					subtaskIDPtr := pgtypeToUUID(agentRun.SubtaskID)
					run := &domain.AgentRun{
						ID:            agentRun.ID,
						SubtaskID:     &subtaskIDPtr,
						AgentType:     domain.AgentTypeWorker,
						AttemptNumber: int(agentRun.AttemptNumber),
						Status:        domain.AgentRunStatusSucceeded,
						StartedAt:     agentRun.StartedAt,
						EndedAt:       &now,
						TokenUsage:    &result.TokenUsage,
					}
					l.services.EventPublisher.PublishAgentCompleted(project.ID, run, subtask.TaskID, prURL)
				}

				log.Info().
					Str("subtask_id", subtask.ID.String()).
					Int("attempt", attempt).
					Msg("worker completed successfully")
				return nil
			}
		}

		// Issue not closed, check exit code
		if result.ExitCode != 0 {
			l.markAgentRunFailed(ctx, agentRun.ID, fmt.Sprintf("exit code: %d", result.ExitCode))
		} else {
			l.markAgentRunFailed(ctx, agentRun.ID, "issue not closed")
		}

		if attempt < l.maxRetries {
			l.backoff(ctx, attempt)
		}
	}

	// Max retries reached, mark subtask as failed
	l.markSubtaskFailed(ctx, subtask.ID)

	log.Warn().
		Str("subtask_id", subtask.ID.String()).
		Int("max_retries", l.maxRetries).
		Msg("worker max retries reached")

	return fmt.Errorf("worker max retries (%d) reached", l.maxRetries)
}

// backoff waits with exponential backoff before the next retry.
// Formula: min(5 * 2^attempt, 120) + jitter (0-20%)
func (l *AgentLoop) backoff(ctx context.Context, attempt int) {
	baseDelay := 5.0           // seconds
	multiplier := 1 << attempt // 2^attempt
	delay := baseDelay * float64(multiplier)
	if delay > 120 {
		delay = 120
	}

	// Add jitter (0-20%)
	jitter := delay * 0.2 * rand.Float64() //nolint:gosec // Non-cryptographic use for backoff jitter
	delay += jitter

	timer := time.NewTimer(time.Duration(delay * float64(time.Second)))
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// markAgentRunSucceeded marks an agent run as succeeded.
func (l *AgentLoop) markAgentRunSucceeded(ctx context.Context, runID uuid.UUID) {
	now := time.Now()
	_, err := l.services.Repo.UpdateAgentRunStatus(ctx, db.UpdateAgentRunStatusParams{
		ID:      runID,
		Status:  string(domain.AgentRunStatusSucceeded),
		EndedAt: repository.PointerToTimestamptz(&now),
	})
	if err != nil {
		log.Error().Err(err).Str("run_id", runID.String()).Msg("failed to mark agent run succeeded")
	}
}

// markAgentRunFailed marks an agent run as failed.
func (l *AgentLoop) markAgentRunFailed(ctx context.Context, runID uuid.UUID, errorMsg string) {
	now := time.Now()
	_, err := l.services.Repo.UpdateAgentRunStatus(ctx, db.UpdateAgentRunStatusParams{
		ID:           runID,
		Status:       string(domain.AgentRunStatusFailed),
		EndedAt:      repository.PointerToTimestamptz(&now),
		ErrorMessage: &errorMsg,
	})
	if err != nil {
		log.Error().Err(err).Str("run_id", runID.String()).Msg("failed to mark agent run failed")
	}
}

// markSubtaskFailed marks a subtask as blocked due to failure.
func (l *AgentLoop) markSubtaskFailed(ctx context.Context, subtaskID uuid.UUID) {
	if err := l.services.SubtaskService.MarkFailed(ctx, subtaskID); err != nil {
		log.Error().Err(err).Str("subtask_id", subtaskID.String()).Msg("failed to mark subtask as failed")
	}
}

// CalculateBackoff calculates the backoff duration for a given attempt.
// Exported for testing.
func CalculateBackoff(attempt int) time.Duration {
	baseDelay := 5.0           // seconds
	multiplier := 1 << attempt // 2^attempt
	delay := baseDelay * float64(multiplier)
	if delay > 120 {
		delay = 120
	}
	return time.Duration(delay * float64(time.Second))
}
