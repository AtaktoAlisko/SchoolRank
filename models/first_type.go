package models

type FirstType struct {
    ID                    int    `json:"first_type_id"`
    FirstSubjectID        int    `json:"first_subject_id"`
    SecondSubjectID       int    `json:"second_subject_id"`
    HistoryOfKazakhstan   int    `json:"history_of_kazakhstan"`
    MathematicalLiteracy  int    `json:"mathematical_literacy"`
    ReadingLiteracy       int    `json:"reading_literacy"`
}
