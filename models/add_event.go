package models

type Event struct {
	ID          int    `json:"id"`
	SchoolID    int    `json:"school_id"`
	UserID      int    `json:"user_id"`
	EventName   string `json:"event_name"`
	Description string `json:"description"`
	Photo       string `json:"photo"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Category    string `json:"category"`
	Location    string `json:"location"`
	Grade       int    `json:"grade"`
	Limit       int    `json:"limit"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	CreatedBy   string `json:"created_by"`
}
