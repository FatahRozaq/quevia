package staff

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
	ErrStaffNotFound                = errors.New("staff profile not found")
	ErrStaffHasActiveAppointments   = errors.New("cannot deactivate staff with active upcoming appointments; reassign bookings first")
	ErrInvalidStatus                = errors.New("invalid status, must be ACTIVE or INACTIVE")
	ErrBranchNotFound               = errors.New("branch not found")
	ErrUserNotFound                 = errors.New("user not found")
)

type ScheduleInput struct {
	DayOfWeek  int16   `json:"day_of_week"`
	StartTime  string  `json:"start_time"`  // HH:MM
	EndTime    string  `json:"end_time"`    // HH:MM
	BreakStart *string `json:"break_start,omitempty"`
	BreakEnd   *string `json:"break_end,omitempty"`
	ValidFrom  string  `json:"valid_from"`  // YYYY-MM-DD
	ValidUntil *string `json:"valid_until,omitempty"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// CreateStaff creates a staff profile bound to a branch.
func (s *Service) CreateStaff(ctx context.Context, tenantID int64, creatorID int64, branchGUID, userGUID uuid.UUID, displayName, spec string) (*repository.CreateStaffProfileRow, error) {
	var row *repository.CreateStaffProfileRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		user, err := q.GetUserByGUID(ctx, pgtype.UUID{Bytes: userGUID, Valid: true})
		if err != nil {
			return ErrUserNotFound
		}

		res, err := q.CreateStaffProfile(ctx, repository.CreateStaffProfileParams{
			Guid:           pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:       tenantID,
			BranchID:       branch.ID,
			UserID:         user.ID,
			DisplayName:    displayName,
			Specialization: pgtype.Text{String: spec, Valid: spec != ""},
			Status:         "ACTIVE",
			CreatedBy:      pgtype.Int8{Int64: creatorID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating staff profile: %w", err)
		}
		row = &res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}

// ListStaff retrieves all staff profiles for a branch.
func (s *Service) ListStaff(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]repository.ListStaffByBranchRow, error) {
	var rows []repository.ListStaffByBranchRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
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

// UpdateStaffStatus updates a staff profile's status, enforcing FR-SVC-06 / BR-19.
func (s *Service) UpdateStaffStatus(ctx context.Context, tenantID int64, updaterID int64, staffGUID uuid.UUID, newStatus string) (*repository.UpdateStaffProfileStatusRow, error) {
	if newStatus != "ACTIVE" && newStatus != "INACTIVE" {
		return nil, ErrInvalidStatus
	}

	var row *repository.UpdateStaffProfileStatusRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		staff, err := q.GetStaffProfileByGUID(ctx, pgtype.UUID{Bytes: staffGUID, Valid: true})
		if err != nil {
			return ErrStaffNotFound
		}

		// Enforce FR-SVC-06 & BR-19: If deactivating, check active upcoming appointments
		if newStatus == "INACTIVE" {
			count, err := q.CountActiveUpcomingAppointmentsByStaff(ctx, repository.CountActiveUpcomingAppointmentsByStaffParams{
				StaffID: staff.ID,
				EndAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			})
			if err != nil {
				return fmt.Errorf("checking active appointments: %w", err)
			}
			if count > 0 {
				return ErrStaffHasActiveAppointments
			}
		}

		res, err := q.UpdateStaffProfileStatus(ctx, repository.UpdateStaffProfileStatusParams{
			Guid:      pgtype.UUID{Bytes: staffGUID, Valid: true},
			Status:    newStatus,
			UpdatedBy: pgtype.Int8{Int64: updaterID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("updating staff status: %w", err)
		}
		row = &res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}

// AssignServices associates a staff member with supported services.
func (s *Service) AssignServices(ctx context.Context, tenantID int64, updaterID int64, staffGUID uuid.UUID, serviceGUIDs []uuid.UUID) error {
	return database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		staff, err := q.GetStaffProfileByGUID(ctx, pgtype.UUID{Bytes: staffGUID, Valid: true})
		if err != nil {
			return ErrStaffNotFound
		}

		// Clear existing associations
		if err := q.RemoveStaffServicesByStaffID(ctx, repository.RemoveStaffServicesByStaffIDParams{
			StaffID:   staff.ID,
			DeletedBy: pgtype.Int8{Int64: updaterID, Valid: true},
		}); err != nil {
			return fmt.Errorf("clearing staff services: %w", err)
		}

		for _, sg := range serviceGUIDs {
			svc, err := q.GetServiceByGUID(ctx, pgtype.UUID{Bytes: sg, Valid: true})
			if err != nil {
				continue
			}

			_, err = q.AssignStaffService(ctx, repository.AssignStaffServiceParams{
				Guid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
				TenantID:  tenantID,
				StaffID:   staff.ID,
				ServiceID: svc.ID,
				CreatedBy: pgtype.Int8{Int64: updaterID, Valid: true},
			})
			if err != nil {
				return fmt.Errorf("assigning service: %w", err)
			}
		}
		return nil
	})
}

// SetStaffSchedules replaces weekly schedules for a staff member.
func (s *Service) SetStaffSchedules(ctx context.Context, tenantID int64, updaterID int64, staffGUID uuid.UUID, schedules []ScheduleInput) error {
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
