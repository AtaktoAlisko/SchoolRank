package models

type SubjectOlympiad struct {
	SubjectName string `json:"subject_name"`
	StartDate   string `json:"start_date"`
	Description string `json:"description"`
	PhotoURL    string `json:"photo_url"`
	SchoolID    int    `json:"school_id"`
	EndDate     string `json:"end_date"`
	Level       string `json:"level"`
	Limit       int    `json:"limit_participants"`
}
