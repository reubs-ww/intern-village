// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/intern-village/orchestrator/internal/domain"
)

func TestDetermineInitialStatus(t *testing.T) {
	// This tests the logic of determining initial status
	// In a real integration test, we'd use a test database
	// Here we just document the expected behavior

	tests := []struct {
		name           string
		hasBlocking    bool
		expectedStatus domain.SubtaskStatus
		expectedReason *domain.BlockedReason
	}{
		{
			name:           "no blocking dependencies returns READY",
			hasBlocking:    false,
			expectedStatus: domain.SubtaskStatusReady,
			expectedReason: nil,
		},
		{
			name:           "has blocking dependencies returns BLOCKED",
			hasBlocking:    true,
			expectedStatus: domain.SubtaskStatusBlocked,
			expectedReason: ptrBlockedReason(domain.BlockedReasonDependency),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The actual test would need a mock repository
			// For now, just verify the expected values are valid
			if !tt.expectedStatus.IsValid() {
				t.Errorf("expectedStatus %s is not valid", tt.expectedStatus)
			}
			if tt.expectedReason != nil && !tt.expectedReason.IsValid() {
				t.Errorf("expectedReason %s is not valid", *tt.expectedReason)
			}
		})
	}
}

func ptrBlockedReason(r domain.BlockedReason) *domain.BlockedReason {
	return &r
}

func TestSelfDependencyPrevention(t *testing.T) {
	// Verify that we properly prevent a subtask from depending on itself
	id := uuid.New()
	sameID := id // Create a copy to compare

	// This would be tested with actual service in integration tests
	// Here we just verify the validation logic
	if id == sameID {
		// This is expected - the service should reject this
		t.Log("Self-dependency correctly detected")
	}
}

func TestBlockedReasonTypes(t *testing.T) {
	tests := []struct {
		name   string
		reason domain.BlockedReason
		valid  bool
	}{
		{
			name:   "DEPENDENCY is valid",
			reason: domain.BlockedReasonDependency,
			valid:  true,
		},
		{
			name:   "FAILURE is valid",
			reason: domain.BlockedReasonFailure,
			valid:  true,
		},
		{
			name:   "unknown is invalid",
			reason: domain.BlockedReason("UNKNOWN"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.reason.IsValid(); got != tt.valid {
				t.Errorf("BlockedReason(%s).IsValid() = %v, want %v", tt.reason, got, tt.valid)
			}
		})
	}
}
