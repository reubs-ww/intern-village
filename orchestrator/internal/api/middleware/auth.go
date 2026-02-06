// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/domain"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// UserContextKey is the key for the authenticated user in the context.
	UserContextKey contextKey = "user"
)

// JWTValidator defines the interface for JWT validation.
type JWTValidator interface {
	ValidateJWT(token string) (*domain.User, error)
}

// AuthMiddleware provides authentication middleware.
type AuthMiddleware struct {
	validator JWTValidator
}

// NewAuthMiddleware creates a new AuthMiddleware.
func NewAuthMiddleware(validator JWTValidator) *AuthMiddleware {
	return &AuthMiddleware{validator: validator}
}

// RequireAuth is a middleware that requires a valid JWT token.
// It checks for the token in:
// 1. Authorization header (Bearer token)
// 2. auth_token cookie
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeUnauthorized(w, "missing authentication")
			return
		}

		// Validate the token
		user, err := m.validator.ValidateJWT(token)
		if err != nil {
			log.Debug().Err(err).Msg("JWT validation failed")
			writeUnauthorized(w, "invalid or expired token")
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken extracts the JWT token from the request.
// It checks the Authorization header first, then the auth_token cookie.
func extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" {
			return parts[1]
		}
	}

	// Fall back to auth_token cookie
	cookie, err := r.Cookie("auth_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// writeUnauthorized writes a 401 Unauthorized response.
func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "UNAUTHORIZED",
		"message": message,
	})
}

// GetUserFromContext retrieves the authenticated user from the request context.
func GetUserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*domain.User)
	return user, ok
}

// SetUserInContext adds the user to the context. Used primarily for testing.
func SetUserInContext(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// GetUserIDFromContext retrieves the authenticated user's ID from the request context.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return user.ID, true
}

// OptionalAuth is a middleware that attempts to authenticate but allows unauthenticated requests.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.validator.ValidateJWT(token)
		if err != nil {
			// Log but continue without authentication
			log.Debug().Err(err).Msg("optional JWT validation failed")
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
