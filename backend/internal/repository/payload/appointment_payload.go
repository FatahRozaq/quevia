package payload

import (
	"time"

	"github.com/google/uuid"
)

type CustomerPayload struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone,omitempty"`
}

type CreateAppointmentPayload struct {
	BranchGUID  uuid.UUID       `json:"branch_guid"`
	ServiceGUID uuid.UUID       `json:"service_guid"`
	StaffGUID   *uuid.UUID      `json:"staff_guid,omitempty"`
	StartAt     time.Time       `json:"start_at"`
	Customer    CustomerPayload `json:"customer"`
	Notes       string          `json:"notes,omitempty"`
}

type AppointmentResponse struct {
	GUID             uuid.UUID `json:"guid"`
	BookingCode      string    `json:"booking_code"`
	QRToken          string    `json:"qr_token"`
	Status           string    `json:"status"`
	StartAt          time.Time `json:"start_at"`
	EndAt            time.Time `json:"end_at"`
	ServiceName      string    `json:"service_name,omitempty"`
	StaffDisplayName string    `json:"staff_display_name,omitempty"`
	StaffGUID        uuid.UUID `json:"staff_guid"`
	CustomerName     string    `json:"customer_name,omitempty"`
	CustomerEmail    string    `json:"customer_email,omitempty"`
}

type LookupBookingPayload struct {
	BookingCode string `json:"booking_code"`
	Email       string `json:"email"`
}

type CancelAppointmentPayload struct {
	Reason string `json:"reason"`
}

type SlotResponse struct {
	StartAt    time.Time   `json:"start_at"`
	EndAt      time.Time   `json:"end_at"`
	StaffGUIDs []uuid.UUID `json:"staff_guids"`
}

type AvailabilityResponse struct {
	BranchTimezone  string         `json:"branch_timezone"`
	Date            string         `json:"date"`
	DurationMinutes int32          `json:"duration_minutes"`
	Slots           []SlotResponse `json:"slots"`
}
