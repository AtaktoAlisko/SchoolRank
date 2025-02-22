package models

type UNTScore struct {
	UNTScoreID int `json:"unt_score_id"`
	Year       int `json:"year"`
	UNTTypeID  int `json:"unt_type_id"`
	StudentID  int `json:"student_id"`
	Score      int `json:"score"`
}
