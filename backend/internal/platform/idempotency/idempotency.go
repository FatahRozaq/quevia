package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

var (
	ErrIdempotencyKeyReused = errors.New("idempotency key reused with different request payload")
	ErrRequestInProgress    = errors.New("request with this idempotency key is already in progress")
)

type CachedResponse struct {
	StatusCode int
	Body       []byte
	Replayed   bool
}

type Manager struct {
	db  *pgxpool.Pool
	ttl time.Duration
}

func NewManager(db *pgxpool.Pool, ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Manager{db: db, ttl: ttl}
}

// ComputeHash generates SHA-256 hash of payload.
func ComputeHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// Execute wraps an operation with idempotency protection.
func (m *Manager) Execute(
	ctx context.Context,
	tenantID int64,
	scope string,
	key string,
	payload []byte,
	operation func(ctx context.Context) (int, interface{}, error),
) (*CachedResponse, error) {
	if key == "" {
		// No key provided, run operation directly
		status, data, err := operation(ctx)
		if err != nil {
			return nil, err
		}
		bodyBytes, _ := json.Marshal(data)
		return &CachedResponse{StatusCode: status, Body: bodyBytes, Replayed: false}, nil
	}

	requestHash := ComputeHash(payload)

	var cached *CachedResponse
	err := database.WithTx(ctx, m.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		existing, err := q.GetIdempotencyKey(ctx, sqlc.GetIdempotencyKeyParams{
			TenantID: tenantID,
			Scope:    scope,
			Key:      key,
		})
		if err == nil {
			if existing.Status == "COMPLETED" {
				if existing.RequestHash != requestHash {
					return ErrIdempotencyKeyReused
				}
				cached = &CachedResponse{
					StatusCode: int(existing.ResponseStatus.Int16),
					Body:       existing.ResponseBody,
					Replayed:   true,
				}
				return nil
			}
			return ErrRequestInProgress
		}

		// Insert IN_PROGRESS
		expiresAt := time.Now().Add(m.ttl)
		_, err = q.CreateIdempotencyKey(ctx, sqlc.CreateIdempotencyKeyParams{
			Guid:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:    tenantID,
			Scope:       scope,
			Key:         key,
			RequestHash: requestHash,
			Status:      "IN_PROGRESS",
			ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
			CreatedBy:   pgtype.Int8{Valid: false},
		})
		if err != nil {
			return fmt.Errorf("creating idempotency key: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if cached != nil {
		return cached, nil
	}

	// Execute business operation
	statusCode, data, opErr := operation(ctx)
	if opErr != nil {
		return nil, opErr
	}

	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling response body: %w", err)
	}

	// Update to COMPLETED
	_ = database.WithTx(ctx, m.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		return q.CompleteIdempotencyKey(ctx, sqlc.CompleteIdempotencyKeyParams{
			TenantID:       tenantID,
			Scope:          scope,
			Key:            key,
			ResponseStatus: pgtype.Int2{Int16: int16(statusCode), Valid: true},
			ResponseBody:   bodyBytes,
			UpdatedBy:      pgtype.Int8{Valid: false},
		})
	})

	return &CachedResponse{
		StatusCode: statusCode,
		Body:       bodyBytes,
		Replayed:   false,
	}, nil
}
