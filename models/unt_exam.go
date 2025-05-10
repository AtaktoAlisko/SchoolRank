package models

// UNTExam represents a unified structure for both Regular and Creative exam types
type UNTExam struct {
	ID                   int    `json:"id"`
	ExamType             string `json:"exam_type"` // "regular" or "creative"
	FirstSubject         string `json:"first_subject,omitempty"`
	FirstSubjectScore    int    `json:"first_subject_score,omitempty"`
	SecondSubject        string `json:"second_subject,omitempty"`
	SecondSubjectScore   int    `json:"second_subject_score,omitempty"`
	HistoryOfKazakhstan  int    `json:"history_of_kazakhstan,omitempty"`
	MathematicalLiteracy int    `json:"mathematical_literacy,omitempty"` // Only for Regular
	ReadingLiteracy      int    `json:"reading_literacy,omitempty"`
	TotalScore           int    `json:"total_score"`
	StudentID            int    `json:"student_id"`
	SchoolID             int    `json:"school_id"`
	StudentTypeID        int    `json:"student_type_id,omitempty"`
	DocumentURL          string `json:"document_url,omitempty"`
	Date                 string `json:"date"`
}
