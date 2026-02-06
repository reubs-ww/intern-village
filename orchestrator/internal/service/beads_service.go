// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
)

// Beads service errors.
var (
	ErrBeadsInitFailed      = errors.New("beads initialization failed")
	ErrBeadsCreateFailed    = errors.New("beads issue creation failed")
	ErrBeadsListFailed      = errors.New("beads list failed")
	ErrBeadsShowFailed      = errors.New("beads show failed")
	ErrBeadsCloseFailed     = errors.New("beads close failed")
	ErrBeadsDepFailed       = errors.New("beads dependency operation failed")
	ErrBeadsWorktreeFailed  = errors.New("beads worktree operation failed")
	ErrBeadsCommandNotFound = errors.New("beads command (bd) not found")
	ErrBeadsInvalidOutput   = errors.New("invalid beads output")
)

// BeadsDependency represents a dependency relationship from Beads.
type BeadsDependency struct {
	IssueID     string `json:"issue_id"`
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"` // "parent-child", "blocks"
}

// BeadsIssue represents an issue from Beads.
type BeadsIssue struct {
	ID           string            `json:"id"`
	Type         string            `json:"issue_type"` // "epic" or "task" - note: field is "issue_type" in JSON
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Status       string            `json:"status"` // "open", "in_progress", "closed"
	ParentID     string            `json:"parent,omitempty"`
	Dependencies []BeadsDependency `json:"dependencies,omitempty"`
}

// GetDependencyIDs returns just the depends_on_id values from dependencies.
func (b *BeadsIssue) GetDependencyIDs() []string {
	var ids []string
	for _, dep := range b.Dependencies {
		// Skip parent-child relationships, only include blocking dependencies
		if dep.Type == "blocks" {
			ids = append(ids, dep.DependsOnID)
		}
	}
	return ids
}

// GetParentFromDeps returns the parent ID from parent-child dependencies.
func (b *BeadsIssue) GetParentFromDeps() string {
	for _, dep := range b.Dependencies {
		if dep.Type == "parent-child" {
			return dep.DependsOnID
		}
	}
	return ""
}

// BeadsService wraps the beads CLI for issue tracking.
type BeadsService struct {
	// bdPath is the path to the bd executable. Empty means use PATH.
	bdPath string
}

// NewBeadsService creates a new BeadsService.
func NewBeadsService() *BeadsService {
	return &BeadsService{
		bdPath: "bd",
	}
}

// NewBeadsServiceWithPath creates a BeadsService with a custom bd path.
func NewBeadsServiceWithPath(bdPath string) *BeadsService {
	return &BeadsService{
		bdPath: bdPath,
	}
}

// runCommand executes a beads command and returns its output.
func (s *BeadsService) runCommand(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, s.bdPath, args...) //nolint:gosec // Args are controlled by the service
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if bd is not found
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrBeadsCommandNotFound
		}
		return "", fmt.Errorf("command failed: %v (output: %s)", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// Init initializes beads in a repository with stealth mode.
// Uses --stealth to avoid committing beads files to the repo.
func (s *BeadsService) Init(ctx context.Context, repoPath, prefix string) error {
	_, err := s.runCommand(ctx, repoPath, "init", "--stealth", "--prefix", prefix)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeadsInitFailed, err)
	}
	return nil
}

// CreateEpic creates a new epic issue and returns its ID.
func (s *BeadsService) CreateEpic(ctx context.Context, repoPath, title string) (string, error) {
	output, err := s.runCommand(ctx, repoPath, "create", "--type", "epic", "--title", title)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBeadsCreateFailed, err)
	}

	// Parse the issue ID from output (e.g., "Created iv-1")
	id := parseCreatedID(output)
	if id == "" {
		return "", fmt.Errorf("%w: could not parse issue ID from: %s", ErrBeadsInvalidOutput, output)
	}

	return id, nil
}

// CreateIssue creates a new task issue under a parent epic.
func (s *BeadsService) CreateIssue(ctx context.Context, repoPath, parentID, title, body string) (string, error) {
	output, err := s.runCommand(ctx, repoPath, "create", "--type", "task", "--parent", parentID, "--title", title, "--description", body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBeadsCreateFailed, err)
	}

	id := parseCreatedID(output)
	if id == "" {
		return "", fmt.Errorf("%w: could not parse issue ID from: %s", ErrBeadsInvalidOutput, output)
	}

	return id, nil
}

// AddDependency adds a dependency between two issues.
// child depends on parent (child is blocked until parent is closed).
func (s *BeadsService) AddDependency(ctx context.Context, repoPath, childID, parentID string) error {
	_, err := s.runCommand(ctx, repoPath, "dep", "add", childID, parentID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeadsDepFailed, err)
	}
	return nil
}

// ListIssues lists all issues under a parent epic.
func (s *BeadsService) ListIssues(ctx context.Context, repoPath, parentID string) ([]BeadsIssue, error) {
	output, err := s.runCommand(ctx, repoPath, "list", "--parent", parentID, "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeadsListFailed, err)
	}

	if output == "" || output == "[]" {
		return []BeadsIssue{}, nil
	}

	var issues []BeadsIssue
	if err := json.Unmarshal([]byte(output), &issues); err != nil {
		return nil, fmt.Errorf("%w: JSON parse error: %v", ErrBeadsInvalidOutput, err)
	}

	return issues, nil
}

// ShowIssue gets details of a specific issue.
func (s *BeadsService) ShowIssue(ctx context.Context, repoPath, issueID string) (*BeadsIssue, error) {
	output, err := s.runCommand(ctx, repoPath, "show", issueID, "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeadsShowFailed, err)
	}

	if output == "" || output == "[]" {
		return nil, fmt.Errorf("%w: issue not found: %s", ErrBeadsShowFailed, issueID)
	}

	// bd show returns an array even for a single issue
	var issues []BeadsIssue
	if err := json.Unmarshal([]byte(output), &issues); err != nil {
		return nil, fmt.Errorf("%w: JSON parse error: %v", ErrBeadsInvalidOutput, err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("%w: issue not found: %s", ErrBeadsShowFailed, issueID)
	}

	return &issues[0], nil
}

// CloseIssue closes an issue with a reason.
func (s *BeadsService) CloseIssue(ctx context.Context, repoPath, issueID, reason string) error {
	_, err := s.runCommand(ctx, repoPath, "close", issueID, "--reason", reason)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeadsCloseFailed, err)
	}
	return nil
}

// DeleteIssue deletes an issue and optionally cascades to dependents.
// If cascade is true, all dependent issues are also deleted.
func (s *BeadsService) DeleteIssue(ctx context.Context, repoPath, issueID string, cascade bool) error {
	args := []string{"delete", issueID, "--force"}
	if cascade {
		args = append(args, "--cascade")
	}
	_, err := s.runCommand(ctx, repoPath, args...)
	if err != nil {
		return fmt.Errorf("failed to delete beads issue %s: %w", issueID, err)
	}
	return nil
}

// UpdateStatus updates the status of an issue.
func (s *BeadsService) UpdateStatus(ctx context.Context, repoPath, issueID, status string) error {
	_, err := s.runCommand(ctx, repoPath, "update", issueID, "--status", status)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeadsShowFailed, err)
	}
	return nil
}

// GetReadyIssues gets issues that have no blocking dependencies.
func (s *BeadsService) GetReadyIssues(ctx context.Context, repoPath, parentID string) ([]BeadsIssue, error) {
	output, err := s.runCommand(ctx, repoPath, "ready", "--parent", parentID, "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeadsListFailed, err)
	}

	if output == "" || output == "[]" {
		return []BeadsIssue{}, nil
	}

	var issues []BeadsIssue
	if err := json.Unmarshal([]byte(output), &issues); err != nil {
		return nil, fmt.Errorf("%w: JSON parse error: %v", ErrBeadsInvalidOutput, err)
	}

	return issues, nil
}

// GetBlockedIssues gets issues that have blocking dependencies.
func (s *BeadsService) GetBlockedIssues(ctx context.Context, repoPath, parentID string) ([]BeadsIssue, error) {
	output, err := s.runCommand(ctx, repoPath, "blocked", "--parent", parentID, "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeadsListFailed, err)
	}

	if output == "" || output == "[]" {
		return []BeadsIssue{}, nil
	}

	var issues []BeadsIssue
	if err := json.Unmarshal([]byte(output), &issues); err != nil {
		return nil, fmt.Errorf("%w: JSON parse error: %v", ErrBeadsInvalidOutput, err)
	}

	return issues, nil
}

// CreateWorktree creates a git worktree for a subtask.
func (s *BeadsService) CreateWorktree(ctx context.Context, repoPath, name, branch string) error {
	_, err := s.runCommand(ctx, repoPath, "worktree", "create", name, "--branch", branch)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeadsWorktreeFailed, err)
	}
	return nil
}

// RemoveWorktree removes a git worktree.
func (s *BeadsService) RemoveWorktree(ctx context.Context, repoPath, name string) error {
	_, err := s.runCommand(ctx, repoPath, "worktree", "remove", name)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeadsWorktreeFailed, err)
	}
	return nil
}

// GenerateBranchName creates a branch name from an issue ID and title.
// Format: iv-{number}-{slug-from-title}
// Example: iv-5-add-oauth-handler
func (s *BeadsService) GenerateBranchName(issueID, title string) string {
	slug := slugify(title)
	// issueID is already in format like "iv-5", so we append the slug
	return fmt.Sprintf("%s-%s", issueID, slug)
}

// AddComment adds a comment to an issue.
func (s *BeadsService) AddComment(ctx context.Context, repoPath, issueID, comment string) error {
	_, err := s.runCommand(ctx, repoPath, "comments", "add", issueID, comment)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeadsShowFailed, err)
	}
	return nil
}

// FindEpicByTaskID finds an epic by the task ID prefix in its title.
// Epic titles are formatted as "[{taskID_prefix}] {title}" by the planner.
// Returns nil if no epic found with that task ID prefix.
// Searches closed epics since the planner closes the epic when planning is complete.
func (s *BeadsService) FindEpicByTaskID(ctx context.Context, repoPath, taskIDPrefix string) (*BeadsIssue, error) {
	// Search for epic with title starting with the task ID prefix pattern
	titlePattern := fmt.Sprintf("[%s]", taskIDPrefix)
	output, err := s.runCommand(ctx, repoPath, "list", "--type", "epic", "--title", titlePattern, "--status", "closed", "--json", "--limit", "1")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeadsListFailed, err)
	}

	if output == "" || output == "[]" {
		return nil, nil
	}

	var issues []BeadsIssue
	if err := json.Unmarshal([]byte(output), &issues); err != nil {
		return nil, fmt.Errorf("%w: JSON parse error: %v", ErrBeadsInvalidOutput, err)
	}

	if len(issues) == 0 {
		return nil, nil
	}

	return &issues[0], nil
}

// GetDependencies lists dependencies for an issue.
func (s *BeadsService) GetDependencies(ctx context.Context, repoPath, issueID string) ([]string, error) {
	output, err := s.runCommand(ctx, repoPath, "dep", "list", issueID, "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeadsDepFailed, err)
	}

	if output == "" || output == "[]" {
		return []string{}, nil
	}

	var deps []string
	if err := json.Unmarshal([]byte(output), &deps); err != nil {
		return nil, fmt.Errorf("%w: JSON parse error: %v", ErrBeadsInvalidOutput, err)
	}

	return deps, nil
}

// parseCreatedID parses the issue ID from beads create output.
// Expected format: "Created iv-1" or similar.
func parseCreatedID(output string) string {
	// Look for patterns like "iv-1", "iv-123", etc.
	re := regexp.MustCompile(`[a-zA-Z]+-\d+`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// slugify converts a title to a URL-safe slug.
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	var result strings.Builder
	lastWasHyphen := false

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
			lastWasHyphen = false
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			if !lastWasHyphen && result.Len() > 0 {
				result.WriteRune('-')
				lastWasHyphen = true
			}
		}
		// Skip other characters
	}

	slug := result.String()
	// Trim trailing hyphens
	slug = strings.TrimSuffix(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
		// Don't end with hyphen
		slug = strings.TrimSuffix(slug, "-")
	}

	return slug
}
