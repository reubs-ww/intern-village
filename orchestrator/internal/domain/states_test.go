// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package domain

import (
	"testing"
)

func TestTaskStatusIsValid(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   bool
	}{
		{TaskStatusPlanning, true},
		{TaskStatusPlanningFailed, true},
		{TaskStatusActive, true},
		{TaskStatusDone, true},
		{TaskStatus("INVALID"), false},
		{TaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("TaskStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestSubtaskStatusIsValid(t *testing.T) {
	tests := []struct {
		status SubtaskStatus
		want   bool
	}{
		{SubtaskStatusPending, true},
		{SubtaskStatusReady, true},
		{SubtaskStatusBlocked, true},
		{SubtaskStatusInProgress, true},
		{SubtaskStatusCompleted, true},
		{SubtaskStatusMerged, true},
		{SubtaskStatus("INVALID"), false},
		{SubtaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("SubtaskStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestBlockedReasonIsValid(t *testing.T) {
	tests := []struct {
		reason BlockedReason
		want   bool
	}{
		{BlockedReasonDependency, true},
		{BlockedReasonFailure, true},
		{BlockedReason("INVALID"), false},
		{BlockedReason(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			if got := tt.reason.IsValid(); got != tt.want {
				t.Errorf("BlockedReason(%q).IsValid() = %v, want %v", tt.reason, got, tt.want)
			}
		})
	}
}

func TestAgentTypeIsValid(t *testing.T) {
	tests := []struct {
		agentType AgentType
		want      bool
	}{
		{AgentTypePlanner, true},
		{AgentTypeWorker, true},
		{AgentType("INVALID"), false},
		{AgentType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			if got := tt.agentType.IsValid(); got != tt.want {
				t.Errorf("AgentType(%q).IsValid() = %v, want %v", tt.agentType, got, tt.want)
			}
		})
	}
}

func TestAgentRunStatusIsValid(t *testing.T) {
	tests := []struct {
		status AgentRunStatus
		want   bool
	}{
		{AgentRunStatusRunning, true},
		{AgentRunStatusSucceeded, true},
		{AgentRunStatusFailed, true},
		{AgentRunStatus("INVALID"), false},
		{AgentRunStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("AgentRunStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestCanTransitionTask(t *testing.T) {
	tests := []struct {
		from TaskStatus
		to   TaskStatus
		want bool
	}{
		// Valid transitions
		{TaskStatusPlanning, TaskStatusActive, true},
		{TaskStatusPlanning, TaskStatusPlanningFailed, true},
		{TaskStatusPlanningFailed, TaskStatusPlanning, true},
		{TaskStatusActive, TaskStatusDone, true},
		// Invalid transitions
		{TaskStatusPlanning, TaskStatusDone, false},
		{TaskStatusActive, TaskStatusPlanning, false},
		{TaskStatusDone, TaskStatusActive, false},
		{TaskStatusDone, TaskStatusPlanning, false},
		{TaskStatusPlanningFailed, TaskStatusActive, false},
		{TaskStatusPlanningFailed, TaskStatusDone, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + " -> " + string(tt.to)
		t.Run(name, func(t *testing.T) {
			if got := CanTransitionTask(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransitionTask(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestCanTransitionSubtask(t *testing.T) {
	tests := []struct {
		from SubtaskStatus
		to   SubtaskStatus
		want bool
	}{
		// Valid transitions
		{SubtaskStatusPending, SubtaskStatusReady, true},
		{SubtaskStatusPending, SubtaskStatusBlocked, true},
		{SubtaskStatusBlocked, SubtaskStatusReady, true},
		{SubtaskStatusReady, SubtaskStatusInProgress, true},
		{SubtaskStatusInProgress, SubtaskStatusCompleted, true},
		{SubtaskStatusInProgress, SubtaskStatusBlocked, true},
		{SubtaskStatusCompleted, SubtaskStatusMerged, true},
		{SubtaskStatusBlocked, SubtaskStatusInProgress, true},
		// Invalid transitions
		{SubtaskStatusPending, SubtaskStatusInProgress, false},
		{SubtaskStatusPending, SubtaskStatusCompleted, false},
		{SubtaskStatusPending, SubtaskStatusMerged, false},
		{SubtaskStatusReady, SubtaskStatusBlocked, false},
		{SubtaskStatusReady, SubtaskStatusCompleted, false},
		{SubtaskStatusReady, SubtaskStatusMerged, false},
		{SubtaskStatusInProgress, SubtaskStatusReady, false},
		{SubtaskStatusInProgress, SubtaskStatusPending, false},
		{SubtaskStatusCompleted, SubtaskStatusReady, false},
		{SubtaskStatusCompleted, SubtaskStatusInProgress, false},
		{SubtaskStatusMerged, SubtaskStatusReady, false},
		{SubtaskStatusMerged, SubtaskStatusCompleted, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + " -> " + string(tt.to)
		t.Run(name, func(t *testing.T) {
			if got := CanTransitionSubtask(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransitionSubtask(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestValidateSubtaskTransition(t *testing.T) {
	dep := BlockedReasonDependency
	failure := BlockedReasonFailure

	tests := []struct {
		name          string
		from          SubtaskStatus
		to            SubtaskStatus
		blockedReason *BlockedReason
		wantErr       bool
	}{
		// Valid transitions
		{"pending to ready", SubtaskStatusPending, SubtaskStatusReady, nil, false},
		{"pending to blocked with reason", SubtaskStatusPending, SubtaskStatusBlocked, &dep, false},
		{"in_progress to blocked with failure", SubtaskStatusInProgress, SubtaskStatusBlocked, &failure, false},
		// Invalid transitions
		{"invalid transition", SubtaskStatusPending, SubtaskStatusMerged, nil, true},
		{"blocked without reason", SubtaskStatusPending, SubtaskStatusBlocked, nil, true},
		{"reason on non-blocked", SubtaskStatusPending, SubtaskStatusReady, &dep, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubtaskTransition(tt.from, tt.to, tt.blockedReason)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSubtaskTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
