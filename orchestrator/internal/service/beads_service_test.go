// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Add OAuth Handler",
			expected: "add-oauth-handler",
		},
		{
			input:    "Fix Bug #123",
			expected: "fix-bug-123",
		},
		{
			input:    "Hello World!",
			expected: "hello-world",
		},
		{
			input:    "  Multiple   Spaces  ",
			expected: "multiple-spaces",
		},
		{
			input:    "Special @#$% Characters",
			expected: "special-characters",
		},
		{
			input:    "Already-Hyphenated",
			expected: "already-hyphenated",
		},
		{
			input:    "underscores_work_too",
			expected: "underscores-work-too",
		},
		{
			input:    "UPPERCASE",
			expected: "uppercase",
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "This is a very long title that should be truncated because it exceeds the maximum allowed length",
			expected: "this-is-a-very-long-title-that-should-be-truncated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateBranchName(t *testing.T) {
	svc := NewBeadsService()

	tests := []struct {
		issueID  string
		title    string
		expected string
	}{
		{
			issueID:  "iv-5",
			title:    "Add OAuth Handler",
			expected: "iv-5-add-oauth-handler",
		},
		{
			issueID:  "iv-123",
			title:    "Fix Bug",
			expected: "iv-123-fix-bug",
		},
		{
			issueID:  "prefix-1",
			title:    "Simple Title",
			expected: "prefix-1-simple-title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := svc.GenerateBranchName(tt.issueID, tt.title)
			if result != tt.expected {
				t.Errorf("GenerateBranchName(%q, %q) = %q, want %q", tt.issueID, tt.title, result, tt.expected)
			}
		})
	}
}
