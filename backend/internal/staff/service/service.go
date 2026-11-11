package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrStaffNotFound              = errors.New("staff profile not found")
	ErrStaffHasActiveAppointments = errors.New("cannot deactivate staff with active upcoming appointments; reassign bookings first")
	ErrInvalidStatus              = errors.New("invalid status, must be ACTIVE or INACTIVE")
	ErrBranchNotFound             = errors.New("branch not found")
	ErrUserNotFound               = errors.New("user not found")
)

type StaffService struct {
	db *pgxpool.Pool
}

func NewStaffService(db *pgxpool.Pool) *StaffService {
	return &StaffService{db: db}
}
