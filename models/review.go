package models

// Review представляет отзыв пользователя о школе.
type Review struct {
	ID         int     `json:"id"`         // Идентификатор отзыва
	SchoolID   int     `json:"school_id"`  // Идентификатор школы
	UserID     int     `json:"user_id"`    // Идентификатор пользователя
	Rating     float64 `json:"rating"`     // Оценка от 1 до 5
	Comment    string  `json:"comment"`    // Текст отзыва
	CreatedAt  string  `json:"created_at"` // Дата и время создания отзыва
	SchoolName string  `json:"school_name"`
}