package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

func (s *StaffService) ListStaff(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]sqlc.ListStaffByBranchRow, error) {
	var rows []sqlc.ListStaffByBranchRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		res, err := q.ListStaffByBranch(ctx, branch.ID)
		if err != nil {
			return err
		}
		rows = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}
