package models

import "time"

type OlympiadRegistration struct {
	OlympiadsRegistrationsID int       `json:"olympiads_registrations_id"`
	StudentID                int       `json:"student_id"`
	SubjectOlympiadID        int       `json:"subject_olympiad_id"`
	RegistrationDate         time.Time `json:"registration_date"`
	Status                   string    `json:"status"`
	SchoolID                 int       `json:"school_id"`

	StudentName       string `json:"student_name"`
	StudentFirstName  string `json:"student_first_name"`
	StudentLastName   string `json:"student_last_name"`
	StudentPatronymic string `json:"student_patronymic"`
	StudentGrade      string `json:"student_grade"`
	StudentLetter     string `json:"student_letter"`

	SchoolName string `json:"school_name"`

	OlympiadName      string `json:"olympiad_name"`
	OlympiadStartDate string `json:"olympiad_start_date"`
	OlympiadEndDate   string `json:"olympiad_end_date"`
	OlympiadPlace     int    `json:"olympiad_place,omitempty"`
	Score             int    `json:"score,omitempty"`
	Level             string `json:"level"`
}
