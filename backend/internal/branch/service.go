package branch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
)

var (
	ErrBranchNotFound  = errors.New("branch not found")
	ErrCounterNotFound = errors.New("service counter not found")
)

type SettingsInput struct {
	ChangeCutoffMinutes       int32  `json:"change_cutoff_minutes"`
	CheckInOpenBeforeMinutes int32  `json:"check_in_open_before_minutes"`
	CheckInCloseAfterMinutes int32  `json:"check_in_close_after_minutes"`
	NoShowAfterMinutes       int32  `json:"no_show_after_minutes"`
	SlotStepMinutes           int16  `json:"slot_step_minutes"`
	AppointmentPriority       int16  `json:"appointment_priority"`
	WalkInPriority            int16  `json:"walk_in_priority"`
	LateGraceMinutes          *int32 `json:"late_grace_minutes,omitempty"`
	NotifyBeforeQueueCount    int16  `json:"notify_before_queue_count"`
}

type BusinessHourInput struct {
	DayOfWeek int16  `json:"day_of_week"`
	OpenTime  string `json:"open_time"`  // HH:MM:SS
	CloseTime string `json:"close_time"` // HH:MM:SS
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) GetSettings(ctx context.Context, tenantID int64, branchGUID uuid.UUID) (*repository.GetBranchSettingsRow, error) {
	var row *repository.GetBranchSettingsRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		res, err := q.GetBranchSettings(ctx, branch.ID)
		if err != nil {
			// If not yet created, return default values
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

func (s *Service) UpsertSettings(ctx context.Context, tenantID int64, userID int64, branchGUID uuid.UUID, input SettingsInput) (*repository.UpsertBranchSettingsRow, error) {
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

func (s *Service) SetBusinessHours(ctx context.Context, tenantID int64, userID int64, branchGUID uuid.UUID, hours []BusinessHourInput) error {
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

func (s *Service) ListBusinessHours(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]repository.ListBusinessHoursByBranchRow, error) {
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

func (s *Service) ListCounters(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]repository.ListCountersByBranchRow, error) {
	var rows []repository.ListCountersByBranchRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
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

func (s *Service) CreateCounter(ctx context.Context, tenantID int64, userID int64, branchGUID uuid.UUID, code, name string, order int16) (*repository.CreateCounterRow, error) {
	var row *repository.CreateCounterRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		res, err := q.CreateCounter(ctx, repository.CreateCounterParams{
			Guid:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:     tenantID,
			BranchID:     branch.ID,
			Code:         code,
			Name:         name,
			DisplayOrder: order,
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
