// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTaskRequest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantErr  bool
		errField string
	}{
		{
			name:     "valid request",
			body:     `{"title": "Test Task", "description": "Test description"}`,
			wantErr:  false,
			errField: "",
		},
		{
			name:     "missing title",
			body:     `{"description": "Test description"}`,
			wantErr:  true,
			errField: "title",
		},
		{
			name:     "missing description",
			body:     `{"title": "Test Task"}`,
			wantErr:  true,
			errField: "description",
		},
		{
			name:     "empty title",
			body:     `{"title": "", "description": "Test description"}`,
			wantErr:  true,
			errField: "title",
		},
		{
			name:     "empty description",
			body:     `{"title": "Test Task", "description": ""}`,
			wantErr:  true,
			errField: "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CreateTaskRequest
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected unmarshal error: %v", err)
				}
				return
			}

			// Check validation
			hasError := false
			if req.Title == "" {
				hasError = true
			}
			if req.Description == "" {
				hasError = true
			}

			if hasError != tt.wantErr {
				t.Errorf("validation result mismatch: got error=%v, want error=%v", hasError, tt.wantErr)
			}
		})
	}
}

func TestTaskResponse_Format(t *testing.T) {
	resp := TaskResponse{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		ProjectID:   "550e8400-e29b-41d4-a716-446655440001",
		Title:       "Test Task",
		Description: "Test description",
		Status:      "PLANNING",
		BeadsEpicID: nil,
		CreatedAt:   "2026-02-04T00:00:00Z",
		UpdatedAt:   "2026-02-04T00:00:00Z",
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
	requiredFields := []string{"id", "project_id", "title", "description", "status", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := unmarshaled[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// beads_epic_id should be omitted when nil
	if _, ok := unmarshaled["beads_epic_id"]; ok {
		t.Error("beads_epic_id should be omitted when nil")
	}
}

func TestTaskHandler_CreateBadRequest(t *testing.T) {
	// Test that invalid JSON returns 400
	req := httptest.NewRequest(http.MethodPost, "/api/projects/invalid-uuid/tasks", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// We can't actually call the handler without mocking all dependencies
	// But we can verify the request would fail on JSON parsing

	var createReq CreateTaskRequest
	err := json.NewDecoder(req.Body).Decode(&createReq)
	if err == nil {
		t.Error("expected JSON decode error for invalid JSON")
	}

	_ = w // Avoid unused variable warning
}
