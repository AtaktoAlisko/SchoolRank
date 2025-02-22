package models

type FirstType struct {
	FirstTypeID          int    `json:"first_type_id"`
	FirstSubjectID       int    `json:"first_subject_id"`
	SecondSubjectID      int    `json:"second_subject_id"`
	HistoryOfKazakhstan  string `json:"history_of_kazakhstan"`
	MathematicalLiteracy string `json:"mathematical_literacy"`
	ReadingLiteracy      string `json:"reading_literacy"`
}
