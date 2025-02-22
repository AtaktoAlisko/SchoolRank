package models

type Student struct {
	StudentID  int    `json:"student_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Patronymic string `json:"patronymic,omitempty"`
	IIN        string `json:"iin"`
}
