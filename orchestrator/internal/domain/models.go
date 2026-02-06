// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

// Package domain contains core domain types and business logic.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a GitHub-authenticated user.
type User struct {
	ID             uuid.UUID `json:"id"`
	GitHubID       int64     `json:"github_id"`
	GitHubUsername string    `json:"github_username"`
	GitHubToken    string    `json:"-"` // Encrypted, never exposed in JSON
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Project represents a GitHub repository the user works on.
type Project struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	GitHubOwner   string    `json:"github_owner"`
	GitHubRepo    string    `json:"github_repo"`
	IsFork        bool      `json:"is_fork"`
	UpstreamOwner *string   `json:"upstream_owner,omitempty"` // Original repo owner (only for forks)
	UpstreamRepo  *string   `json:"upstream_repo,omitempty"`  // Original repo name (only for forks)
	DefaultBranch string    `json:"default_branch"`
	ClonePath     string    `json:"clone_path"`
	BeadsPrefix   string    `json:"beads_prefix"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Task represents a user-submitted work item.
type Task struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	BeadsEpicID *string    `json:"beads_epic_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Subtask represents a broken-down work unit.
type Subtask struct {
	ID                 uuid.UUID      `json:"id"`
	TaskID             uuid.UUID      `json:"task_id"`
	Title              string         `json:"title"`
	Spec               *string        `json:"spec,omitempty"`
	ImplementationPlan *string        `json:"implementation_plan,omitempty"`
	Status             SubtaskStatus  `json:"status"`
	BlockedReason      *BlockedReason `json:"blocked_reason,omitempty"`
	BranchName         *string        `json:"branch_name,omitempty"`
	PRUrl              *string        `json:"pr_url,omitempty"`
	PRNumber           *int           `json:"pr_number,omitempty"`
	RetryCount         int            `json:"retry_count"`
	TokenUsage         int            `json:"token_usage"`
	Position           int            `json:"position"`
	BeadsIssueID       *string        `json:"beads_issue_id,omitempty"`
	WorktreePath       *string        `json:"worktree_path,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// SubtaskDependency tracks which subtasks block others.
type SubtaskDependency struct {
	ID          uuid.UUID `json:"id"`
	SubtaskID   uuid.UUID `json:"subtask_id"`
	DependsOnID uuid.UUID `json:"depends_on_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// AgentRun represents history of agent executions.
// For Planner agents: TaskID is set, SubtaskID is nil
// For Worker agents: SubtaskID is set, TaskID can be derived from subtask
type AgentRun struct {
	ID            uuid.UUID      `json:"id"`
	SubtaskID     *uuid.UUID     `json:"subtask_id,omitempty"` // nil for Planner runs
	TaskID        *uuid.UUID     `json:"task_id,omitempty"`    // set for Planner runs
	AgentType     AgentType      `json:"agent_type"`
	AttemptNumber int            `json:"attempt_number"`
	Status        AgentRunStatus `json:"status"`
	StartedAt     time.Time      `json:"started_at"`
	EndedAt       *time.Time     `json:"ended_at,omitempty"`
	TokenUsage    *int           `json:"token_usage,omitempty"`
	ErrorMessage  *string        `json:"error_message,omitempty"`
	LogPath       string         `json:"log_path"`
	PromptText    string         `json:"prompt_text"`
	CreatedAt     time.Time      `json:"created_at"`
}

// NewUser creates a new User with a generated UUID.
func NewUser(githubID int64, githubUsername, encryptedToken string) *User {
	now := time.Now()
	return &User{
		ID:             uuid.New(),
		GitHubID:       githubID,
		GitHubUsername: githubUsername,
		GitHubToken:    encryptedToken,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewProject creates a new Project with a generated UUID.
func NewProject(userID uuid.UUID, owner, repo string, isFork bool, defaultBranch, clonePath, beadsPrefix string) *Project {
	now := time.Now()
	return &Project{
		ID:            uuid.New(),
		UserID:        userID,
		GitHubOwner:   owner,
		GitHubRepo:    repo,
		IsFork:        isFork,
		DefaultBranch: defaultBranch,
		ClonePath:     clonePath,
		BeadsPrefix:   beadsPrefix,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// NewProjectWithUpstream creates a new Project with upstream info (for forks).
func NewProjectWithUpstream(userID uuid.UUID, owner, repo string, isFork bool, upstreamOwner, upstreamRepo, defaultBranch, clonePath, beadsPrefix string) *Project {
	now := time.Now()
	return &Project{
		ID:            uuid.New(),
		UserID:        userID,
		GitHubOwner:   owner,
		GitHubRepo:    repo,
		IsFork:        isFork,
		UpstreamOwner: &upstreamOwner,
		UpstreamRepo:  &upstreamRepo,
		DefaultBranch: defaultBranch,
		ClonePath:     clonePath,
		BeadsPrefix:   beadsPrefix,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// NewTask creates a new Task with a generated UUID.
func NewTask(projectID uuid.UUID, title, description string) *Task {
	now := time.Now()
	return &Task{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Status:      TaskStatusPlanning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewSubtask creates a new Subtask with a generated UUID.
func NewSubtask(taskID uuid.UUID, title string, position int) *Subtask {
	now := time.Now()
	return &Subtask{
		ID:         uuid.New(),
		TaskID:     taskID,
		Title:      title,
		Status:     SubtaskStatusPending,
		RetryCount: 0,
		TokenUsage: 0,
		Position:   position,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewAgentRun creates a new AgentRun for Worker agents with a generated UUID.
func NewAgentRun(subtaskID uuid.UUID, agentType AgentType, attemptNumber int, logPath, promptText string) *AgentRun {
	now := time.Now()
	return &AgentRun{
		ID:            uuid.New(),
		SubtaskID:     &subtaskID,
		AgentType:     agentType,
		AttemptNumber: attemptNumber,
		Status:        AgentRunStatusRunning,
		StartedAt:     now,
		LogPath:       logPath,
		PromptText:    promptText,
		CreatedAt:     now,
	}
}

// NewAgentRunForTask creates a new AgentRun for Planner agents (task-level) with a generated UUID.
func NewAgentRunForTask(taskID uuid.UUID, agentType AgentType, attemptNumber int, logPath, promptText string) *AgentRun {
	now := time.Now()
	return &AgentRun{
		ID:            uuid.New(),
		TaskID:        &taskID,
		AgentType:     agentType,
		AttemptNumber: attemptNumber,
		Status:        AgentRunStatusRunning,
		StartedAt:     now,
		LogPath:       logPath,
		PromptText:    promptText,
		CreatedAt:     now,
	}
}
