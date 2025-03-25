package models

type FirstType struct {
	ID                   int     `json:"id"`
	FirstSubjectID       *int    `json:"first_subject_id,omitempty"`    // Указатель на int
	SecondSubjectID      *int    `json:"second_subject_id,omitempty"`   // Указатель на int
	HistoryOfKazakhstan  *int    `json:"history_of_kazakhstan,omitempty"`// Указатель на int
	MathematicalLiteracy *int    `json:"mathematical_literacy,omitempty"`// Указатель на int
	ReadingLiteracy      *int    `json:"reading_literacy,omitempty"`     // Указатель на int
	FirstSubjectName     *string `json:"first_subject_name,omitempty"`   // Указатель на string
	SecondSubjectName    *string `json:"second_subject_name,omitempty"`  // Указатель на string
	FirstSubjectScore    *int    `json:"first_subject_score,omitempty"`  // Указатель на int
	SecondSubjectScore   *int    `json:"second_subject_score,omitempty"` // Указатель на int
	TotalScore           *int    `json:"total_score,omitempty"`          // Новый параметр для общего балла
}
