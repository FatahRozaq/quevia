package payload

import "github.com/google/uuid"

type CreateStaffPayload struct {
	BranchGUID     uuid.UUID `json:"branch_guid"`
	UserGUID       uuid.UUID `json:"user_guid"`
	DisplayName    string    `json:"display_name"`
	Specialization string    `json:"specialization"`
}

type UpdateStaffStatusPayload struct {
	Status string `json:"status"` // ACTIVE or INACTIVE
}

type AssignServicesPayload struct {
	ServiceGUIDs []uuid.UUID `json:"service_guids"`
}

type StaffSchedulePayload struct {
	DayOfWeek  int16   `json:"day_of_week"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
	BreakStart *string `json:"break_start,omitempty"`
	BreakEnd   *string `json:"break_end,omitempty"`
	ValidFrom  string  `json:"valid_from"`
	ValidUntil *string `json:"valid_until,omitempty"`
}
