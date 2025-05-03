package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"ranking-school/models" 
	"ranking-school/utils"  
)

// CreateStudentType — создание записи типа экзамена для студента
func CreateStudentType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var studentType models.StudentType

		// Декодируем данные из тела запроса
		if err := json.NewDecoder(r.Body).Decode(&studentType); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request data"})
			return
		}

		// Проверка на обязательные параметры
		if studentType.StudentID == 0 || studentType.ExamType == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "StudentID and ExamType are required"})
			return
		}

		// Вставка данных в таблицу student_types
		query := `
			INSERT INTO student_types (student_id, exam_type)
			VALUES (?, ?);
		`

		result, err := db.Exec(query, studentType.StudentID, studentType.ExamType)
		if err != nil {
			log.Printf("Error saving student type: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to save exam type"})
			return
		}

		// Получаем ID только что вставленной записи
		studentTypeID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Error getting last insert ID: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve last insert ID"})
			return
		}

		// Отправляем успешный ответ
		response := map[string]interface{}{
			"message":         "Student type created successfully",
			"student_type_id": studentTypeID,
		}
		utils.ResponseJSON(w, response)
	}
}
