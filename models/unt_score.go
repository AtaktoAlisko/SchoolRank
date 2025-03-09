package models

type UNTScore struct {
    ID                   int    `json:"id"`
    Year                 int    `json:"year"`
    UNTTypeID            int    `json:"unt_type_id"`
    StudentID            int    `json:"student_id"`
    TotalScore           int    `json:"score"`

    // Предметы
    FirstSubjectName     string `json:"first_subject_name"`
    FirstSubjectScore    int    `json:"first_subject_score"`
    SecondSubjectName    string `json:"second_subject_name"`
    SecondSubjectScore   int    `json:"second_subject_score"`

    // История и грамотность
    HistoryKazakhstan    int    `json:"history_of_kazakhstan"`
    MathematicalLiteracy int    `json:"mathematical_literacy"`  
    ReadingLiteracy      int    `json:"reading_literacy"`

    // Данные о студенте
    FirstName            string `json:"first_name"`
    LastName             string `json:"last_name"`
    IIN                     string `json:"iin"`
    Rating              float64 `json:"rating"` 
}
