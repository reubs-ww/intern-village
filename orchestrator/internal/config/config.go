// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

// Package config handles configuration loading from environment variables.
package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration values for the orchestrator.
// Values are loaded from environment variables.
type Config struct {
	// Database
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// GitHub OAuth
	GitHubClientID     string `envconfig:"GITHUB_CLIENT_ID" required:"true"`
	GitHubClientSecret string `envconfig:"GITHUB_CLIENT_SECRET" required:"true"`

	// Security
	JWTSecret     string `envconfig:"JWT_SECRET" required:"true"`
	EncryptionKey string `envconfig:"ENCRYPTION_KEY" required:"true"`

	// Directories
	DataDir    string `envconfig:"DATA_DIR" default:"/data"`
	PromptsDir string `envconfig:"PROMPTS_DIR" default:"./prompts"`

	// Server
	Port     int    `envconfig:"PORT" default:"8080"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Agent settings
	AgentMaxRetries     int `envconfig:"AGENT_MAX_RETRIES" default:"10"`
	SyncIntervalSeconds int `envconfig:"SYNC_INTERVAL_SECONDS" default:"30"`

	// SSE settings
	SSEHeartbeatIntervalS     int `envconfig:"SSE_HEARTBEAT_INTERVAL_S" default:"30"`
	SSEConnectionTimeoutM     int `envconfig:"SSE_CONNECTION_TIMEOUT_M" default:"60"`
	SSEMaxConnectionsPerUser  int `envconfig:"SSE_MAX_CONNECTIONS_PER_USER" default:"5"`
	EventChannelBuffer        int `envconfig:"EVENT_CHANNEL_BUFFER" default:"100"`
	LogTailPollMS             int `envconfig:"LOG_TAIL_POLL_MS" default:"100"`
	LogTailMaxLineBytes       int `envconfig:"LOG_TAIL_MAX_LINE_BYTES" default:"1048576"`
}

// Load reads configuration from environment variables.
// It returns an error if required variables are missing.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// validate checks that the configuration values are valid.
func (c *Config) validate() error {
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes for AES-256")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}

	if c.AgentMaxRetries < 1 {
		return fmt.Errorf("AGENT_MAX_RETRIES must be at least 1")
	}

	if c.SyncIntervalSeconds < 1 {
		return fmt.Errorf("SYNC_INTERVAL_SECONDS must be at least 1")
	}

	return nil
}
