// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/intern-village/orchestrator/internal/domain"
)

func TestEventHub_Subscribe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()

	// Subscribe
	connID, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	assert.NotEmpty(t, connID)
	assert.NotNil(t, eventChan)
	assert.Equal(t, 1, hub.ConnectionCount(projectID))
	assert.Equal(t, 1, hub.UserConnectionCount(userID))
}

func TestEventHub_SubscribeMultiple(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	// Subscribe two connections
	_, _, cleanup1 := hub.Subscribe(projectID, userID1, nil)
	defer cleanup1()

	_, _, cleanup2 := hub.Subscribe(projectID, userID2, nil)
	defer cleanup2()

	assert.Equal(t, 2, hub.ConnectionCount(projectID))
	assert.Equal(t, 1, hub.UserConnectionCount(userID1))
	assert.Equal(t, 1, hub.UserConnectionCount(userID2))
}

func TestEventHub_Cleanup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()

	// Subscribe and cleanup
	_, _, cleanup := hub.Subscribe(projectID, userID, nil)
	assert.Equal(t, 1, hub.ConnectionCount(projectID))

	cleanup()
	assert.Equal(t, 0, hub.ConnectionCount(projectID))
	assert.Equal(t, 0, hub.UserConnectionCount(userID))
}

func TestEventHub_PublishTaskStatusChanged(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()
	taskID := uuid.New()

	_, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	// Publish event
	hub.PublishTaskStatusChanged(projectID, taskID, "PLANNING", "ACTIVE")

	// Receive event
	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeTaskStatusChanged, event.Type)
		data, ok := event.Data.(TaskStatusChangedData)
		require.True(t, ok)
		assert.Equal(t, taskID, data.TaskID)
		assert.Equal(t, "PLANNING", data.OldStatus)
		assert.Equal(t, "ACTIVE", data.NewStatus)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventHub_PublishToCorrectProjectOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID1 := uuid.New()
	projectID2 := uuid.New()
	userID := uuid.New()
	taskID := uuid.New()

	_, eventChan1, cleanup1 := hub.Subscribe(projectID1, userID, nil)
	defer cleanup1()

	_, eventChan2, cleanup2 := hub.Subscribe(projectID2, userID, nil)
	defer cleanup2()

	// Publish event to project1 only
	hub.PublishTaskStatusChanged(projectID1, taskID, "PLANNING", "ACTIVE")

	// Project1 should receive the event
	select {
	case event := <-eventChan1:
		assert.Equal(t, EventTypeTaskStatusChanged, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event on project1")
	}

	// Project2 should NOT receive the event
	select {
	case <-eventChan2:
		t.Fatal("project2 should not receive events from project1")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event
	}
}

func TestEventHub_LogSubscriptions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()
	runID := uuid.New()

	// Subscribe without log subscription
	connID, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	// Publish log event - should not be received (not subscribed)
	hub.PublishAgentLog(projectID, runID, "test line", 1, "14:32:05")

	select {
	case <-eventChan:
		t.Fatal("should not receive log events when not subscribed")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event
	}

	// Update subscription to include this run
	hub.UpdateLogSubscriptions(connID, []uuid.UUID{runID})

	// Publish log event - should be received now
	hub.PublishAgentLog(projectID, runID, "test line 2", 2, "14:32:06")

	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeAgentLog, event.Type)
		data, ok := event.Data.(AgentLogData)
		require.True(t, ok)
		assert.Equal(t, "test line 2", data.Line)
		assert.Equal(t, 2, data.LineNumber)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for log event")
	}
}

func TestEventHub_LogSubscriptionInitial(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()
	runID := uuid.New()

	// Subscribe with initial log subscription
	_, eventChan, cleanup := hub.Subscribe(projectID, userID, []uuid.UUID{runID})
	defer cleanup()

	// Publish log event - should be received
	hub.PublishAgentLog(projectID, runID, "test line", 1, "14:32:05")

	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeAgentLog, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for log event")
	}
}

func TestEventHub_ChannelFull(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	// Small buffer to test overflow
	hub := NewEventHub(2, logger)

	projectID := uuid.New()
	userID := uuid.New()
	taskID := uuid.New()

	_, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	// Fill the buffer
	hub.PublishTaskStatusChanged(projectID, taskID, "PLANNING", "ACTIVE")
	hub.PublishTaskStatusChanged(projectID, taskID, "ACTIVE", "DONE")

	// This should be dropped (buffer full)
	hub.PublishTaskStatusChanged(projectID, taskID, "DONE", "ARCHIVED")

	// Drain events
	events := make([]Event, 0)
	for i := 0; i < 3; i++ {
		select {
		case event := <-eventChan:
			events = append(events, event)
		case <-time.After(50 * time.Millisecond):
			break
		}
	}

	// Should only have 2 events (third was dropped)
	assert.Equal(t, 2, len(events))
}

func TestEventHub_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	taskID := uuid.New()

	done := make(chan bool)

	// Spawn multiple goroutines that subscribe and unsubscribe
	for i := 0; i < 10; i++ {
		go func() {
			userID := uuid.New()
			_, _, cleanup := hub.Subscribe(projectID, userID, nil)
			time.Sleep(10 * time.Millisecond)
			cleanup()
			done <- true
		}()
	}

	// Concurrently publish events
	go func() {
		for i := 0; i < 20; i++ {
			hub.PublishTaskStatusChanged(projectID, taskID, "PLANNING", "ACTIVE")
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 11; i++ {
		<-done
	}

	// Should not panic and final count should be 0
	assert.Equal(t, 0, hub.ConnectionCount(projectID))
}

func TestEventHub_PublishAgentStarted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()
	taskID := uuid.New()
	subtaskID := uuid.New()

	_, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	run := &domain.AgentRun{
		ID:            uuid.New(),
		SubtaskID:     &subtaskID,
		AgentType:     domain.AgentTypeWorker,
		AttemptNumber: 1,
		StartedAt:     time.Now(),
	}

	hub.PublishAgentStarted(projectID, run, taskID)

	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeAgentStarted, event.Type)
		data, ok := event.Data.(AgentStartedData)
		require.True(t, ok)
		assert.Equal(t, run.ID, data.RunID)
		assert.Equal(t, "WORKER", data.AgentType)
		assert.Equal(t, taskID, data.TaskID)
		assert.NotNil(t, data.SubtaskID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventHub_PublishAgentCompleted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()
	taskID := uuid.New()

	_, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	startedAt := time.Now().Add(-10 * time.Second)
	endedAt := time.Now()
	tokenUsage := 1500

	run := &domain.AgentRun{
		ID:            uuid.New(),
		TaskID:        &taskID, // For planner, TaskID is set instead of SubtaskID
		AgentType:     domain.AgentTypePlanner,
		AttemptNumber: 1,
		StartedAt:     startedAt,
		EndedAt:       &endedAt,
		TokenUsage:    &tokenUsage,
	}

	hub.PublishAgentCompleted(projectID, run, taskID, "")

	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeAgentCompleted, event.Type)
		data, ok := event.Data.(AgentCompletedData)
		require.True(t, ok)
		assert.Equal(t, run.ID, data.RunID)
		assert.Equal(t, "PLANNER", data.AgentType)
		assert.InDelta(t, 10000, data.DurationMs, 100)
		assert.Equal(t, 1500, data.TokenUsage)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventHub_PublishAgentFailed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()
	taskID := uuid.New()
	subtaskID := uuid.New()

	_, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	run := &domain.AgentRun{
		ID:            uuid.New(),
		SubtaskID:     &subtaskID,
		AgentType:     domain.AgentTypeWorker,
		AttemptNumber: 3,
		StartedAt:     time.Now(),
	}

	nextAttempt := time.Now().Add(30 * time.Second)
	hub.PublishAgentFailed(projectID, run, taskID, "exit code: 1", true, &nextAttempt)

	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeAgentFailed, event.Type)
		data, ok := event.Data.(AgentFailedData)
		require.True(t, ok)
		assert.Equal(t, run.ID, data.RunID)
		assert.Equal(t, "exit code: 1", data.Error)
		assert.True(t, data.WillRetry)
		assert.NotNil(t, data.NextAttemptAt)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventHub_PublishSubtaskStatusChanged(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()

	_, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	prURL := "https://github.com/owner/repo/pull/42"
	prNumber := 42

	subtask := &domain.Subtask{
		ID:       uuid.New(),
		TaskID:   uuid.New(),
		Status:   domain.SubtaskStatusCompleted,
		PRUrl:    &prURL,
		PRNumber: &prNumber,
	}

	hub.PublishSubtaskStatusChanged(projectID, subtask, "IN_PROGRESS")

	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeSubtaskStatusChanged, event.Type)
		data, ok := event.Data.(SubtaskStatusChangedData)
		require.True(t, ok)
		assert.Equal(t, subtask.ID, data.SubtaskID)
		assert.Equal(t, "IN_PROGRESS", data.OldStatus)
		assert.Equal(t, "COMPLETED", data.NewStatus)
		assert.Equal(t, &prURL, data.PRUrl)
		assert.Equal(t, &prNumber, data.PRNumber)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventHub_PublishSubtaskUnblocked(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := NewEventHub(100, logger)

	projectID := uuid.New()
	userID := uuid.New()
	taskID := uuid.New()
	subtaskID := uuid.New()
	unblockedByID := uuid.New()

	_, eventChan, cleanup := hub.Subscribe(projectID, userID, nil)
	defer cleanup()

	hub.PublishSubtaskUnblocked(projectID, taskID, subtaskID, unblockedByID)

	select {
	case event := <-eventChan:
		assert.Equal(t, EventTypeSubtaskUnblocked, event.Type)
		data, ok := event.Data.(SubtaskUnblockedData)
		require.True(t, ok)
		assert.Equal(t, subtaskID, data.SubtaskID)
		assert.Equal(t, taskID, data.TaskID)
		assert.Equal(t, unblockedByID, data.UnblockedBy)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventHub_DefaultBufferSize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	// Test with 0 buffer size - should default to 100
	hub := NewEventHub(0, logger).(*eventHub)
	assert.Equal(t, 100, hub.bufferSize)
}
