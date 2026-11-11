package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

func (s *BranchService) ListCounters(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]sqlc.ListCountersByBranchRow, error) {
	var rows []sqlc.ListCountersByBranchRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		res, err := q.ListCountersByBranch(ctx, branch.ID)
		if err != nil {
			return fmt.Errorf("listing counters: %w", err)
		}
		rows = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *BranchService) CreateCounter(ctx context.Context, tenantID int64, userID int64, input payload.CreateCounterPayload) (*sqlc.CreateCounterRow, error) {
	var row *sqlc.CreateCounterRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: input.BranchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		res, err := q.CreateCounter(ctx, sqlc.CreateCounterParams{
			Guid:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:     tenantID,
			BranchID:     branch.ID,
			Code:         input.Code,
			Name:         input.Name,
			DisplayOrder: input.DisplayOrder,
			Status:       "ACTIVE",
			CreatedBy:    pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating counter: %w", err)
		}
		row = &res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}
