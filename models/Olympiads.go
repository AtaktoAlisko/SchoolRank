package models

type Olympiads struct {
	OlympiadID    int    `json:"olympiad_id"`
	StudentID     int    `json:"student_id"`
	OlympiadPlace int    `json:"olympiad_place"`
	Score         int    `json:"score"`
	SchoolID      int    `json:"school_id"`
	Level         string `json:"level"` // Тип олимпиады: 'city', 'region', 'republican'
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Patronymic    string `json:"patronymic"`
}
