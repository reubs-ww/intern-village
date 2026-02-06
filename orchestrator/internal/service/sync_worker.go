// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package service

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// SyncWorker runs periodic sync operations in the background.
type SyncWorker struct {
	syncService    *SyncService
	projectService *ProjectService
	interval       time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup
	running        bool
	mu             sync.Mutex
}

// NewSyncWorker creates a new SyncWorker.
func NewSyncWorker(
	syncService *SyncService,
	projectService *ProjectService,
	intervalSeconds int,
) *SyncWorker {
	interval := time.Duration(intervalSeconds) * time.Second
	if interval < time.Second*5 {
		interval = time.Second * 30 // Default to 30 seconds
	}

	return &SyncWorker{
		syncService:    syncService,
		projectService: projectService,
		interval:       interval,
		stopCh:         make(chan struct{}),
	}
}

// Start starts the periodic sync worker.
func (w *SyncWorker) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return
	}

	w.running = true
	w.wg.Add(1)
	go w.run()

	log.Info().
		Dur("interval", w.interval).
		Msg("sync worker started")
}

// Stop stops the periodic sync worker gracefully.
func (w *SyncWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()

	log.Info().Msg("sync worker stopped")
}

// run is the main loop for the sync worker.
func (w *SyncWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.syncInProgressSubtasks()
		}
	}
}

// syncInProgressSubtasks syncs all subtasks that are currently in progress.
func (w *SyncWorker) syncInProgressSubtasks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subtasks, err := w.syncService.GetInProgressSubtasks(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get in-progress subtasks")
		return
	}

	if len(subtasks) == 0 {
		return
	}

	log.Debug().
		Int("count", len(subtasks)).
		Msg("syncing in-progress subtasks")

	for _, subtask := range subtasks {
		if subtask.BeadsIssueID == nil {
			continue
		}

		// Get the task to find the repo path
		task, err := w.syncService.taskService.GetTaskByIDInternal(ctx, subtask.TaskID)
		if err != nil {
			log.Error().
				Err(err).
				Str("task_id", subtask.TaskID.String()).
				Msg("failed to get task for sync")
			continue
		}

		project, err := w.projectService.GetProjectInternal(ctx, task.ProjectID)
		if err != nil {
			log.Error().
				Err(err).
				Str("project_id", task.ProjectID.String()).
				Msg("failed to get project for sync")
			continue
		}

		// Sync the subtask
		if err := w.syncService.SyncSubtaskFromBeads(ctx, subtask.ID, project.ClonePath); err != nil {
			log.Error().
				Err(err).
				Str("subtask_id", subtask.ID.String()).
				Msg("failed to sync subtask from beads")
		}
	}
}
