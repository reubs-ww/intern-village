// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/agent"
	"log/slog"

	"github.com/intern-village/orchestrator/internal/api/handlers"
	"github.com/intern-village/orchestrator/internal/api/middleware"
	"github.com/intern-village/orchestrator/internal/api/response"
	"github.com/intern-village/orchestrator/internal/config"
	"github.com/intern-village/orchestrator/internal/repository"
	"github.com/intern-village/orchestrator/internal/repository/postgres"
	"github.com/intern-village/orchestrator/internal/service"
)

// Server represents the HTTP server.
type Server struct {
	router       *chi.Mux
	httpServer   *http.Server
	cfg          *config.Config
	db           *postgres.DB
	repo         *repository.Repository
	crypto       *repository.Crypto
	agentManager *agent.AgentManager
	syncWorker   *service.SyncWorker
	eventHub     service.EventHub
}

// NewServer creates a new HTTP server with all routes configured.
func NewServer(cfg *config.Config, db *postgres.DB, repo *repository.Repository, crypto *repository.Crypto) (*Server, error) {
	s := &Server{
		router: chi.NewRouter(),
		cfg:    cfg,
		db:     db,
		repo:   repo,
		crypto: crypto,
	}

	s.setupMiddleware()
	if err := s.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: s.router,
		// Extended timeouts to support long-running operations like repo cloning
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 11 * time.Minute, // Slightly longer than longest request timeout (10 min)
		IdleTimeout:  120 * time.Second,
	}

	// Start sync worker if configured
	if s.syncWorker != nil {
		s.syncWorker.Start()
	}

	return s, nil
}

// setupMiddleware configures the middleware chain.
func (s *Server) setupMiddleware() {
	// Request ID
	s.router.Use(chimw.RequestID)

	// Real IP (for behind reverse proxy)
	s.router.Use(chimw.RealIP)

	// Custom request logging with zerolog
	s.router.Use(middleware.Logger)

	// Panic recovery
	s.router.Use(chimw.Recoverer)

	// Request timeout (applied selectively, not globally - see setupRoutes)

	// CORS
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "https://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() error {
	// Set up frontend handler (SPA)
	frontendHandler, err := NewFrontendHandler()
	if err != nil {
		// If frontend is not embedded (development mode), skip it
		log.Warn().Err(err).Msg("frontend not available, serving API only")
		frontendHandler = nil
	}

	// Create event hub for real-time events
	logger := slog.Default()
	s.eventHub = service.NewEventHub(s.cfg.EventChannelBuffer, logger)

	// Create log tailer for streaming agent logs
	logTailerConfig := service.LogTailerConfig{
		PollInterval: time.Duration(s.cfg.LogTailPollMS) * time.Millisecond,
		MaxLineBytes: s.cfg.LogTailMaxLineBytes,
	}
	logTailer := service.NewLogTailer(s.eventHub, logTailerConfig, logger)

	// Create services
	authService, err := service.NewAuthService(
		s.cfg.GitHubClientID,
		s.cfg.GitHubClientSecret,
		s.cfg.JWTSecret,
		s.repo,
		s.crypto,
	)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %w", err)
	}

	githubService := service.NewGitHubService()
	beadsService := service.NewBeadsService()
	projectService := service.NewProjectService(s.repo, s.crypto, githubService, beadsService, s.cfg.DataDir)
	dependencyService := service.NewDependencyService(s.repo, s.eventHub)
	taskService := service.NewTaskService(s.repo, projectService, githubService, beadsService, s.eventHub)
	subtaskService := service.NewSubtaskService(s.repo, taskService, dependencyService, beadsService, projectService, githubService, s.eventHub)
	syncService := service.NewSyncService(s.repo, beadsService, subtaskService, dependencyService, taskService)

	// Initialize agent components (Phase 7)
	promptRenderer, err := agent.NewPromptRenderer(s.cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to create prompt renderer: %w", err)
	}

	executor := agent.NewExecutor(s.cfg.DataDir)

	// Create agent loop with service adapters
	agentLoop := agent.NewAgentLoop(
		executor,
		promptRenderer,
		agent.LoopServices{
			Repo:           s.repo,
			BeadsService:   newBeadsServiceAdapter(beadsService),
			GitHubService:  newGitHubServiceAdapter(githubService),
			SyncService:    newSyncServiceAdapter(syncService),
			TaskService:    newTaskServiceAdapter(taskService),
			SubtaskService: newSubtaskServiceAdapter(subtaskService),
			EventPublisher: newEventPublisherAdapter(s.eventHub),
			LogTailer:      newLogTailerAdapter(logTailer),
		},
		s.cfg.AgentMaxRetries,
	)

	// Create and store agent manager
	s.agentManager = agent.NewAgentManager(agentLoop, s.repo, projectService, s.crypto, s.eventHub)

	// Wire agent spawners into services
	taskService.SetAgentSpawner(s.agentManager)
	subtaskService.SetWorkerSpawner(s.agentManager)

	// Create sync worker
	s.syncWorker = service.NewSyncWorker(
		syncService,
		projectService,
		s.cfg.SyncIntervalSeconds,
	)

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService, s.cfg)
	projectHandler := handlers.NewProjectHandler(projectService, authService)
	taskHandler := handlers.NewTaskHandler(taskService)
	subtaskHandler := handlers.NewSubtaskHandler(subtaskService)
	agentHandler := handlers.NewAgentHandler(s.repo, subtaskService)
	eventHandler := handlers.NewEventHandler(s.eventHub, s.repo, projectService, s.cfg)

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Health check (no auth required)
	s.router.Get("/health", s.handleHealth)

	// API routes
	s.router.Route("/api", func(r chi.Router) {
		// Auth endpoints (no auth required)
		r.Route("/auth", func(r chi.Router) {
			r.Get("/github", authHandler.InitiateOAuth)
			r.Get("/github/callback", authHandler.HandleCallback)
			r.Post("/logout", authHandler.Logout)

			// Protected auth endpoints
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireAuth)
				r.Get("/me", authHandler.GetCurrentUser)
			})
		})

		// Project creation with extended timeout (10 min for cloning large repos)
		// Defined outside the 60s timeout group to avoid timeout being overridden
		r.With(authMiddleware.RequireAuth, chimw.Timeout(10*time.Minute)).Post("/projects", projectHandler.Create)

		// SSE Events - no timeout middleware (SSE connections are long-lived, managed internally)
		// Defined outside the 60s timeout group to avoid premature connection termination
		r.With(authMiddleware.RequireAuth).Get("/projects/{project_id}/events", eventHandler.StreamEvents)

		// Protected API routes (require auth) with default 60s timeout
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			// Default timeout for most API routes
			r.Use(chimw.Timeout(60 * time.Second))

			// Projects (Phase 4) - Note: POST /projects is defined above with extended timeout
			r.Get("/projects", projectHandler.List)
			r.Get("/projects/{id}", projectHandler.Get)
			r.Delete("/projects/{id}", projectHandler.Delete)
			r.Post("/projects/{id}/cleanup", projectHandler.Cleanup)

			// Tasks under projects (Phase 5)
			r.Get("/projects/{project_id}/tasks", taskHandler.List)
			// Extended timeout for task creation (syncs repo before planning)
			r.With(chimw.Timeout(5 * time.Minute)).Post("/projects/{project_id}/tasks", taskHandler.Create)

			// Active runs endpoint (not SSE, can have normal timeout)
			r.Get("/projects/{project_id}/active-runs", eventHandler.GetActiveRuns)

			// Tasks by ID (Phase 5)
			r.Route("/tasks", func(r chi.Router) {
				r.Get("/{id}", taskHandler.Get)
				r.Delete("/{id}", taskHandler.Delete)
				r.Post("/{id}/retry-planning", taskHandler.RetryPlanning)

				// Subtasks under tasks
				r.Get("/{task_id}/subtasks", subtaskHandler.List)
			})

			// Subtasks by ID (Phase 5)
			r.Route("/subtasks", func(r chi.Router) {
				r.Get("/{id}", subtaskHandler.Get)
				r.Post("/{id}/start", subtaskHandler.Start)
				r.Post("/{id}/mark-merged", subtaskHandler.MarkMerged)
				r.Post("/{id}/retry", subtaskHandler.Retry)
				r.Patch("/{id}/position", subtaskHandler.UpdatePosition)

				// Agent runs for subtask (Phase 8)
				r.Get("/{id}/runs", agentHandler.ListRuns)
			})

			// Agent runs by ID (Phase 8)
			r.Route("/runs", func(r chi.Router) {
				r.Get("/{id}/logs", agentHandler.GetLogs)
			})
		})
	})

	// Serve frontend SPA for all non-API routes (Phase 9)
	if frontendHandler != nil {
		s.router.Handle("/*", frontendHandler)
	}

	return nil
}

// handleHealth returns the server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		log.Error().Err(err).Msg("health check failed: database unreachable")
		response.Error(w, http.StatusServiceUnavailable, "UNHEALTHY", "database unreachable")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Info().
		Int("port", s.cfg.Port).
		Msg("starting HTTP server")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server, agent manager, and sync worker.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("shutting down server components")

	// Stop sync worker first
	if s.syncWorker != nil {
		s.syncWorker.Stop()
	}

	// Stop agent manager (waits for running agents)
	if s.agentManager != nil {
		if err := s.agentManager.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("error shutting down agent manager")
		}
	}

	// Finally, shut down HTTP server
	log.Info().Msg("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// Router returns the underlying chi router for testing.
func (s *Server) Router() *chi.Mux {
	return s.router
}
