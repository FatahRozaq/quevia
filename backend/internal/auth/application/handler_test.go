package application

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/auth/service"
	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

func TestAuthEndpointsHTTP(t *testing.T) {
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
		t.Skipf("skipping test: db not available: %v", err)
	}
	defer pool.Close()

	sessionStore, err := cache.NewSessionStore(redisURL)
	if err != nil {
		t.Skipf("skipping test: redis not available: %v", err)
	}

	tokenManager := token.NewTokenManager("test-super-secret-key-at-least-32-chars!", 15*time.Minute)
	svc := service.NewAuthService(pool, sessionStore, tokenManager, 14*24*time.Hour)

	h := server.Default()
	v1 := h.Group("/api/v1")
	authMw := middleware.AuthMiddleware(tokenManager)
	AddRouteAuth(v1, svc, authMw)

	slug := "http-biz-" + uuid.NewString()[:8]
	email := "admin-" + uuid.NewString()[:8] + "@http.com"
	password := "SecureP@ss123!"

	// 1. POST /api/v1/tenants
	regBody, _ := json.Marshal(payload.RegisterTenantPayload{
		TenantName:    "Bengkel Auto HTTP",
		TenantSlug:    slug,
		BusinessType:  "WORKSHOP",
		ContactEmail:  email,
		BranchCode:    "HTTP-01",
		BranchName:    "Cabang HTTP",
		AdminFullName: "HTTP Admin",
		AdminEmail:    email,
		AdminPassword: password,
	})

	wReg := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/tenants",
		&ut.Body{Body: bytes.NewBuffer(regBody), Len: len(regBody)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	assert.DeepEqual(t, consts.StatusCreated, wReg.Result().StatusCode())

	// 2. POST /api/v1/auth/login
	loginBody, _ := json.Marshal(payload.LoginPayload{
		Email:    email,
		Password: password,
	})

	wLogin := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/auth/login",
		&ut.Body{Body: bytes.NewBuffer(loginBody), Len: len(loginBody)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	assert.DeepEqual(t, consts.StatusOK, wLogin.Result().StatusCode())

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Result().Body(), &loginResp)
	tokenStr := loginResp.Data.AccessToken
	if tokenStr == "" {
		t.Fatal("expected access token in response")
	}

	// 3. GET /api/v1/me (Authenticated)
	wMe := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/me", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + tokenStr},
	)
	assert.DeepEqual(t, consts.StatusOK, wMe.Result().StatusCode())
}
