package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/google/uuid"

	"github.com/FatahRozaq/quevia/backend/internal/platform/middleware"
)

func TestAuthEndpointsHTTP(t *testing.T) {
	svc, pool := getTestDBAndRedis(t)
	defer pool.Close()

	handler := NewHandler(svc)
	h := server.Default()
	v1 := h.Group("/api/v1")
	authMw := middleware.AuthMiddleware(svc.tokenManager)
	handler.RegisterRoutes(v1, authMw)

	slug := "http-biz-" + uuid.NewString()[:8]
	email := "admin-" + uuid.NewString()[:8] + "@http.com"
	password := "SecureP@ss123!"

	// 1. POST /api/v1/tenants
	regBody, _ := json.Marshal(RegisterTenantInput{
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
	loginBody, _ := json.Marshal(LoginRequest{
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
