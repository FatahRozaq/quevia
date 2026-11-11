package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
)

var (
	ErrServiceNotFound  = errors.New("service not found")
	ErrCategoryNotFound = errors.New("service category not found")
)

type CreateServiceInput struct {
	BranchGUID             uuid.UUID  `json:"branch_guid"`
	CategoryGUID           *uuid.UUID `json:"category_guid,omitempty"`
	Name                   string     `json:"name"`
	Description            string     `json:"description"`
	DurationMinutes        int32      `json:"duration_minutes"`
	BufferMinutes          int32      `json:"buffer_minutes"`
	Price                  *float64   `json:"price,omitempty"`
	QueuePrefix            string     `json:"queue_prefix"`
	RequiredSpecialization string     `json:"required_specialization"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) ListCategories(ctx context.Context, tenantID int64) ([]repository.ListServiceCategoriesRow, error) {
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

func (s *Service) CreateCategory(ctx context.Context, tenantID int64, userID int64, name string, sortOrder int16) (*repository.CreateServiceCategoryRow, error) {
	var row *repository.CreateServiceCategoryRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		res, err := q.CreateServiceCategory(ctx, repository.CreateServiceCategoryParams{
			Guid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
			TenantID:  tenantID,
			Name:      name,
			SortOrder: sortOrder,
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

func (s *Service) ListServices(ctx context.Context, tenantID int64, branchGUID uuid.UUID) ([]repository.ListServicesByBranchRow, error) {
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

func (s *Service) CreateService(ctx context.Context, tenantID int64, userID int64, input CreateServiceInput) (*repository.CreateServiceRow, error) {
	var row *repository.CreateServiceRow
	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: input.BranchGUID, Valid: true})
		if err != nil {
			return errors.New("branch not found")
		}

		var categoryID pgtype.Int8
		// If category provided, lookup ID (or keep null)

		var priceNumeric pgtype.Numeric
		if input.Price != nil {
			_ = priceNumeric.Scan(fmt.Sprintf("%.2f", *input.Price))
		}

		res, err := q.CreateService(ctx, repository.CreateServiceParams{
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
