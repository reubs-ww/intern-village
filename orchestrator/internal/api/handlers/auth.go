// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

// Package handlers contains HTTP request handlers.
package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/api/middleware"
	"github.com/intern-village/orchestrator/internal/api/response"
	"github.com/intern-village/orchestrator/internal/config"
	"github.com/intern-village/orchestrator/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService *service.AuthService
	cfg         *config.Config
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
	}
}

// UserResponse represents the user information returned in API responses.
type UserResponse struct {
	ID             string `json:"id"`
	GitHubUsername string `json:"github_username"`
	CreatedAt      string `json:"created_at"`
}

// AuthResponse represents the response after successful authentication.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// InitiateOAuth redirects the user to GitHub OAuth authorization.
// GET /api/auth/github
func (h *AuthHandler) InitiateOAuth(w http.ResponseWriter, r *http.Request) {
	// Generate a random state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		log.Error().Err(err).Msg("failed to generate OAuth state")
		response.InternalError(w, err)
		return
	}

	// Store state in a secure, HttpOnly cookie for validation on callback
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	// Redirect to GitHub OAuth
	authURL := h.authService.GetAuthURL(state)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles the GitHub OAuth callback.
// GET /api/auth/github/callback
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify state for CSRF protection
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		log.Warn().Msg("missing OAuth state cookie")
		response.BadRequest(w, "missing OAuth state")
		return
	}

	queryState := r.URL.Query().Get("state")
	if queryState == "" || queryState != stateCookie.Value {
		log.Warn().Msg("OAuth state mismatch")
		response.BadRequest(w, "invalid OAuth state")
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})

	// Check for OAuth error
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		log.Warn().
			Str("error", errMsg).
			Str("description", errDesc).
			Msg("GitHub OAuth error")
		response.BadRequest(w, "GitHub authentication failed: "+errDesc)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Warn().Msg("missing OAuth code")
		response.BadRequest(w, "missing authorization code")
		return
	}

	token, err := h.authService.ExchangeCode(ctx, code)
	if err != nil {
		log.Error().Err(err).Msg("failed to exchange OAuth code")
		response.InternalError(w, err)
		return
	}

	// Fetch GitHub user info
	ghUser, err := h.authService.FetchGitHubUser(ctx, token.AccessToken)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch GitHub user")
		response.InternalError(w, err)
		return
	}

	// Create or update user in database
	user, err := h.authService.CreateOrUpdateUser(ctx, ghUser, token.AccessToken)
	if err != nil {
		log.Error().Err(err).Msg("failed to create/update user")
		response.InternalError(w, err)
		return
	}

	// Generate JWT
	jwtToken, err := h.authService.GenerateJWT(user)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate JWT")
		response.InternalError(w, err)
		return
	}

	log.Info().
		Str("user_id", user.ID.String()).
		Str("github_username", user.GitHubUsername).
		Msg("user authenticated")

	// Set JWT as HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	// Redirect to frontend
	// In development, frontend is on port 5173; in production, same origin
	frontendURL := "http://localhost:5173"
	http.Redirect(w, r, frontendURL, http.StatusTemporaryRedirect)
}

// Logout handles user logout.
// POST /api/auth/logout
// For MVP, logout is client-side (discard token). This endpoint is a placeholder.
func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	// Clear any auth-related cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})

	response.OK(w, map[string]string{"message": "logged out"})
}

// GetCurrentUser returns the current authenticated user's information.
// GET /api/auth/me
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "not authenticated")
		return
	}

	response.OK(w, UserResponse{
		ID:             user.ID.String(),
		GitHubUsername: user.GitHubUsername,
		CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// generateRandomState generates a cryptographically secure random state string.
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
