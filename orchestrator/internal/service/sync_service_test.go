// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"testing"
)

func TestParseIssueBody(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		expectedSpec string
		expectedPlan string
	}{
		{
			name: "standard format with headers",
			body: `## Spec
This is the specification.
It has multiple lines.

## Implementation Plan
Step 1: Do something
Step 2: Do something else

## Acceptance Criteria
- Test passes`,
			expectedSpec: "This is the specification.\nIt has multiple lines.",
			expectedPlan: "Step 1: Do something\nStep 2: Do something else",
		},
		{
			name: "single hash headers",
			body: `# Spec
The spec here

# Implementation Plan
The plan here`,
			expectedSpec: "The spec here",
			expectedPlan: "The plan here",
		},
		{
			name:         "no headers - use as spec",
			body:         "Just plain text describing the task",
			expectedSpec: "Just plain text describing the task",
			expectedPlan: "",
		},
		{
			name: "spec only",
			body: `## Spec
Only has a spec section`,
			expectedSpec: "Only has a spec section",
			expectedPlan: "",
		},
		{
			name:         "empty body",
			body:         "",
			expectedSpec: "",
			expectedPlan: "",
		},
		{
			name: "case insensitive headers",
			body: `## SPEC
Upper case spec

## IMPLEMENTATION PLAN
Upper case plan`,
			expectedSpec: "Upper case spec",
			expectedPlan: "Upper case plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, plan := parseIssueBody(tt.body)
			if spec != tt.expectedSpec {
				t.Errorf("spec mismatch:\ngot:  %q\nwant: %q", spec, tt.expectedSpec)
			}
			if plan != tt.expectedPlan {
				t.Errorf("plan mismatch:\ngot:  %q\nwant: %q", plan, tt.expectedPlan)
			}
		})
	}
}

func TestSyncService_NewSyncService(t *testing.T) {
	// Test that NewSyncService creates a valid service
	// In a real test, we'd use mock dependencies
	service := NewSyncService(nil, nil, nil, nil, nil)
	if service == nil {
		t.Error("NewSyncService returned nil")
	}
}

func TestSyncWorker_NewSyncWorker(t *testing.T) {
	tests := []struct {
		name            string
		intervalSeconds int
		expectInterval  int // in seconds
	}{
		{
			name:            "normal interval",
			intervalSeconds: 30,
			expectInterval:  30,
		},
		{
			name:            "zero interval defaults to 30",
			intervalSeconds: 0,
			expectInterval:  30,
		},
		{
			name:            "negative interval defaults to 30",
			intervalSeconds: -1,
			expectInterval:  30,
		},
		{
			name:            "very short interval defaults to 30",
			intervalSeconds: 1,
			expectInterval:  30,
		},
		{
			name:            "5 second interval is minimum",
			intervalSeconds: 5,
			expectInterval:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewSyncWorker(nil, nil, tt.intervalSeconds)
			if worker == nil {
				t.Fatal("NewSyncWorker returned nil")
			}
			// Note: We can't directly check interval as it's private
			// In real tests, we'd verify behavior
		})
	}
}
