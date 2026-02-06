// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for common domain error conditions.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict indicates a conflict with the current state.
	ErrConflict = errors.New("conflict with current state")

	// ErrForbidden indicates the user does not have permission.
	ErrForbidden = errors.New("forbidden")

	// ErrUnprocessable indicates the request cannot be processed.
	ErrUnprocessable = errors.New("request cannot be processed")

	// ErrInvalidTransition indicates an invalid state machine transition.
	ErrInvalidTransition = errors.New("invalid state transition")

	// ErrAlreadyExists indicates the resource already exists.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates the input is invalid.
	ErrInvalidInput = errors.New("invalid input")
)

// NotFoundError represents a not found error with details.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %s not found", e.Resource, e.ID)
}

func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// NewNotFoundError creates a new NotFoundError.
func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{Resource: resource, ID: id}
}

// ConflictError represents a conflict error with details.
type ConflictError struct {
	Resource string
	Reason   string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict on %s: %s", e.Resource, e.Reason)
}

func (e *ConflictError) Unwrap() error {
	return ErrConflict
}

// NewConflictError creates a new ConflictError.
func NewConflictError(resource, reason string) *ConflictError {
	return &ConflictError{Resource: resource, Reason: reason}
}

// ForbiddenError represents a forbidden error with details.
type ForbiddenError struct {
	Resource string
	Reason   string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("forbidden access to %s: %s", e.Resource, e.Reason)
}

func (e *ForbiddenError) Unwrap() error {
	return ErrForbidden
}

// NewForbiddenError creates a new ForbiddenError.
func NewForbiddenError(resource, reason string) *ForbiddenError {
	return &ForbiddenError{Resource: resource, Reason: reason}
}

// UnprocessableError represents an unprocessable error with details.
type UnprocessableError struct {
	Resource string
	Reason   string
}

func (e *UnprocessableError) Error() string {
	return fmt.Sprintf("cannot process %s: %s", e.Resource, e.Reason)
}

func (e *UnprocessableError) Unwrap() error {
	return ErrUnprocessable
}

// NewUnprocessableError creates a new UnprocessableError.
func NewUnprocessableError(resource, reason string) *UnprocessableError {
	return &UnprocessableError{Resource: resource, Reason: reason}
}

// InvalidTransitionError represents an invalid state transition.
type InvalidTransitionError struct {
	Resource    string
	From        string
	To          string
	Description string
}

func (e *InvalidTransitionError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("invalid %s transition from %s to %s: %s", e.Resource, e.From, e.To, e.Description)
	}
	return fmt.Sprintf("invalid %s transition from %s to %s", e.Resource, e.From, e.To)
}

func (e *InvalidTransitionError) Unwrap() error {
	return ErrInvalidTransition
}

// NewInvalidTransitionError creates a new InvalidTransitionError.
func NewInvalidTransitionError(resource, from, to, description string) *InvalidTransitionError {
	return &InvalidTransitionError{
		Resource:    resource,
		From:        from,
		To:          to,
		Description: description,
	}
}

// ValidationError represents input validation errors.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsConflict checks if an error is a conflict error.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsForbidden checks if an error is a forbidden error.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsUnprocessable checks if an error is an unprocessable error.
func IsUnprocessable(err error) bool {
	return errors.Is(err, ErrUnprocessable)
}

// IsInvalidTransition checks if an error is an invalid transition error.
func IsInvalidTransition(err error) bool {
	return errors.Is(err, ErrInvalidTransition)
}

// IsInvalidInput checks if an error is an invalid input error.
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}
