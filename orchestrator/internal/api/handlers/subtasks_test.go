// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package handlers

import (
	"encoding/json"
	"testing"

	"github.com/intern-village/orchestrator/internal/domain"
)

func TestSubtaskResponse_Format(t *testing.T) {
	prURL := "https://github.com/owner/repo/pull/1"
	prNumber := 1
	branchName := "iv-1-add-oauth"
	spec := "Test spec"
	blockedReason := "DEPENDENCY"

	resp := SubtaskResponse{
		ID:                 "550e8400-e29b-41d4-a716-446655440000",
		TaskID:             "550e8400-e29b-41d4-a716-446655440001",
		Title:              "Test Subtask",
		Spec:               &spec,
		ImplementationPlan: nil,
		Status:             "READY",
		BlockedReason:      nil,
		BranchName:         &branchName,
		PRUrl:              &prURL,
		PRNumber:           &prNumber,
		RetryCount:         0,
		TokenUsage:         100,
		Position:           1,
		BeadsIssueID:       nil,
		WorktreePath:       nil,
		CreatedAt:          "2026-02-04T00:00:00Z",
		UpdatedAt:          "2026-02-04T00:00:00Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check required fields are present
	requiredFields := []string{"id", "task_id", "title", "status", "retry_count", "token_usage", "position", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := unmarshaled[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Optional fields with values should be present
	if _, ok := unmarshaled["spec"]; !ok {
		t.Error("spec should be present when not nil")
	}
	if _, ok := unmarshaled["pr_url"]; !ok {
		t.Error("pr_url should be present when not nil")
	}

	// Optional fields with nil values should be omitted
	if _, ok := unmarshaled["implementation_plan"]; ok {
		t.Error("implementation_plan should be omitted when nil")
	}

	// Test with blocked reason
	resp.Status = "BLOCKED"
	resp.BlockedReason = &blockedReason

	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal blocked response: %v", err)
	}

	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal blocked response: %v", err)
	}

	if _, ok := unmarshaled["blocked_reason"]; !ok {
		t.Error("blocked_reason should be present when not nil")
	}
}

func TestUpdatePositionRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid position",
			body:    `{"position": 5}`,
			wantErr: false,
		},
		{
			name:    "zero position is valid",
			body:    `{"position": 0}`,
			wantErr: false,
		},
		{
			name:    "negative position",
			body:    `{"position": -1}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req UpdatePositionRequest
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			hasError := req.Position < 0
			if hasError != tt.wantErr {
				t.Errorf("validation result mismatch: got error=%v, want error=%v", hasError, tt.wantErr)
			}
		})
	}
}

func TestSubtaskToResponse(t *testing.T) {
	reason := domain.BlockedReasonFailure
	prURL := "https://github.com/owner/repo/pull/1"
	prNumber := 1
	spec := "Test spec"
	plan := "Test plan"
	branch := "iv-1-test"
	beadsID := "iv-1"
	worktreePath := "/data/worktrees/iv-1"

	subtask := &domain.Subtask{
		ID:                 [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		TaskID:             [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 17},
		Title:              "Test Subtask",
		Spec:               &spec,
		ImplementationPlan: &plan,
		Status:             domain.SubtaskStatusBlocked,
		BlockedReason:      &reason,
		BranchName:         &branch,
		PRUrl:              &prURL,
		PRNumber:           &prNumber,
		RetryCount:         3,
		TokenUsage:         5000,
		Position:           2,
		BeadsIssueID:       &beadsID,
		WorktreePath:       &worktreePath,
	}

	resp := subtaskToResponse(subtask)

	if resp.Title != subtask.Title {
		t.Errorf("Title mismatch: got %s, want %s", resp.Title, subtask.Title)
	}
	if resp.Status != string(subtask.Status) {
		t.Errorf("Status mismatch: got %s, want %s", resp.Status, subtask.Status)
	}
	if *resp.BlockedReason != string(*subtask.BlockedReason) {
		t.Errorf("BlockedReason mismatch: got %s, want %s", *resp.BlockedReason, *subtask.BlockedReason)
	}
	if resp.RetryCount != subtask.RetryCount {
		t.Errorf("RetryCount mismatch: got %d, want %d", resp.RetryCount, subtask.RetryCount)
	}
	if resp.TokenUsage != subtask.TokenUsage {
		t.Errorf("TokenUsage mismatch: got %d, want %d", resp.TokenUsage, subtask.TokenUsage)
	}
	if resp.Position != subtask.Position {
		t.Errorf("Position mismatch: got %d, want %d", resp.Position, subtask.Position)
	}
}

func TestPtrStr(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil returns empty",
			input:    nil,
			expected: "",
		},
		{
			name:     "non-nil returns value",
			input:    strPtr("test"),
			expected: "test",
		},
		{
			name:     "empty string returns empty",
			input:    strPtr(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ptrStr(tt.input)
			if got != tt.expected {
				t.Errorf("ptrStr(%v) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
