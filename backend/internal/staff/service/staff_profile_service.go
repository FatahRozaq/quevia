package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

func (s *StaffService) CreateStaff(ctx context.Context, tenantID int64, creatorID int64, input payload.CreateStaffPayload) (*sqlc.CreateStaffProfileRow, error) {
	var row *sqlc.CreateStaffProfileRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: input.BranchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		user, err := q.GetUserByGUID(ctx, pgtype.UUID{Bytes: input.UserGUID, Valid: true})
		if err != nil {
			return ErrUserNotFound
		}

		res, err := q.CreateStaffProfile(ctx, sqlc.CreateStaffProfileParams{
			Guid:           pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:       tenantID,
			BranchID:       branch.ID,
			UserID:         user.ID,
			DisplayName:    input.DisplayName,
			Specialization: pgtype.Text{String: input.Specialization, Valid: input.Specialization != ""},
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

func (s *StaffService) UpdateStaffStatus(ctx context.Context, tenantID int64, updaterID int64, staffGUID uuid.UUID, newStatus string) (*sqlc.UpdateStaffProfileStatusRow, error) {
	if newStatus != "ACTIVE" && newStatus != "INACTIVE" {
		return nil, ErrInvalidStatus
	}

	var row *sqlc.UpdateStaffProfileStatusRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		staff, err := q.GetStaffProfileByGUID(ctx, pgtype.UUID{Bytes: staffGUID, Valid: true})
		if err != nil {
			return ErrStaffNotFound
		}

		// Enforce FR-SVC-06 & BR-19: If deactivating, check active upcoming appointments
		if newStatus == "INACTIVE" {
			count, err := q.CountActiveUpcomingAppointmentsByStaff(ctx, sqlc.CountActiveUpcomingAppointmentsByStaffParams{
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

		res, err := q.UpdateStaffProfileStatus(ctx, sqlc.UpdateStaffProfileStatusParams{
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

func (s *StaffService) AssignServices(ctx context.Context, tenantID int64, updaterID int64, staffGUID uuid.UUID, serviceGUIDs []uuid.UUID) error {
	return database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		staff, err := q.GetStaffProfileByGUID(ctx, pgtype.UUID{Bytes: staffGUID, Valid: true})
		if err != nil {
			return ErrStaffNotFound
		}

		if err := q.RemoveStaffServicesByStaffID(ctx, sqlc.RemoveStaffServicesByStaffIDParams{
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

			_, err = q.AssignStaffService(ctx, sqlc.AssignStaffServiceParams{
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
