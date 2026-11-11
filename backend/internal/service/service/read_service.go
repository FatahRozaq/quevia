package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
)

func (s *CatalogService) ListCategories(ctx context.Context, tenantID int64) ([]repository.ListServiceCategoriesRow, error) {
	var rows []repository.ListServiceCategoriesRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		res, err := q.ListServiceCategories(ctx, tenantID)
		if err != nil {
			return err
		}
		rows = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *CatalogService) ListServices(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]repository.ListServicesByBranchRow, error) {
	var rows []repository.ListServicesByBranchRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return errors.New("branch not found")
		}

		res, err := q.ListServicesByBranch(ctx, branch.ID)
		if err != nil {
			return err
		}
		rows = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}
