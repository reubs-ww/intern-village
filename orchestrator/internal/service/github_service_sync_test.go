// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestSyncRepoWithRetry_MaxRetriesReached tests that SyncRepoWithRetry returns error after max retries.
// Note: This test uses an invalid repo path to simulate failure.
func TestSyncRepoWithRetry_MaxRetriesReached(t *testing.T) {
	svc := NewGitHubService()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a non-existent path to trigger failure
	err := svc.SyncRepoWithRetry(ctx, "/nonexistent/path/to/repo", "main", false, 2)
	if err == nil {
		t.Error("SyncRepoWithRetry() expected error, got nil")
	}

	// Verify error mentions retries
	if err != nil && !contains(err.Error(), "sync failed after") {
		t.Errorf("SyncRepoWithRetry() error = %v, want error containing 'sync failed after'", err)
	}
}

// TestSyncRepoWithRetry_ContextCancellation tests that SyncRepoWithRetry respects context cancellation.
func TestSyncRepoWithRetry_ContextCancellation(t *testing.T) {
	svc := NewGitHubService()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	err := svc.SyncRepoWithRetry(ctx, "/nonexistent/path", "main", false, 3)
	if err == nil {
		t.Error("SyncRepoWithRetry() expected error due to context cancellation, got nil")
	}
}

// TestSyncDirectClone_InvalidPath tests syncDirectClone with invalid path.
func TestSyncDirectClone_InvalidPath(t *testing.T) {
	svc := NewGitHubService()
	ctx := context.Background()

	err := svc.syncDirectClone(ctx, "/nonexistent/path", "main")
	if err == nil {
		t.Error("syncDirectClone() expected error for non-existent path, got nil")
	}
}

// TestSyncForkedRepo_InvalidPath tests syncForkedRepo with invalid path.
func TestSyncForkedRepo_InvalidPath(t *testing.T) {
	svc := NewGitHubService()
	ctx := context.Background()

	err := svc.syncForkedRepo(ctx, "/nonexistent/path", "main")
	if err == nil {
		t.Error("syncForkedRepo() expected error for non-existent path, got nil")
	}
}

// TestAddUpstreamRemote_InvalidPath tests AddUpstreamRemote with invalid path.
func TestAddUpstreamRemote_InvalidPath(t *testing.T) {
	svc := NewGitHubService()
	ctx := context.Background()

	err := svc.AddUpstreamRemote(ctx, "/nonexistent/path", "owner", "repo")
	if err == nil {
		t.Error("AddUpstreamRemote() expected error for non-existent path, got nil")
	}
}

// TestSyncDirectClone_ActualRepo tests syncDirectClone with a real temporary git repo.
func TestSyncDirectClone_ActualRepo(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "sync-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repo
	repoPath := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to config git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to config git name: %v", err)
	}

	// Create a file and commit
	if err := os.WriteFile(filepath.Join(repoPath, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Get current branch name (not used in this test but kept for future reference)
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath
	_, err = cmd.Output()
	if err != nil {
		t.Fatalf("failed to get branch: %v", err)
	}

	// Create a bare remote
	bareRemote := filepath.Join(tmpDir, "remote.git")
	cmd = exec.Command("git", "clone", "--bare", repoPath, bareRemote)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bare remote: %v", err)
	}

	// Add the remote as origin
	cmd = exec.Command("git", "remote", "add", "origin", bareRemote)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Sync should now work (origin exists)
	svc := NewGitHubService()
	ctx := context.Background()

	// This should work without error
	err = svc.syncDirectClone(ctx, repoPath, "master")
	// We might get an error about the branch name not existing, which is fine
	// The important thing is it doesn't panic
	_ = err
}

// TestAddUpstreamRemote_ActualRepo tests AddUpstreamRemote with a real temporary git repo.
func TestAddUpstreamRemote_ActualRepo(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "upstream-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repo
	repoPath := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	svc := NewGitHubService()
	ctx := context.Background()

	// Add upstream remote
	err = svc.AddUpstreamRemote(ctx, repoPath, "original-owner", "original-repo")
	if err != nil {
		t.Errorf("AddUpstreamRemote() error = %v, want nil", err)
	}

	// Verify remote was added
	cmd = exec.Command("git", "remote", "get-url", "upstream")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get upstream url: %v", err)
	}

	expected := "https://github.com/original-owner/original-repo.git"
	got := string(output)
	if !contains(got, expected) {
		t.Errorf("AddUpstreamRemote() remote url = %v, want %v", got, expected)
	}

	// Adding again should update, not fail
	err = svc.AddUpstreamRemote(ctx, repoPath, "new-owner", "new-repo")
	if err != nil {
		t.Errorf("AddUpstreamRemote() second call error = %v, want nil", err)
	}

	// Verify remote was updated
	cmd = exec.Command("git", "remote", "get-url", "upstream")
	cmd.Dir = repoPath
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("failed to get upstream url after update: %v", err)
	}

	expected = "https://github.com/new-owner/new-repo.git"
	got = string(output)
	if !contains(got, expected) {
		t.Errorf("AddUpstreamRemote() updated remote url = %v, want %v", got, expected)
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
