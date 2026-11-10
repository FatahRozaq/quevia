package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserClaims defines JWT token payload.
type UserClaims struct {
	UserGUID   uuid.UUID `json:"user_guid"`
	TenantGUID uuid.UUID `json:"tenant_guid"`
	TenantID   int64     `json:"tenant_id"`
	Role       string    `json:"role"`
	BranchGUID uuid.UUID `json:"branch_guid,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager issues and validates JWT tokens.
type TokenManager struct {
	signingKey []byte
	accessTTL  time.Duration
}

// NewTokenManager initializes a JWT token manager.
func NewTokenManager(secret string, accessTTL time.Duration) *TokenManager {
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	return &TokenManager{
		signingKey: []byte(secret),
		accessTTL:  accessTTL,
	}
}

// GenerateAccessToken signs a new JWT access token.
func (m *TokenManager) GenerateAccessToken(
	userGUID uuid.UUID,
	tenantGUID uuid.UUID,
	tenantID int64,
	role string,
	branchGUID uuid.UUID,
) (string, error) {
	now := time.Now()
	claims := UserClaims{
		UserGUID:   userGUID,
		TenantGUID: tenantGUID,
		TenantID:   tenantID,
		Role:       role,
		BranchGUID: branchGUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.signingKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

// ValidateAccessToken parses and validates a signed JWT token.
func (m *TokenManager) ValidateAccessToken(tokenStr string) (*UserClaims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.signingKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := parsed.Claims.(*UserClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// GenerateRefreshToken creates a secure random hex token.
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
