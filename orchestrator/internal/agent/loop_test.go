// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"testing"
	"time"
)

func TestCalculateBackoffValues(t *testing.T) {
	// Test that the backoff formula is correct: min(5 * 2^attempt, 120)
	tests := []struct {
		attempt       int
		expectedBase  float64 // before cap
		expectedFinal float64 // after cap
	}{
		{1, 10, 10},   // 5 * 2^1 = 10
		{2, 20, 20},   // 5 * 2^2 = 20
		{3, 40, 40},   // 5 * 2^3 = 40
		{4, 80, 80},   // 5 * 2^4 = 80
		{5, 160, 120}, // 5 * 2^5 = 160, capped at 120
		{6, 320, 120}, // 5 * 2^6 = 320, capped at 120
		{7, 640, 120}, // 5 * 2^7 = 640, capped at 120
	}

	for _, tt := range tests {
		delay := CalculateBackoff(tt.attempt)
		expectedDuration := time.Duration(tt.expectedFinal * float64(time.Second))

		// Allow 1 second tolerance for precision
		if delay < expectedDuration || delay > expectedDuration+time.Second {
			t.Errorf("CalculateBackoff(%d) = %v, want ~%v", tt.attempt, delay, expectedDuration)
		}
	}
}

func TestCalculateBackoffNeverExceeds120Seconds(t *testing.T) {
	maxBackoff := 120 * time.Second

	for attempt := 1; attempt <= 20; attempt++ {
		delay := CalculateBackoff(attempt)
		// Allow 1 second for any jitter
		if delay > maxBackoff+time.Second {
			t.Errorf("CalculateBackoff(%d) = %v, exceeds max of %v", attempt, delay, maxBackoff)
		}
	}
}

func TestCalculateBackoffAlwaysPositive(t *testing.T) {
	for attempt := 0; attempt <= 20; attempt++ {
		delay := CalculateBackoff(attempt)
		if delay < 0 {
			t.Errorf("CalculateBackoff(%d) = %v, should be positive", attempt, delay)
		}
	}
}
