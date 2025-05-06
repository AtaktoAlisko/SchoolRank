package models

type FirstType struct {
	ID                   int    `json:"id"`                    // Идентификатор записи
	FirstSubject         string `json:"first_subject"`         // Название первого предмета
	FirstSubjectScore    int    `json:"first_subject_score"`   // Баллы по первому предмету
	SecondSubject        string `json:"second_subject"`        // Название второго предмета
	SecondSubjectScore   int    `json:"second_subject_score"`  // Баллы по второму предмету
	HistoryOfKazakhstan  int    `json:"history_of_kazakhstan"` // Баллы по истории Казахстана
	MathematicalLiteracy int    `json:"mathematical_literacy"` // Баллы по математической грамотности
	ReadingLiteracy      int    `json:"reading_literacy"`      // Баллы по читательской грамотности
	TotalScore           int    `json:"total_score"`           // Общий балл
	Type                 string `json:"type"`                  // Тип экзамена
	StudentID            int    `json:"student_id"`            // ID студента
	SchoolID             int    `json:"school_id"`
	StudentTypeID        int    `json:"student_type_id"`
	DocumentURL          string `json:"document_url,omitempty"`
	Date                 string `json:"date"`
}
