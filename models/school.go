package models

// School представляет школу.
type School struct {
    SchoolID      int            `json:"school_id"`           // ID школы
    SchoolName    string         `json:"school_name"`         // Название школы
    SchoolAddress string `json:"school_address"`      // Адрес школы (может быть NULL)
    City          string         `json:"city"`                // Город
    AboutSchool   string `json:"about_school"`        // Описание школы (может быть NULL)
    PhotoURL      string         `json:"photo_url"`           // URL фотографии школы
    SchoolEmail   string `json:"school_email"`        // Email школы (может быть NULL)
    SchoolPhone   string `json:"school_phone"`               // Телефон школы (может быть NULL)
    SchoolAdminLogin string      `json:"school_admin_login"`  // Логин школьного администратора
    CreatedAt     string         `json:"created_at"`          // Дата создания
    UpdatedAt     string         `json:"updated_at"`          // Дата обновления
    Specializations  []string       `json:"specializations"`      // Специализации школы (напр. физ-мат, гео-мат)
    Achievements     []string       `json:"achievements"`         // Достижения школы
}
