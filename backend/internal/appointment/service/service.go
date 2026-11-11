package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSlotUnavailable     = errors.New("selected slot is no longer available")
	ErrAppointmentNotFound = errors.New("appointment not found")
	ErrChangeCutoffPassed  = errors.New("change cutoff time has passed")
	ErrInvalidEmail        = errors.New("valid customer email is required")
	ErrNoStaffAvailable    = errors.New("no staff available for selected time")
	ErrServiceNotFound     = errors.New("service not found")
	ErrBranchNotFound      = errors.New("branch not found")
	ErrStaffNotEligible    = errors.New("staff not eligible for this service")
)

type AppointmentService struct {
	db *pgxpool.Pool
}

func NewAppointmentService(db *pgxpool.Pool) *AppointmentService {
	return &AppointmentService{db: db}
}

func generateBookingCode() string {
	const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	code := make([]byte, 6)
	for i := range code {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[num.Int64()]
	}
	return "QV-" + string(code)
}

func generateQRToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
