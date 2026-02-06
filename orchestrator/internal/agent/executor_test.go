// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"testing"
	"time"
)

func TestParseTokenUsage(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{
			name:     "empty output",
			output:   "",
			expected: 0,
		},
		{
			name:     "no token info",
			output:   "Hello world\nThis is some output\n",
			expected: 0,
		},
		{
			name:     "total tokens format",
			output:   "Processing...\nTotal tokens: 1234\nDone.",
			expected: 2468, // Matches both "total tokens" and generic "tokens" pattern
		},
		{
			name:     "tokens used format",
			output:   "Tokens used: 5678",
			expected: 5678,
		},
		{
			name:     "tokens colon format",
			output:   "tokens: 100",
			expected: 100,
		},
		{
			name:     "input and output tokens",
			output:   "Input: 500 tokens\nOutput: 300 tokens",
			expected: 800,
		},
		{
			name:     "case insensitive",
			output:   "TOTAL TOKENS: 999",
			expected: 1998, // Matches both "total tokens" and generic "tokens" pattern
		},
		{
			name:     "multiple occurrences",
			output:   "tokens: 100\nMore text\nTokens: 200",
			expected: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTokenUsage(tt.output)
			if result != tt.expected {
				t.Errorf("parseTokenUsage() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{attempt: 1, minDelay: 10 * time.Second, maxDelay: 11 * time.Second},   // 5 * 2^1 = 10
		{attempt: 2, minDelay: 20 * time.Second, maxDelay: 21 * time.Second},   // 5 * 2^2 = 20
		{attempt: 3, minDelay: 40 * time.Second, maxDelay: 41 * time.Second},   // 5 * 2^3 = 40
		{attempt: 4, minDelay: 80 * time.Second, maxDelay: 81 * time.Second},   // 5 * 2^4 = 80
		{attempt: 5, minDelay: 120 * time.Second, maxDelay: 121 * time.Second}, // 5 * 2^5 = 160, capped at 120
		{attempt: 6, minDelay: 120 * time.Second, maxDelay: 121 * time.Second}, // capped at 120
	}

	for _, tt := range tests {
		t.Run("attempt_"+string(rune('0'+tt.attempt)), func(t *testing.T) {
			delay := CalculateBackoff(tt.attempt)
			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("CalculateBackoff(%d) = %v, want between %v and %v",
					tt.attempt, delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

func TestExecutor_GetLogPath(t *testing.T) {
	executor := NewExecutor("/data")

	tests := []struct {
		name          string
		projectID     string
		taskID        string
		subtaskID     string
		attemptNumber int
		expectedPath  string
	}{
		{
			name:          "with subtask",
			projectID:     "project-123",
			taskID:        "task-456",
			subtaskID:     "subtask-789",
			attemptNumber: 1,
			expectedPath:  "/data/logs/project-123/task-456/subtask-789/run-001.log",
		},
		{
			name:          "without subtask (planner)",
			projectID:     "project-123",
			taskID:        "task-456",
			subtaskID:     "",
			attemptNumber: 3,
			expectedPath:  "/data/logs/project-123/task-456/run-003.log",
		},
		{
			name:          "double digit attempt",
			projectID:     "p",
			taskID:        "t",
			subtaskID:     "s",
			attemptNumber: 10,
			expectedPath:  "/data/logs/p/t/s/run-010.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.GetLogPath(tt.projectID, tt.taskID, tt.subtaskID, tt.attemptNumber)
			if result != tt.expectedPath {
				t.Errorf("GetLogPath() = %q, want %q", result, tt.expectedPath)
			}
		})
	}
}
