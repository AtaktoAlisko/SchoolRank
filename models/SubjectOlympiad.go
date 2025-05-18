package models

type SubjectOlympiad struct {
	ID               int    `json:"subject_olympiad_id"`
	SubjectName      string `json:"subject_name"`
	StartDate        string `json:"start_date"`
	Description      string `json:"description"`
	SchoolID         int    `json:"school_id"`
	EndDate          string `json:"end_date,omitempty"` // Made omitempty to hide when expired
	Level            string `json:"level"`
	Limit            int    `json:"limit_participants"`
	CreatorID        int    `json:"creator_id,omitempty"`
	CreatorFirstName string `json:"creator_first_name,omitempty"`
	CreatorLastName  string `json:"creator_last_name,omitempty"`
	SchoolName       string `json:"school_name,omitempty"`
	Expired          bool   `json:"expired"` // New field to indicate if the olympiad has expired
}
