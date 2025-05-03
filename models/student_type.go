package models

type StudentType struct {
	ID        int    `json:"id"`         // Идентификатор записи
	StudentID int    `json:"student_id"` // ID студента
	ExamType  string `json:"exam_type"`  // Тип экзамена: "regular" или "creative"
}
