package staff

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/auth"
	"github.com/FatahRozaq/quevia/backend/internal/branch"
	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
	"github.com/FatahRozaq/quevia/backend/internal/service"
)

func setupTestServices(t *testing.T) (*Service, *branch.Service, *service.Service, *auth.Service, *pgxpool.Pool) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quevia_app:quevia_app_password@127.0.0.1:5432/db-quevia?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping test: database not available: %v", err)
	}

	sessionStore, err := cache.NewSessionStore(redisURL)
	if err != nil {
		t.Skipf("skipping test: redis not available: %v", err)
	}

	tokenManager := token.NewTokenManager("test-secret-at-least-32-chars-long!", 15*time.Minute)
	authSvc := auth.NewService(pool, sessionStore, tokenManager, 14*24*time.Hour)
	branchSvc := branch.NewService(pool)
	serviceSvc := service.NewService(pool)
	staffSvc := NewService(pool)

	return staffSvc, branchSvc, serviceSvc, authSvc, pool
}

func createTenantAndBranch(t *testing.T, ctx context.Context, authSvc *auth.Service, pool *pgxpool.Pool) (int64, int64, uuid.UUID, uuid.UUID) {
	t.Helper()

	slug := "test-org-" + uuid.NewString()[:8]
	email := "owner-" + uuid.NewString()[:8] + "@org.com"

	reg, err := authSvc.RegisterTenant(ctx, auth.RegisterTenantInput{
		TenantName:    "Test Org",
		TenantSlug:    slug,
		BusinessType:  "WORKSHOP",
		ContactEmail:  email,
		BranchCode:    "HQ-01",
		BranchName:    "Headquarters",
		AdminFullName: "Owner Admin",
		AdminEmail:    email,
		AdminPassword: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	var tenantID, userID int64
	_ = database.WithTx(ctx, pool, nil, func(tx pgx.Tx, q *repository.Queries) error {
		tRow, _ := q.GetTenantBySlug(ctx, slug)
		tenantID = tRow.ID
		uRow, _ := q.GetUserByEmail(ctx, email)
		userID = uRow.ID
		return nil
	})

	return tenantID, userID, reg.BranchGUID, reg.UserGUID
}

func TestStaffSchedulesAndBusinessHours(t *testing.T) {
	staffSvc, branchSvc, _, authSvc, pool := setupTestServices(t)
	defer pool.Close()

	ctx := context.Background()
	tenantID, userID, branchGUID, userGUID := createTenantAndBranch(t, ctx, authSvc, pool)

	// 1. Admin sets branch business hours
	err := branchSvc.SetBusinessHours(ctx, tenantID, userID, branchGUID, []branch.BusinessHourInput{
		{DayOfWeek: 1, OpenTime: "08:00", CloseTime: "17:00"},
		{DayOfWeek: 2, OpenTime: "08:00", CloseTime: "17:00"},
	})
	if err != nil {
		t.Fatalf("failed to set business hours: %v", err)
	}

	bHours, err := branchSvc.ListBusinessHours(ctx, tenantID, branchGUID)
	if err != nil || len(bHours) != 2 {
		t.Fatalf("expected 2 business hours rows, got %v (err: %v)", len(bHours), err)
	}

	// 2. Create staff profile
	staff, err := staffSvc.CreateStaff(ctx, tenantID, userID, branchGUID, userGUID, "Budi Mekanik", "Transmisi")
	if err != nil {
		t.Fatalf("failed to create staff: %v", err)
	}

	// 3. Admin sets staff working schedules
	err = staffSvc.SetStaffSchedules(ctx, tenantID, userID, staff.Guid.Bytes, []ScheduleInput{
		{
			DayOfWeek: 1,
			StartTime: "08:00",
			EndTime:   "16:00",
			ValidFrom: "2026-01-01",
		},
	})
	if err != nil {
		t.Fatalf("failed to set staff schedules: %v", err)
	}
}

func TestStaffServiceAssociation(t *testing.T) {
	staffSvc, _, serviceSvc, authSvc, pool := setupTestServices(t)
	defer pool.Close()

	ctx := context.Background()
	tenantID, userID, branchGUID, userGUID := createTenantAndBranch(t, ctx, authSvc, pool)

	// Create 2 services
	svc1, err := serviceSvc.CreateService(ctx, tenantID, userID, service.CreateServiceInput{
		BranchGUID:      branchGUID,
		Name:            "Tune Up",
		DurationMinutes: 45,
		BufferMinutes:   10,
		QueuePrefix:     "A",
	})
	if err != nil {
		t.Fatalf("failed to create service 1: %v", err)
	}

	svc2, err := serviceSvc.CreateService(ctx, tenantID, userID, service.CreateServiceInput{
		BranchGUID:      branchGUID,
		Name:            "Ganti Oli",
		DurationMinutes: 30,
		BufferMinutes:   5,
		QueuePrefix:     "B",
	})
	if err != nil {
		t.Fatalf("failed to create service 2: %v", err)
	}

	// Create staff
	staff, err := staffSvc.CreateStaff(ctx, tenantID, userID, branchGUID, userGUID, "Siti Mekanik", "Engine")
	if err != nil {
		t.Fatalf("failed to create staff: %v", err)
	}

	// Associate staff with both services
	err = staffSvc.AssignServices(ctx, tenantID, userID, staff.Guid.Bytes, []uuid.UUID{
		svc1.Guid.Bytes,
		svc2.Guid.Bytes,
	})
	if err != nil {
		t.Fatalf("failed to assign services: %v", err)
	}

	// Verify staff services in repository
	var staffSvcRows []repository.ListStaffServicesRow
	err = database.WithTx(ctx, pool, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		rows, err := q.ListStaffServices(ctx, staff.ID)
		if err != nil {
			return err
		}
		staffSvcRows = rows
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list staff services: %v", err)
	}
	if len(staffSvcRows) != 2 {
		t.Fatalf("expected 2 associated services, got %d", len(staffSvcRows))
	}
}

func TestStaffDeactivationEnforcesActiveAppointmentsRule(t *testing.T) {
	staffSvc, _, serviceSvc, authSvc, pool := setupTestServices(t)
	defer pool.Close()

	ctx := context.Background()
	tenantID, userID, branchGUID, userGUID := createTenantAndBranch(t, ctx, authSvc, pool)

	// 1. Create staff
	staff, err := staffSvc.CreateStaff(ctx, tenantID, userID, branchGUID, userGUID, "Agus Teknisi", "AC")
	if err != nil {
		t.Fatalf("failed to create staff: %v", err)
	}

	// 2. Deactivating staff without any appointment should SUCCEED
	upd, err := staffSvc.UpdateStaffStatus(ctx, tenantID, userID, staff.Guid.Bytes, "INACTIVE")
	if err != nil {
		t.Fatalf("expected deactivation without appointments to succeed, got: %v", err)
	}
	if upd.Status != "INACTIVE" {
		t.Errorf("expected status INACTIVE, got %s", upd.Status)
	}

	// Reactivate staff for next test
	_, err = staffSvc.UpdateStaffStatus(ctx, tenantID, userID, staff.Guid.Bytes, "ACTIVE")
	if err != nil {
		t.Fatalf("reactivating staff failed: %v", err)
	}

	// 3. Create a service and a customer, then book an active upcoming appointment for this staff
	svcRow, _ := serviceSvc.CreateService(ctx, tenantID, userID, service.CreateServiceInput{
		BranchGUID:      branchGUID,
		Name:            "Service AC",
		DurationMinutes: 60,
		QueuePrefix:     "C",
	})

	var appointmentID int64
	err = database.WithTx(ctx, pool, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		cust, err := q.CreateCustomer(ctx, repository.CreateCustomerParams{
			Guid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:  tenantID,
			UserID:    pgtype.Int8{Valid: false},
			FullName:  "Customer One",
			Email:     pgtype.Text{String: "cust@client.com", Valid: true},
			Phone:     pgtype.Text{String: "08123456789", Valid: true},
			CreatedBy: pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return err
		}

		// Insert appointment row directly
		aptStart := time.Now().Add(24 * time.Hour)
		aptEnd := aptStart.Add(1 * time.Hour)
		apt, err := q.CreateAppointment(ctx, repository.CreateAppointmentParams{
			Guid:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:    tenantID,
			BranchID:    staff.BranchID,
			CustomerID:  cust.ID,
			ServiceID:   svcRow.ID,
			StaffID:     staff.ID,
			BookingCode: "QV-" + uuid.NewString()[:8],
			QrToken:     uuid.NewString(),
			StartAt:     pgtype.Timestamptz{Time: aptStart, Valid: true},
			EndAt:       pgtype.Timestamptz{Time: aptEnd, Valid: true},
			Status:      "CONFIRMED",
			Notes:       pgtype.Text{String: "Test Booking", Valid: true},
			CreatedBy:   pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return err
		}
		appointmentID = apt.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to insert test appointment: %v", err)
	}

	// 4. ATTEMPT DEACTIVATION: Must be REJECTED per FR-SVC-06 / BR-19
	_, err = staffSvc.UpdateStaffStatus(ctx, tenantID, userID, staff.Guid.Bytes, "INACTIVE")
	if err == nil {
		t.Fatal("CRITICAL SECURITY FAILURE: Staff deactivation succeeded despite having active upcoming appointments!")
	}
	if err != ErrStaffHasActiveAppointments {
		t.Fatalf("expected ErrStaffHasActiveAppointments, got: %v", err)
	}

	// 5. Verify staff status is STILL ACTIVE
	staffCheck, err := staffSvc.ListStaff(ctx, tenantID, branchGUID)
	if err != nil || len(staffCheck) == 0 {
		t.Fatalf("failed to list staff: %v", err)
	}
	for _, s := range staffCheck {
		if s.Guid.Bytes == staff.Guid.Bytes && s.Status != "ACTIVE" {
			t.Fatalf("expected staff status to remain ACTIVE, got: %s", s.Status)
		}
	}

	// 6. Cancel the appointment
	_ = database.WithTx(ctx, pool, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		_, err := tx.Exec(ctx, "UPDATE appointments SET status = 'CANCELLED' WHERE id = $1", appointmentID)
		return err
	})

	// 7. Now deactivating staff should SUCCEED
	updFinal, err := staffSvc.UpdateStaffStatus(ctx, tenantID, userID, staff.Guid.Bytes, "INACTIVE")
	if err != nil {
		t.Fatalf("expected deactivation after appointment cancelled to succeed, got: %v", err)
	}
	if updFinal.Status != "INACTIVE" {
		t.Errorf("expected status to be INACTIVE, got: %s", updFinal.Status)
	}
}
