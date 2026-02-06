// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LogTailer provides functionality to tail log files and publish lines to the EventHub.
type LogTailer interface {
	// StartTailing begins tailing a log file and publishing lines to the EventHub.
	// It blocks until the context is cancelled or the log file indicates completion.
	StartTailing(ctx context.Context, projectID, runID uuid.UUID, logPath string) error

	// StopTailing stops tailing a specific run's log file.
	StopTailing(runID uuid.UUID)

	// IsActive returns true if the tailer is currently tailing a specific run.
	IsActive(runID uuid.UUID) bool
}

// logTailer is the default implementation of LogTailer.
type logTailer struct {
	eventHub     EventHub
	mu           sync.Mutex
	activeTails  map[uuid.UUID]context.CancelFunc
	pollInterval time.Duration
	maxLineBytes int
	logger       *slog.Logger
}

// LogTailerConfig contains configuration for the log tailer.
type LogTailerConfig struct {
	PollInterval time.Duration
	MaxLineBytes int
}

// DefaultLogTailerConfig returns the default configuration for the log tailer.
func DefaultLogTailerConfig() LogTailerConfig {
	return LogTailerConfig{
		PollInterval: 100 * time.Millisecond,
		MaxLineBytes: 1048576, // 1MB
	}
}

// NewLogTailer creates a new LogTailer.
func NewLogTailer(eventHub EventHub, cfg LogTailerConfig, logger *slog.Logger) LogTailer {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 100 * time.Millisecond
	}
	if cfg.MaxLineBytes <= 0 {
		cfg.MaxLineBytes = 1048576
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &logTailer{
		eventHub:     eventHub,
		activeTails:  make(map[uuid.UUID]context.CancelFunc),
		pollInterval: cfg.PollInterval,
		maxLineBytes: cfg.MaxLineBytes,
		logger:       logger,
	}
}

// timestampRegex matches log line timestamps like [14:32:05]
var timestampRegex = regexp.MustCompile(`^\[(\d{2}:\d{2}:\d{2})\]`)

// StartTailing begins tailing a log file and publishing lines to the EventHub.
func (t *logTailer) StartTailing(ctx context.Context, projectID, runID uuid.UUID, logPath string) error {
	// Check if already tailing this run
	t.mu.Lock()
	if _, exists := t.activeTails[runID]; exists {
		t.mu.Unlock()
		t.logger.Debug("already tailing this run", "run_id", runID)
		return nil
	}

	// Create a cancellable context for this tailer
	tailCtx, cancel := context.WithCancel(ctx)
	t.activeTails[runID] = cancel
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.activeTails, runID)
		t.mu.Unlock()
	}()

	t.logger.Debug("starting log tail",
		"run_id", runID,
		"log_path", logPath,
	)

	// Wait for the file to exist (up to 5 seconds)
	var file *os.File
	var err error
	for i := 0; i < 50; i++ {
		file, err = os.Open(logPath)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		select {
		case <-tailCtx.Done():
			return tailCtx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	if file == nil {
		t.logger.Warn("log file not found after waiting", "log_path", logPath)
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	lineNumber := 0

	for {
		select {
		case <-tailCtx.Done():
			t.logger.Debug("log tail cancelled", "run_id", runID)
			return tailCtx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Check if the run is complete
				if strings.Contains(line, "=== Run Complete ===") {
					t.publishLine(projectID, runID, line, lineNumber+1)
					t.logger.Debug("log tail complete (sentinel found)", "run_id", runID)
					return nil
				}

				// No new data, wait and try again
				select {
				case <-tailCtx.Done():
					return tailCtx.Err()
				case <-time.After(t.pollInterval):
				}
				continue
			}
			t.logger.Error("error reading log file", "error", err)
			return err
		}

		lineNumber++
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		// Truncate very long lines
		if len(line) > t.maxLineBytes {
			line = line[:t.maxLineBytes] + "... (truncated)"
		}

		t.publishLine(projectID, runID, line, lineNumber)

		// Check for completion sentinel
		if strings.Contains(line, "=== Run Complete ===") {
			t.logger.Debug("log tail complete (sentinel found)", "run_id", runID)
			return nil
		}
	}
}

// publishLine publishes a single log line to the EventHub.
func (t *logTailer) publishLine(projectID, runID uuid.UUID, line string, lineNumber int) {
	timestamp := ""
	if matches := timestampRegex.FindStringSubmatch(line); len(matches) > 1 {
		timestamp = matches[1]
	}

	t.eventHub.PublishAgentLog(projectID, runID, line, lineNumber, timestamp)
}

// StopTailing stops tailing a specific run's log file.
func (t *logTailer) StopTailing(runID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if cancel, exists := t.activeTails[runID]; exists {
		cancel()
		delete(t.activeTails, runID)
		t.logger.Debug("stopped log tail", "run_id", runID)
	}
}

// IsActive returns true if the tailer is currently tailing a specific run.
func (t *logTailer) IsActive(runID uuid.UUID) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, exists := t.activeTails[runID]
	return exists
}
