// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"testing"

	"github.com/intern-village/orchestrator/internal/domain"
)

func TestDbTaskToDomain(t *testing.T) {
	// This tests the conversion function
	// In real usage, this would be tested with actual DB fixtures
}

func TestTaskStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     domain.TaskStatus
		to       domain.TaskStatus
		expected bool
	}{
		{
			name:     "planning to active is valid",
			from:     domain.TaskStatusPlanning,
			to:       domain.TaskStatusActive,
			expected: true,
		},
		{
			name:     "planning to planning_failed is valid",
			from:     domain.TaskStatusPlanning,
			to:       domain.TaskStatusPlanningFailed,
			expected: true,
		},
		{
			name:     "planning_failed to planning is valid (retry)",
			from:     domain.TaskStatusPlanningFailed,
			to:       domain.TaskStatusPlanning,
			expected: true,
		},
		{
			name:     "active to done is valid",
			from:     domain.TaskStatusActive,
			to:       domain.TaskStatusDone,
			expected: true,
		},
		{
			name:     "planning to done is invalid",
			from:     domain.TaskStatusPlanning,
			to:       domain.TaskStatusDone,
			expected: false,
		},
		{
			name:     "done to active is invalid",
			from:     domain.TaskStatusDone,
			to:       domain.TaskStatusActive,
			expected: false,
		},
		{
			name:     "active to planning is invalid",
			from:     domain.TaskStatusActive,
			to:       domain.TaskStatusPlanning,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.CanTransitionTask(tt.from, tt.to)
			if got != tt.expected {
				t.Errorf("CanTransitionTask(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.expected)
			}
		})
	}
}

func TestSubtaskStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     domain.SubtaskStatus
		to       domain.SubtaskStatus
		expected bool
	}{
		{
			name:     "pending to ready is valid",
			from:     domain.SubtaskStatusPending,
			to:       domain.SubtaskStatusReady,
			expected: true,
		},
		{
			name:     "pending to blocked is valid",
			from:     domain.SubtaskStatusPending,
			to:       domain.SubtaskStatusBlocked,
			expected: true,
		},
		{
			name:     "ready to in_progress is valid",
			from:     domain.SubtaskStatusReady,
			to:       domain.SubtaskStatusInProgress,
			expected: true,
		},
		{
			name:     "blocked to ready is valid (unblock)",
			from:     domain.SubtaskStatusBlocked,
			to:       domain.SubtaskStatusReady,
			expected: true,
		},
		{
			name:     "blocked to in_progress is valid (retry)",
			from:     domain.SubtaskStatusBlocked,
			to:       domain.SubtaskStatusInProgress,
			expected: true,
		},
		{
			name:     "in_progress to completed is valid",
			from:     domain.SubtaskStatusInProgress,
			to:       domain.SubtaskStatusCompleted,
			expected: true,
		},
		{
			name:     "in_progress to blocked is valid (failure)",
			from:     domain.SubtaskStatusInProgress,
			to:       domain.SubtaskStatusBlocked,
			expected: true,
		},
		{
			name:     "completed to merged is valid",
			from:     domain.SubtaskStatusCompleted,
			to:       domain.SubtaskStatusMerged,
			expected: true,
		},
		{
			name:     "pending to in_progress is invalid",
			from:     domain.SubtaskStatusPending,
			to:       domain.SubtaskStatusInProgress,
			expected: false,
		},
		{
			name:     "ready to completed is invalid",
			from:     domain.SubtaskStatusReady,
			to:       domain.SubtaskStatusCompleted,
			expected: false,
		},
		{
			name:     "merged to ready is invalid",
			from:     domain.SubtaskStatusMerged,
			to:       domain.SubtaskStatusReady,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.CanTransitionSubtask(tt.from, tt.to)
			if got != tt.expected {
				t.Errorf("CanTransitionSubtask(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.expected)
			}
		})
	}
}

func TestValidateSubtaskTransition(t *testing.T) {
	depReason := domain.BlockedReasonDependency
	failReason := domain.BlockedReasonFailure

	tests := []struct {
		name          string
		from          domain.SubtaskStatus
		to            domain.SubtaskStatus
		blockedReason *domain.BlockedReason
		expectError   bool
	}{
		{
			name:          "valid transition with no blocked reason",
			from:          domain.SubtaskStatusReady,
			to:            domain.SubtaskStatusInProgress,
			blockedReason: nil,
			expectError:   false,
		},
		{
			name:          "transition to blocked requires reason",
			from:          domain.SubtaskStatusPending,
			to:            domain.SubtaskStatusBlocked,
			blockedReason: nil,
			expectError:   true,
		},
		{
			name:          "transition to blocked with reason is valid",
			from:          domain.SubtaskStatusPending,
			to:            domain.SubtaskStatusBlocked,
			blockedReason: &depReason,
			expectError:   false,
		},
		{
			name:          "non-blocked transition with reason is invalid",
			from:          domain.SubtaskStatusReady,
			to:            domain.SubtaskStatusInProgress,
			blockedReason: &failReason,
			expectError:   true,
		},
		{
			name:          "invalid transition returns error",
			from:          domain.SubtaskStatusPending,
			to:            domain.SubtaskStatusCompleted,
			blockedReason: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateSubtaskTransition(tt.from, tt.to, tt.blockedReason)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateSubtaskTransition(%s, %s, %v) error = %v, expectError %v",
					tt.from, tt.to, tt.blockedReason, err, tt.expectError)
			}
		})
	}
}
