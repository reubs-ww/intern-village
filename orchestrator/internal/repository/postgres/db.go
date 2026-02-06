// Package postgres provides PostgreSQL database connectivity and migrations.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/intern-village/orchestrator/internal/repository"
)

// PoolConfig contains configuration options for the connection pool.
type PoolConfig struct {
	// MaxConns is the maximum number of connections in the pool.
	// Default: 10
	MaxConns int32

	// MinConns is the minimum number of connections in the pool.
	// Default: 2
	MinConns int32

	// MaxConnLifetime is the maximum duration a connection may be reused.
	// Default: 1 hour
	MaxConnLifetime time.Duration

	// MaxConnIdleTime is the maximum duration a connection may be idle.
	// Default: 30 minutes
	MaxConnIdleTime time.Duration

	// HealthCheckPeriod is the duration between health checks.
	// Default: 1 minute
	HealthCheckPeriod time.Duration
}

// DefaultPoolConfig returns the default pool configuration.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          10,
		MinConns:          2,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
	}
}

// DB wraps a pgxpool.Pool and provides database operations.
type DB struct {
	pool *pgxpool.Pool
}

// Connect establishes a connection to the PostgreSQL database.
func Connect(ctx context.Context, databaseURL string, cfg PoolConfig) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Apply pool configuration
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().
		Int32("max_conns", cfg.MaxConns).
		Int32("min_conns", cfg.MinConns).
		Msg("Connected to PostgreSQL")

	return &DB{pool: pool}, nil
}

// Close closes the database connection pool.
func (d *DB) Close() {
	if d.pool != nil {
		d.pool.Close()
		log.Info().Msg("Closed PostgreSQL connection pool")
	}
}

// Pool returns the underlying connection pool.
func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

// Repository creates a new Repository using this database connection.
func (d *DB) Repository() *repository.Repository {
	return repository.New(d.pool)
}

// Ping verifies the database connection is still alive.
func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// RunMigrations applies all pending database migrations.
// Note: For MVP, migrations should be applied manually using:
//
//	psql -f migrations/001_initial.sql
//
// In production, consider using golang-migrate or goose.
func RunMigrations(_ context.Context, _ *DB) error {
	// For now, we'll just log that migrations should be run manually
	// This is a placeholder for a proper migration runner
	log.Info().Msg("Migrations should be applied using: psql -f migrations/001_initial.sql")
	return nil
}
