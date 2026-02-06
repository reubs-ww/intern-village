// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/domain"
	"github.com/intern-village/orchestrator/internal/repository"
	"github.com/intern-village/orchestrator/internal/service"
)

// runningAgent represents a running agent with its cancel function.
type runningAgent struct {
	taskID    uuid.UUID
	subtaskID uuid.UUID
	agentType domain.AgentType
	cancel    context.CancelFunc
}

// AgentManager manages spawning and tracking of agents.
type AgentManager struct {
	loop           *AgentLoop
	repo           *repository.Repository
	projectService *service.ProjectService
	crypto         *repository.Crypto
	eventHub       service.EventHub

	// Track running agents
	mu            sync.RWMutex
	runningAgents map[uuid.UUID]*runningAgent // keyed by task/subtask ID

	// Shutdown handling
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAgentManager creates a new AgentManager.
func NewAgentManager(
	loop *AgentLoop,
	repo *repository.Repository,
	projectService *service.ProjectService,
	crypto *repository.Crypto,
	eventHub service.EventHub,
) *AgentManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentManager{
		loop:           loop,
		repo:           repo,
		projectService: projectService,
		crypto:         crypto,
		eventHub:       eventHub,
		runningAgents:  make(map[uuid.UUID]*runningAgent),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// SpawnPlanner spawns a Planner agent for a task.
func (m *AgentManager) SpawnPlanner(ctx context.Context, task *domain.Task, project *domain.Project) error {
	m.mu.Lock()
	if _, exists := m.runningAgents[task.ID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("planner already running for task %s", task.ID)
	}

	// Create context for this agent
	agentCtx, agentCancel := context.WithCancel(m.ctx)

	m.runningAgents[task.ID] = &runningAgent{
		taskID:    task.ID,
		agentType: domain.AgentTypePlanner,
		cancel:    agentCancel,
	}
	m.mu.Unlock()

	// Get user token for GitHub operations
	userToken, err := m.getUserToken(ctx, project.UserID)
	if err != nil {
		m.removeRunningAgent(task.ID)
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Run in goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer m.removeRunningAgent(task.ID)

		log.Info().
			Str("task_id", task.ID.String()).
			Msg("planner agent started")

		if err := m.loop.RunPlannerLoop(agentCtx, task, project, userToken); err != nil {
			log.Error().
				Err(err).
				Str("task_id", task.ID.String()).
				Msg("planner agent failed")
		}
	}()

	return nil
}

// SpawnWorker spawns a Worker agent for a subtask.
func (m *AgentManager) SpawnWorker(ctx context.Context, subtask *domain.Subtask, project *domain.Project) error {
	m.mu.Lock()
	if _, exists := m.runningAgents[subtask.ID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("worker already running for subtask %s", subtask.ID)
	}

	// Create context for this agent
	agentCtx, agentCancel := context.WithCancel(m.ctx)

	m.runningAgents[subtask.ID] = &runningAgent{
		subtaskID: subtask.ID,
		taskID:    subtask.TaskID,
		agentType: domain.AgentTypeWorker,
		cancel:    agentCancel,
	}
	m.mu.Unlock()

	// Get user token for GitHub operations
	userToken, err := m.getUserToken(ctx, project.UserID)
	if err != nil {
		m.removeRunningAgent(subtask.ID)
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Run in goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer m.removeRunningAgent(subtask.ID)

		log.Info().
			Str("subtask_id", subtask.ID.String()).
			Msg("worker agent started")

		if err := m.loop.RunWorkerLoop(agentCtx, subtask, project, userToken); err != nil {
			log.Error().
				Err(err).
				Str("subtask_id", subtask.ID.String()).
				Msg("worker agent failed")
		}
	}()

	return nil
}

// KillAgentsForTask kills all agents running for a task.
func (m *AgentManager) KillAgentsForTask(ctx context.Context, taskID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	killed := 0
	for id, agent := range m.runningAgents {
		if agent.taskID == taskID {
			agent.cancel()
			delete(m.runningAgents, id)
			killed++
			log.Info().
				Str("task_id", taskID.String()).
				Str("agent_id", id.String()).
				Msg("killed agent for task")
		}
	}

	if killed == 0 {
		log.Debug().
			Str("task_id", taskID.String()).
			Msg("no running agents found for task")
	}

	return nil
}

// KillAgentsForSubtask kills agents running for a subtask.
func (m *AgentManager) KillAgentsForSubtask(ctx context.Context, subtaskID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agent, exists := m.runningAgents[subtaskID]; exists {
		agent.cancel()
		delete(m.runningAgents, subtaskID)
		log.Info().
			Str("subtask_id", subtaskID.String()).
			Msg("killed worker agent")
	}

	return nil
}

// GetRunningAgents returns a list of running agent IDs and their types.
func (m *AgentManager) GetRunningAgents() map[uuid.UUID]domain.AgentType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[uuid.UUID]domain.AgentType, len(m.runningAgents))
	for id, agent := range m.runningAgents {
		result[id] = agent.agentType
	}
	return result
}

// IsRunning checks if an agent is running for the given ID.
func (m *AgentManager) IsRunning(id uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.runningAgents[id]
	return exists
}

// Shutdown gracefully shuts down the agent manager.
// It cancels all running agents and waits for them to complete.
func (m *AgentManager) Shutdown(ctx context.Context) error {
	log.Info().Msg("shutting down agent manager")

	// Cancel all running agents
	m.cancel()

	// Wait for all agents to complete with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("all agents stopped")
		return nil
	case <-ctx.Done():
		log.Warn().Msg("timeout waiting for agents to stop")
		return ctx.Err()
	}
}

// removeRunningAgent removes an agent from the tracking map.
func (m *AgentManager) removeRunningAgent(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runningAgents, id)
}

// getUserToken retrieves and decrypts the user's GitHub token.
func (m *AgentManager) getUserToken(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := m.repo.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	token, err := m.crypto.DecryptToken(user.GithubToken)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return token, nil
}
