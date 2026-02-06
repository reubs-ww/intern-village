// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// GitHub service errors.
var (
	ErrNoRepoAccess     = errors.New("no access to repository")
	ErrRepoNotFound     = errors.New("repository not found")
	ErrForkFailed       = errors.New("fork operation failed")
	ErrCloneFailed      = errors.New("clone operation failed")
	ErrPushFailed       = errors.New("push operation failed")
	ErrPRCreationFailed = errors.New("pull request creation failed")
	ErrInvalidRepoURL   = errors.New("invalid repository URL")
)

// RepoInfo contains information about a repository.
type RepoInfo struct {
	Owner         string
	Repo          string
	DefaultBranch string
	HasPushAccess bool
	IsFork        bool
	ParentOwner   string // Only set if IsFork is true
	ParentRepo    string // Only set if IsFork is true
}

// ForkInfo contains information about a forked repository.
type ForkInfo struct {
	Owner         string
	Repo          string
	DefaultBranch string
	CloneURL      string
}

// PRInfo contains information about a created pull request.
type PRInfo struct {
	Number  int
	URL     string
	HTMLURL string
}

// GitHubService handles GitHub API operations.
type GitHubService struct {
	// Nothing stored here - clients are created per-request with user tokens
}

// NewGitHubService creates a new GitHubService.
func NewGitHubService() *GitHubService {
	return &GitHubService{}
}

// newClient creates a GitHub client with the provided access token.
func (s *GitHubService) newClient(ctx context.Context, accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// ParseRepoURL parses a GitHub repository URL and returns owner and repo.
// Supports formats:
//   - github.com/owner/repo
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo.git
//   - git@github.com:owner/repo.git
func (s *GitHubService) ParseRepoURL(repoURL string) (owner, repo string, err error) {
	// Normalize the URL
	url := strings.TrimSpace(repoURL)
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH format: git@github.com:owner/repo
	if path, found := strings.CutPrefix(url, "git@github.com:"); found {
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return "", "", ErrInvalidRepoURL
		}
		return parts[0], parts[1], nil
	}

	// Handle HTTPS and plain formats
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", ErrInvalidRepoURL
	}

	owner = parts[0]
	repo = parts[1]

	if owner == "" || repo == "" {
		return "", "", ErrInvalidRepoURL
	}

	return owner, repo, nil
}

// GetRepoInfo fetches repository information including push access.
func (s *GitHubService) GetRepoInfo(ctx context.Context, owner, repo, accessToken string) (*RepoInfo, error) {
	client := s.newClient(ctx, accessToken)

	repository, resp, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, ErrRepoNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrGitHubAPIFailed, err)
	}

	info := &RepoInfo{
		Owner:         repository.GetOwner().GetLogin(),
		Repo:          repository.GetName(),
		DefaultBranch: repository.GetDefaultBranch(),
		HasPushAccess: repository.GetPermissions()["push"],
		IsFork:        repository.GetFork(),
	}

	// If it's a fork, get parent info
	if info.IsFork && repository.GetParent() != nil {
		info.ParentOwner = repository.GetParent().GetOwner().GetLogin()
		info.ParentRepo = repository.GetParent().GetName()
	}

	return info, nil
}

// CheckPushAccess checks if the user has push access to a repository.
func (s *GitHubService) CheckPushAccess(ctx context.Context, owner, repo, accessToken string) (bool, error) {
	info, err := s.GetRepoInfo(ctx, owner, repo, accessToken)
	if err != nil {
		return false, err
	}
	return info.HasPushAccess, nil
}

// ForkRepo forks a repository to the authenticated user's account.
func (s *GitHubService) ForkRepo(ctx context.Context, owner, repo, accessToken string) (*ForkInfo, error) {
	client := s.newClient(ctx, accessToken)

	// Create the fork
	fork, _, err := client.Repositories.CreateFork(ctx, owner, repo, &github.RepositoryCreateForkOptions{})
	if err != nil {
		// Check if it's an AcceptedError (HTTP 202) - fork is being created asynchronously
		// The library still returns fork data in this case, so we treat it as success
		var acceptedErr *github.AcceptedError
		if errors.As(err, &acceptedErr) {
			// Fork data should still be available, continue to polling
		} else if strings.Contains(err.Error(), "already exists") {
			// Check if it's an "already exists" error
			// Fetch the existing fork
			user, _, userErr := client.Users.Get(ctx, "")
			if userErr != nil {
				return nil, fmt.Errorf("%w: %v", ErrForkFailed, userErr)
			}
			existingFork, _, forkErr := client.Repositories.Get(ctx, user.GetLogin(), repo)
			if forkErr != nil {
				return nil, fmt.Errorf("%w: %v", ErrForkFailed, forkErr)
			}
			return &ForkInfo{
				Owner:         existingFork.GetOwner().GetLogin(),
				Repo:          existingFork.GetName(),
				DefaultBranch: existingFork.GetDefaultBranch(),
				CloneURL:      existingFork.GetCloneURL(),
			}, nil
		} else {
			return nil, fmt.Errorf("%w: %v", ErrForkFailed, err)
		}
	}

	// Fork creation is asynchronous, wait for it to be ready
	forkOwner := fork.GetOwner().GetLogin()
	forkRepo := fork.GetName()

	// Poll until the fork is ready with exponential backoff (max 2 minutes)
	// Large repos like anthropics/claude-code can take longer to fork
	maxDuration := 120 * time.Second
	startTime := time.Now()
	backoff := 1 * time.Second
	maxBackoff := 16 * time.Second

	for time.Since(startTime) < maxDuration {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: %v", ErrForkFailed, ctx.Err())
		case <-time.After(backoff):
		}

		forkedRepo, _, err := client.Repositories.Get(ctx, forkOwner, forkRepo)
		if err == nil && (forkedRepo.GetSize() > 0 || !forkedRepo.GetFork()) {
			return &ForkInfo{
				Owner:         forkedRepo.GetOwner().GetLogin(),
				Repo:          forkedRepo.GetName(),
				DefaultBranch: forkedRepo.GetDefaultBranch(),
				CloneURL:      forkedRepo.GetCloneURL(),
			}, nil
		}

		// Exponential backoff: 1s, 2s, 4s, 8s, 16s (capped)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	// Return the info even if not fully ready - clone might work
	return &ForkInfo{
		Owner:         fork.GetOwner().GetLogin(),
		Repo:          fork.GetName(),
		DefaultBranch: fork.GetDefaultBranch(),
		CloneURL:      fork.GetCloneURL(),
	}, nil
}

// CloneRepo clones a repository to the specified destination path.
// Uses the provided access token for authentication.
func (s *GitHubService) CloneRepo(ctx context.Context, owner, repo, accessToken, destPath string) error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("%w: failed to create parent directory: %v", ErrCloneFailed, err)
	}

	// Build clone URL with embedded token for authentication
	// Format: https://x-access-token:{token}@github.com/{owner}/{repo}.git
	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", accessToken, owner, repo)

	// Execute git clone
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v (output: %s)", ErrCloneFailed, err, string(output))
	}

	return nil
}

// PushBranch pushes a branch to the remote origin.
// The repo must have been cloned with token authentication.
func (s *GitHubService) PushBranch(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", branch)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v (output: %s)", ErrPushFailed, err, string(output))
	}
	return nil
}

// CreatePR creates a pull request.
func (s *GitHubService) CreatePR(ctx context.Context, owner, repo, accessToken, head, base, title, body string) (*PRInfo, error) {
	client := s.newClient(ctx, accessToken)

	newPR := &github.NewPullRequest{
		Title: github.Ptr(title),
		Head:  github.Ptr(head),
		Base:  github.Ptr(base),
		Body:  github.Ptr(body),
	}

	pr, _, err := client.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPRCreationFailed, err)
	}

	return &PRInfo{
		Number:  pr.GetNumber(),
		URL:     pr.GetURL(),
		HTMLURL: pr.GetHTMLURL(),
	}, nil
}

// GetCommitMessages gets commit messages for a branch compared to the base.
func (s *GitHubService) GetCommitMessages(ctx context.Context, repoPath, baseBranch string) ([]string, error) {
	// Get the list of commits that differ from the base branch
	cmd := exec.CommandContext(ctx, "git", "log", fmt.Sprintf("%s..HEAD", baseBranch), "--oneline") //nolint:gosec // baseBranch is validated
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If no commits, that's fine
		if strings.Contains(string(output), "unknown revision") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get commit messages: %v (output: %s)", err, string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var messages []string
	for _, line := range lines {
		if line != "" {
			messages = append(messages, line)
		}
	}

	return messages, nil
}

// GetCurrentBranch returns the current branch name in the repository.
func (s *GitHubService) GetCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %v (output: %s)", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// Sync errors.
var (
	ErrSyncFailed = errors.New("repository sync failed")
)

// AddUpstreamRemote adds the upstream remote to a forked repository.
// This is called once after cloning a fork to set up syncing from the original repo.
func (s *GitHubService) AddUpstreamRemote(ctx context.Context, repoPath, upstreamOwner, upstreamRepo string) error {
	upstreamURL := fmt.Sprintf("https://github.com/%s/%s.git", upstreamOwner, upstreamRepo)

	// Check if upstream remote already exists
	checkCmd := exec.CommandContext(ctx, "git", "remote", "get-url", "upstream")
	checkCmd.Dir = repoPath
	if _, err := checkCmd.CombinedOutput(); err == nil {
		// Upstream already exists, update it
		cmd := exec.CommandContext(ctx, "git", "remote", "set-url", "upstream", upstreamURL)
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%w: failed to update upstream remote: %v (output: %s)", ErrSyncFailed, err, string(output))
		}
		return nil
	}

	// Add new upstream remote
	cmd := exec.CommandContext(ctx, "git", "remote", "add", "upstream", upstreamURL)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to add upstream remote: %v (output: %s)", ErrSyncFailed, err, string(output))
	}

	return nil
}

// SyncRepo synchronizes the repository to the latest state.
// For direct clones: fetches origin and resets to origin/{defaultBranch}
// For forks: fetches upstream, resets to upstream/{defaultBranch}, and force pushes to origin
func (s *GitHubService) SyncRepo(ctx context.Context, repoPath, defaultBranch string, isFork bool) error {
	if isFork {
		return s.syncForkedRepo(ctx, repoPath, defaultBranch)
	}
	return s.syncDirectClone(ctx, repoPath, defaultBranch)
}

// syncDirectClone syncs a direct clone from origin.
func (s *GitHubService) syncDirectClone(ctx context.Context, repoPath, defaultBranch string) error {
	// Fetch origin
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	fetchCmd.Dir = repoPath
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to fetch origin: %v (output: %s)", ErrSyncFailed, err, string(output))
	}

	// Checkout default branch
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", defaultBranch)
	checkoutCmd.Dir = repoPath
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to checkout %s: %v (output: %s)", ErrSyncFailed, defaultBranch, err, string(output))
	}

	// Reset to origin/defaultBranch
	resetTarget := fmt.Sprintf("origin/%s", defaultBranch)
	resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", resetTarget)
	resetCmd.Dir = repoPath
	if output, err := resetCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to reset to %s: %v (output: %s)", ErrSyncFailed, resetTarget, err, string(output))
	}

	return nil
}

// syncForkedRepo syncs a forked repo from upstream.
func (s *GitHubService) syncForkedRepo(ctx context.Context, repoPath, defaultBranch string) error {
	// Fetch upstream
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "upstream")
	fetchCmd.Dir = repoPath
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to fetch upstream: %v (output: %s)", ErrSyncFailed, err, string(output))
	}

	// Checkout default branch
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", defaultBranch)
	checkoutCmd.Dir = repoPath
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to checkout %s: %v (output: %s)", ErrSyncFailed, defaultBranch, err, string(output))
	}

	// Reset to upstream/defaultBranch
	resetTarget := fmt.Sprintf("upstream/%s", defaultBranch)
	resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", resetTarget)
	resetCmd.Dir = repoPath
	if output, err := resetCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to reset to %s: %v (output: %s)", ErrSyncFailed, resetTarget, err, string(output))
	}

	// Force push to origin to keep fork in sync
	pushCmd := exec.CommandContext(ctx, "git", "push", "origin", defaultBranch, "--force")
	pushCmd.Dir = repoPath
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to push to origin: %v (output: %s)", ErrSyncFailed, err, string(output))
	}

	return nil
}

// SyncRepoWithRetry calls SyncRepo with retry logic.
// Retries up to maxRetries times with exponential backoff on failure.
func (s *GitHubService) SyncRepoWithRetry(ctx context.Context, repoPath, defaultBranch string, isFork bool, maxRetries int) error {
	var lastErr error
	for attempt := range maxRetries {
		if err := s.SyncRepo(ctx, repoPath, defaultBranch, isFork); err != nil {
			lastErr = err
			// Exponential backoff: 1s, 2s, 4s
			//nolint:gosec // attempt is bounded by maxRetries which is small
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("sync failed after %d attempts: %w", maxRetries, lastErr)
}
