package models

type SubjectOlympiad struct {
	ID            int    `json:"id"`
	SubjectName   string `json:"subject_name"`    // Название предмета
	EventName     string `json:"event_name"`      // Название события (олимпиады)
	Date          string `json:"date"`            // Дата проведения олимпиады
	Duration      string `json:"duration"`        // Продолжительность
	Description   string `json:"description"`     // Описание события
	SchoolAdminID int    `json:"school_admin_id"` // ID школьного администратора
	PhotoURL      string `json:"photo_url"`       // URL фотографии олимпиады
	City          string `json:"city"`            // Город проведения олимпиады
}
