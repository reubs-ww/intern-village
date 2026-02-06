// Package repository defines the data access layer interfaces and types.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/intern-village/orchestrator/generated/db"
)

// Repository aggregates all database query interfaces.
// It wraps the sqlc-generated Queries and provides a clean interface
// for the service layer.
type Repository struct {
	*db.Queries
	pool DBTX
}

// DBTX is the database transaction interface.
type DBTX interface {
	db.DBTX
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// New creates a new Repository with the given database connection.
func New(pool DBTX) *Repository {
	return &Repository{
		Queries: db.New(pool),
		pool:    pool,
	}
}

// WithTx returns a new Repository that uses the given transaction.
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		Queries: r.Queries.WithTx(tx),
		pool:    r.pool,
	}
}

// BeginTx starts a new database transaction.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{})
}

// Transaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (r *Repository) Transaction(ctx context.Context, fn func(*Repository) error) error {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	txRepo := r.WithTx(tx)
	if err := fn(txRepo); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// TimestamptzToPointer converts a pgtype.Timestamptz to a *time.Time.
func TimestamptzToPointer(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
}

// PointerToTimestamptz converts a *time.Time to a pgtype.Timestamptz.
func PointerToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// NullUUID represents a nullable UUID.
type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}
