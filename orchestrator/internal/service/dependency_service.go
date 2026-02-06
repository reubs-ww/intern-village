// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/intern-village/orchestrator/generated/db"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
)

// DependencyService handles subtask dependency operations.
type DependencyService struct {
	repo     *repository.Repository
	eventHub EventHub
}

// NewDependencyService creates a new DependencyService.
func NewDependencyService(repo *repository.Repository, eventHub EventHub) *DependencyService {
	return &DependencyService{
		repo:     repo,
		eventHub: eventHub,
	}
}

// AddDependency creates a dependency between subtasks.
// The subtaskID is blocked by dependsOnID.
func (s *DependencyService) AddDependency(ctx context.Context, subtaskID, dependsOnID uuid.UUID) (*domain.SubtaskDependency, error) {
	// Validate that both subtasks exist
	_, err := s.repo.GetSubtaskByID(ctx, subtaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}

	_, err = s.repo.GetSubtaskByID(ctx, dependsOnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency subtask: %w", err)
	}

	// Prevent self-dependency
	if subtaskID == dependsOnID {
		return nil, domain.NewValidationError("subtask_id", "subtask cannot depend on itself")
	}

	// Create the dependency (ON CONFLICT DO NOTHING handles duplicates)
	dep, err := s.repo.CreateDependency(ctx, db.CreateDependencyParams{
		SubtaskID:   subtaskID,
		DependsOnID: dependsOnID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dependency: %w", err)
	}

	return &domain.SubtaskDependency{
		ID:          dep.ID,
		SubtaskID:   dep.SubtaskID,
		DependsOnID: dep.DependsOnID,
		CreatedAt:   dep.CreatedAt,
	}, nil
}

// GetBlockingDependencies returns all dependencies that are blocking a subtask.
// A dependency is blocking if its status is not MERGED.
func (s *DependencyService) GetBlockingDependencies(ctx context.Context, subtaskID uuid.UUID) ([]db.GetDependenciesForSubtaskRow, error) {
	deps, err := s.repo.GetDependenciesForSubtask(ctx, subtaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	// Filter to only blocking (non-MERGED) dependencies
	var blocking []db.GetDependenciesForSubtaskRow
	for _, dep := range deps {
		if dep.DependencyStatus != string(domain.SubtaskStatusMerged) {
			blocking = append(blocking, dep)
		}
	}

	return blocking, nil
}

// HasBlockingDependencies checks if a subtask has any unmerged dependencies.
func (s *DependencyService) HasBlockingDependencies(ctx context.Context, subtaskID uuid.UUID) (bool, error) {
	return s.repo.HasBlockingDependencies(ctx, subtaskID)
}

// UnblockDependents checks all subtasks that depend on the given subtask
// and unblocks them if all their dependencies are now merged.
// Returns the list of subtask IDs that were unblocked.
func (s *DependencyService) UnblockDependents(ctx context.Context, subtaskID uuid.UUID) ([]uuid.UUID, error) {
	// Get all subtasks that depend on this one
	dependents, err := s.repo.GetDependentsOfSubtask(ctx, subtaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependents: %w", err)
	}

	var unblocked []uuid.UUID

	for _, dep := range dependents {
		// Only check subtasks that are currently BLOCKED with DEPENDENCY reason
		if dep.DependentStatus != string(domain.SubtaskStatusBlocked) {
			continue
		}

		// Check if all dependencies are now merged
		hasBlocking, err := s.repo.HasBlockingDependencies(ctx, dep.SubtaskID)
		if err != nil {
			return nil, fmt.Errorf("failed to check dependencies for %s: %w", dep.SubtaskID, err)
		}

		if !hasBlocking {
			// Transition from BLOCKED to READY
			_, err := s.repo.UpdateSubtaskStatus(ctx, db.UpdateSubtaskStatusParams{
				ID:            dep.SubtaskID,
				Status:        string(domain.SubtaskStatusReady),
				BlockedReason: nil,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to unblock subtask %s: %w", dep.SubtaskID, err)
			}
			unblocked = append(unblocked, dep.SubtaskID)

			// Publish subtask:unblocked event
			if s.eventHub != nil {
				// Get subtask to find task ID, then task to find project ID
				subtask, err := s.repo.GetSubtaskByID(ctx, dep.SubtaskID)
				if err == nil {
					task, err := s.repo.GetTaskByID(ctx, subtask.TaskID)
					if err == nil {
						s.eventHub.PublishSubtaskUnblocked(task.ProjectID, subtask.TaskID, dep.SubtaskID, subtaskID)
					}
				}
			}
		}
	}

	return unblocked, nil
}

// DetermineInitialStatus determines whether a subtask should be READY or BLOCKED
// based on its dependencies.
func (s *DependencyService) DetermineInitialStatus(ctx context.Context, subtaskID uuid.UUID) (domain.SubtaskStatus, *domain.BlockedReason, error) {
	hasBlocking, err := s.repo.HasBlockingDependencies(ctx, subtaskID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to check dependencies: %w", err)
	}

	if hasBlocking {
		reason := domain.BlockedReasonDependency
		return domain.SubtaskStatusBlocked, &reason, nil
	}

	return domain.SubtaskStatusReady, nil, nil
}

// GetDependencies returns all dependencies for a subtask.
func (s *DependencyService) GetDependencies(ctx context.Context, subtaskID uuid.UUID) ([]db.GetDependenciesForSubtaskRow, error) {
	return s.repo.GetDependenciesForSubtask(ctx, subtaskID)
}

// DeleteDependency removes a dependency between subtasks.
func (s *DependencyService) DeleteDependency(ctx context.Context, subtaskID, dependsOnID uuid.UUID) error {
	return s.repo.DeleteDependency(ctx, db.DeleteDependencyParams{
		SubtaskID:   subtaskID,
		DependsOnID: dependsOnID,
	})
}

// DeleteDependenciesForSubtask removes all dependencies for a subtask.
func (s *DependencyService) DeleteDependenciesForSubtask(ctx context.Context, subtaskID uuid.UUID) error {
	return s.repo.DeleteDependenciesForSubtask(ctx, subtaskID)
}
