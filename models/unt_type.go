package models

type UNTType struct {
    UNTTypeID         int     `json:"unt_type_id"`
    FirstTypeID       *int    `json:"first_type_id,omitempty"`   
    SecondTypeID      *int    `json:"second_type_id,omitempty"`  
    FirstSubjectID    *int    `json:"first_subject_id,omitempty"`
    FirstSubjectName  *string `json:"first_subject_name,omitempty"`
    FirstSubjectScore *int    `json:"first_subject_score,omitempty"`
    SecondSubjectName *string `json:"second_subject_name,omitempty"`
    SecondSubjectScore *int   `json:"second_subject_score,omitempty"`
    HistoryKazakhstan *int    `json:"history_of_kazakhstan,omitempty"`
    MathematicalLiteracy *int `json:"mathematical_literacy,omitempty"`
    ReadingLiteracy   *int    `json:"reading_literacy,omitempty"`
}
