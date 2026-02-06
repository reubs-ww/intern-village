// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package domain

import (
	"errors"
	"testing"
)

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("project", "123")

	if err.Error() != "project with ID 123 not found" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !errors.Is(err, ErrNotFound) {
		t.Error("expected error to wrap ErrNotFound")
	}

	if !IsNotFound(err) {
		t.Error("IsNotFound should return true")
	}
}

func TestConflictError(t *testing.T) {
	err := NewConflictError("subtask", "already in progress")

	if err.Error() != "conflict on subtask: already in progress" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !errors.Is(err, ErrConflict) {
		t.Error("expected error to wrap ErrConflict")
	}

	if !IsConflict(err) {
		t.Error("IsConflict should return true")
	}
}

func TestForbiddenError(t *testing.T) {
	err := NewForbiddenError("project", "user does not own this project")

	if err.Error() != "forbidden access to project: user does not own this project" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !errors.Is(err, ErrForbidden) {
		t.Error("expected error to wrap ErrForbidden")
	}

	if !IsForbidden(err) {
		t.Error("IsForbidden should return true")
	}
}

func TestUnprocessableError(t *testing.T) {
	err := NewUnprocessableError("subtask", "cannot start blocked subtask")

	if err.Error() != "cannot process subtask: cannot start blocked subtask" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !errors.Is(err, ErrUnprocessable) {
		t.Error("expected error to wrap ErrUnprocessable")
	}

	if !IsUnprocessable(err) {
		t.Error("IsUnprocessable should return true")
	}
}

func TestInvalidTransitionError(t *testing.T) {
	t.Run("with description", func(t *testing.T) {
		err := NewInvalidTransitionError("task", "PLANNING", "DONE", "must complete subtasks first")

		expected := "invalid task transition from PLANNING to DONE: must complete subtasks first"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s, expected: %s", err.Error(), expected)
		}

		if !errors.Is(err, ErrInvalidTransition) {
			t.Error("expected error to wrap ErrInvalidTransition")
		}

		if !IsInvalidTransition(err) {
			t.Error("IsInvalidTransition should return true")
		}
	})

	t.Run("without description", func(t *testing.T) {
		err := NewInvalidTransitionError("task", "PLANNING", "DONE", "")

		expected := "invalid task transition from PLANNING to DONE"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s, expected: %s", err.Error(), expected)
		}
	})
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("title", "cannot be empty")

	if err.Error() != "validation error on title: cannot be empty" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !errors.Is(err, ErrInvalidInput) {
		t.Error("expected error to wrap ErrInvalidInput")
	}

	if !IsInvalidInput(err) {
		t.Error("IsInvalidInput should return true")
	}
}

func TestErrorCheckersWithNilAndOtherErrors(t *testing.T) {
	otherErr := errors.New("some other error")

	if IsNotFound(nil) {
		t.Error("IsNotFound(nil) should be false")
	}
	if IsNotFound(otherErr) {
		t.Error("IsNotFound(other) should be false")
	}

	if IsConflict(nil) {
		t.Error("IsConflict(nil) should be false")
	}
	if IsConflict(otherErr) {
		t.Error("IsConflict(other) should be false")
	}

	if IsForbidden(nil) {
		t.Error("IsForbidden(nil) should be false")
	}
	if IsForbidden(otherErr) {
		t.Error("IsForbidden(other) should be false")
	}

	if IsUnprocessable(nil) {
		t.Error("IsUnprocessable(nil) should be false")
	}
	if IsUnprocessable(otherErr) {
		t.Error("IsUnprocessable(other) should be false")
	}

	if IsInvalidTransition(nil) {
		t.Error("IsInvalidTransition(nil) should be false")
	}
	if IsInvalidTransition(otherErr) {
		t.Error("IsInvalidTransition(other) should be false")
	}

	if IsInvalidInput(nil) {
		t.Error("IsInvalidInput(nil) should be false")
	}
	if IsInvalidInput(otherErr) {
		t.Error("IsInvalidInput(other) should be false")
	}
}
