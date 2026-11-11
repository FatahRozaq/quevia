package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	authSvc "github.com/FatahRozaq/quevia/backend/internal/auth/service"
	branchSvc "github.com/FatahRozaq/quevia/backend/internal/branch/service"
	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/idempotency"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
	serviceSvc "github.com/FatahRozaq/quevia/backend/internal/service/service"
	staffSvc "github.com/FatahRozaq/quevia/backend/internal/staff/service"
)

func setupTestEnvironment(t *testing.T) (*AppointmentService, *idempotency.Manager, *branchSvc.BranchService, *serviceSvc.CatalogService, *staffSvc.StaffService, *authSvc.AuthService, *pgxpool.Pool) {
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

	tokenManager := token.NewTokenManager("test-key-32-bytes-minimum-length!", 15*time.Minute)
	idempMgr := idempotency.NewManager(pool, 24*time.Hour)

	authService := authSvc.NewAuthService(pool, sessionStore, tokenManager, 14*24*time.Hour)
	branchService := branchSvc.NewBranchService(pool)
	catalogService := serviceSvc.NewCatalogService(pool)
	staffService := staffSvc.NewStaffService(pool)
	aptService := NewAppointmentService(pool)

	return aptService, idempMgr, branchService, catalogService, staffService, authService, pool
}

func setupTenantBranchServiceStaff(
	t *testing.T,
	ctx context.Context,
	authService *authSvc.AuthService,
	branchService *branchSvc.BranchService,
	catalogService *serviceSvc.CatalogService,
	staffService *staffSvc.StaffService,
	pool *pgxpool.Pool,
) (int64, int64, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	slug := "apt-biz-" + uuid.NewString()[:8]
	email := "owner-" + uuid.NewString()[:8] + "@biz.com"

	reg, err := authService.RegisterTenant(ctx, payload.RegisterTenantPayload{
		TenantName:    "Bengkel Concurrency",
		TenantSlug:    slug,
		BusinessType:  "WORKSHOP",
		ContactEmail:  email,
		BranchCode:    "CNC-01",
		BranchName:    "Concurrency Branch",
		AdminFullName: "Boss",
		AdminEmail:    email,
		AdminPassword: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	var tenantID, userID int64
	_ = database.WithTx(ctx, pool, nil, func(tx pgx.Tx, q *sqlc.Queries) error {
		tRow, _ := q.GetTenantBySlug(ctx, slug)
		tenantID = tRow.ID
		uRow, _ := q.GetUserByEmail(ctx, email)
		userID = uRow.ID
		return nil
	})

	// Configure branch business hours (Every day 08:00 - 18:00)
	var bHours []payload.BusinessHourPayload
	for d := int16(1); d <= 7; d++ {
		bHours = append(bHours, payload.BusinessHourPayload{
			DayOfWeek: d,
			OpenTime:  "08:00",
			CloseTime: "18:00",
		})
	}
	_ = branchService.SetBusinessHours(ctx, tenantID, userID, reg.BranchGUID, bHours)

	// Create service (30 mins duration)
	svc, err := catalogService.CreateService(ctx, tenantID, userID, payload.CreateServicePayload{
		BranchGUID:      reg.BranchGUID,
		Name:            "Quick Service",
		DurationMinutes: 30,
		QueuePrefix:     "Q",
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Create staff
	staffRow, err := staffService.CreateStaff(ctx, tenantID, userID, payload.CreateStaffPayload{
		BranchGUID:     reg.BranchGUID,
		UserGUID:       reg.UserGUID,
		DisplayName:    "Fast Mechanic",
		Specialization: "Engine",
	})
	if err != nil {
		t.Fatalf("failed to create staff: %v", err)
	}

	// Assign staff to service
	_ = staffService.AssignServices(ctx, tenantID, userID, staffRow.Guid.Bytes, []uuid.UUID{svc.Guid.Bytes})

	// Set staff schedules (Every day 08:00 - 17:00, break 12:00 - 13:00)
	var sSchedules []payload.StaffSchedulePayload
	bStart := "12:00"
	bEnd := "13:00"
	for d := int16(1); d <= 7; d++ {
		sSchedules = append(sSchedules, payload.StaffSchedulePayload{
			DayOfWeek:  d,
			StartTime:  "08:00",
			EndTime:    "17:00",
			BreakStart: &bStart,
			BreakEnd:   &bEnd,
			ValidFrom:  "2026-01-01",
		})
	}
	_ = staffService.SetStaffSchedules(ctx, tenantID, userID, staffRow.Guid.Bytes, sSchedules)

	return tenantID, userID, reg.BranchGUID, svc.Guid.Bytes, staffRow.Guid.Bytes
}

func TestSlotAvailabilityCalculation(t *testing.T) {
	aptService, _, branchService, catalogService, staffService, authService, pool := setupTestEnvironment(t)
	defer pool.Close()

	ctx := context.Background()
	tenantID, _, branchGUID, serviceGUID, staffGUID := setupTenantBranchServiceStaff(
		t, ctx, authService, branchService, catalogService, staffService, pool,
	)

	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	avail, err := aptService.CalculateAvailability(ctx, tenantID, branchGUID, serviceGUID, tomorrow, &staffGUID)
	if err != nil {
		t.Fatalf("calculating availability failed: %v", err)
	}

	if len(avail.Slots) == 0 {
		t.Fatal("expected available slots, got none")
	}

	for _, sl := range avail.Slots {
		if sl.StartAt.Hour() == 12 && sl.StartAt.Minute() == 0 {
			t.Errorf("slot at %v overlaps with lunch break 12:00-13:00", sl.StartAt)
		}
	}
}

func TestConcurrency50GoroutinesAntiDoubleBooking(t *testing.T) {
	aptService, _, branchService, catalogService, staffService, authService, pool := setupTestEnvironment(t)
	defer pool.Close()

	ctx := context.Background()
	tenantID, _, branchGUID, serviceGUID, staffGUID := setupTenantBranchServiceStaff(
		t, ctx, authService, branchService, catalogService, staffService, pool,
	)

	targetSlot := time.Now().Add(48 * time.Hour).Truncate(time.Hour)

	const concurrencyCount = 50
	var successCount int64
	var conflictCount int64

	var wg sync.WaitGroup
	wg.Add(concurrencyCount)

	for i := range concurrencyCount {
		go func(workerID int) {
			defer wg.Done()

			_, err := aptService.CreateAppointment(context.Background(), tenantID, nil, payload.CreateAppointmentPayload{
				BranchGUID:  branchGUID,
				ServiceGUID: serviceGUID,
				StaffGUID:   &staffGUID,
				StartAt:     targetSlot,
				Customer: payload.CustomerPayload{
					FullName: "Competitor",
					Email:    "comp-" + uuid.NewString()[:8] + "@race.com",
				},
				Notes: "Concurrent Booking Attempt",
			})

			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(err, ErrSlotUnavailable) {
				atomic.AddInt64(&conflictCount, 1)
			} else {
				t.Errorf("unexpected error in worker %d: %v", workerID, err)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("CRITICAL GIST EXCLUSION VIOLATION: expected exactly 1 booking to succeed, got %d!", successCount)
	}
	if conflictCount != concurrencyCount-1 {
		t.Fatalf("expected %d conflicts, got %d", concurrencyCount-1, conflictCount)
	}

	var dbCount int64
	_ = database.WithTx(ctx, pool, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		row := tx.QueryRow(ctx, "SELECT count(*) FROM appointments WHERE branch_id = (SELECT id FROM branches WHERE guid = $1) AND start_at = $2 AND status = 'CONFIRMED'", branchGUID, targetSlot)
		return row.Scan(&dbCount)
	})

	if dbCount != 1 {
		t.Fatalf("database contains %d records for the same slot, expected exactly 1!", dbCount)
	}
}

func TestIdempotencyReplay(t *testing.T) {
	aptService, idempMgr, branchService, catalogService, staffService, authService, pool := setupTestEnvironment(t)
	defer pool.Close()

	ctx := context.Background()
	tenantID, _, branchGUID, serviceGUID, staffGUID := setupTenantBranchServiceStaff(
		t, ctx, authService, branchService, catalogService, staffService, pool,
	)

	idempotencyKey := "idemp-key-" + uuid.NewString()
	targetSlot := time.Now().Add(72 * time.Hour).Truncate(time.Hour)

	reqPayload, _ := json.Marshal(map[string]interface{}{
		"branch": branchGUID,
		"slot":   targetSlot,
	})

	// First execution
	resp1, err := idempMgr.Execute(
		ctx,
		tenantID,
		"POST /api/v1/appointments",
		idempotencyKey,
		reqPayload,
		func(ctx context.Context) (int, interface{}, error) {
			res, err := aptService.CreateAppointment(ctx, tenantID, nil, payload.CreateAppointmentPayload{
				BranchGUID:  branchGUID,
				ServiceGUID: serviceGUID,
				StaffGUID:   &staffGUID,
				StartAt:     targetSlot,
				Customer: payload.CustomerPayload{
					FullName: "Idempotent User",
					Email:    "idemp@example.com",
				},
			})
			return 201, res, err
		},
	)
	if err != nil {
		t.Fatalf("initial execution failed: %v", err)
	}
	if resp1.Replayed {
		t.Fatal("expected first execution not to be replayed")
	}

	// Replay execution
	resp2, err := idempMgr.Execute(
		ctx,
		tenantID,
		"POST /api/v1/appointments",
		idempotencyKey,
		reqPayload,
		func(ctx context.Context) (int, interface{}, error) {
			t.Fatal("operation should NOT be executed on replayed idempotency key!")
			return 0, nil, nil
		},
	)
	if err != nil {
		t.Fatalf("replay execution failed: %v", err)
	}
	if !resp2.Replayed {
		t.Fatal("expected response to be marked as Replayed")
	}

	var m1, m2 map[string]interface{}
	_ = json.Unmarshal(resp1.Body, &m1)
	_ = json.Unmarshal(resp2.Body, &m2)
	if m1["booking_code"] != m2["booking_code"] || m1["guid"] != m2["guid"] {
		t.Fatalf("replayed body mismatch: %v vs %v", m1, m2)
	}

	differentPayload := []byte(`{"different":"payload"}`)
	_, err = idempMgr.Execute(
		ctx,
		tenantID,
		"POST /api/v1/appointments",
		idempotencyKey,
		differentPayload,
		func(ctx context.Context) (int, interface{}, error) {
			return 0, nil, nil
		},
	)
	if err != idempotency.ErrIdempotencyKeyReused {
		t.Fatalf("expected ErrIdempotencyKeyReused, got: %v", err)
	}
}
