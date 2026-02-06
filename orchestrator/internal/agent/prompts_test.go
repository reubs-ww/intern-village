// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/intern-village/orchestrator/internal/domain"
)

func TestNewPromptRenderer(t *testing.T) {
	// Create temp directory for data
	tmpDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	renderer, err := NewPromptRenderer(tmpDir)
	if err != nil {
		t.Fatalf("NewPromptRenderer() error = %v", err)
	}

	if renderer == nil {
		t.Fatal("NewPromptRenderer() returned nil")
	}

	if renderer.dataDir != tmpDir {
		t.Errorf("dataDir = %q, want %q", renderer.dataDir, tmpDir)
	}
}

func TestRenderPlannerPrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	renderer, err := NewPromptRenderer(tmpDir)
	if err != nil {
		t.Fatalf("NewPromptRenderer() error = %v", err)
	}

	task := &domain.Task{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		Title:       "Add user authentication",
		Description: "Implement OAuth login with GitHub",
		Status:      domain.TaskStatusPlanning,
	}

	project := &domain.Project{
		ID:            uuid.New(),
		GitHubOwner:   "testowner",
		GitHubRepo:    "testrepo",
		DefaultBranch: "main",
		ClonePath:     "/data/projects/test",
	}

	prompt, err := renderer.RenderPlannerPrompt(task, project)
	if err != nil {
		t.Fatalf("RenderPlannerPrompt() error = %v", err)
	}

	// Verify the prompt contains expected content
	expectedContents := []string{
		"Add user authentication",
		"Implement OAuth login with GitHub",
		"testowner/testrepo",
		"main",
		"/data/projects/test",
		"Planner Agent",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(prompt, expected) {
			t.Errorf("prompt does not contain expected content: %q", expected)
		}
	}
}

func TestRenderWorkerPrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	renderer, err := NewPromptRenderer(tmpDir)
	if err != nil {
		t.Fatalf("NewPromptRenderer() error = %v", err)
	}

	beadsID := "iv-123"
	spec := "Implement the OAuth handler"
	plan := "1. Create handler\n2. Add routes\n3. Test"
	branch := "iv-123-add-oauth"
	worktree := "/data/projects/test/iv-123"

	subtask := &domain.Subtask{
		ID:                 uuid.New(),
		TaskID:             uuid.New(),
		Title:              "Add OAuth handler",
		Spec:               &spec,
		ImplementationPlan: &plan,
		BeadsIssueID:       &beadsID,
		BranchName:         &branch,
		WorktreePath:       &worktree,
	}

	project := &domain.Project{
		ID:            uuid.New(),
		GitHubOwner:   "testowner",
		GitHubRepo:    "testrepo",
		DefaultBranch: "main",
		ClonePath:     "/data/projects/test",
	}

	prompt, err := renderer.RenderWorkerPrompt(subtask, project)
	if err != nil {
		t.Fatalf("RenderWorkerPrompt() error = %v", err)
	}

	// Verify the prompt contains expected content
	expectedContents := []string{
		"Add OAuth handler",
		"iv-123",
		"Implement the OAuth handler",
		"1. Create handler",
		"iv-123-add-oauth",
		"/data/projects/test/iv-123",
		"Worker Agent",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(prompt, expected) {
			t.Errorf("prompt does not contain expected content: %q", expected)
		}
	}
}

func TestSavePrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	renderer, err := NewPromptRenderer(tmpDir)
	if err != nil {
		t.Fatalf("NewPromptRenderer() error = %v", err)
	}

	t.Run("save planner prompt", func(t *testing.T) {
		content := "# Planner Prompt\nThis is a test"
		projectID := uuid.New().String()
		taskID := uuid.New().String()

		path, err := renderer.SavePrompt(content, projectID, taskID, "")
		if err != nil {
			t.Fatalf("SavePrompt() error = %v", err)
		}

		expectedPath := filepath.Join(tmpDir, "prompts", projectID, taskID, "planner.md")
		if path != expectedPath {
			t.Errorf("SavePrompt() path = %q, want %q", path, expectedPath)
		}

		// Verify file was written
		savedContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read saved file: %v", err)
		}

		if string(savedContent) != content {
			t.Errorf("saved content = %q, want %q", string(savedContent), content)
		}
	})

	t.Run("save worker prompt", func(t *testing.T) {
		content := "# Worker Prompt\nThis is a test"
		projectID := uuid.New().String()
		taskID := uuid.New().String()
		subtaskID := uuid.New().String()

		path, err := renderer.SavePrompt(content, projectID, taskID, subtaskID)
		if err != nil {
			t.Fatalf("SavePrompt() error = %v", err)
		}

		expectedPath := filepath.Join(tmpDir, "prompts", projectID, taskID, subtaskID+".md")
		if path != expectedPath {
			t.Errorf("SavePrompt() path = %q, want %q", path, expectedPath)
		}
	})
}

func TestGetPromptPath(t *testing.T) {
	renderer := &PromptRenderer{dataDir: "/data"}

	tests := []struct {
		name        string
		projectID   string
		taskID      string
		subtaskID   string
		expectedEnd string
	}{
		{
			name:        "planner prompt",
			projectID:   "proj-1",
			taskID:      "task-1",
			subtaskID:   "",
			expectedEnd: "proj-1/task-1/planner.md",
		},
		{
			name:        "worker prompt",
			projectID:   "proj-1",
			taskID:      "task-1",
			subtaskID:   "sub-1",
			expectedEnd: "proj-1/task-1/sub-1.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := renderer.GetPromptPath(tt.projectID, tt.taskID, tt.subtaskID)
			if !strings.HasSuffix(path, tt.expectedEnd) {
				t.Errorf("GetPromptPath() = %q, want suffix %q", path, tt.expectedEnd)
			}
		})
	}
}
