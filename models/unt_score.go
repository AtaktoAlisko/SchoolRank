package models

type UNTScore struct {
    ID                    int     `json:"id"`                     // Идентификатор записи
    Year                  int     `json:"year"`                   // Год сдачи экзамена
    UNTTypeID             int     `json:"unt_type_id"`            // Идентификатор типа экзамена (1 — профильный, 2 — творческий)
    StudentID             int     `json:"student_id"`             // Идентификатор студента
    TotalScore            int     `json:"total_score"`            // Баллы по основным экзаменам для первого типа (total_score)
    TotalScoreCreative    int     `json:"total_score_creative"`   // Баллы по творческому экзамену для второго типа (total_score_creative)
    CreativeExam1         int     `json:"creative_exam1"`         // Творческий экзамен 1
    CreativeExam2         int     `json:"creative_exam2"`         // Творческий экзамен 2
    FirstSubjectName      string  `json:"first_subject_name"`     // Название первого предмета
    FirstSubjectScore     int     `json:"first_subject_score"`    // Баллы по первому предмету
    SecondSubjectName     string  `json:"second_subject_name"`    // Название второго предмета
    SecondSubjectScore    int     `json:"second_subject_score"`   // Баллы по второму предмету
    HistoryKazakhstan     int     `json:"history_of_kazakhstan"`  // Баллы по предмету "История Казахстана"
    MathematicalLiteracy  int     `json:"mathematical_literacy"`  // Баллы по математической грамотности
    ReadingLiteracy       int     `json:"reading_literacy"`       // Баллы по читательской грамотности
    FirstName             string  `json:"first_name"`             // Имя студента
    LastName              string  `json:"last_name"`              // Фамилия студента
    IIN                   string  `json:"iin"`                    // ИИН студента

    // Средний рейтинг для первого типа (профильного экзамена)
    AverageRating         float64 `json:"average_rating"`          // Средний рейтинг для первого типа (для всех студентов, сдавших первый тип)

    // Средний рейтинг для второго типа (творческого экзамена)
    AverageRatingSecond   float64 `json:"average_rating_second"`   // Средний рейтинг для второго типа (для всех студентов, сдавших второй тип)
}
