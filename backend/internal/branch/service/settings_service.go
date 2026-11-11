package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

func (s *BranchService) GetSettings(ctx context.Context, tenantID int64, branchGUID uuid.UUID) (*repository.GetBranchSettingsRow, error) {
	var row *repository.GetBranchSettingsRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		res, err := q.GetBranchSettings(ctx, branch.ID)
		if err != nil {
			res = repository.GetBranchSettingsRow{
				BranchID:                 branch.ID,
				ChangeCutoffMinutes:       180,
				CheckInOpenBeforeMinutes: 30,
				CheckInCloseAfterMinutes: 15,
				NoShowAfterMinutes:       10,
				SlotStepMinutes:           30,
				AppointmentPriority:       2,
				WalkInPriority:            1,
				NotifyBeforeQueueCount:    2,
			}
		}
		row = &res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (s *BranchService) UpsertSettings(ctx context.Context, tenantID int64, userID int64, branchGUID uuid.UUID, input payload.BranchSettingsPayload) (*repository.UpsertBranchSettingsRow, error) {
	var row *repository.UpsertBranchSettingsRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		var lateGrace pgtype.Int4
		if input.LateGraceMinutes != nil {
			lateGrace = pgtype.Int4{Int32: *input.LateGraceMinutes, Valid: true}
		}

		res, err := q.UpsertBranchSettings(ctx, repository.UpsertBranchSettingsParams{
			Guid:                     pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:                 tenantID,
			BranchID:                 branch.ID,
			ChangeCutoffMinutes:       input.ChangeCutoffMinutes,
			CheckInOpenBeforeMinutes: input.CheckInOpenBeforeMinutes,
			CheckInCloseAfterMinutes: input.CheckInCloseAfterMinutes,
			NoShowAfterMinutes:       input.NoShowAfterMinutes,
			SlotStepMinutes:           input.SlotStepMinutes,
			AppointmentPriority:       input.AppointmentPriority,
			WalkInPriority:            input.WalkInPriority,
			LateGraceMinutes:          lateGrace,
			NotifyBeforeQueueCount:    input.NotifyBeforeQueueCount,
			CreatedBy:                pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("upserting branch settings: %w", err)
		}
		row = &res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}
