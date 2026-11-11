package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
	"github.com/FatahRozaq/quevia/backend/internal/repository/sqlc"
)

func (s *CatalogService) CreateCategory(ctx context.Context, tenantID int64, userID int64, input payload.CreateCategoryPayload) (*sqlc.CreateServiceCategoryRow, error) {
	var row *sqlc.CreateServiceCategoryRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		res, err := q.CreateServiceCategory(ctx, sqlc.CreateServiceCategoryParams{
			Guid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:  tenantID,
			Name:      input.Name,
			SortOrder: input.SortOrder,
			CreatedBy: pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return err
		}
		row = &res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (s *CatalogService) CreateService(ctx context.Context, tenantID int64, userID int64, input payload.CreateServicePayload) (*sqlc.CreateServiceRow, error) {
	var row *sqlc.CreateServiceRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *sqlc.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: input.BranchGUID, Valid: true})
		if err != nil {
			return errors.New("branch not found")
		}

		var categoryID pgtype.Int8
		if input.CategoryGUID != nil {
			// Optional category link
		}

		var priceNumeric pgtype.Numeric
		if input.Price != nil {
			_ = priceNumeric.Scan(fmt.Sprintf("%.2f", *input.Price))
		}

		res, err := q.CreateService(ctx, sqlc.CreateServiceParams{
			Guid:                   pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:               tenantID,
			BranchID:               branch.ID,
			CategoryID:             categoryID,
			Name:                   input.Name,
			Description:            pgtype.Text{String: input.Description, Valid: input.Description != ""},
			DurationMinutes:        input.DurationMinutes,
			BufferMinutes:          input.BufferMinutes,
			Price:                  priceNumeric,
			QueuePrefix:            input.QueuePrefix,
			RequiredSpecialization: pgtype.Text{String: input.RequiredSpecialization, Valid: input.RequiredSpecialization != ""},
			Status:                 "ACTIVE",
			CreatedBy:              pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating service: %w", err)
		}
		row = &res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}
