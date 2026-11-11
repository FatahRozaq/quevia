package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
	"github.com/FatahRozaq/quevia/backend/internal/platform/security"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

func (s *AuthService) RegisterTenant(ctx context.Context, input payload.RegisterTenantPayload) (*payload.RegisterTenantResponse, error) {
	input.Normalize()

	hashedPassword, err := security.HashPassword(input.AdminPassword)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	result := &payload.RegisterTenantResponse{}

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
			BranchID:  pgtype.Int8{Valid: false},
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
