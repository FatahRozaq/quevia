package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

func (s *BranchService) SetBusinessHours(ctx context.Context, tenantID int64, userID int64, branchGUID uuid.UUID, hours []payload.BusinessHourPayload) error {
	return database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		if err := q.DeleteBusinessHoursByBranch(ctx, repository.DeleteBusinessHoursByBranchParams{
			BranchID:  branch.ID,
			DeletedBy: pgtype.Int8{Int64: userID, Valid: true},
		}); err != nil {
			return fmt.Errorf("clearing old business hours: %w", err)
		}

		for _, h := range hours {
			openT, err := time.Parse("15:04:05", h.OpenTime)
			if err != nil {
				openT, err = time.Parse("15:04", h.OpenTime)
				if err != nil {
					return fmt.Errorf("invalid open_time: %w", err)
				}
			}
			closeT, err := time.Parse("15:04:05", h.CloseTime)
			if err != nil {
				closeT, err = time.Parse("15:04", h.CloseTime)
				if err != nil {
					return fmt.Errorf("invalid close_time: %w", err)
				}
			}

			openMicro := int64(openT.Hour()*3600+openT.Minute()*60+openT.Second()) * 1000000
			closeMicro := int64(closeT.Hour()*3600+closeT.Minute()*60+closeT.Second()) * 1000000

			_, err = q.CreateBusinessHour(ctx, repository.CreateBusinessHourParams{
				Guid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
				TenantID:  tenantID,
				BranchID:  branch.ID,
				DayOfWeek: h.DayOfWeek,
				OpenTime:  pgtype.Time{Microseconds: openMicro, Valid: true},
				CloseTime: pgtype.Time{Microseconds: closeMicro, Valid: true},
				CreatedBy: pgtype.Int8{Int64: userID, Valid: true},
			})
			if err != nil {
				return fmt.Errorf("creating business hour: %w", err)
			}
		}
		return nil
	})
}

func (s *BranchService) ListBusinessHours(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]repository.ListBusinessHoursByBranchRow, error) {
	var rows []repository.ListBusinessHoursByBranchRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		res, err := q.ListBusinessHoursByBranch(ctx, branch.ID)
		if err != nil {
			return fmt.Errorf("listing business hours: %w", err)
		}
		rows = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}
