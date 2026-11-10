package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/cache"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
	"github.com/FatahRozaq/quevia/backend/internal/platform/security"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTenantSlugExists   = errors.New("tenant slug already registered")
	ErrUserEmailExists    = errors.New("user email already registered")
	ErrNoActiveMembership = errors.New("no active membership found for user")
)

type RegisterTenantInput struct {
	TenantName    string `json:"tenant_name"`
	TenantSlug    string `json:"tenant_slug"`
	BusinessType  string `json:"business_type"`
	ContactEmail  string `json:"contact_email"`
	BranchCode    string `json:"branch_code"`
	BranchName    string `json:"branch_name"`
	AdminFullName string `json:"admin_full_name"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	AdminPhone    string `json:"admin_phone"`
}

type RegisterTenantResult struct {
	TenantGUID uuid.UUID `json:"tenant_guid"`
	TenantSlug string    `json:"tenant_slug"`
	BranchGUID uuid.UUID `json:"branch_guid"`
	UserGUID   uuid.UUID `json:"user_guid"`
}

type LoginResult struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"-"`
	UserGUID     uuid.UUID `json:"user_guid"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	TenantGUID   uuid.UUID `json:"tenant_guid"`
	TenantSlug   string    `json:"tenant_slug"`
	BranchGUID   uuid.UUID `json:"branch_guid,omitempty"`
	Role         string    `json:"role"`
}

type Service struct {
	db           *pgxpool.Pool
	sessionStore *cache.SessionStore
	tokenManager *token.TokenManager
	refreshTTL   time.Duration
}

func NewService(
	db *pgxpool.Pool,
	sessionStore *cache.SessionStore,
	tokenManager *token.TokenManager,
	refreshTTL time.Duration,
) *Service {
	if refreshTTL <= 0 {
		refreshTTL = 14 * 24 * time.Hour
	}
	return &Service{
		db:           db,
		sessionStore: sessionStore,
		tokenManager: tokenManager,
		refreshTTL:   refreshTTL,
	}
}

// RegisterTenant creates a tenant, first branch, admin user, and membership atomically.
func (s *Service) RegisterTenant(ctx context.Context, input RegisterTenantInput) (*RegisterTenantResult, error) {
	input.TenantSlug = strings.ToLower(strings.TrimSpace(input.TenantSlug))
	input.AdminEmail = strings.ToLower(strings.TrimSpace(input.AdminEmail))

	hashedPassword, err := security.HashPassword(input.AdminPassword)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	result := &RegisterTenantResult{}

	err = database.WithTx(ctx, s.db, nil, func(tx pgx.Tx, q *repository.Queries) error {
		// 1. Check existing tenant slug
		if _, err := q.GetTenantBySlug(ctx, input.TenantSlug); err == nil {
			return ErrTenantSlugExists
		}

		// 2. Check existing user email
		if _, err := q.GetUserByEmail(ctx, input.AdminEmail); err == nil {
			return ErrUserEmailExists
		}

		// 3. Create Tenant
		tenantGUID := uuid.New()
		tenant, err := q.CreateTenant(ctx, repository.CreateTenantParams{
			Guid:         pgtype.UUID{Bytes: tenantGUID, Valid: true},
			Name:         input.TenantName,
			Slug:         input.TenantSlug,
			BusinessType: pgtype.Text{String: input.BusinessType, Valid: input.BusinessType != ""},
			ContactEmail: input.ContactEmail,
			Status:       "ACTIVE",
		})
		if err != nil {
			return fmt.Errorf("creating tenant: %w", err)
		}

		// 4. Create User
		userGUID := uuid.New()
		user, err := q.CreateUser(ctx, repository.CreateUserParams{
			Guid:         pgtype.UUID{Bytes: userGUID, Valid: true},
			Email:        input.AdminEmail,
			PasswordHash: pgtype.Text{String: hashedPassword, Valid: true},
			FullName:     input.AdminFullName,
			Phone:        pgtype.Text{String: input.AdminPhone, Valid: input.AdminPhone != ""},
			PlatformRole: pgtype.Text{Valid: false},
			Status:       "ACTIVE",
			IsSystem:     false,
		})
		if err != nil {
			return fmt.Errorf("creating user: %w", err)
		}

		// 5. Inject tenant RLS context for subsequent tenant-scoped inserts
		if err := database.SetTenantContext(ctx, tx, tenant.ID); err != nil {
			return err
		}

		// 6. Create Default Branch
		branchGUID := uuid.New()
		_, err = q.CreateBranch(ctx, repository.CreateBranchParams{
			Guid:      pgtype.UUID{Bytes: branchGUID, Valid: true},
			TenantID:  tenant.ID,
			Code:      input.BranchCode,
			Name:      input.BranchName,
			Address:   pgtype.Text{Valid: false},
			Phone:     pgtype.Text{Valid: false},
			Timezone:  "Asia/Jakarta",
			Status:    "ACTIVE",
			CreatedBy: pgtype.Int8{Int64: user.ID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating branch: %w", err)
		}

		// 7. Create Membership (TENANT_ADMIN)
		membershipGUID := uuid.New()
		_, err = q.CreateMembership(ctx, repository.CreateMembershipParams{
			Guid:      pgtype.UUID{Bytes: membershipGUID, Valid: true},
			TenantID:  tenant.ID,
			UserID:    user.ID,
			BranchID:  pgtype.Int8{Valid: false}, // null branch_id = access to all branches
			Role:      "TENANT_ADMIN",
			Status:    "ACTIVE",
			CreatedBy: pgtype.Int8{Int64: user.ID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating membership: %w", err)
		}

		result.TenantGUID = tenantGUID
		result.TenantSlug = tenant.Slug
		result.BranchGUID = branchGUID
		result.UserGUID = userGUID
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Login authenticates a user and returns an access token along with a refresh token.
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var result *LoginResult
	err := database.WithTx(ctx, s.db, nil, func(tx pgx.Tx, q *repository.Queries) error {
		user, err := q.GetUserByEmail(ctx, email)
		if err != nil {
			return ErrInvalidCredentials
		}

		if !user.PasswordHash.Valid {
			return ErrInvalidCredentials
		}

		match, err := security.VerifyPassword(password, user.PasswordHash.String)
		if err != nil || !match {
			return ErrInvalidCredentials
		}

		memberships, err := q.GetMembershipsByUserID(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("getting memberships: %w", err)
		}
		if len(memberships) == 0 {
			return ErrNoActiveMembership
		}

		// Primary membership (first active)
		m := memberships[0]
		var branchGUID uuid.UUID
		if m.BranchGuid.Valid {
			branchGUID = uuid.UUID(m.BranchGuid.Bytes)
		}
		tenantGUID := uuid.UUID(m.TenantGuid.Bytes)
		userGUID := uuid.UUID(user.Guid.Bytes)

		accessToken, err := s.tokenManager.GenerateAccessToken(
			userGUID,
			tenantGUID,
			m.TenantID,
			m.Role,
			branchGUID,
		)
		if err != nil {
			return fmt.Errorf("generating access token: %w", err)
		}

		refreshToken, err := token.GenerateRefreshToken()
		if err != nil {
			return fmt.Errorf("generating refresh token: %w", err)
		}

		if err := s.sessionStore.StoreRefreshToken(ctx, refreshToken, userGUID.String(), s.refreshTTL); err != nil {
			return fmt.Errorf("storing refresh token in redis: %w", err)
		}

		result = &LoginResult{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			UserGUID:     userGUID,
			FullName:     user.FullName,
			Email:        user.Email,
			TenantGUID:   tenantGUID,
			TenantSlug:   m.TenantSlug,
			BranchGUID:   branchGUID,
			Role:         m.Role,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
