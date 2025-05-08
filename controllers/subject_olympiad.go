package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type SubjectOlympiadController struct{}

func (c *SubjectOlympiadController) CreateSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Проверка токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
			return
		}

		// Шаг 2: Получение роли пользователя
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Шаг 3: Проверка прав доступа
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create an olympiad"})
			return
		}

		// Шаг 4: Получение school_id из URL
		var schoolID int
		schoolIDStr := r.URL.Path[len("/api/subject-olympiads/create/"):]
		schoolID, err = strconv.Atoi(schoolIDStr)
		if err != nil || schoolID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid or missing school_id in URL"})
			return
		}

		// Шаг 5: Парсинг формы
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error parsing form data"})
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		// Шаг 6: Парсинг остальных полей
		subjectName := r.FormValue("subject_name")
		startDate := r.FormValue("date")
		endDate := r.FormValue("end_date")
		description := r.FormValue("description")
		level := r.FormValue("level")
		limitStr := r.FormValue("limit")

		if subjectName == "" || startDate == "" || endDate == "" || description == "" || level == "" || limitStr == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "subject_name, start date, end date, description, level, and limit are required fields."})
			return
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid participant limit format or value"})
			return
		}

		// Шаг 7: Загрузка фото
		file, _, err := r.FormFile("photo_url")
		if err != nil {
			log.Println("Error reading file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
			return
		}
		defer file.Close()

		uniqueFileName := fmt.Sprintf("olympiad-%d.jpg", time.Now().Unix())
		photoURL, err := utils.UploadFileToS3(file, uniqueFileName, "schoolphoto")
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload photo"})
			log.Println("Error uploading file:", err)
			return
		}

		olympiad := models.SubjectOlympiad{
			SubjectName: subjectName,
			StartDate:   startDate,
			EndDate:     endDate,
			Description: description,
			PhotoURL:    photoURL,
			SchoolID:    schoolID,
			Level:       level,
			Limit:       limit,
		}

		// Шаг 8: Вставка в БД
		query := `INSERT INTO subject_olympiads 
			(subject_name, date, end_date, description, photo_url, school_id, level, limit_participants) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = db.Exec(query,
			olympiad.SubjectName,
			olympiad.StartDate,
			olympiad.EndDate,
			olympiad.Description,
			olympiad.PhotoURL,
			olympiad.SchoolID,
			olympiad.Level,
			olympiad.Limit)

		if err != nil {
			log.Println("Error inserting olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create olympiad"})
			return
		}

		utils.ResponseJSON(w, olympiad)
	}
}
func (c *SubjectOlympiadController) GetSubjectOlympiads(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Получение school_id из URL
		schoolIDStr := mux.Vars(r)["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil || schoolID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid or missing school_id in URL"})
			return
		}

		// Шаг 2: Извлечение параметров фильтрации из строки запроса
		subjectName := r.URL.Query().Get("subject_name")
		startDate := r.URL.Query().Get("start_date")
		endDate := r.URL.Query().Get("end_date")
		level := r.URL.Query().Get("level")

		// Шаг 3: Строим SQL-запрос с учетом фильтров
		query := `SELECT subject_name, date, end_date, description, photo_url, school_id, level, limit_participants
				  FROM subject_olympiads 
				  WHERE school_id = ?`

		var args []interface{}
		args = append(args, schoolID)

		// Добавляем фильтры в запрос, если они были переданы
		if subjectName != "" {
			query += " AND subject_name LIKE ?"
			args = append(args, "%"+subjectName+"%")
		}
		if startDate != "" {
			query += " AND date >= ?"
			args = append(args, startDate)
		}
		if endDate != "" {
			query += " AND end_date <= ?"
			args = append(args, endDate)
		}
		if level != "" {
			query += " AND level = ?"
			args = append(args, level)
		}

		// Шаг 4: Выполнение запроса
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println("Error querying olympiads:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiads"})
			return
		}
		defer rows.Close()

		// Шаг 5: Чтение результатов
		var olympiads []models.SubjectOlympiad
		for rows.Next() {
			var olympiad models.SubjectOlympiad
			err := rows.Scan(&olympiad.SubjectName, &olympiad.StartDate, &olympiad.EndDate, &olympiad.Description, &olympiad.PhotoURL, &olympiad.SchoolID, &olympiad.Level, &olympiad.Limit)
			if err != nil {
				log.Println("Error scanning row:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to process olympiad data"})
				return
			}
			olympiads = append(olympiads, olympiad)
		}

		// Шаг 6: Ответ в формате JSON
		if len(olympiads) == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No olympiads found"})
			return
		}

		utils.ResponseJSON(w, olympiads)
	}
}
func (c *SubjectOlympiadController) UpdateSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Проверка токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
			return
		}

		// Шаг 2: Получение роли пользователя
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Шаг 3: Проверка прав доступа
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to update an olympiad"})
			return
		}

		// Шаг 4: Получение subject_olympiad_id из URL
		var olympiadID int
		vars := mux.Vars(r)
		olympiadID, err = strconv.Atoi(vars["id"])
		if err != nil || olympiadID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid or missing olympiad_id in URL"})
			return
		}

		// Шаг 5: Парсинг формы
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error parsing form data"})
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		// Шаг 6: Парсинг остальных полей
		subjectName := r.FormValue("subject_name")
		startDate := r.FormValue("date")
		endDate := r.FormValue("end_date")
		description := r.FormValue("description")
		level := r.FormValue("level")
		limitStr := r.FormValue("limit")

		if subjectName == "" || startDate == "" || endDate == "" || description == "" || level == "" || limitStr == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "subject_name, start date, end date, description, level, and limit are required fields."})
			return
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid participant limit format or value"})
			return
		}

		// Шаг 7: Загрузка фото (опционально)
		var photoURL string
		file, _, err := r.FormFile("photo_url")
		if err == nil {
			defer file.Close()
			uniqueFileName := fmt.Sprintf("olympiad-%d.jpg", time.Now().Unix())
			photoURL, err = utils.UploadFileToS3(file, uniqueFileName, "schoolphoto")
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload photo"})
				log.Println("Error uploading file:", err)
				return
			}
		} else if err != http.ErrMissingFile {
			log.Println("Error reading file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
			return
		}

		// Шаг 8: Обновление олимпиады в БД
		query := `UPDATE subject_olympiads
				  SET subject_name = ?, date = ?, end_date = ?, description = ?, level = ?, limit_participants = ?`
		var args []interface{}
		args = append(args, subjectName, startDate, endDate, description, level, limit)

		// Если файл был загружен, обновляем photo_url
		if photoURL != "" {
			query += ", photo_url = ?"
			args = append(args, photoURL)
		}

		query += " WHERE id = ?"
		args = append(args, olympiadID)

		_, err = db.Exec(query, args...)
		if err != nil {
			log.Println("Error updating olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update olympiad"})
			return
		}

		// Шаг 9: Ответ в формате JSON
		utils.ResponseJSON(w, models.SubjectOlympiad{
			SubjectName: subjectName,
			StartDate:   startDate,
			EndDate:     endDate,
			Description: description,
			PhotoURL:    photoURL,
			SchoolID:    0, // Можно оставить или обновить SchoolID по необходимости
			Level:       level,
			Limit:       limit,
		})
	}
}
func (c *SubjectOlympiadController) DeleteSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Проверка токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
			return
		}

		// Шаг 2: Получение роли пользователя
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Шаг 3: Проверка прав доступа
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to delete an olympiad"})
			return
		}

		// Шаг 4: Получение olympiad_id из URL
		vars := mux.Vars(r)
		olympiadIDStr := vars["id"]
		olympiadID, err := strconv.Atoi(olympiadIDStr)
		if err != nil || olympiadID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID"})
			return
		}

		// Шаг 5: Удаление олимпиады из БД
		query := "DELETE FROM subject_olympiads WHERE id = ?"
		result, err := db.Exec(query, olympiadID)
		if err != nil {
			log.Println("Error deleting olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete olympiad"})
			return
		}

		// Шаг 6: Проверка, был ли удален хотя бы один ряд
		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad not found"})
			return
		}

		// Шаг 7: Ответ на успешное удаление
		utils.ResponseJSON(w, map[string]string{"message": "Olympiad successfully deleted"})
	}
}
