// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/intern-village/orchestrator/internal/domain"
)

func TestGenerateAndValidateJWT(t *testing.T) {
	// Create a mock auth service with just JWT functionality
	jwtSecret := "test-secret-key-for-jwt-testing"

	user := &domain.User{
		ID:             uuid.New(),
		GitHubID:       12345,
		GitHubUsername: "testuser",
		GitHubToken:    "encrypted-token",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Test token generation
	t.Run("generate JWT", func(t *testing.T) {
		token, err := generateTestJWT(user, jwtSecret)
		if err != nil {
			t.Fatalf("failed to generate JWT: %v", err)
		}
		if token == "" {
			t.Error("expected non-empty token")
		}
	})

	// Test token validation
	t.Run("validate valid JWT", func(t *testing.T) {
		token, err := generateTestJWT(user, jwtSecret)
		if err != nil {
			t.Fatalf("failed to generate JWT: %v", err)
		}

		claims, err := validateTestJWT(token, jwtSecret)
		if err != nil {
			t.Fatalf("failed to validate JWT: %v", err)
		}

		if claims.UserID != user.ID {
			t.Errorf("expected user ID %v, got %v", user.ID, claims.UserID)
		}
		if claims.GitHubUsername != user.GitHubUsername {
			t.Errorf("expected username %s, got %s", user.GitHubUsername, claims.GitHubUsername)
		}
	})

	// Test invalid token
	t.Run("reject invalid JWT", func(t *testing.T) {
		_, err := validateTestJWT("invalid-token", jwtSecret)
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	// Test wrong secret
	t.Run("reject JWT with wrong secret", func(t *testing.T) {
		token, err := generateTestJWT(user, jwtSecret)
		if err != nil {
			t.Fatalf("failed to generate JWT: %v", err)
		}

		_, err = validateTestJWT(token, "wrong-secret")
		if err == nil {
			t.Error("expected error for wrong secret")
		}
	})

	// Test expired token
	t.Run("reject expired JWT", func(t *testing.T) {
		token, err := generateExpiredTestJWT(user, jwtSecret)
		if err != nil {
			t.Fatalf("failed to generate expired JWT: %v", err)
		}

		_, err = validateTestJWT(token, jwtSecret)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})
}

func TestJWTClaimsContent(t *testing.T) {
	jwtSecret := "test-secret-key-for-jwt-testing"
	user := &domain.User{
		ID:             uuid.New(),
		GitHubID:       12345,
		GitHubUsername: "testuser",
		GitHubToken:    "encrypted-token",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	token, err := generateTestJWT(user, jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	claims, err := validateTestJWT(token, jwtSecret)
	if err != nil {
		t.Fatalf("failed to validate JWT: %v", err)
	}

	t.Run("issuer is correct", func(t *testing.T) {
		if claims.Issuer != "intern-village" {
			t.Errorf("expected issuer 'intern-village', got '%s'", claims.Issuer)
		}
	})

	t.Run("subject is user ID", func(t *testing.T) {
		if claims.Subject != user.ID.String() {
			t.Errorf("expected subject %s, got %s", user.ID.String(), claims.Subject)
		}
	})

	t.Run("expiry is 24 hours", func(t *testing.T) {
		expiresAt := claims.ExpiresAt.Time
		issuedAt := claims.IssuedAt.Time
		duration := expiresAt.Sub(issuedAt)

		if duration != 24*time.Hour {
			t.Errorf("expected 24h duration, got %v", duration)
		}
	})
}

// Helper functions for testing

func generateTestJWT(user *domain.User, secret string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "intern-village",
			Subject:   user.ID.String(),
		},
		UserID:         user.ID,
		GitHubUsername: user.GitHubUsername,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateExpiredTestJWT(user *domain.User, secret string) (string, error) {
	past := time.Now().Add(-48 * time.Hour)
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(past.Add(24 * time.Hour)), // expired 24h ago
			IssuedAt:  jwt.NewNumericDate(past),
			NotBefore: jwt.NewNumericDate(past),
			Issuer:    "intern-village",
			Subject:   user.ID.String(),
		},
		UserID:         user.ID,
		GitHubUsername: user.GitHubUsername,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func validateTestJWT(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
