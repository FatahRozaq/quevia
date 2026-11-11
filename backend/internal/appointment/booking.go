package appointment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
)

var (
	ErrSlotUnavailable        = errors.New("selected slot is no longer available")
	ErrAppointmentNotFound    = errors.New("appointment not found")
	ErrChangeCutoffPassed     = errors.New("change cutoff time has passed")
	ErrInvalidEmail           = errors.New("valid customer email is required")
	ErrNoStaffAvailable       = errors.New("no staff available for selected time")
)

type CustomerInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone,omitempty"`
}

type CreateAppointmentInput struct {
	TenantID    int64         `json:"tenant_id"`
	BranchGUID  uuid.UUID     `json:"branch_guid"`
	ServiceGUID uuid.UUID     `json:"service_guid"`
	StaffGUID   *uuid.UUID    `json:"staff_guid,omitempty"`
	StartAt     time.Time     `json:"start_at"`
	Customer    CustomerInput `json:"customer"`
	Notes       string        `json:"notes,omitempty"`
	UserID      *int64        `json:"user_id,omitempty"` // null for guest
}

type AppointmentResult struct {
	GUID             uuid.UUID `json:"guid"`
	BookingCode      string    `json:"booking_code"`
	QRToken          string    `json:"qr_token"`
	Status           string    `json:"status"`
	StartAt          time.Time `json:"start_at"`
	EndAt            time.Time `json:"end_at"`
	ServiceName      string    `json:"service_name,omitempty"`
	StaffDisplayName string    `json:"staff_display_name,omitempty"`
	StaffGUID        uuid.UUID `json:"staff_guid"`
	CustomerName     string    `json:"customer_name,omitempty"`
	CustomerEmail    string    `json:"customer_email,omitempty"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func generateBookingCode() string {
	const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	code := make([]byte, 6)
	for i := range code {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[num.Int64()]
	}
	return "QV-" + string(code)
}

func generateQRToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CreateAppointment books an appointment with PostgreSQL GiST anti-double booking protection.
func (s *Service) CreateAppointment(ctx context.Context, input CreateAppointmentInput) (*AppointmentResult, error) {
	// 1. Validate customer email
	input.Customer.Email = strings.ToLower(strings.TrimSpace(input.Customer.Email))
	if _, err := mail.ParseAddress(input.Customer.Email); err != nil {
		return nil, ErrInvalidEmail
	}

	var result *AppointmentResult

	err := database.WithTx(ctx, s.db, &input.TenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: input.BranchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		svc, err := q.GetServiceByGUID(ctx, pgtype.UUID{Bytes: input.ServiceGUID, Valid: true})
		if err != nil {
			return ErrServiceNotFound
		}

		endAt := input.StartAt.Add(time.Duration(svc.DurationMinutes) * time.Minute)

		// 2. Resolve Customer (find by email or create)
		var customerID int64
		custRow, err := q.GetCustomerByEmail(ctx, repository.GetCustomerByEmailParams{
			TenantID: input.TenantID,
			Lower:    input.Customer.Email,
		})
		if err == nil {
			customerID = custRow.ID
		} else {
			newCust, err := q.CreateCustomer(ctx, repository.CreateCustomerParams{
				Guid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
				TenantID:  input.TenantID,
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

		// 3. Resolve Staff Member
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
			// Auto-assign: find eligible staff without overlapping appointments
			eligibleStaff, err := q.ListStaffByBranchAndService(ctx, repository.ListStaffByBranchAndServiceParams{
				BranchID:  branch.ID,
				ServiceID: svc.ID,
			})
			if err != nil || len(eligibleStaff) == 0 {
				return ErrNoStaffAvailable
			}

			found := false
			for _, st := range eligibleStaff {
				overlaps, err := q.CheckStaffOverlap(ctx, repository.CheckStaffOverlapParams{
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

		// 4. Generate Unique Booking Code and QR Token
		bookingCode := generateBookingCode()
		qrToken := generateQRToken()
		aptGUID := uuid.New()

		// 5. Insert Appointment (Database GiST exclusion constraint protects concurrency)
		createdByID := pgtype.Int8{Valid: false}
		if input.UserID != nil {
			createdByID = pgtype.Int8{Int64: *input.UserID, Valid: true}
		}

		apt, err := q.CreateAppointment(ctx, repository.CreateAppointmentParams{
			Guid:        pgtype.UUID{Bytes: aptGUID, Valid: true},
			TenantID:    input.TenantID,
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
				// PostgreSQL GiST Exclusion Constraint ex_appointments_no_overlap violation
				return ErrSlotUnavailable
			}
			return fmt.Errorf("creating appointment: %w", err)
		}

		// 6. Record Audit Log
		auditData, _ := json.Marshal(map[string]interface{}{
			"booking_code": bookingCode,
			"customer":     input.Customer.Email,
			"start_at":     input.StartAt,
			"end_at":       endAt,
		})
		_, _ = q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
			Guid:       pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:   pgtype.Int8{Int64: input.TenantID, Valid: true},
			BranchID:   pgtype.Int8{Int64: branch.ID, Valid: true},
			Action:     "appointment.created",
			EntityType: "appointment",
			EntityGuid: pgtype.UUID{Bytes: aptGUID, Valid: true},
			AfterData:  auditData,
			CreatedBy:  createdByID,
		})

		// 7. Write Outbox Event
		outboxPayload, _ := json.Marshal(map[string]interface{}{
			"appointment_guid": aptGUID,
			"booking_code":     bookingCode,
			"customer_email":   input.Customer.Email,
			"start_at":         input.StartAt,
		})
		_, _ = q.CreateOutboxEvent(ctx, repository.CreateOutboxEventParams{
			Guid:          pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:      input.TenantID,
			AggregateType: "appointment",
			AggregateGuid: pgtype.UUID{Bytes: aptGUID, Valid: true},
			EventType:     "appointment.created",
			Payload:       outboxPayload,
			CreatedBy:     createdByID,
		})

		result = &AppointmentResult{
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

// CancelAppointment cancels an appointment and frees the slot.
func (s *Service) CancelAppointment(ctx context.Context, tenantID int64, userID *int64, appointmentGUID uuid.UUID, reason string) (*AppointmentResult, error) {
	var result *AppointmentResult

	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		apt, err := q.GetAppointmentByGUID(ctx, pgtype.UUID{Bytes: appointmentGUID, Valid: true})
		if err != nil {
			return ErrAppointmentNotFound
		}

		// Check change cutoff minutes
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

		cancelled, err := q.CancelAppointment(ctx, repository.CancelAppointmentParams{
			Guid:         pgtype.UUID{Bytes: appointmentGUID, Valid: true},
			CancelReason: pgtype.Text{String: reason, Valid: reason != ""},
			UpdatedBy:    updatedByID,
		})
		if err != nil {
			return fmt.Errorf("cancelling appointment: %w", err)
		}

		result = &AppointmentResult{
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

// LookupBooking finds booking details using booking code and customer email.
func (s *Service) LookupBooking(ctx context.Context, tenantID int64, bookingCode, email string) (*AppointmentResult, error) {
	var result *AppointmentResult

	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		apt, err := q.GetAppointmentByBookingCodeAndEmail(ctx, repository.GetAppointmentByBookingCodeAndEmailParams{
			BookingCode: strings.TrimSpace(bookingCode),
			Lower:       strings.ToLower(strings.TrimSpace(email)),
		})
		if err != nil {
			return ErrAppointmentNotFound
		}

		result = &AppointmentResult{
			GUID:             apt.Guid.Bytes,
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
