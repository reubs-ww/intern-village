// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/api"
	"github.com/intern-village/orchestrator/internal/config"
	"github.com/intern-village/orchestrator/internal/repository"
	"github.com/intern-village/orchestrator/internal/repository/postgres"
)

func main() {
	// Initialize logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Configure log level
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	log.Info().
		Str("log_level", cfg.LogLevel).
		Int("port", cfg.Port).
		Msg("starting orchestrator")

	// Initialize crypto for token encryption
	crypto, err := repository.NewCrypto([]byte(cfg.EncryptionKey))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize crypto")
	}

	// Connect to database
	ctx := context.Background()
	db, err := postgres.Connect(ctx, cfg.DatabaseURL, postgres.DefaultPoolConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Create repository
	repo := db.Repository()

	// Create and start HTTP server
	server, err := api.NewServer(cfg, db, repo, crypto)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Agent manager and sync worker are now handled by server.Shutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server stopped")
}
