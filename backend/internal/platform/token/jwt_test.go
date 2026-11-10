package token

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTTokenManager(t *testing.T) {
	secret := "test-secret-key-that-is-at-least-32-bytes!"
	tm := NewTokenManager(secret, 5*time.Minute)

	userGUID := uuid.New()
	tenantGUID := uuid.New()
	tenantID := int64(42)
	role := "TENANT_ADMIN"
	branchGUID := uuid.New()

	tokenStr, err := tm.GenerateAccessToken(userGUID, tenantGUID, tenantID, role, branchGUID)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	claims, err := tm.ValidateAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserGUID != userGUID {
		t.Errorf("expected user guid %v, got %v", userGUID, claims.UserGUID)
	}
	if claims.TenantID != tenantID {
		t.Errorf("expected tenant id %d, got %d", tenantID, claims.TenantID)
	}
	if claims.Role != role {
		t.Errorf("expected role %s, got %s", role, claims.Role)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	tok1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}
	tok2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("failed to generate second refresh token: %v", err)
	}

	if tok1 == "" || tok2 == "" {
		t.Fatal("empty refresh token generated")
	}
	if tok1 == tok2 {
		t.Fatal("expected unique refresh tokens")
	}
}
