package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
)

// NewPool initializes a thread-safe connection pool for PostgreSQL.
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing db config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating db pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging db: %w", err)
	}

	return pool, nil
}

// SetTenantContext sets the PostgreSQL session configuration for Row-Level Security.
func SetTenantContext(ctx context.Context, tx pgx.Tx, tenantID int64) error {
	_, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", strconv.FormatInt(tenantID, 10))
	if err != nil {
		return fmt.Errorf("setting tenant context: %w", err)
	}
	return nil
}

// WithTx executes a database transaction with optional tenant isolation context.
func WithTx(ctx context.Context, pool *pgxpool.Pool, tenantID *int64, fn func(tx pgx.Tx, q *repository.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if tenantID != nil {
		if err := SetTenantContext(ctx, tx, *tenantID); err != nil {
			return err
		}
	}

	q := repository.New(tx)
	if err := fn(tx, q); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}

	return nil
}
