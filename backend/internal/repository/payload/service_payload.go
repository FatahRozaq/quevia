package payload

import "github.com/google/uuid"

type CreateCategoryPayload struct {
	Name      string `json:"name"`
	SortOrder int16  `json:"sort_order"`
}

type CreateServicePayload struct {
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
