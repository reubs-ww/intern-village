// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/intern-village/orchestrator/generated/db"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
)

// SyncService synchronizes Beads state to Postgres.
// Beads is the source of truth for dependencies and agent state.
type SyncService struct {
	repo              *repository.Repository
	beadsService      *BeadsService
	subtaskService    *SubtaskService
	dependencyService *DependencyService
	taskService       *TaskService
}

// NewSyncService creates a new SyncService.
func NewSyncService(
	repo *repository.Repository,
	beadsService *BeadsService,
	subtaskService *SubtaskService,
	dependencyService *DependencyService,
	taskService *TaskService,
) *SyncService {
	return &SyncService{
		repo:              repo,
		beadsService:      beadsService,
		subtaskService:    subtaskService,
		dependencyService: dependencyService,
		taskService:       taskService,
	}
}

// SyncTaskFromBeads syncs all subtasks for a task from Beads.
// This is called after the Planner agent completes.
func (s *SyncService) SyncTaskFromBeads(ctx context.Context, taskID uuid.UUID, repoPath string) error {
	// Get the task
	task, err := s.taskService.GetTaskByIDInternal(ctx, taskID)
	if err != nil {
		return err
	}

	if task.BeadsEpicID == nil || *task.BeadsEpicID == "" {
		return fmt.Errorf("task %s has no beads epic ID", taskID)
	}

	// List all issues under the epic
	issues, err := s.beadsService.ListIssues(ctx, repoPath, *task.BeadsEpicID)
	if err != nil {
		return fmt.Errorf("failed to list issues from beads: %w", err)
	}

	// Track created subtask IDs for dependency sync
	beadsIDToSubtaskID := make(map[string]uuid.UUID)

	// Create or update subtasks
	for _, issue := range issues {
		subtask, err := s.syncIssueToSubtask(ctx, taskID, issue)
		if err != nil {
			return fmt.Errorf("failed to sync issue %s: %w", issue.ID, err)
		}
		beadsIDToSubtaskID[issue.ID] = subtask.ID
	}

	// Sync dependencies
	for _, issue := range issues {
		// Get blocking dependencies (not parent-child)
		depIDs := issue.GetDependencyIDs()
		if len(depIDs) > 0 {
			subtaskID := beadsIDToSubtaskID[issue.ID]
			for _, depID := range depIDs {
				if depSubtaskID, ok := beadsIDToSubtaskID[depID]; ok {
					_, err := s.dependencyService.AddDependency(ctx, subtaskID, depSubtaskID)
					if err != nil {
						// Log but don't fail - might be duplicate
						fmt.Printf("warning: failed to add dependency %s -> %s: %v\n", issue.ID, depID, err)
					}
				}
			}
		}
	}

	// Determine initial status for each subtask
	for _, subtaskID := range beadsIDToSubtaskID {
		status, reason, err := s.dependencyService.DetermineInitialStatus(ctx, subtaskID)
		if err != nil {
			return fmt.Errorf("failed to determine status for subtask %s: %w", subtaskID, err)
		}
		if err := s.subtaskService.UpdateSubtaskStatus(ctx, subtaskID, status, reason); err != nil {
			return fmt.Errorf("failed to update subtask status %s: %w", subtaskID, err)
		}
	}

	return nil
}

// syncIssueToSubtask creates or updates a subtask from a Beads issue.
func (s *SyncService) syncIssueToSubtask(ctx context.Context, taskID uuid.UUID, issue BeadsIssue) (*domain.Subtask, error) {
	// Check if subtask already exists
	existing, err := s.subtaskService.GetSubtaskByBeadsID(ctx, issue.ID)
	if err == nil {
		// Update existing subtask
		// For now, we don't update existing subtasks - they're managed by the agent
		return existing, nil
	}

	if !domain.IsNotFound(err) {
		return nil, err
	}

	// Parse spec and implementation plan from description
	spec, plan := parseIssueBody(issue.Description)

	// Create new subtask
	return s.subtaskService.CreateSubtask(ctx, CreateSubtaskInput{
		TaskID:             taskID,
		Title:              issue.Title,
		Spec:               &spec,
		ImplementationPlan: &plan,
		BeadsIssueID:       &issue.ID,
	})
}

// SyncSubtaskFromBeads syncs a single subtask from Beads.
// This is called during the agent loop to check if the issue is closed.
func (s *SyncService) SyncSubtaskFromBeads(ctx context.Context, subtaskID uuid.UUID, repoPath string) error {
	subtask, err := s.subtaskService.GetSubtaskByIDInternal(ctx, subtaskID)
	if err != nil {
		return err
	}

	if subtask.BeadsIssueID == nil {
		return fmt.Errorf("subtask %s has no beads issue ID", subtaskID)
	}

	issue, err := s.beadsService.ShowIssue(ctx, repoPath, *subtask.BeadsIssueID)
	if err != nil {
		return fmt.Errorf("failed to get issue from beads: %w", err)
	}

	// Check if issue is closed
	if issue.Status == "closed" {
		// The agent loop will handle transitioning to COMPLETED
		return nil
	}

	return nil
}

// SyncDependencies syncs all dependencies for subtasks under a task.
func (s *SyncService) SyncDependencies(ctx context.Context, taskID uuid.UUID, repoPath string) error {
	// Get all subtasks for this task
	subtasks, err := s.repo.ListSubtasksByTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to list subtasks: %w", err)
	}

	for _, subtask := range subtasks {
		if subtask.BeadsIssueID == nil {
			continue
		}

		// Get dependencies from beads
		deps, err := s.beadsService.GetDependencies(ctx, repoPath, *subtask.BeadsIssueID)
		if err != nil {
			return fmt.Errorf("failed to get dependencies for %s: %w", *subtask.BeadsIssueID, err)
		}

		// Find subtask IDs for each dependency
		for _, depBeadsID := range deps {
			depSubtask, err := s.repo.GetSubtaskByBeadsID(ctx, &depBeadsID)
			if err != nil {
				// Dependency might not be synced yet
				continue
			}

			_, err = s.dependencyService.AddDependency(ctx, subtask.ID, depSubtask.ID)
			if err != nil {
				// Ignore duplicate errors
				continue
			}
		}
	}

	return nil
}

// IsBeadsIssueClosed checks if a beads issue is closed.
func (s *SyncService) IsBeadsIssueClosed(ctx context.Context, repoPath, issueID string) (bool, error) {
	issue, err := s.beadsService.ShowIssue(ctx, repoPath, issueID)
	if err != nil {
		return false, err
	}
	return issue.Status == "closed", nil
}

// GetInProgressSubtasks returns all subtasks that are currently in progress.
func (s *SyncService) GetInProgressSubtasks(ctx context.Context) ([]db.Subtask, error) {
	return s.repo.ListInProgressSubtasks(ctx)
}

// parseIssueBody extracts spec and implementation plan from the issue body.
// The expected format is:
// ## Spec
// <spec content>
// ## Implementation Plan
// <plan content>
func parseIssueBody(body string) (spec, plan string) {
	// Simple parsing - look for markdown headers
	lines := strings.Split(body, "\n")
	var currentSection string
	var specLines, planLines []string

	for _, line := range lines {
		lowerLine := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lowerLine, "## spec") || strings.HasPrefix(lowerLine, "# spec") {
			currentSection = "spec"
			continue
		}
		if strings.HasPrefix(lowerLine, "## implementation") || strings.HasPrefix(lowerLine, "# implementation") {
			currentSection = "plan"
			continue
		}
		if strings.HasPrefix(lowerLine, "## acceptance") || strings.HasPrefix(lowerLine, "# acceptance") {
			// End of plan section
			currentSection = ""
			continue
		}

		switch currentSection {
		case "spec":
			specLines = append(specLines, line)
		case "plan":
			planLines = append(planLines, line)
		}
	}

	spec = strings.TrimSpace(strings.Join(specLines, "\n"))
	plan = strings.TrimSpace(strings.Join(planLines, "\n"))

	// If no structured format, use the whole body as spec
	if spec == "" && plan == "" {
		spec = strings.TrimSpace(body)
	}

	return spec, plan
}
