package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrBranchNotFound  = errors.New("branch not found")
	ErrCounterNotFound = errors.New("service counter not found")
)

type BranchService struct {
	db *pgxpool.Pool
}

func NewBranchService(db *pgxpool.Pool) *BranchService {
	return &BranchService{db: db}
}
