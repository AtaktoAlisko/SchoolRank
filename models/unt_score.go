package models

type UNTScore struct {
    ID                    int     `json:"id"`
    Year                  int     `json:"year"`
    UNTTypeID             int     `json:"unt_type_id"`
    StudentID             int     `json:"student_id"`
    TotalScore            int     `json:"total_score"`
    TotalScoreCreative    int     `json:"total_score_creative"`  // для творческого экзамена
    CreativeExam1         int     `json:"creative_exam1"`        // Творческий экзамен 1
    CreativeExam2         int     `json:"creative_exam2"`        // Творческий экзамен 2
    FirstSubjectName      string  `json:"first_subject_name"`
    FirstSubjectScore     int     `json:"first_subject_score"`
    SecondSubjectName     string  `json:"second_subject_name"`
    SecondSubjectScore    int     `json:"second_subject_score"`
    HistoryKazakhstan     int     `json:"history_of_kazakhstan"`
    MathematicalLiteracy  int     `json:"mathematical_literacy"`
    ReadingLiteracy       int     `json:"reading_literacy"`
    FirstName             string  `json:"first_name"`
    LastName              string  `json:"last_name"`
    IIN                   string  `json:"iin"`
    Rating               float64 `json:"rating"`  // Рейтинг студента
    AverageRating        float64 `json:"average_rating"`  // Средний рейтинг для первого типа
    AverageRatingSecond  float64 `json:"average_rating_second"`  // Средний рейтинг для второго типа
}
