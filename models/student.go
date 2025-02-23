package models

type Student struct {
	ID         int    `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Patronymic string `json:"patronymic"`
	IIN        string `json:"iin"`
	SchoolID   int    `json:"school_id"`
}
