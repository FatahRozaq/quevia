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

func (s *StaffService) SetStaffSchedules(ctx context.Context, tenantID int64, updaterID int64, staffGUID uuid.UUID, schedules []payload.StaffSchedulePayload) error {
	return database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		staff, err := q.GetStaffProfileByGUID(ctx, pgtype.UUID{Bytes: staffGUID, Valid: true})
		if err != nil {
			return ErrStaffNotFound
		}

		if err := q.DeleteStaffSchedulesByStaffID(ctx, repository.DeleteStaffSchedulesByStaffIDParams{
			StaffID:   staff.ID,
			DeletedBy: pgtype.Int8{Int64: updaterID, Valid: true},
		}); err != nil {
			return fmt.Errorf("clearing staff schedules: %w", err)
		}

		for _, sc := range schedules {
			startT, err := time.Parse("15:04", sc.StartTime)
			if err != nil {
				return fmt.Errorf("parsing start_time: %w", err)
			}
			endT, err := time.Parse("15:04", sc.EndTime)
			if err != nil {
				return fmt.Errorf("parsing end_time: %w", err)
			}

			startMicro := int64(startT.Hour()*3600+startT.Minute()*60) * 1000000
			endMicro := int64(endT.Hour()*3600+endT.Minute()*60) * 1000000

			var breakStartMicro, breakEndMicro pgtype.Time
			if sc.BreakStart != nil && sc.BreakEnd != nil {
				bStart, _ := time.Parse("15:04", *sc.BreakStart)
				bEnd, _ := time.Parse("15:04", *sc.BreakEnd)
				breakStartMicro = pgtype.Time{Microseconds: int64(bStart.Hour()*3600+bStart.Minute()*60) * 1000000, Valid: true}
				breakEndMicro = pgtype.Time{Microseconds: int64(bEnd.Hour()*3600+bEnd.Minute()*60) * 1000000, Valid: true}
			}

			validFromDate, err := time.Parse("2006-01-02", sc.ValidFrom)
			if err != nil {
				validFromDate = time.Now()
			}

			_, err = q.CreateStaffSchedule(ctx, repository.CreateStaffScheduleParams{
				Guid:       pgtype.UUID{Bytes: uuid.New(), Valid: true},
				TenantID:   tenantID,
				StaffID:    staff.ID,
				DayOfWeek:  sc.DayOfWeek,
				StartTime:  pgtype.Time{Microseconds: startMicro, Valid: true},
				EndTime:    pgtype.Time{Microseconds: endMicro, Valid: true},
				BreakStart: breakStartMicro,
				BreakEnd:   breakEndMicro,
				ValidFrom:  pgtype.Date{Time: validFromDate, Valid: true},
				CreatedBy:  pgtype.Int8{Int64: updaterID, Valid: true},
			})
			if err != nil {
				return fmt.Errorf("creating staff schedule: %w", err)
			}
		}
		return nil
	})
}
