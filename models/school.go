package models

type School struct {
    SchoolID    int    `json:"school_id"`
    Name        string `json:"name"`         // Название школы
    Address     string `json:"address"`      // Адрес школы
    Title       string `json:"title"`        // Заголовок школы
    Description string `json:"description"`  // Описание школы
    Contacts    string `json:"contacts"`     // Контактная информация
    PhotoURL    string `json:"photo_url"`    // URL фотографии школы
}
