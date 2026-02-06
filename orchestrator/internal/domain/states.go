// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package domain

import "fmt"

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	// TaskStatusPlanning indicates the Planner agent is working.
	TaskStatusPlanning TaskStatus = "PLANNING"
	// TaskStatusPlanningFailed indicates the Planner exceeded max retries.
	TaskStatusPlanningFailed TaskStatus = "PLANNING_FAILED"
	// TaskStatusActive indicates planning is complete, subtasks are being worked on.
	TaskStatusActive TaskStatus = "ACTIVE"
	// TaskStatusDone indicates all subtasks are merged.
	TaskStatusDone TaskStatus = "DONE"
)

// IsValid checks if the TaskStatus is a known value.
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusPlanning, TaskStatusPlanningFailed, TaskStatusActive, TaskStatusDone:
		return true
	}
	return false
}

// String returns the string representation of the TaskStatus.
func (s TaskStatus) String() string {
	return string(s)
}

// SubtaskStatus represents the status of a subtask.
type SubtaskStatus string

const (
	// SubtaskStatusPending indicates the subtask was just created.
	SubtaskStatusPending SubtaskStatus = "PENDING"
	// SubtaskStatusReady indicates the subtask has no unmerged dependencies.
	SubtaskStatusReady SubtaskStatus = "READY"
	// SubtaskStatusBlocked indicates the subtask is blocked.
	SubtaskStatusBlocked SubtaskStatus = "BLOCKED"
	// SubtaskStatusInProgress indicates the Worker agent is working.
	SubtaskStatusInProgress SubtaskStatus = "IN_PROGRESS"
	// SubtaskStatusCompleted indicates the Worker succeeded and PR was created.
	SubtaskStatusCompleted SubtaskStatus = "COMPLETED"
	// SubtaskStatusMerged indicates the PR was merged.
	SubtaskStatusMerged SubtaskStatus = "MERGED"
)

// IsValid checks if the SubtaskStatus is a known value.
func (s SubtaskStatus) IsValid() bool {
	switch s {
	case SubtaskStatusPending, SubtaskStatusReady, SubtaskStatusBlocked,
		SubtaskStatusInProgress, SubtaskStatusCompleted, SubtaskStatusMerged:
		return true
	}
	return false
}

// String returns the string representation of the SubtaskStatus.
func (s SubtaskStatus) String() string {
	return string(s)
}

// BlockedReason indicates why a subtask is blocked.
type BlockedReason string

const (
	// BlockedReasonDependency indicates waiting for dependencies to be merged.
	BlockedReasonDependency BlockedReason = "DEPENDENCY"
	// BlockedReasonFailure indicates the agent failed after max retries.
	BlockedReasonFailure BlockedReason = "FAILURE"
)

// IsValid checks if the BlockedReason is a known value.
func (r BlockedReason) IsValid() bool {
	switch r {
	case BlockedReasonDependency, BlockedReasonFailure:
		return true
	}
	return false
}

// String returns the string representation of the BlockedReason.
func (r BlockedReason) String() string {
	return string(r)
}

// AgentType represents the type of agent.
type AgentType string

const (
	// AgentTypePlanner is the planning agent that creates subtasks.
	AgentTypePlanner AgentType = "PLANNER"
	// AgentTypeWorker is the implementation agent that codes.
	AgentTypeWorker AgentType = "WORKER"
)

// IsValid checks if the AgentType is a known value.
func (t AgentType) IsValid() bool {
	switch t {
	case AgentTypePlanner, AgentTypeWorker:
		return true
	}
	return false
}

// String returns the string representation of the AgentType.
func (t AgentType) String() string {
	return string(t)
}

// AgentRunStatus represents the status of an agent run.
type AgentRunStatus string

const (
	// AgentRunStatusRunning indicates the agent is currently executing.
	AgentRunStatusRunning AgentRunStatus = "RUNNING"
	// AgentRunStatusSucceeded indicates the agent completed successfully.
	AgentRunStatusSucceeded AgentRunStatus = "SUCCEEDED"
	// AgentRunStatusFailed indicates the agent failed.
	AgentRunStatusFailed AgentRunStatus = "FAILED"
)

// IsValid checks if the AgentRunStatus is a known value.
func (s AgentRunStatus) IsValid() bool {
	switch s {
	case AgentRunStatusRunning, AgentRunStatusSucceeded, AgentRunStatusFailed:
		return true
	}
	return false
}

// String returns the string representation of the AgentRunStatus.
func (s AgentRunStatus) String() string {
	return string(s)
}

// TaskTransition represents a valid state transition for tasks.
type TaskTransition struct {
	From TaskStatus
	To   TaskStatus
}

// ValidTaskTransitions defines all valid task state transitions.
var ValidTaskTransitions = []TaskTransition{
	{TaskStatusPlanning, TaskStatusActive},         // Planner completes successfully
	{TaskStatusPlanning, TaskStatusPlanningFailed}, // Planner fails after max retries
	{TaskStatusPlanningFailed, TaskStatusPlanning}, // User retries planning
	{TaskStatusActive, TaskStatusDone},             // All subtasks merged
}

// CanTransitionTask checks if a task can transition from one status to another.
func CanTransitionTask(from, to TaskStatus) bool {
	for _, t := range ValidTaskTransitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}

// SubtaskTransition represents a valid state transition for subtasks.
type SubtaskTransition struct {
	From          SubtaskStatus
	To            SubtaskStatus
	BlockedReason *BlockedReason // Only relevant when transitioning to BLOCKED
}

// ValidSubtaskTransitions defines all valid subtask state transitions.
var ValidSubtaskTransitions = []SubtaskTransition{
	{SubtaskStatusPending, SubtaskStatusReady, nil},                            // No dependencies
	{SubtaskStatusPending, SubtaskStatusBlocked, ptr(BlockedReasonDependency)}, // Has dependencies
	{SubtaskStatusBlocked, SubtaskStatusReady, nil},                            // Dependencies merged (was DEPENDENCY blocked)
	{SubtaskStatusReady, SubtaskStatusInProgress, nil},                         // User starts subtask
	{SubtaskStatusInProgress, SubtaskStatusCompleted, nil},                     // Worker succeeds
	{SubtaskStatusInProgress, SubtaskStatusBlocked, ptr(BlockedReasonFailure)}, // Worker fails after max retries
	{SubtaskStatusCompleted, SubtaskStatusMerged, nil},                         // User marks merged
	{SubtaskStatusBlocked, SubtaskStatusInProgress, nil},                       // User retries (was FAILURE blocked)
}

func ptr(r BlockedReason) *BlockedReason {
	return &r
}

// CanTransitionSubtask checks if a subtask can transition from one status to another.
func CanTransitionSubtask(from, to SubtaskStatus) bool {
	for _, t := range ValidSubtaskTransitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}

// ValidateSubtaskTransition validates a subtask transition and returns an error if invalid.
func ValidateSubtaskTransition(from, to SubtaskStatus, blockedReason *BlockedReason) error {
	if !CanTransitionSubtask(from, to) {
		return fmt.Errorf("invalid transition from %s to %s", from, to)
	}

	// Check blocked reason requirements
	if to == SubtaskStatusBlocked && blockedReason == nil {
		return fmt.Errorf("blocked reason required when transitioning to BLOCKED")
	}
	if to != SubtaskStatusBlocked && blockedReason != nil {
		return fmt.Errorf("blocked reason should not be set for non-BLOCKED status")
	}

	return nil
}
