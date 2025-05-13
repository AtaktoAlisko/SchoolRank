package models

// EventsParticipant represents a participant in a school event
type EventsParticipant struct {
	ID         int    `json:"id"`
	SchoolID   int    `json:"school_id"`
	Grade      string `json:"grade"`
	Letter     string `json:"letter"`
	StudentID  int    `json:"student_id"`
	EventsName string `json:"events_name"`
	Document   string `json:"document"`
	Category   string `json:"category"`
	Role       string `json:"role"`
	Date       string `json:"date"`
	// Additional fields for response enrichment
	StudentName      string `json:"student_name,omitempty"`
	StudentLastName  string `json:"student_lastname,omitempty"`
	SchoolName       string `json:"school_name,omitempty"`
	CreatorID        int    `json:"creator_id,omitempty"`
	CreatorFirstName string `json:"creator_first_name,omitempty"`
	CreatorLastName  string `json:"creator_last_name,omitempty"`
}
