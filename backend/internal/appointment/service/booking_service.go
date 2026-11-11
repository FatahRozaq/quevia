package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

func (s *AppointmentService) CreateAppointment(ctx context.Context, tenantID int64, userID *int64, input payload.CreateAppointmentPayload) (*payload.AppointmentResponse, error) {
	input.Customer.Email = strings.ToLower(strings.TrimSpace(input.Customer.Email))
	if _, err := mail.ParseAddress(input.Customer.Email); err != nil {
		return nil, ErrInvalidEmail
	}

	var result *payload.AppointmentResponse

	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: input.BranchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		svc, err := q.GetServiceByGUID(ctx, pgtype.UUID{Bytes: input.ServiceGUID, Valid: true})
		if err != nil {
			return ErrServiceNotFound
		}

		endAt := input.StartAt.Add(time.Duration(svc.DurationMinutes) * time.Minute)

		var customerID int64
		custRow, err := q.GetCustomerByEmail(ctx, sqlc.GetCustomerByEmailParams{
			TenantID: tenantID,
			Lower:    input.Customer.Email,
		})
		if err == nil {
			customerID = custRow.ID
		} else {
			newCust, err := q.CreateCustomer(ctx, sqlc.CreateCustomerParams{
				Guid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
				TenantID:  tenantID,
				UserID:    pgtype.Int8{Valid: false},
				FullName:  input.Customer.FullName,
				Email:     pgtype.Text{String: input.Customer.Email, Valid: true},
				Phone:     pgtype.Text{String: input.Customer.Phone, Valid: input.Customer.Phone != ""},
				CreatedBy: pgtype.Int8{Valid: false},
			})
			if err != nil {
				return fmt.Errorf("creating customer: %w", err)
			}
			customerID = newCust.ID
		}

		var staffID int64
		var staffGUID uuid.UUID
		var staffDisplayName string

		if input.StaffGUID != nil {
			sp, err := q.GetStaffProfileByGUID(ctx, pgtype.UUID{Bytes: *input.StaffGUID, Valid: true})
			if err != nil || sp.Status != "ACTIVE" {
				return ErrStaffNotEligible
			}
			staffID = sp.ID
			staffGUID = sp.Guid.Bytes
			staffDisplayName = sp.DisplayName
		} else {
			eligibleStaff, err := q.ListStaffByBranchAndService(ctx, sqlc.ListStaffByBranchAndServiceParams{
				BranchID:  branch.ID,
				ServiceID: svc.ID,
			})
			if err != nil || len(eligibleStaff) == 0 {
				return ErrNoStaffAvailable
			}

			found := false
			for _, st := range eligibleStaff {
				overlaps, err := q.CheckStaffOverlap(ctx, sqlc.CheckStaffOverlapParams{
					StaffID: st.ID,
					StartAt: pgtype.Timestamptz{Time: input.StartAt, Valid: true},
					EndAt:   pgtype.Timestamptz{Time: endAt, Valid: true},
				})
				if err == nil && len(overlaps) == 0 {
					staffID = st.ID
					staffGUID = st.Guid.Bytes
					staffDisplayName = st.DisplayName
					found = true
					break
				}
			}
			if !found {
				return ErrSlotUnavailable
			}
		}

		bookingCode := generateBookingCode()
		qrToken := generateQRToken()
		aptGUID := uuid.New()

		createdByID := pgtype.Int8{Valid: false}
		if userID != nil {
			createdByID = pgtype.Int8{Int64: *userID, Valid: true}
		}

		apt, err := q.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
			Guid:        pgtype.UUID{Bytes: aptGUID, Valid: true},
			TenantID:    tenantID,
			BranchID:    branch.ID,
			CustomerID:  customerID,
			ServiceID:   svc.ID,
			StaffID:     staffID,
			BookingCode: bookingCode,
			QrToken:     qrToken,
			StartAt:     pgtype.Timestamptz{Time: input.StartAt, Valid: true},
			EndAt:       pgtype.Timestamptz{Time: endAt, Valid: true},
			Status:      "CONFIRMED",
			Notes:       pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
			CreatedBy:   createdByID,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
				return ErrSlotUnavailable
			}
			return fmt.Errorf("creating appointment: %w", err)
		}

		// Audit Log
		auditData, _ := json.Marshal(map[string]interface{}{
			"booking_code": bookingCode,
			"customer":     input.Customer.Email,
			"start_at":     input.StartAt,
			"end_at":       endAt,
		})
		_, _ = q.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
			Guid:       pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:   pgtype.Int8{Int64: tenantID, Valid: true},
			BranchID:   pgtype.Int8{Int64: branch.ID, Valid: true},
			Action:     "appointment.created",
			EntityType: "appointment",
			EntityGuid: pgtype.UUID{Bytes: aptGUID, Valid: true},
			AfterData:  auditData,
			CreatedBy:  createdByID,
		})

		// Outbox Event
		outboxPayload, _ := json.Marshal(map[string]interface{}{
			"appointment_guid": aptGUID,
			"booking_code":     bookingCode,
			"customer_email":   input.Customer.Email,
			"start_at":         input.StartAt,
		})
		_, _ = q.CreateOutboxEvent(ctx, sqlc.CreateOutboxEventParams{
			Guid:          pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:      tenantID,
			AggregateType: "appointment",
			AggregateGuid: pgtype.UUID{Bytes: aptGUID, Valid: true},
			EventType:     "appointment.created",
			Payload:       outboxPayload,
			CreatedBy:     createdByID,
		})

		result = &payload.AppointmentResponse{
			GUID:             aptGUID,
			BookingCode:      apt.BookingCode,
			QRToken:          apt.QrToken,
			Status:           apt.Status,
			StartAt:          apt.StartAt.Time,
			EndAt:            apt.EndAt.Time,
			ServiceName:      svc.Name,
			StaffDisplayName: staffDisplayName,
			StaffGUID:        staffGUID,
			CustomerName:     input.Customer.FullName,
			CustomerEmail:    input.Customer.Email,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *AppointmentService) CancelAppointment(ctx context.Context, tenantID int64, userID *int64, appointmentGUID uuid.UUID, reason string) (*payload.AppointmentResponse, error) {
	var result *payload.AppointmentResponse

	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		apt, err := q.GetAppointmentByGUID(ctx, pgtype.UUID{Bytes: appointmentGUID, Valid: true})
		if err != nil {
			return ErrAppointmentNotFound
		}

		bSettings, err := q.GetBranchSettings(ctx, apt.BranchID)
		cutoffMinutes := int32(180)
		if err == nil && bSettings.ChangeCutoffMinutes > 0 {
			cutoffMinutes = bSettings.ChangeCutoffMinutes
		}

		cutoffTime := apt.StartAt.Time.Add(-time.Duration(cutoffMinutes) * time.Minute)
		if time.Now().After(cutoffTime) {
			return ErrChangeCutoffPassed
		}

		updatedByID := pgtype.Int8{Valid: false}
		if userID != nil {
			updatedByID = pgtype.Int8{Int64: *userID, Valid: true}
		}

		cancelled, err := q.CancelAppointment(ctx, sqlc.CancelAppointmentParams{
			Guid:         pgtype.UUID{Bytes: appointmentGUID, Valid: true},
			CancelReason: pgtype.Text{String: reason, Valid: reason != ""},
			UpdatedBy:    updatedByID,
		})
		if err != nil {
			return fmt.Errorf("cancelling appointment: %w", err)
		}

		result = &payload.AppointmentResponse{
			GUID:        appointmentGUID,
			BookingCode: cancelled.BookingCode,
			Status:      cancelled.Status,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *AppointmentService) LookupBooking(ctx context.Context, tenantID int64, bookingCode, email string) (*payload.AppointmentResponse, error) {
	var result *payload.AppointmentResponse

	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		apt, err := q.GetAppointmentByBookingCodeAndEmail(ctx, sqlc.GetAppointmentByBookingCodeAndEmailParams{
			BookingCode: strings.TrimSpace(bookingCode),
			Lower:       strings.ToLower(strings.TrimSpace(email)),
		})
		if err != nil {
			return ErrAppointmentNotFound
		}

		result = &payload.AppointmentResponse{
			GUID:             uuid.UUID(apt.Guid.Bytes),
			BookingCode:      apt.BookingCode,
			QRToken:          apt.QrToken,
			Status:           apt.Status,
			StartAt:          apt.StartAt.Time,
			EndAt:            apt.EndAt.Time,
			ServiceName:      apt.ServiceName,
			StaffDisplayName: apt.StaffName,
			StaffGUID:        uuid.UUID(apt.StaffGuid.Bytes),
			CustomerName:     apt.CustomerName,
			CustomerEmail:    apt.CustomerEmail.String,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
