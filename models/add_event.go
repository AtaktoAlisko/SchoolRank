package models

type Event struct {
	ID                int    `json:"id"`
	SchoolID          int    `json:"school_id"`
	SchoolName        string `json:"school_name"`
	UserID            int    `json:"user_id"`
	EventName         string `json:"event_name"`
	Description       string `json:"description"`
	Photo             string `json:"photo"`
	Participants      int    `json:"participants"`
	Limit             int    `json:"limit"`
	LimitParticipants int    `json:"limit_participants"`
	StartDate         string `json:"start_date"`
	EndDate           string `json:"end_date"`
	Location          string `json:"location"`
	Grade             int    `json:"grade"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	CreatedBy         string `json:"created_by"`
	Category          string `json:"category"` // Renamed from Type
}
