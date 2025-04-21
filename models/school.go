package models

type School struct {
    SchoolID      int    `json:"school_id"`      // Идентификатор школы
    Name          string `json:"name"`           // Название школы
    Address       string `json:"address"`        // Адрес школы
    City          string `json:"city"`           // Город
    Title         string `json:"title"`          // Заголовок школы
    Description   string `json:"description"`    // Описание школы
    PhotoURL      string `json:"photo_url"`      // URL фотографии школы
    Email         string `json:"email"`          // Электронная почта школы
    Phone         string `json:"phone"`          // Телефон школы
    DirectorEmail string `json:"director_email"` // Электронная почта директора
    CreatedAt     string `json:"created_at"`     // Дата создания записи
    UpdatedAt     string `json:"updated_at"`     // Дата последнего обновления записи
}
