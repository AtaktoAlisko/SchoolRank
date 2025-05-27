package models

type StudentType struct {
	ID        int    `json:"id"`         
	StudentID int    `json:"student_id"` 
	ExamType  string `json:"exam_type"`  
}
