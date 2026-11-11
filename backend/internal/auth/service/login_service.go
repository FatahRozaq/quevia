package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/security"
	"github.com/FatahRozaq/quevia/backend/internal/platform/token"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

func (s *AuthService) Login(ctx context.Context, input payload.LoginPayload) (*payload.LoginResponse, error) {
	input.Normalize()

	var result *payload.LoginResponse
	err := database.WithTx(ctx, s.db, nil, func(tx pgx.Tx, q *sqlc.Queries) error {
		user, err := q.GetUserByEmail(ctx, input.Email)
		if err != nil {
			return ErrInvalidCredentials
		}

		if !user.PasswordHash.Valid {
			return ErrInvalidCredentials
		}

		match, err := security.VerifyPassword(input.Password, user.PasswordHash.String)
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

		result = &payload.LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			User: payload.UserResponse{
				GUID:       userGUID,
				FullName:   user.FullName,
				Email:      user.Email,
				TenantGUID: tenantGUID,
				TenantSlug: m.TenantSlug,
				BranchGUID: branchGUID,
				Role:       m.Role,
			},
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
