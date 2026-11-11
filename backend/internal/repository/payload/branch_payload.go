package payload

import "github.com/google/uuid"

type BranchSettingsPayload struct {
	ChangeCutoffMinutes       int32  `json:"change_cutoff_minutes"`
	CheckInOpenBeforeMinutes int32  `json:"check_in_open_before_minutes"`
	CheckInCloseAfterMinutes int32  `json:"check_in_close_after_minutes"`
	NoShowAfterMinutes       int32  `json:"no_show_after_minutes"`
	SlotStepMinutes           int16  `json:"slot_step_minutes"`
	AppointmentPriority       int16  `json:"appointment_priority"`
	WalkInPriority            int16  `json:"walk_in_priority"`
	LateGraceMinutes          *int32 `json:"late_grace_minutes,omitempty"`
	NotifyBeforeQueueCount    int16  `json:"notify_before_queue_count"`
}

type BusinessHourPayload struct {
	DayOfWeek int16  `json:"day_of_week"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

type CreateCounterPayload struct {
	BranchGUID   uuid.UUID `json:"branch_guid"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	DisplayOrder int16     `json:"display_order"`
}
