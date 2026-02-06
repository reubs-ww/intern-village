// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/prompts"
)

// templateFuncs provides custom functions for prompt templates.
var templateFuncs = template.FuncMap{
	// short returns the first 8 characters of a UUID for use as a short identifier.
	"short": func(id uuid.UUID) string {
		s := id.String()
		if len(s) >= 8 {
			return s[:8]
		}
		return s
	},
	// shortStr returns the first 8 characters of a string.
	"shortStr": func(s string) string {
		// Handle UUID format (xxxxxxxx-xxxx-...)
		if len(s) >= 8 {
			return strings.Split(s, "-")[0]
		}
		return s
	},
}

// PlannerContext is the template context for the Planner prompt.
type PlannerContext struct {
	Task    *domain.Task
	Project *domain.Project
}

// WorkerContext is the template context for the Worker prompt.
type WorkerContext struct {
	Subtask *domain.Subtask
	Project *domain.Project
}

// PromptRenderer renders agent prompts from templates.
type PromptRenderer struct {
	plannerTmpl *template.Template
	workerTmpl  *template.Template
	dataDir     string
}

// NewPromptRenderer creates a new PromptRenderer.
func NewPromptRenderer(dataDir string) (*PromptRenderer, error) {
	// Load planner template
	plannerContent, err := prompts.FS.ReadFile("planner.md")
	if err != nil {
		return nil, fmt.Errorf("failed to read planner template: %w", err)
	}

	plannerTmpl, err := template.New("planner").Funcs(templateFuncs).Parse(string(plannerContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse planner template: %w", err)
	}

	// Load worker template
	workerContent, err := prompts.FS.ReadFile("worker.md")
	if err != nil {
		return nil, fmt.Errorf("failed to read worker template: %w", err)
	}

	workerTmpl, err := template.New("worker").Funcs(templateFuncs).Parse(string(workerContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse worker template: %w", err)
	}

	return &PromptRenderer{
		plannerTmpl: plannerTmpl,
		workerTmpl:  workerTmpl,
		dataDir:     dataDir,
	}, nil
}

// RenderPlannerPrompt renders the Planner prompt template.
func (r *PromptRenderer) RenderPlannerPrompt(task *domain.Task, project *domain.Project) (string, error) {
	ctx := PlannerContext{
		Task:    task,
		Project: project,
	}

	var buf bytes.Buffer
	if err := r.plannerTmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("failed to render planner prompt: %w", err)
	}

	return buf.String(), nil
}

// RenderWorkerPrompt renders the Worker prompt template.
func (r *PromptRenderer) RenderWorkerPrompt(subtask *domain.Subtask, project *domain.Project) (string, error) {
	ctx := WorkerContext{
		Subtask: subtask,
		Project: project,
	}

	var buf bytes.Buffer
	if err := r.workerTmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("failed to render worker prompt: %w", err)
	}

	return buf.String(), nil
}

// SavePrompt saves a rendered prompt to the data directory for audit purposes.
// Returns the path to the saved file.
func (r *PromptRenderer) SavePrompt(content, projectID, taskID, subtaskID string) (string, error) {
	// Create directory structure: /data/prompts/{project_id}/{task_id}/
	promptDir := filepath.Join(r.dataDir, "prompts", projectID, taskID)
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create prompt directory: %w", err)
	}

	// Determine filename based on whether this is for a task (planner) or subtask (worker)
	var filename string
	if subtaskID != "" {
		filename = fmt.Sprintf("%s.md", subtaskID)
	} else {
		filename = "planner.md"
	}

	promptPath := filepath.Join(promptDir, filename)
	if err := os.WriteFile(promptPath, []byte(content), 0o644); err != nil { //nolint:gosec // Prompt files are for audit and can be world-readable
		return "", fmt.Errorf("failed to write prompt file: %w", err)
	}

	return promptPath, nil
}

// GetPromptPath returns the expected path for a prompt file.
func (r *PromptRenderer) GetPromptPath(projectID, taskID, subtaskID string) string {
	promptDir := filepath.Join(r.dataDir, "prompts", projectID, taskID)
	if subtaskID != "" {
		return filepath.Join(promptDir, fmt.Sprintf("%s.md", subtaskID))
	}
	return filepath.Join(promptDir, "planner.md")
}
