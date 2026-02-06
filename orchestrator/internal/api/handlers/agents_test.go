// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package handlers

import (
	"encoding/json"
	"testing"
)

func TestAgentRunResponse_Format(t *testing.T) {
	endedAt := "2026-02-04T00:05:00Z"
	tokenUsage := 1500
	errorMsg := "exit code: 1"

	resp := AgentRunResponse{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		SubtaskID:     "550e8400-e29b-41d4-a716-446655440001",
		AgentType:     "WORKER",
		AttemptNumber: 1,
		Status:        "RUNNING",
		StartedAt:     "2026-02-04T00:00:00Z",
		EndedAt:       nil,
		TokenUsage:    nil,
		ErrorMessage:  nil,
		LogPath:       "/data/logs/project/task/subtask/run-001.log",
		CreatedAt:     "2026-02-04T00:00:00Z",
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
	requiredFields := []string{"id", "subtask_id", "agent_type", "attempt_number", "status", "started_at", "log_path", "created_at"}
	for _, field := range requiredFields {
		if _, ok := unmarshaled[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Optional fields with nil values should be omitted
	if _, ok := unmarshaled["ended_at"]; ok {
		t.Error("ended_at should be omitted when nil")
	}
	if _, ok := unmarshaled["token_usage"]; ok {
		t.Error("token_usage should be omitted when nil")
	}
	if _, ok := unmarshaled["error_message"]; ok {
		t.Error("error_message should be omitted when nil")
	}

	// Test with optional fields populated
	resp.EndedAt = &endedAt
	resp.TokenUsage = &tokenUsage
	resp.ErrorMessage = &errorMsg
	resp.Status = "FAILED"

	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal completed response: %v", err)
	}

	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal completed response: %v", err)
	}

	if _, ok := unmarshaled["ended_at"]; !ok {
		t.Error("ended_at should be present when not nil")
	}
	if _, ok := unmarshaled["token_usage"]; !ok {
		t.Error("token_usage should be present when not nil")
	}
	if _, ok := unmarshaled["error_message"]; !ok {
		t.Error("error_message should be present when not nil")
	}
}

func TestAgentRunLogsResponse_Format(t *testing.T) {
	resp := AgentRunLogsResponse{
		RunID:   "550e8400-e29b-41d4-a716-446655440000",
		LogPath: "/data/logs/project/task/subtask/run-001.log",
		Content: "Log line 1\nLog line 2\nLog line 3\n",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check all fields are present
	requiredFields := []string{"run_id", "log_path", "content"}
	for _, field := range requiredFields {
		if _, ok := unmarshaled[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Check content matches
	if unmarshaled["content"] != resp.Content {
		t.Errorf("content mismatch: got %s, want %s", unmarshaled["content"], resp.Content)
	}
}

func TestAgentRunLogsResponse_EmptyContent(t *testing.T) {
	resp := AgentRunLogsResponse{
		RunID:   "550e8400-e29b-41d4-a716-446655440000",
		LogPath: "/data/logs/project/task/subtask/run-001.log",
		Content: "",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Empty content should still be present (not omitted)
	if _, ok := unmarshaled["content"]; !ok {
		t.Error("content should be present even when empty")
	}

	if unmarshaled["content"] != "" {
		t.Errorf("content should be empty string, got: %s", unmarshaled["content"])
	}
}

func TestAgentRunResponse_AgentTypes(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
	}{
		{"planner agent", "PLANNER"},
		{"worker agent", "WORKER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := AgentRunResponse{
				ID:            "550e8400-e29b-41d4-a716-446655440000",
				SubtaskID:     "550e8400-e29b-41d4-a716-446655440001",
				AgentType:     tt.agentType,
				AttemptNumber: 1,
				Status:        "SUCCEEDED",
				StartedAt:     "2026-02-04T00:00:00Z",
				LogPath:       "/data/logs/run.log",
				CreatedAt:     "2026-02-04T00:00:00Z",
			}

			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal response: %v", err)
			}

			var unmarshaled map[string]interface{}
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if unmarshaled["agent_type"] != tt.agentType {
				t.Errorf("agent_type mismatch: got %s, want %s", unmarshaled["agent_type"], tt.agentType)
			}
		})
	}
}

func TestAgentRunResponse_StatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"running status", "RUNNING"},
		{"succeeded status", "SUCCEEDED"},
		{"failed status", "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := AgentRunResponse{
				ID:            "550e8400-e29b-41d4-a716-446655440000",
				SubtaskID:     "550e8400-e29b-41d4-a716-446655440001",
				AgentType:     "WORKER",
				AttemptNumber: 1,
				Status:        tt.status,
				StartedAt:     "2026-02-04T00:00:00Z",
				LogPath:       "/data/logs/run.log",
				CreatedAt:     "2026-02-04T00:00:00Z",
			}

			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal response: %v", err)
			}

			var unmarshaled map[string]interface{}
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if unmarshaled["status"] != tt.status {
				t.Errorf("status mismatch: got %s, want %s", unmarshaled["status"], tt.status)
			}
		})
	}
}
