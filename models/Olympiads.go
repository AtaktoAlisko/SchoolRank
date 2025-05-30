package models

type Olympiads struct {
	ID                int    `json:"id"`          // Added missing ID field
	OlympiadID        int    `json:"olympiad_id"` // Keep if needed for compatibility
	OlympiadName      string `json:"olympiad_name"`
	StudentID         int    `json:"student_id"`
	OlympiadPlace     int    `json:"olympiad_place"`
	Score             int    `json:"score"`
	SchoolID          int    `json:"school_id"`
	Level             string `json:"level"`
	FirstName         string `json:"first_name"`         // Maps to StudentFirstName
	LastName          string `json:"last_name"`          // Maps to StudentLastName
	Patronymic        string `json:"patronymic"`         // Maps to StudentPatronymic
	StudentFirstName  string `json:"student_first_name"` // Added explicit field
	StudentLastName   string `json:"student_last_name"`  // Added explicit field
	StudentPatronymic string `json:"student_patronymic"` // Added explicit field
	Grade             int    `json:"grade"`
	Letter            string `json:"letter"`
	DocumentURL       string `json:"document_url"`
	Date              string `json:"date"`
	SchoolName        string `json:"school_name"`
}
