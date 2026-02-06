// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

// Package service contains business logic services.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v68/github"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"

	"github.com/intern-village/orchestrator/generated/db"
	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
)

// JWT-related errors.
var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrExpiredToken    = errors.New("token has expired")
	ErrUserNotFound    = errors.New("user not found")
	ErrOAuthFailed     = errors.New("OAuth exchange failed")
	ErrGitHubAPIFailed = errors.New("GitHub API request failed")
)

// JWTClaims represents the claims stored in a JWT token.
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID         uuid.UUID `json:"user_id"`
	GitHubUsername string    `json:"github_username"`
}

// AuthService handles authentication-related operations.
type AuthService struct {
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	repo        *repository.Repository
	crypto      *repository.Crypto
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	clientID, clientSecret, jwtSecret string,
	repo *repository.Repository,
	crypto *repository.Crypto,
) (*AuthService, error) {
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("GitHub OAuth credentials are required")
	}
	if jwtSecret == "" {
		return nil, errors.New("JWT secret is required")
	}
	if repo == nil {
		return nil, errors.New("repository is required")
	}
	if crypto == nil {
		return nil, errors.New("crypto is required")
	}

	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"read:user", "repo"},
		Endpoint:     githuboauth.Endpoint,
	}

	return &AuthService{
		oauthConfig: oauthConfig,
		jwtSecret:   []byte(jwtSecret),
		repo:        repo,
		crypto:      crypto,
	}, nil
}

// GetAuthURL returns the GitHub OAuth authorization URL.
func (s *AuthService) GetAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode exchanges an OAuth authorization code for an access token.
func (s *AuthService) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthFailed, err)
	}
	return token, nil
}

// FetchGitHubUser fetches the GitHub user profile using an access token.
func (s *AuthService) FetchGitHubUser(ctx context.Context, accessToken string) (*github.User, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGitHubAPIFailed, err)
	}

	return user, nil
}

// CreateOrUpdateUser creates or updates a user after GitHub OAuth.
func (s *AuthService) CreateOrUpdateUser(ctx context.Context, ghUser *github.User, accessToken string) (*domain.User, error) {
	// Encrypt the access token for storage
	encryptedToken, err := s.crypto.EncryptToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	githubID := ghUser.GetID()
	username := ghUser.GetLogin()

	// Try to find existing user
	dbUser, err := s.repo.GetUserByGitHubID(ctx, githubID)
	if err != nil {
		// User doesn't exist, create new one
		newUser, err := s.repo.CreateUser(ctx, db.CreateUserParams{
			GithubID:       githubID,
			GithubUsername: username,
			GithubToken:    encryptedToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		return dbUserToDomain(newUser), nil
	}

	// User exists, update token and username
	updatedUser, err := s.repo.UpdateUser(ctx, db.UpdateUserParams{
		ID:             dbUser.ID,
		GithubToken:    encryptedToken,
		GithubUsername: username,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return dbUserToDomain(updatedUser), nil
}

// GenerateJWT generates a JWT token for a user.
func (s *AuthService) GenerateJWT(user *domain.User) (string, error) {
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
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateJWT validates a JWT token and returns the associated user.
func (s *AuthService) ValidateJWT(tokenString string) (*domain.User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Fetch user from database
	ctx := context.Background()
	dbUser, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return dbUserToDomain(dbUser), nil
}

// GetUserByID retrieves a user by ID.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	dbUser, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return dbUserToDomain(dbUser), nil
}

// DecryptUserToken decrypts and returns a user's GitHub access token.
func (s *AuthService) DecryptUserToken(user *domain.User) (string, error) {
	return s.crypto.DecryptToken(user.GitHubToken)
}

// dbUserToDomain converts a database User to a domain User.
func dbUserToDomain(u db.User) *domain.User {
	return &domain.User{
		ID:             u.ID,
		GitHubID:       u.GithubID,
		GitHubUsername: u.GithubUsername,
		GitHubToken:    u.GithubToken,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}
