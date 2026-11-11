package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrServiceNotFound  = errors.New("service not found")
	ErrCategoryNotFound = errors.New("service category not found")
)

type CatalogService struct {
	db *pgxpool.Pool
}

func NewCatalogService(db *pgxpool.Pool) *CatalogService {
	return &CatalogService{db: db}
}
