package models

// UNTType - структура для хранения данных о типах экзаменов
type UNTType struct {
	UNTTypeID      int `json:"unt_type_id"`      // ID записи
	FirstTypeID    int `json:"first_type_id"`    // ID для First_Type
	SecondTypeID   int `json:"second_type_id"`   // ID для Second_Type
	HistoryKazakh  int `json:"history_of_kazakhstan"` // Оценка по истории Казахстана
	MathLiteracy   int `json:"mathematical_literacy"` // Оценка по математической грамотности
	ReadingLiteracy int `json:"reading_literacy"`  // Оценка по грамотности
}
