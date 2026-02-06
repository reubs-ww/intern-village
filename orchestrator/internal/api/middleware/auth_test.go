// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/intern-village/orchestrator/internal/domain"
)

// mockValidator is a test implementation of JWTValidator.
type mockValidator struct {
	user *domain.User
	err  error
}

func (m *mockValidator) ValidateJWT(token string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func TestRequireAuth_ValidToken(t *testing.T) {
	testUser := &domain.User{
		ID:             uuid.New(),
		GitHubID:       12345,
		GitHubUsername: "testuser",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	validator := &mockValidator{user: testUser}
	middleware := NewAuthMiddleware(validator)

	// Create a test handler that checks for user in context
	var capturedUser *domain.User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if ok {
			capturedUser = user
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()

	middleware.RequireAuth(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if capturedUser == nil {
		t.Error("expected user in context, got nil")
	} else if capturedUser.ID != testUser.ID {
		t.Errorf("expected user ID %v, got %v", testUser.ID, capturedUser.ID)
	}
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	validator := &mockValidator{user: nil, err: nil}
	middleware := NewAuthMiddleware(validator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header
	rr := httptest.NewRecorder()

	middleware.RequireAuth(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestRequireAuth_InvalidHeaderFormat(t *testing.T) {
	validator := &mockValidator{user: nil, err: nil}
	middleware := NewAuthMiddleware(validator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	testCases := []struct {
		name   string
		header string
	}{
		{"no bearer", "token-without-bearer"},
		{"basic auth", "Basic dGVzdDp0ZXN0"},
		{"empty bearer", "Bearer "},
		{"only bearer", "Bearer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tc.header)
			rr := httptest.NewRecorder()

			middleware.RequireAuth(handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rr.Code)
			}
		})
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	validator := &mockValidator{user: nil, err: ErrInvalidToken}
	middleware := NewAuthMiddleware(validator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()

	middleware.RequireAuth(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestOptionalAuth_WithValidToken(t *testing.T) {
	testUser := &domain.User{
		ID:             uuid.New(),
		GitHubID:       12345,
		GitHubUsername: "testuser",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	validator := &mockValidator{user: testUser}
	middleware := NewAuthMiddleware(validator)

	var capturedUser *domain.User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if ok {
			capturedUser = user
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()

	middleware.OptionalAuth(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if capturedUser == nil {
		t.Error("expected user in context, got nil")
	}
}

func TestOptionalAuth_WithoutToken(t *testing.T) {
	validator := &mockValidator{user: nil}
	middleware := NewAuthMiddleware(validator)

	var capturedUser *domain.User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if ok {
			capturedUser = user
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header
	rr := httptest.NewRecorder()

	middleware.OptionalAuth(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if capturedUser != nil {
		t.Error("expected no user in context")
	}
}

func TestOptionalAuth_WithInvalidToken(t *testing.T) {
	validator := &mockValidator{user: nil, err: ErrInvalidToken}
	middleware := NewAuthMiddleware(validator)

	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()

	middleware.OptionalAuth(handler).ServeHTTP(rr, req)

	// Should still proceed without auth
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}

func TestGetUserFromContext(t *testing.T) {
	t.Run("with user", func(t *testing.T) {
		testUser := &domain.User{
			ID:             uuid.New(),
			GitHubUsername: "testuser",
		}

		validator := &mockValidator{user: testUser}
		middleware := NewAuthMiddleware(validator)

		var ctxUser *domain.User
		var found bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctxUser, found = GetUserFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer token")
		rr := httptest.NewRecorder()

		middleware.RequireAuth(handler).ServeHTTP(rr, req)

		if !found {
			t.Error("expected to find user")
		}
		if ctxUser.ID != testUser.ID {
			t.Errorf("expected user ID %v, got %v", testUser.ID, ctxUser.ID)
		}
	})

	t.Run("without user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, found := GetUserFromContext(r.Context())
			if found {
				t.Error("expected not to find user")
			}
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	testID := uuid.New()
	testUser := &domain.User{
		ID:             testID,
		GitHubUsername: "testuser",
	}

	validator := &mockValidator{user: testUser}
	middleware := NewAuthMiddleware(validator)

	var ctxUserID uuid.UUID
	var found bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, found = GetUserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	rr := httptest.NewRecorder()

	middleware.RequireAuth(handler).ServeHTTP(rr, req)

	if !found {
		t.Error("expected to find user ID")
	}
	if ctxUserID != testID {
		t.Errorf("expected user ID %v, got %v", testID, ctxUserID)
	}
}

// ErrInvalidToken for testing
var ErrInvalidToken = errorString("invalid token")

type errorString string

func (e errorString) Error() string { return string(e) }
