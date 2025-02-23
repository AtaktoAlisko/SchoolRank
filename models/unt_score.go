package models

type UNTScore struct {
    ID          int    `json:"unt_score_id"`
    Year                int    `json:"year"`
    UNTTypeID           int    `json:"unt_type_id"`
    StudentID           int    `json:"student_id"`
    FirstSubjectScore   int    `json:"first_subject_score"`
    SecondSubjectScore  int    `json:"second_subject_score"`
    HistoryKazakhstan       int    `json:"history_of_kazakhstan"`
    MathLiteracy        int    `json:"mathematical_literacy"`
    ReadingLiteracy     int    `json:"reading_literacy"`
    TotalScore               int    `json:"score"` 
}
