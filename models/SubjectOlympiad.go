package models

type SubjectOlympiad struct {
	ID           int    `json:"id"`
	OlympiadName string `json:"subject_name"`  // Название предмета
	StartDate    string `json:"date"`          // Дата проведения олимпиады
	Description  string `json:"description"`   // Описание события
	PhotoURL     string `json:"photo_url"`     // URL фотографии олимпиады
	City         string `json:"city"`          // Город проведения олимпиады
	SchoolID     int    `json:"school_id"`     // ID школы
	EndDate      string `json:"end_date"`      // Дата окончания олимпиады
	Level        string `json:"level"`         // Уровень олимпиады
	OlympiadType string `json:"olympiad_name"` // Название олимпиады
	Limit        int    `json:"limit"`         // Лимит участников
}
