package models

type Event struct {
	ID                int    `json:"id"`
	SchoolID          int    `json:"school_id"`
	SchoolName        string `json:"school_name"` // S
	UserID            int    `json:"user_id"`
	EventName         string `json:"event_name"` // S
	Description       string `json:"description"`
	Photo             string `json:"photo"`        // S
	Participants      int    `json:"participants"` // S
	Limit             int    `json:"limit"`        // S
	LimitParticipants int    `json:"limit_participants"`
	StartDate         string `json:"start_date"` // S
	EndDate           string `json:"end_date"`   // S
	Location          string `json:"location"`   // S
	Grade             int    `json:"grade"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	CreatedBy         string `json:"created_by"`
	Category          string `json:"category"`
}
