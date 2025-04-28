package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type SubjectOlympiadController struct{}

// Метод для создания олимпиады по предмету
func (c *SubjectOlympiadController) CreateOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var olympiad models.SubjectOlympiad
		var error models.Error

		// Декодируем запрос
		err := json.NewDecoder(r.Body).Decode(&olympiad)
		if err != nil {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверка обязательных полей
		if olympiad.SubjectName == "" || olympiad.EventName == "" || olympiad.Date == "" || olympiad.City == "" {
			error.Message = "Subject name, event name, date, and city are required fields."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Вставка новой олимпиады в базу данных
		query := `INSERT INTO subject_olympiads (subject_name, event_name, date, duration, description, photo_url, city, school_admin_id) 
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = db.Exec(query, olympiad.SubjectName, olympiad.EventName, olympiad.Date, olympiad.Duration, olympiad.Description, olympiad.PhotoURL, olympiad.City, olympiad.SchoolAdminID)
		if err != nil {
			log.Println("Error inserting olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create olympiad"})
			return
		}

		utils.ResponseJSON(w, olympiad)
	}
}

// Метод для регистрации студента на олимпиаду
func (c *SubjectOlympiadController) RegisterStudentToOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var registrationData struct {
			StudentID  int `json:"student_id"`  // ID студента
			OlympiadID int `json:"olympiad_id"` // ID олимпиады
		}

		var error models.Error

		// Декодируем запрос
		err := json.NewDecoder(r.Body).Decode(&registrationData)
		if err != nil {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверка, что студент существует
		var studentExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND role = 'student')", registrationData.StudentID).Scan(&studentExists)
		if err != nil || !studentExists {
			error.Message = "Student not found or incorrect role."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверка, что олимпиада существует
		var olympiadExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM subject_olympiads WHERE id = ?)", registrationData.OlympiadID).Scan(&olympiadExists)
		if err != nil || !olympiadExists {
			error.Message = "Olympiad not found."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверка, что студент уже зарегистрирован на олимпиаду
		var alreadyRegistered bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student_olympiads WHERE student_id = ? AND olympiad_id = ?)", registrationData.StudentID, registrationData.OlympiadID).Scan(&alreadyRegistered)
		if err != nil || alreadyRegistered {
			error.Message = "Student is already registered for this olympiad."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Регистрируем студента на олимпиаду
		query := `INSERT INTO student_olympiads (student_id, olympiad_id) VALUES (?, ?)`
		_, err = db.Exec(query, registrationData.StudentID, registrationData.OlympiadID)
		if err != nil {
			log.Println("Error registering student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to register student"})
			return
		}

		// Ответ с подтверждением
		utils.ResponseJSON(w, map[string]string{"message": "Student successfully registered for the olympiad."})
	}
}

// Метод для получения всех олимпиад для студентов
func (c *SubjectOlympiadController) GetAllOlympiadsForStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка токена и роль студента
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Запрос для получения всех олимпиад
		rows, err := db.Query("SELECT id, subject_name, event_name, date, duration, description, photo_url, city FROM subject_olympiads")
		if err != nil {
			log.Println("Error fetching olympiads:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiads"})
			return
		}
		defer rows.Close()

		var olympiads []models.SubjectOlympiad

		// Проходим по всем строкам в результатах запроса
		for rows.Next() {
			var olympiad models.SubjectOlympiad
			err := rows.Scan(&olympiad.ID, &olympiad.SubjectName, &olympiad.EventName, &olympiad.Date, &olympiad.Duration, &olympiad.Description, &olympiad.PhotoURL, &olympiad.City)
			if err != nil {
				log.Println("Error scanning olympiad data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning olympiad data"})
				return
			}

			olympiads = append(olympiads, olympiad)
		}

		// Проверка на ошибки после итерации
		if err = rows.Err(); err != nil {
			log.Println("Error during iteration:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error during iteration"})
			return
		}

		// Возвращаем список всех олимпиад для студентов в формате JSON
		utils.ResponseJSON(w, olympiads)
	}
}
