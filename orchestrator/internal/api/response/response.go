// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

// Package response provides HTTP response utilities.
package response

import (
	"encoding/json"
	"net/http"

	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/rs/zerolog/log"
)

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes matching the spec.
const (
	CodeInvalidRequest = "INVALID_REQUEST"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeNotFound       = "NOT_FOUND"
	CodeConflict       = "CONFLICT"
	CodeUnprocessable  = "UNPROCESSABLE"
	CodeInternalError  = "INTERNAL_ERROR"
)

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Error().Err(err).Msg("failed to encode JSON response")
		}
	}
}

// Error writes an error response.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// ErrorFromDomain converts a domain error to an HTTP error response.
func ErrorFromDomain(w http.ResponseWriter, err error) {
	switch {
	case domain.IsNotFound(err):
		Error(w, http.StatusNotFound, CodeNotFound, err.Error())
	case domain.IsConflict(err):
		Error(w, http.StatusConflict, CodeConflict, err.Error())
	case domain.IsForbidden(err):
		Error(w, http.StatusForbidden, CodeForbidden, err.Error())
	case domain.IsUnprocessable(err):
		Error(w, http.StatusUnprocessableEntity, CodeUnprocessable, err.Error())
	case domain.IsInvalidInput(err):
		Error(w, http.StatusBadRequest, CodeInvalidRequest, err.Error())
	case domain.IsInvalidTransition(err):
		Error(w, http.StatusConflict, CodeConflict, err.Error())
	default:
		log.Error().Err(err).Msg("internal error")
		Error(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
	}
}

// BadRequest writes a 400 Bad Request error.
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, CodeInvalidRequest, message)
}

// Unauthorized writes a 401 Unauthorized error.
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden writes a 403 Forbidden error.
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, CodeForbidden, message)
}

// NotFound writes a 404 Not Found error.
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, CodeNotFound, message)
}

// InternalError writes a 500 Internal Server Error.
func InternalError(w http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("internal server error")
	Error(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
}

// Created writes a 201 Created response.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// OK writes a 200 OK response.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
