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
		limitStr := r.FormValue("limit_participants")

		if subjectName == "" || startDate == "" || endDate == "" || description == "" || level == "" || limitStr == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "subject_name, start date, end date, description, level, and limit are required fields."})
			return
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid participant limit format or value"})
			return
		}

		// Шаг 7: Вставка в БД и получение ID
		query := `INSERT INTO subject_olympiads 
			(subject_name, date, end_date, description, school_id, level, limit_participants, creator_id) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := db.Exec(query,
			subjectName,
			startDate,
			endDate,
			description,
			schoolID,
			level,
			limit,
			userID)

		if err != nil {
			log.Println("Error inserting olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create olympiad"})
			return
		}

		// Получение ID созданной олимпиады
		olympiadID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error getting last insert ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiad ID"})
			return
		}

		// Получаем полные данные олимпиады из БД для корректного ответа
		var olympiad models.SubjectOlympiad

		query = `SELECT so.id, so.subject_name, so.date, so.end_date, so.description, 
			so.school_id, so.level, so.limit_participants, 
			u.id as creator_id, u.first_name, u.last_name, s.name as school_name
			FROM subject_olympiads so
			LEFT JOIN users u ON so.creator_id = u.id
			LEFT JOIN Schools s ON so.school_id = s.id
			WHERE so.id = ?`

		err = db.QueryRow(query, olympiadID).Scan(
			&olympiad.ID,
			&olympiad.SubjectName,
			&olympiad.StartDate,
			&olympiad.EndDate,
			&olympiad.Description,
			&olympiad.SchoolID,
			&olympiad.Level,
			&olympiad.Limit,
			&olympiad.CreatorID,
			&olympiad.CreatorFirstName,
			&olympiad.CreatorLastName,
			&olympiad.SchoolName,
		)

		if err != nil {
			log.Println("Error fetching created olympiad:", err)
			// Если не удалось получить полные данные, вернем базовую информацию
			utils.ResponseJSON(w, models.SubjectOlympiad{
				ID:          int(olympiadID),
				SubjectName: subjectName,
				StartDate:   startDate,
				EndDate:     endDate,
				Description: description,
				SchoolID:    schoolID,
				Level:       level,
				Limit:       limit,
				CreatorID:   userID,
			})
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
			log.Printf("⚠️ Invalid or missing school_id in URL: '%s'. Using default value 1", schoolIDStr)
			schoolID = 1 // значение по умолчанию
		}

		// Шаг 2: Извлечение параметров фильтрации из строки запроса
		subjectName := r.URL.Query().Get("subject_name")
		startDate := r.URL.Query().Get("start_date")
		endDate := r.URL.Query().Get("end_date")
		level := r.URL.Query().Get("level")

		// Шаг 3: Строим SQL-запрос с учетом фильтров
		query := `SELECT subject_olympiad_id, subject_name, date, end_date, description, school_id, level, limit_participants
                  FROM subject_olympiads 
                  WHERE school_id = ?`

		var args []interface{}
		args = append(args, schoolID)

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
		log.Printf("Executing query: %s with args: %v", query, args)
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
			err := rows.Scan(
				&olympiad.ID,
				&olympiad.SubjectName,
				&olympiad.StartDate,
				&olympiad.EndDate,
				&olympiad.Description,
				&olympiad.SchoolID,
				&olympiad.Level,
				&olympiad.Limit,
			)
			if err != nil {
				log.Println("Error scanning row:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to process olympiad data"})
				return
			}
			olympiads = append(olympiads, olympiad)
		}

		// Шаг 6: Ответ в формате JSON
		log.Printf("Found %d olympiads", len(olympiads))
		if len(olympiads) == 0 {
			var count int
			countQuery := "SELECT COUNT(*) FROM subject_olympiads"
			err := db.QueryRow(countQuery).Scan(&count)
			if err != nil {
				log.Printf("Error checking table data: %v", err)
			} else {
				log.Printf("Total records in subject_olympiads: %d", count)

				// Get list of available school_ids
				schoolIdsQuery := "SELECT DISTINCT school_id FROM subject_olympiads"
				rows, err := db.Query(schoolIdsQuery)
				if err != nil {
					log.Printf("Error getting school IDs: %v", err)
				} else {
					defer rows.Close()
					var schoolIds []int
					for rows.Next() {
						var id int
						rows.Scan(&id)
						schoolIds = append(schoolIds, id)
					}
					log.Printf("Available school_ids in database: %v", schoolIds)
				}
			}

			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No olympiads found for school_id " + schoolIDStr})
			return
		}

		utils.ResponseJSON(w, olympiads)
	}
}
func (c *SubjectOlympiadController) EditOlympiadsCreated(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
			return
		}

		// Step 2: Fetch user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Printf("Error fetching user role for userID %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to edit olympiads"})
			return
		}

		// Step 4: Get olympiad_id from URL
		vars := mux.Vars(r)
		olympiadID, err := strconv.Atoi(vars["id"])
		if err != nil || olympiadID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID"})
			return
		}

		// Step 5: Verify olympiad ownership for schooladmin
		if userRole == "schooladmin" {
			var count int
			err = db.QueryRow(`
				SELECT COUNT(*) 
				FROM subject_olympiads so
				JOIN Schools s ON so.school_id = s.school_id
				WHERE so.subject_olympiad_id = ? AND s.user_id = ?
			`, olympiadID, userID).Scan(&count)

			if err != nil {
				log.Printf("Error verifying olympiad ownership for olympiadID %d, userID %d: %v", olympiadID, userID, err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to verify olympiad ownership"})
				return
			}

			if count == 0 {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only edit olympiads for your school"})
				return
			}
		}

		// Step 6: Parse form data
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			log.Printf("Error parsing multipart form: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error parsing form data"})
			return
		}

		// Log all form field names for debugging
		log.Printf("Received form fields:")
		for key, values := range r.Form {
			log.Printf("Field: %s, value: %v", key, values)
		}

		// Step 7: Fetch current olympiad data
		var currentOlympiad models.SubjectOlympiad
		err = db.QueryRow(`
			SELECT 
				subject_name, date, end_date, description, school_id, level, limit_participants
			FROM 
				subject_olympiads
			WHERE 
				subject_olympiad_id = ?
		`, olympiadID).Scan(
			&currentOlympiad.SubjectName,
			&currentOlympiad.StartDate,
			&currentOlympiad.EndDate,
			&currentOlympiad.Description,
			&currentOlympiad.SchoolID,
			&currentOlympiad.Level,
			&currentOlympiad.Limit,
		)
		if err != nil {
			log.Printf("Error fetching current olympiad data for olympiadID %d: %v", olympiadID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch current olympiad data"})
			return
		}

		// Step 8: Get form data (use current data as defaults)
		// Try both possible field names for subject name
		subjectName := r.FormValue("subject_name")
		if subjectName == "" {
			subjectName = r.FormValue("olympiad_name")
			if subjectName == "" {
				subjectName = currentOlympiad.SubjectName
			}
		}

		// Try both possible field names for start date
		startDate := r.FormValue("start_date")
		if startDate == "" {
			startDate = r.FormValue("date")
			if startDate == "" {
				startDate = currentOlympiad.StartDate
			}
		}

		endDate := r.FormValue("end_date")
		if endDate == "" {
			endDate = currentOlympiad.EndDate
		}

		description := r.FormValue("description")
		if description == "" {
			description = currentOlympiad.Description
		}

		level := r.FormValue("level")
		if level == "" {
			level = currentOlympiad.Level
		}

		var limit int
		limitStr := r.FormValue("limit_participants")
		if limitStr == "" {
			limitStr = r.FormValue("limit") // Try alternative field name
			if limitStr == "" {
				limit = currentOlympiad.Limit
			} else {
				limit, err = strconv.Atoi(limitStr)
				if err != nil || limit <= 0 {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid participant limit format or value"})
					return
				}
			}
		} else {
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid participant limit format or value"})
				return
			}
		}

		// Handle school_id (allow superadmin to change, restrict schooladmin)
		var schoolID int = currentOlympiad.SchoolID
		schoolIDStr := r.FormValue("school_id")
		if schoolIDStr != "" {
			newSchoolID, err := strconv.Atoi(schoolIDStr)
			if err != nil || newSchoolID <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
				return
			}

			// Validate school_id exists
			var count int
			err = db.QueryRow("SELECT COUNT(*) FROM Schools WHERE school_id = ?", newSchoolID).Scan(&count)
			if err != nil || count == 0 {
				log.Printf("Invalid school_id %d: %v", newSchoolID, err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School does not exist"})
				return
			}

			if userRole == "schooladmin" {
				// Ensure schooladmin can only set school_id to their own school
				err = db.QueryRow(`
					SELECT COUNT(*) 
					FROM Schools 
					WHERE school_id = ? AND user_id = ?
				`, newSchoolID, userID).Scan(&count)
				if err != nil || count == 0 {
					log.Printf("Schooladmin %d attempted to set invalid school_id %d", userID, newSchoolID)
					utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only set your own school"})
					return
				}
			}

			schoolID = newSchoolID
		}

		// Log updated form data for debugging
		log.Printf("Processed form data - olympiadID: %d, subject_name: %s, start_date: %s, end_date: %s, school_id: %d, level: %s, limit: %d",
			olympiadID, subjectName, startDate, endDate, schoolID, level, limit)

		// Step 9: Update olympiad (including school_id and creator_id)
		_, err = db.Exec(`
			UPDATE subject_olympiads
			SET subject_name = ?, date = ?, end_date = ?, description = ?, 
				school_id = ?, level = ?, limit_participants = ?, creator_id = ?
			WHERE subject_olympiad_id = ?
		`, subjectName, startDate, endDate, description, schoolID, level, limit, userID, olympiadID)

		if err != nil {
			log.Printf("Error updating olympiad for olympiadID %d: %v", olympiadID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update olympiad"})
			return
		}

		// Step 10: Fetch updated olympiad data
		var updatedOlympiad models.SubjectOlympiad
		err = db.QueryRow(`
			SELECT 
				so.subject_olympiad_id, 
				so.subject_name, 
				so.date, 
				so.end_date, 
				so.description, 
				so.school_id, 
				so.level, 
				so.limit_participants,
				COALESCE(so.creator_id, 0) as creator_id,
				COALESCE(u.first_name, '') as creator_first_name,
				COALESCE(u.last_name, '') as creator_last_name,
				COALESCE(s.name, '') as school_name
			FROM 
				subject_olympiads so
			LEFT JOIN 
				Schools s ON so.school_id = s.school_id
			LEFT JOIN 
				users u ON so.creator_id = u.id
			WHERE 
				so.subject_olympiad_id = ?
		`, olympiadID).Scan(
			&updatedOlympiad.ID,
			&updatedOlympiad.SubjectName,
			&updatedOlympiad.StartDate,
			&updatedOlympiad.EndDate,
			&updatedOlympiad.Description,
			&updatedOlympiad.SchoolID,
			&updatedOlympiad.Level,
			&updatedOlympiad.Limit,
			&updatedOlympiad.CreatorID,
			&updatedOlympiad.CreatorFirstName,
			&updatedOlympiad.CreatorLastName,
			&updatedOlympiad.SchoolName,
		)

		if err != nil {
			log.Printf("Error fetching updated olympiad for olympiadID %d: %v", olympiadID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Olympiad updated but failed to fetch updated data"})
			return
		}

		utils.ResponseJSON(w, updatedOlympiad)
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
		query := "DELETE FROM subject_olympiads WHERE subject_olympiad_id = ?"
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
func (c *SubjectOlympiadController) GetAllSubjectOlympiads(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		subject := r.URL.Query().Get("subject")
		query := `
			SELECT subject_olympiad_id, subject_name, date, end_date, description
			FROM subject_olympiads
		`
		var rows *sql.Rows
		if subject != "" {
			query += " WHERE subject_name = ?"
			rows, err = db.Query(query, subject)
		} else {
			rows, err = db.Query(query)
		}
		if err != nil {
			log.Println("Query error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Query failed"})
			return
		}
		defer rows.Close()

		var olympiads []models.SubjectOlympiad
		for rows.Next() {
			var o models.SubjectOlympiad
			if err := rows.Scan(&o.ID, &o.SubjectName, &o.StartDate, &o.EndDate, &o.Description); err != nil {
				continue
			}
			olympiads = append(olympiads, o)
		}

		utils.ResponseJSON(w, olympiads)
	}
}
func (c *SubjectOlympiadController) GetOlympiadsBySubjectID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify token
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Extract subject_olympiad_id from URL parameters
		vars := mux.Vars(r)
		subjectOlympiadID, ok := vars["subject_olympiad_id"]
		if !ok || subjectOlympiadID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing subject_olympiad_id"})
			return
		}

		// Query to fetch olympiads by subject_olympiad_id
		query := `
            SELECT subject_olympiad_id, subject_name, date, end_date, description
            FROM subject_olympiads
            WHERE subject_olympiad_id = ?
        `
		rows, err := db.Query(query, subjectOlympiadID)
		if err != nil {
			log.Println("Query error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Query failed"})
			return
		}
		defer rows.Close()

		// Collect olympiads
		var olympiads []models.SubjectOlympiad
		for rows.Next() {
			var o models.SubjectOlympiad
			if err := rows.Scan(&o.ID, &o.SubjectName, &o.StartDate, &o.EndDate, &o.Description); err != nil {
				log.Println("Scan error:", err)
				continue
			}
			olympiads = append(olympiads, o)
		}

		// Check for errors from iterating over rows
		if err = rows.Err(); err != nil {
			log.Println("Row iteration error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing results"})
			return
		}

		// Respond with the results
		utils.ResponseJSON(w, olympiads)
	}
}
func (c *SubjectOlympiadController) GetSubjectOlympiadsByNamePhoto(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		subject := r.URL.Query().Get("subject")
		query := `
			SELECT subject_olympiad_id, subject_name, photo_url
			FROM subject_olympiads
		`

		var rows *sql.Rows
		if subject != "" {
			query += " WHERE subject_name = ?"
			rows, err = db.Query(query, subject)
		} else {
			rows, err = db.Query(query)
		}
		if err != nil {
			log.Println("Query error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Query failed"})
			return
		}
		defer rows.Close()

		var olympiads []map[string]interface{}
		for rows.Next() {
			var id int
			var name, photo string

			if err := rows.Scan(&id, &name, &photo); err != nil {
				continue
			}

			olympiad := map[string]interface{}{
				"subject_olympiad_id": id,
				"subject_name":        name,
				"photo_url":           photo,
			}
			olympiads = append(olympiads, olympiad)
		}

		utils.ResponseJSON(w, olympiads)
	}
}
func parseOlympiadsRows(rows *sql.Rows) []models.SubjectOlympiad {
	var olympiads []models.SubjectOlympiad

	for rows.Next() {
		var olympiad models.SubjectOlympiad
		err := rows.Scan(
			&olympiad.ID,
			&olympiad.SubjectName,
			&olympiad.StartDate,
			&olympiad.EndDate,
			&olympiad.Description,
			&olympiad.SchoolID,
			&olympiad.Level,
			&olympiad.Limit,
			&olympiad.CreatorID,
			&olympiad.CreatorFirstName,
			&olympiad.CreatorLastName,
			&olympiad.SchoolName,
		)
		if err != nil {
			log.Println("Error scanning olympiad row:", err)
			continue
		}
		olympiads = append(olympiads, olympiad)
	}

	return olympiads
}
func (c *SubjectOlympiadController) GetOlympiadsBySubjectName(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify token first
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Get subject name from query parameter
		subjectName := r.URL.Query().Get("subject_name")
		if subjectName == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "subject_name parameter is required"})
			return
		}

		// Query to get olympiads with school information and participant count
		query := `
			SELECT 
				so.subject_olympiad_id,
				so.date as start_date,
				so.end_date,
				so.description as location,
				so.limit_participants,
				so.level,
				s.school_id,
				s.school_name,
				COUNT(reg.olympiads_registrations_id) as current_participants
			FROM 
				subject_olympiads so
			LEFT JOIN 
				Schools s ON so.school_id = s.school_id
			LEFT JOIN 
				olympiad_registrations reg ON so.subject_olympiad_id = reg.subject_olympiad_id 
				AND reg.status = 'registered'
			WHERE 
				so.subject_name = ?
			GROUP BY 
				so.subject_olympiad_id, so.date, so.end_date, so.description, so.limit_participants, so.level, s.school_id, s.school_name
			ORDER BY 
				s.school_name, so.subject_olympiad_id
		`

		// Execute the query
		log.Printf("Executing query: %s with subject_name: %s", query, subjectName)
		rows, err := db.Query(query, subjectName)
		if err != nil {
			log.Println("Error querying olympiads:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiads"})
			return
		}
		defer rows.Close()

		// Define olympiad structure with participant count
		type OlympiadData struct {
			ID              int    `json:"subject_olympiad_id"`
			StartDate       string `json:"start_date"`
			EndDate         string `json:"end_date,omitempty"` // Only included if not expired
			Location        string `json:"location"`
			Limit           int    `json:"limit_participants"`
			Level           string `json:"level"`
			Expired         bool   `json:"expired"`
			ParticipantInfo string `json:"participants"` // Format: "current/limit"
		}

		// Define school with olympiads structure
		type SchoolWithOlympiads struct {
			SchoolID   int            `json:"school_id"`
			SchoolName string         `json:"school_name"`
			Olympiads  []OlympiadData `json:"olympiads"`
		}

		// Read results and group by school
		schoolMap := make(map[int]*SchoolWithOlympiads)
		currentTime := time.Now()

		for rows.Next() {
			var olympiad OlympiadData
			var schoolID int
			var schoolName sql.NullString
			var endDateStr sql.NullString
			var currentParticipants int

			// Scan the row - FIXED: Changed ¤tParticipants to &currentParticipants
			err := rows.Scan(
				&olympiad.ID,
				&olympiad.StartDate,
				&endDateStr,
				&olympiad.Location,
				&olympiad.Limit,
				&olympiad.Level,
				&schoolID,
				&schoolName,
				&currentParticipants, // FIXED: This was ¤tParticipants
			)
			if err != nil {
				log.Println("Error scanning row:", err)
				continue
			}

			// Set EndDate only if endDateStr is valid
			if endDateStr.Valid {
				olympiad.EndDate = endDateStr.String
			} else {
				// Default to start date + 1 day if end_date is NULL
				start, _ := time.Parse("2006-01-02", olympiad.StartDate)
				olympiad.EndDate = start.AddDate(0, 0, 1).Format("2006-01-02")
			}

			// Format participant info as "current/limit"
			olympiad.ParticipantInfo = fmt.Sprintf("%d/%d", currentParticipants, olympiad.Limit)

			// Parse end date to check if it's expired
			endDate, err := time.Parse("2006-01-02", olympiad.EndDate)
			if err != nil {
				log.Printf("Error parsing end date '%s': %v", olympiad.EndDate, err)
				olympiad.Expired = false
			} else {
				olympiad.Expired = currentTime.After(endDate)
			}

			// Get the school name string value
			schoolNameStr := "Unknown School"
			if schoolName.Valid {
				schoolNameStr = schoolName.String
			}

			// Add to school map
			if _, exists := schoolMap[schoolID]; !exists {
				schoolMap[schoolID] = &SchoolWithOlympiads{
					SchoolID:   schoolID,
					SchoolName: schoolNameStr,
					Olympiads:  []OlympiadData{},
				}
			}

			// Add olympiad to the school's list
			schoolMap[schoolID].Olympiads = append(schoolMap[schoolID].Olympiads, olympiad)
		}

		// Check for errors from iterating over rows
		if err := rows.Err(); err != nil {
			log.Println("Error iterating over rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing olympiad data"})
			return
		}

		// Convert map to slice for response
		var response []SchoolWithOlympiads
		for _, school := range schoolMap {
			response = append(response, *school)
		}

		// Return appropriate response
		if len(response) == 0 {
			log.Printf("No olympiads found for subject_name %s", subjectName)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No olympiads found for the given subject"})
			return
		}

		log.Printf("Found olympiads from %d Schools for subject %s", len(response), subjectName)
		utils.ResponseJSON(w, response)
	}
}
