package service

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTenantSlugExists   = errors.New("tenant slug already registered")
	ErrUserEmailExists    = errors.New("user email already registered")
	ErrNoActiveMembership = errors.New("no active membership found for user")
)

type AuthService struct {
	db           *pgxpool.Pool
	sessionStore *cache.SessionStore
	tokenManager *token.TokenManager
	refreshTTL   time.Duration
}

func NewAuthService(
	db *pgxpool.Pool,
	sessionStore *cache.SessionStore,
	tokenManager *token.TokenManager,
	refreshTTL time.Duration,
) *AuthService {
	if refreshTTL <= 0 {
		refreshTTL = 14 * 24 * time.Hour
	}
	return &AuthService{
		db:           db,
		sessionStore: sessionStore,
		tokenManager: tokenManager,
		refreshTTL:   refreshTTL,
	}
}

func (s *AuthService) GetSessionStore() *cache.SessionStore {
	return s.sessionStore
}

func (s *AuthService) GetTokenManager() *token.TokenManager {
	return s.tokenManager
}
