package models

import "database/sql"

// School представляет школу.
type School struct {
	SchoolID         int            `json:"school_id"`          // ID школы
	UserID           int            `json:"user_id"`            // ID пользователя (директора) связанного с этой школой
	SchoolName       string         `json:"school_name"`        // Название школы
	SchoolAddress    sql.NullString `json:"school_address"`     // Адрес школы (может быть NULL)
	City             string         `json:"city"`               // Город
	AboutSchool      sql.NullString `json:"about_school"`       // Описание школы (может быть NULL)
	PhotoURL         sql.NullString `json:"photo_url"`          // URL фотографии школы (может быть NULL)
	SchoolEmail      sql.NullString `json:"school_email"`       // Email школы (может быть NULL)
	SchoolPhone      sql.NullString `json:"school_phone"`       // Телефон школы (может быть NULL)
	SchoolAdminLogin sql.NullString `json:"school_admin_login"` // Логин школьного администратора (может быть NULL)
	CreatedAt        sql.NullString `json:"created_at"`         // Дата создания
	UpdatedAt        sql.NullString `json:"updated_at"`         // Дата обновления
	Specializations  []string       `json:"specializations"`
}
