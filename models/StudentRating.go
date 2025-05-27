package models


type StudentRating struct {
    StudentID          int     `json:"student_id"`
    Year               int     `json:"year"`
    TotalScore         int     `json:"total_score"`
    FirstTypeID        *int    `json:"first_type_id"`
    SecondTypeID       *int    `json:"second_type_id"`
    FirstSubjectName   string  `json:"first_subject_name"`
    FirstSubjectScore  int     `json:"first_subject_score"`
    SecondSubjectName  string  `json:"second_subject_name"`
    SecondSubjectScore int     `json:"second_subject_score"`
    HistoryKazakhstan  int     `json:"history_of_kazakhstan"`
    ReadingLiteracy    int     `json:"reading_literacy"`
 MathematicalLiteracy int    `json:"mathematical_literacy"`
    Rating             float64 `json:"rating"` 
    
}
