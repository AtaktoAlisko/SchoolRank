package models

type FirstType struct {
	ID                   int     `json:"id"`
	FirstSubjectID       int     `json:"first_subject_id"`
	SecondSubjectID      int     `json:"second_subject_id"`
	HistoryOfKazakhstan  int     `json:"history_of_kazakhstan"`
	MathematicalLiteracy int     `json:"mathematical_literacy"`
	ReadingLiteracy      int     `json:"reading_literacy"`
	FirstSubjectName     string  `json:"first_subject_name"`
	SecondSubjectName    string  `json:"second_subject_name"`
	FirstSubjectScore    int     `json:"first_subject_score"`
	SecondSubjectScore   int     `json:"second_subject_score"`
}
