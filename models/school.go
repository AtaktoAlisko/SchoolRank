package models

import "database/sql"

type School struct {
    SchoolID    int            `json:"school_id"`
    Name        string         `json:"name"`         // Название школы
    Address     string         `json:"address"`      // Адрес школы
    Title       string         `json:"title"`        // Заголовок школы
    Description string         `json:"description"`  // Описание школы
    PhotoURL    string         `json:"photo_url"`    // URL фотографии школы
    Email       sql.NullString `json:"email"`        // Используем sql.NullString для nullable поля
    Phone       sql.NullString `json:"phone"`
}
