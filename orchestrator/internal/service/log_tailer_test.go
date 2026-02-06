// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/intern-village/orchestrator/internal/domain"
)

// mockEventHub is a test implementation of EventHub for log tailer tests.
type mockEventHub struct {
	logs []AgentLogData
}

func (m *mockEventHub) Subscribe(projectID, userID uuid.UUID, logSubscriptions []uuid.UUID) (string, <-chan Event, func()) {
	return "", nil, func() {}
}

func (m *mockEventHub) UpdateLogSubscriptions(connID string, runIDs []uuid.UUID) {}
func (m *mockEventHub) ConnectionCount(projectID uuid.UUID) int                  { return 0 }
func (m *mockEventHub) UserConnectionCount(userID uuid.UUID) int                 { return 0 }

func (m *mockEventHub) PublishAgentStarted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID) {
}
func (m *mockEventHub) PublishAgentLog(projectID, runID uuid.UUID, line string, lineNumber int, timestamp string) {
	m.logs = append(m.logs, AgentLogData{
		RunID:      runID,
		Line:       line,
		LineNumber: lineNumber,
		Timestamp:  timestamp,
	})
}
func (m *mockEventHub) PublishAgentCompleted(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, prURL string) {
}
func (m *mockEventHub) PublishAgentFailed(projectID uuid.UUID, run *domain.AgentRun, taskID uuid.UUID, errMsg string, willRetry bool, nextAttemptAt *time.Time) {
}
func (m *mockEventHub) PublishTaskStatusChanged(projectID, taskID uuid.UUID, oldStatus, newStatus string) {
}
func (m *mockEventHub) PublishSubtaskStatusChanged(projectID uuid.UUID, subtask *domain.Subtask, oldStatus string) {
}
func (m *mockEventHub) PublishSubtaskUnblocked(projectID, taskID, subtaskID, unblockedByID uuid.UUID) {
}

func TestLogTailer_TailsNewLines(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockHub := &mockEventHub{logs: make([]AgentLogData, 0)}

	// Type assertion to satisfy EventHub interface
	var hub EventHub = mockHub

	tailer := NewLogTailer(hub, LogTailerConfig{
		PollInterval: 10 * time.Millisecond,
		MaxLineBytes: 1024,
	}, logger)

	projectID := uuid.New()
	runID := uuid.New()

	// Create temp file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Write initial content
	f, err := os.Create(logPath)
	require.NoError(t, err)

	_, err = f.WriteString("[14:32:05] First line\n")
	require.NoError(t, err)
	_, err = f.WriteString("[14:32:06] Second line\n")
	require.NoError(t, err)
	f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)

	go func() {
		_ = tailer.StartTailing(ctx, projectID, runID, logPath)
		done <- true
	}()

	// Wait a bit then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	// Verify logs were received
	assert.GreaterOrEqual(t, len(mockHub.logs), 2)
	assert.Equal(t, "[14:32:05] First line", mockHub.logs[0].Line)
	assert.Equal(t, "14:32:05", mockHub.logs[0].Timestamp)
	assert.Equal(t, "[14:32:06] Second line", mockHub.logs[1].Line)
}

func TestLogTailer_WaitsForFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockHub := &mockEventHub{logs: make([]AgentLogData, 0)}
	var hub EventHub = mockHub

	tailer := NewLogTailer(hub, LogTailerConfig{
		PollInterval: 10 * time.Millisecond,
	}, logger)

	projectID := uuid.New()
	runID := uuid.New()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "delayed.log")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)

	go func() {
		_ = tailer.StartTailing(ctx, projectID, runID, logPath)
		done <- true
	}()

	// Create file after a short delay (within the 5s wait window)
	time.Sleep(100 * time.Millisecond)
	f, err := os.Create(logPath)
	require.NoError(t, err)
	_, err = f.WriteString("[14:32:05] Delayed line\n")
	require.NoError(t, err)
	_, err = f.WriteString("=== Run Complete ===\n")
	require.NoError(t, err)
	f.Close()

	// Wait for tailer to complete
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		cancel()
		<-done
	}

	require.GreaterOrEqual(t, len(mockHub.logs), 1, "expected at least one log line")
	assert.Equal(t, "[14:32:05] Delayed line", mockHub.logs[0].Line)
}

func TestLogTailer_StopsOnContextCancel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockHub := &mockEventHub{logs: make([]AgentLogData, 0)}
	var hub EventHub = mockHub

	tailer := NewLogTailer(hub, DefaultLogTailerConfig(), logger)

	projectID := uuid.New()
	runID := uuid.New()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	f, err := os.Create(logPath)
	require.NoError(t, err)
	f.WriteString("[14:32:05] Line\n")
	f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)

	go func() {
		err := tailer.StartTailing(ctx, projectID, runID, logPath)
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err = <-done
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLogTailer_StopsOnSentinel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockHub := &mockEventHub{logs: make([]AgentLogData, 0)}
	var hub EventHub = mockHub

	tailer := NewLogTailer(hub, LogTailerConfig{
		PollInterval: 10 * time.Millisecond,
	}, logger)

	projectID := uuid.New()
	runID := uuid.New()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	f, err := os.Create(logPath)
	require.NoError(t, err)
	f.WriteString("[14:32:05] Working...\n")
	f.WriteString("=== Run Complete ===\n")
	f.Close()

	ctx := context.Background()
	err = tailer.StartTailing(ctx, projectID, runID, logPath)

	// Should return nil (completed successfully, not cancelled)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(mockHub.logs), 2)
}

func TestLogTailer_TruncatesLongLines(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockHub := &mockEventHub{logs: make([]AgentLogData, 0)}
	var hub EventHub = mockHub

	tailer := NewLogTailer(hub, LogTailerConfig{
		PollInterval: 10 * time.Millisecond,
		MaxLineBytes: 20,
	}, logger)

	projectID := uuid.New()
	runID := uuid.New()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	f, err := os.Create(logPath)
	require.NoError(t, err)
	f.WriteString("This is a very long line that should be truncated\n")
	f.WriteString("=== Run Complete ===\n")
	f.Close()

	err = tailer.StartTailing(context.Background(), projectID, runID, logPath)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, len(mockHub.logs), 1)
	assert.Contains(t, mockHub.logs[0].Line, "... (truncated)")
	assert.LessOrEqual(t, len(mockHub.logs[0].Line), 40) // 20 + truncation message
}

func TestLogTailer_DuplicateStartTailing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockHub := &mockEventHub{logs: make([]AgentLogData, 0)}
	var hub EventHub = mockHub

	tailer := NewLogTailer(hub, DefaultLogTailerConfig(), logger)

	projectID := uuid.New()
	runID := uuid.New()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	f, err := os.Create(logPath)
	require.NoError(t, err)
	f.WriteString("[14:32:05] Line\n")
	f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done1 := make(chan bool)
	go func() {
		_ = tailer.StartTailing(ctx, projectID, runID, logPath)
		done1 <- true
	}()

	time.Sleep(50 * time.Millisecond)

	// Second call should return immediately (already tailing)
	done2 := make(chan bool)
	go func() {
		_ = tailer.StartTailing(ctx, projectID, runID, logPath)
		done2 <- true
	}()

	select {
	case <-done2:
		// Expected - second call returns immediately
	case <-time.After(100 * time.Millisecond):
		t.Fatal("second StartTailing should return immediately")
	}

	cancel()
	<-done1
}

func TestLogTailer_StopTailing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockHub := &mockEventHub{logs: make([]AgentLogData, 0)}
	var hub EventHub = mockHub

	tailer := NewLogTailer(hub, DefaultLogTailerConfig(), logger)

	projectID := uuid.New()
	runID := uuid.New()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	f, err := os.Create(logPath)
	require.NoError(t, err)
	f.WriteString("[14:32:05] Line\n")
	f.Close()

	ctx := context.Background()
	done := make(chan bool)

	go func() {
		_ = tailer.StartTailing(ctx, projectID, runID, logPath)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	assert.True(t, tailer.IsActive(runID))

	tailer.StopTailing(runID)

	select {
	case <-done:
		// Expected - tailer stopped
	case <-time.After(500 * time.Millisecond):
		t.Fatal("StopTailing should stop the tailer")
	}

	assert.False(t, tailer.IsActive(runID))
}

func TestLogTailer_DefaultConfig(t *testing.T) {
	cfg := DefaultLogTailerConfig()
	assert.Equal(t, 100*time.Millisecond, cfg.PollInterval)
	assert.Equal(t, 1048576, cfg.MaxLineBytes)
}
