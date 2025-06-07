package models

import "time"

type EventRegistration struct {
	EventRegistrationID int       `json:"event_registration_id"`
	StudentID           int       `json:"student_id"`
	EventID             int       `json:"event_id"`
	RegistrationDate    time.Time `json:"registration_date"`
	Status              string    `json:"status"`
	SchoolID            int       `json:"school_id"`
	Message             string    `json:"message"`

	StudentName       string `json:"student_name"`
	StudentFirstName  string `json:"student_first_name"`
	StudentLastName   string `json:"student_last_name"`
	StudentPatronymic string `json:"student_patronymic"`
	StudentGrade      string `json:"student_grade"`
	StudentLetter     string `json:"student_letter"`
	StudentRole       string `json:"student_role"`
	SchoolName        string `json:"school_name"`

	EventName      string `json:"event_name,omitempty"`
	EventLocation  string `json:"event_location,omitempty"`
	EventStartDate string `json:"event_start_date,omitempty"`
	EventEndDate   string `json:"event_end_date,omitempty"`
	EventCategory  string `json:"event_category,omitempty"`
	EventPhoto     string `json:"event_photo,omitempty"`
}
