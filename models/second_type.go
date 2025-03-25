package models

type SecondType struct {
    ID                  int             `json:"second_type_id"`
    HistoryOfKazakhstan int             `json:"history_of_kazakhstan"`
    ReadingLiteracy     int             `json:"reading_literacy"`
    CreativeExam1       int             `json:"creative_exam1"`  // Новый экзамен 1
    CreativeExam2       int             `json:"creative_exam2"`  // Новый экзамен 2
    TotalScore          int             `json:"total_score"`      // Новый параметр для общего балла
}
