package models

import "database/sql"

// School представляет школу.
type School struct {
    SchoolID      int            `json:"school_id"`           // ID школы
    SchoolName    string         `json:"school_name"`         // Название школы
    SchoolAddress sql.NullString `json:"school_address"`      // Адрес школы (может быть NULL)
    City          string         `json:"city"`                // Город
    AboutSchool   sql.NullString `json:"about_school"`        // Описание школы (может быть NULL)
    PhotoURL      string         `json:"photo_url"`           // URL фотографии школы
    SchoolEmail   sql.NullString `json:"school_email"`        // Email школы (может быть NULL)
    SchoolPhone   sql.NullString `json:"phone"`               // Телефон школы (может быть NULL)
    SchoolAdminLogin string      `json:"school_admin_login"`  // Логин школьного администратора
    CreatedAt     string         `json:"created_at"`          // Дата создания
    UpdatedAt     string         `json:"updated_at"`          // Дата обновления
}
