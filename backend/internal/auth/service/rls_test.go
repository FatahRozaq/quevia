package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

func getTestDBAndRedis(t *testing.T) (*AuthService, *pgxpool.Pool) {
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
		t.Skipf("skipping db integration test (cannot connect to postgres): %v", err)
	}

	sessionStore, err := cache.NewSessionStore(redisURL)
	if err != nil {
		t.Skipf("skipping redis integration test (cannot connect to redis): %v", err)
	}

	tokenManager := token.NewTokenManager("test-super-secret-key-at-least-32-chars!", 15*time.Minute)
	svc := NewAuthService(pool, sessionStore, tokenManager, 14*24*time.Hour)

	return svc, pool
}

func TestRegisterTenantAndLogin(t *testing.T) {
	svc, pool := getTestDBAndRedis(t)
	defer pool.Close()

	ctx := context.Background()
	slug := "test-biz-" + uuid.NewString()[:8]
	email := "admin-" + uuid.NewString()[:8] + "@example.com"
	password := "Secret123!"

	// 1. Register Tenant
	regResult, err := svc.RegisterTenant(ctx, payload.RegisterTenantPayload{
		TenantName:    "Bengkel Maju Jaya",
		TenantSlug:    slug,
		BusinessType:  "WORKSHOP",
		ContactEmail:  email,
		BranchCode:    "JKT-01",
		BranchName:    "Cabang Jakarta Pusat",
		AdminFullName: "Budi Santoso",
		AdminEmail:    email,
		AdminPassword: password,
		AdminPhone:    "081234567890",
	})
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	if regResult.TenantSlug != slug {
		t.Errorf("expected slug %s, got %s", slug, regResult.TenantSlug)
	}

	// 2. Login
	loginResult, err := svc.Login(ctx, payload.LoginPayload{
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}

	if loginResult.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if loginResult.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if loginResult.User.Role != "TENANT_ADMIN" {
		t.Errorf("expected role TENANT_ADMIN, got %s", loginResult.User.Role)
	}

	// 3. Verify Refresh Token in Redis
	userGUID, err := svc.sessionStore.ValidateRefreshToken(ctx, loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("refresh token validation in redis failed: %v", err)
	}
	if userGUID != loginResult.User.GUID.String() {
		t.Errorf("expected user guid in redis %s, got %s", loginResult.User.GUID.String(), userGUID)
	}
}

func TestMultiTenantRLSIsolation(t *testing.T) {
	svc, pool := getTestDBAndRedis(t)
	defer pool.Close()

	ctx := context.Background()

	// Register Tenant A
	slugA := "tenant-a-" + uuid.NewString()[:8]
	emailA := "admin-" + uuid.NewString()[:8] + "@alpha.com"
	resA, err := svc.RegisterTenant(ctx, payload.RegisterTenantPayload{
		TenantName:    "Tenant Alpha",
		TenantSlug:    slugA,
		ContactEmail:  emailA,
		BranchCode:    "ALPHA-01",
		BranchName:    "Alpha Central Branch",
		AdminFullName: "Alpha Admin",
		AdminEmail:    emailA,
		AdminPassword: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to register tenant A: %v", err)
	}

	// Register Tenant B
	slugB := "tenant-b-" + uuid.NewString()[:8]
	emailB := "admin-" + uuid.NewString()[:8] + "@beta.com"
	resB, err := svc.RegisterTenant(ctx, payload.RegisterTenantPayload{
		TenantName:    "Tenant Beta",
		TenantSlug:    slugB,
		ContactEmail:  emailB,
		BranchCode:    "BETA-01",
		BranchName:    "Beta Central Branch",
		AdminFullName: "Beta Admin",
		AdminEmail:    emailB,
		AdminPassword: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to register tenant B: %v", err)
	}

	// Retrieve internal tenant IDs
	var tenantIDA, tenantIDB int64
	_ = database.WithTx(ctx, pool, nil, func(tx pgx.Tx, q *sqlc.Queries) error {
		tA, _ := q.GetTenantBySlug(ctx, slugA)
		tenantIDA = tA.ID
		tB, _ := q.GetTenantBySlug(ctx, slugB)
		tenantIDB = tB.ID
		return nil
	})

	// Query with Tenant A context:
	err = database.WithTx(ctx, pool, &tenantIDA, func(tx pgx.Tx, q *sqlc.Queries) error {
		branches, err := q.ListBranchesByTenant(ctx, tenantIDA)
		if err != nil {
			return err
		}
		if len(branches) != 1 || branches[0].Code != "ALPHA-01" {
			t.Fatalf("expected 1 branch for Tenant A with code ALPHA-01, got %v", branches)
		}

		// Malicious attempt: Query Tenant B's ID while in Tenant A's RLS session
		leakBranches, err := q.ListBranchesByTenant(ctx, tenantIDB)
		if err != nil {
			return err
		}
		if len(leakBranches) != 0 {
			t.Fatalf("RLS VIOLATION: Tenant A saw %d branches belonging to Tenant B!", len(leakBranches))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Tenant A transaction failed: %v", err)
	}

	// Query with Tenant B context:
	err = database.WithTx(ctx, pool, &tenantIDB, func(tx pgx.Tx, q *sqlc.Queries) error {
		leakBranches, err := q.ListBranchesByTenant(ctx, tenantIDA)
		if err != nil {
			return err
		}
		if len(leakBranches) != 0 {
			t.Fatalf("RLS VIOLATION: Tenant B saw %d branches belonging to Tenant A!", len(leakBranches))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Tenant B transaction failed: %v", err)
	}

	_ = resA
	_ = resB
}
