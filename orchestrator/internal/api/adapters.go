// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package api

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/intern-village/orchestrator/internal/agent"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/service"
)

// beadsServiceAdapter adapts service.BeadsService to agent.BeadsServiceInterface.
type beadsServiceAdapter struct {
	svc *service.BeadsService
}

func newBeadsServiceAdapter(svc *service.BeadsService) agent.BeadsServiceInterface {
	return &beadsServiceAdapter{svc: svc}
}

func (a *beadsServiceAdapter) ShowIssue(ctx context.Context, repoPath, issueID string) (*agent.BeadsIssue, error) {
	issue, err := a.svc.ShowIssue(ctx, repoPath, issueID)
	if err != nil {
		return nil, err
	}
	return convertBeadsIssue(issue), nil
}

// convertBeadsIssue converts a service.BeadsIssue to an agent.BeadsIssue.
func convertBeadsIssue(issue *service.BeadsIssue) *agent.BeadsIssue {
	if issue == nil {
		return nil
	}
	deps := make([]agent.BeadsDependency, len(issue.Dependencies))
	for i, d := range issue.Dependencies {
		deps[i] = agent.BeadsDependency{
			IssueID:     d.IssueID,
			DependsOnID: d.DependsOnID,
			Type:        d.Type,
		}
	}
	return &agent.BeadsIssue{
		ID:           issue.ID,
		Type:         issue.Type,
		Title:        issue.Title,
		Description:  issue.Description,
		Status:       issue.Status,
		ParentID:     issue.ParentID,
		Dependencies: deps,
	}
}

func (a *beadsServiceAdapter) CloseIssue(ctx context.Context, repoPath, issueID, reason string) error {
	return a.svc.CloseIssue(ctx, repoPath, issueID, reason)
}

func (a *beadsServiceAdapter) FindEpicByTaskID(ctx context.Context, repoPath, taskIDPrefix string) (*agent.BeadsIssue, error) {
	issue, err := a.svc.FindEpicByTaskID(ctx, repoPath, taskIDPrefix)
	if err != nil {
		return nil, err
	}
	return convertBeadsIssue(issue), nil
}

// gitHubServiceAdapter adapts service.GitHubService to agent.GitHubServiceInterface.
type gitHubServiceAdapter struct {
	svc *service.GitHubService
}

func newGitHubServiceAdapter(svc *service.GitHubService) agent.GitHubServiceInterface {
	return &gitHubServiceAdapter{svc: svc}
}

func (a *gitHubServiceAdapter) PushBranch(ctx context.Context, repoPath, branch string) error {
	return a.svc.PushBranch(ctx, repoPath, branch)
}

func (a *gitHubServiceAdapter) CreatePR(ctx context.Context, owner, repo, accessToken, head, base, title, body string) (*agent.PRInfo, error) {
	prInfo, err := a.svc.CreatePR(ctx, owner, repo, accessToken, head, base, title, body)
	if err != nil {
		return nil, err
	}
	return &agent.PRInfo{
		Number:  prInfo.Number,
		URL:     prInfo.URL,
		HTMLURL: prInfo.HTMLURL,
	}, nil
}

func (a *gitHubServiceAdapter) GetCommitMessages(ctx context.Context, repoPath, baseBranch string) ([]string, error) {
	return a.svc.GetCommitMessages(ctx, repoPath, baseBranch)
}

// syncServiceAdapter adapts service.SyncService to agent.SyncServiceInterface.
type syncServiceAdapter struct {
	svc *service.SyncService
}

func newSyncServiceAdapter(svc *service.SyncService) agent.SyncServiceInterface {
	return &syncServiceAdapter{svc: svc}
}

func (a *syncServiceAdapter) SyncTaskFromBeads(ctx context.Context, taskID uuid.UUID, repoPath string) error {
	return a.svc.SyncTaskFromBeads(ctx, taskID, repoPath)
}

// taskServiceAdapter adapts service.TaskService to agent.TaskServiceInterface.
type taskServiceAdapter struct {
	svc *service.TaskService
}

func newTaskServiceAdapter(svc *service.TaskService) agent.TaskServiceInterface {
	return &taskServiceAdapter{svc: svc}
}

func (a *taskServiceAdapter) TransitionToActive(ctx context.Context, taskID uuid.UUID) error {
	return a.svc.TransitionToActive(ctx, taskID)
}

func (a *taskServiceAdapter) MarkPlanningFailed(ctx context.Context, taskID uuid.UUID) error {
	return a.svc.MarkPlanningFailed(ctx, taskID)
}

func (a *taskServiceAdapter) UpdateBeadsEpicID(ctx context.Context, taskID uuid.UUID, epicID string) error {
	return a.svc.UpdateBeadsEpicID(ctx, taskID, epicID)
}

// subtaskServiceAdapter adapts service.SubtaskService to agent.SubtaskServiceInterface.
type subtaskServiceAdapter struct {
	svc *service.SubtaskService
}

func newSubtaskServiceAdapter(svc *service.SubtaskService) agent.SubtaskServiceInterface {
	return &subtaskServiceAdapter{svc: svc}
}

func (a *subtaskServiceAdapter) MarkCompleted(ctx context.Context, subtaskID uuid.UUID, prURL string, prNumber int) error {
	return a.svc.MarkCompleted(ctx, subtaskID, prURL, prNumber)
}

func (a *subtaskServiceAdapter) MarkFailed(ctx context.Context, subtaskID uuid.UUID) error {
	return a.svc.MarkFailed(ctx, subtaskID)
}

func (a *subtaskServiceAdapter) IncrementRetryCount(ctx context.Context, subtaskID uuid.UUID) (int, error) {
	return a.svc.IncrementRetryCount(ctx, subtaskID)
}

func (a *subtaskServiceAdapter) UpdateTokenUsage(ctx context.Context, subtaskID uuid.UUID, tokens int) error {
	return a.svc.UpdateTokenUsage(ctx, subtaskID, tokens)
}

// eventPublisherAdapter adapts service.EventHub to agent.EventPublisherInterface.
type eventPublisherAdapter struct {
	hub service.EventHub
}

func newEventPublisherAdapter(hub service.EventHub) agent.EventPublisherInterface {
	if hub == nil {
		return nil
	}
	return &eventPublisherAdapter{hub: hub}
}

func (a *eventPublisherAdapter) PublishAgentStarted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID) {
	a.hub.PublishAgentStarted(projectID, run, taskID)
}

func (a *eventPublisherAdapter) PublishAgentCompleted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, prURL string) {
	a.hub.PublishAgentCompleted(projectID, run, taskID, prURL)
}

func (a *eventPublisherAdapter) PublishAgentFailed(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, errMsg string, willRetry bool, nextAttemptAt *time.Time) {
	a.hub.PublishAgentFailed(projectID, run, taskID, errMsg, willRetry, nextAttemptAt)
}

// logTailerAdapter adapts service.LogTailer to agent.LogTailerInterface.
type logTailerAdapter struct {
	tailer service.LogTailer
}

func newLogTailerAdapter(tailer service.LogTailer) agent.LogTailerInterface {
	if tailer == nil {
		return nil
	}
	return &logTailerAdapter{tailer: tailer}
}

func (a *logTailerAdapter) StartTailing(ctx context.Context, projectID, runID uuid.UUID, logPath string) error {
	return a.tailer.StartTailing(ctx, projectID, runID, logPath)
}

func (a *logTailerAdapter) StopTailing(runID uuid.UUID) {
	a.tailer.StopTailing(runID)
}
