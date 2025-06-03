package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
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
			u.id as creator_id, u.first_name, u.last_name, s.school_name as school_name
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
func (c *SubjectOlympiadController) GetSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Println("Ошибка проверки токена:", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Неверный токен"})
			return
		}
		log.Printf("Проверенный userID: %d", userID)

		vars := mux.Vars(r)
		olympiadIDStr := vars["olympiad_id"]
		olympiadID, err := strconv.Atoi(olympiadIDStr)
		if err != nil || olympiadID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Неверный ID олимпиады"})
			return
		}

		var olympiad models.SubjectOlympiad
		var firstName, lastName, schoolName sql.NullString

		query := `
			SELECT 
				so.subject_olympiad_id,
				so.subject_name,
				so.date,
				so.end_date,
				COALESCE(so.description, '') as description,
				so.school_id,
				so.level,
				so.limit_participants,
				COALESCE(so.creator_id, 0),
				u.first_name,
				u.last_name,
				s.school_name,
				CASE WHEN so.end_date < CURRENT_DATE THEN true ELSE false END AS expired,
				(
					SELECT COUNT(*) FROM olympiad_registrations reg
					WHERE reg.subject_olympiad_id = so.subject_olympiad_id AND reg.status = 'accepted'
				) AS participants
			FROM subject_olympiads so
			LEFT JOIN users u ON so.creator_id = u.id
			LEFT JOIN Schools s ON so.school_id = s.school_id
			WHERE so.subject_olympiad_id = ?
		`

		err = db.QueryRow(query, olympiadID).Scan(
			&olympiad.ID,
			&olympiad.SubjectName,
			&olympiad.StartDate,
			&olympiad.EndDate,
			&olympiad.Location,
			&olympiad.Description,
			&olympiad.SchoolID,
			&olympiad.Level,
			&olympiad.Limit,
			&olympiad.CreatorID,
			&firstName,
			&lastName,
			&schoolName,
			&olympiad.Expired,
			&olympiad.Participants,
		)

		if err == sql.ErrNoRows {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Олимпиада не найдена"})
			return
		} else if err != nil {
			log.Println("Ошибка запроса:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка запроса к базе данных"})
			return
		}

		olympiad.CreatorFirstName = utils.NullStringToString(firstName)
		olympiad.CreatorLastName = utils.NullStringToString(lastName)
		olympiad.SchoolName = utils.NullStringToString(schoolName)

		log.Printf("DEBUG: %+v", olympiad)
		utils.ResponseJSON(w, olympiad)
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
		// Step 1: Verify token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check permissions
		if userRole != "superadmin" && userRole != "schooladmin" && userRole != "teacher" && userRole != "student" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view olympiads"})
			return
		}

		// Step 4: Build base query with photo_url
		query := `
			SELECT 
				so.subject_olympiad_id,
				so.subject_name,
				so.date AS start_date,
				so.end_date,
				so.description,
				so.school_id,
				so.level,
				so.limit_participants,
				COALESCE(u.id, 0) AS creator_id,
				COALESCE(u.first_name, '') AS creator_first_name,
				COALESCE(u.last_name, '') AS creator_last_name,
				COALESCE(s.school_name, '') AS school_name,
				so.photo_url
			FROM subject_olympiads so
			LEFT JOIN users u ON so.creator_id = u.id
			LEFT JOIN Schools s ON so.school_id = s.school_id
		`

		var rows *sql.Rows

		// Step 5: Restrict query for non-superadmin
		if userRole == "superadmin" {
			rows, err = db.Query(query)
		} else {
			var schoolID int
			err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", userID).Scan(&schoolID)
			if err != nil {
				log.Println("Error fetching user school_id:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user school details"})
				return
			}
			query += " WHERE so.school_id = ?"
			rows, err = db.Query(query, schoolID)
		}

		if err != nil {
			log.Println("Error querying olympiads:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch olympiads"})
			return
		}
		defer rows.Close()

		// Step 6: Read and map rows
		var olympiads []map[string]interface{}
		for rows.Next() {
			var (
				id                int
				subjectName       string
				startDate         string
				endDate           sql.NullString
				description       sql.NullString
				schoolID          sql.NullInt64
				level             sql.NullString
				limitParticipants sql.NullInt64
				creatorID         int
				creatorFirstName  string
				creatorLastName   string
				schoolName        string
				photo             sql.NullString
			)

			err := rows.Scan(
				&id,
				&subjectName,
				&startDate,
				&endDate,
				&description,
				&schoolID,
				&level,
				&limitParticipants,
				&creatorID,
				&creatorFirstName,
				&creatorLastName,
				&schoolName,
				&photo,
			)
			if err != nil {
				log.Println("Error scanning olympiad row:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing olympiad data"})
				return
			}

			// Check if olympiad expired
			isExpired := false
			if endDate.Valid && endDate.String != "" {
				endDateParsed, err := time.Parse("2006-01-02", endDate.String)
				if err == nil && time.Now().After(endDateParsed) {
					isExpired = true
				}
			}

			// Convert photo_url
			var photoURL interface{}
			if photo.Valid && photo.String != "" {
				photoURL = photo.String
			} else {
				photoURL = nil
			}

			// Convert end_date to proper format
			var endDateValue interface{}
			if endDate.Valid {
				endDateValue = endDate.String
			} else {
				endDateValue = nil
			}

			// Convert school_id to proper format
			var schoolIDValue interface{}
			if schoolID.Valid {
				schoolIDValue = int(schoolID.Int64)
			} else {
				schoolIDValue = nil
			}

			// Convert level to proper format
			var levelValue interface{}
			if level.Valid {
				levelValue = level.String
			} else {
				levelValue = nil
			}

			// Convert limit_participants to proper format
			var limitParticipantsValue interface{}
			if limitParticipants.Valid {
				limitParticipantsValue = int(limitParticipants.Int64)
			} else {
				limitParticipantsValue = nil
			}

			// Build result object
			olympiad := map[string]interface{}{
				"subject_olympiad_id": id,
				"id":                  id,
				"subject_name":        subjectName,
				"start_date":          startDate,
				"end_date":            endDateValue,
				"description":         description.String,
				"school_id":           schoolIDValue,
				"level":               levelValue,
				"limit_participants":  limitParticipantsValue,
				"photo_url":           photoURL,
				"creator_id":          creatorID,
				"creator_first_name":  creatorFirstName,
				"creator_last_name":   creatorLastName,
				"school_name":         schoolName,
				"expired":             isExpired,
			}

			olympiads = append(olympiads, olympiad)
		}

		// Step 7: Return response
		utils.ResponseJSON(w, olympiads)
	}
}
func (c *SubjectOlympiadController) GetOlympiadsBySubjectID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		subjectName := r.URL.Query().Get("subject_name")
		if subjectName == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "subject_name parameter is required"})
			return
		}

		query := `
            SELECT 
                so.subject_olympiad_id,
                so.date as start_date,
                so.end_date,
                COALESCE(so.description, '') as location,
                so.level,
                so.limit_participants,
                s.school_id,
                COALESCE(s.school_name, '') as school_name,
                COUNT(reg.olympiads_registrations_id) as current_participants
            FROM 
                subject_olympiads so
            LEFT JOIN 
                Schools s ON so.school_id = s.school_id
            LEFT JOIN 
                olympiad_registrations reg ON so.subject_olympiad_id = reg.subject_olympiad_id 
                AND reg.status = 'accepted'
            WHERE 
                so.subject_name = ?
            GROUP BY 
                so.subject_olympiad_id, so.date, so.end_date, so.description, so.level, so.limit_participants, s.school_id, s.school_name
            ORDER BY 
                s.school_name, so.subject_olympiad_id
        `

		log.Printf("Executing query: %s with subject_name: %s", query, subjectName)
		rows, err := db.Query(query, subjectName)
		if err != nil {
			log.Println("Error querying olympiads:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiads"})
			return
		}
		defer rows.Close()

		type OlympiadData struct {
			ID                int    `json:"subject_olympiad_id"`
			StartDate         string `json:"start_date"`
			EndDate           string `json:"end_date,omitempty"`
			Location          string `json:"location"`
			Level             string `json:"level"`
			LimitParticipants int    `json:"limit_participants"`
			Expired           bool   `json:"expired"`
			Participants      int    `json:"participants"`
		}

		type SchoolWithOlympiads struct {
			SchoolID   int            `json:"school_id"`
			SchoolName string         `json:"school_name"`
			Olympiads  []OlympiadData `json:"olympiads"`
		}

		schoolMap := make(map[int]*SchoolWithOlympiads)
		currentTime := time.Now()

		for rows.Next() {
			var olympiad OlympiadData
			var schoolID int
			var schoolName string
			var endDateStr sql.NullString
			var currentParticipants int

			err := rows.Scan(
				&olympiad.ID,
				&olympiad.StartDate,
				&endDateStr,
				&olympiad.Location,
				&olympiad.Level,
				&olympiad.LimitParticipants,
				&schoolID,
				&schoolName,
				&currentParticipants,
			)
			if err != nil {
				log.Println("Error scanning row:", err)
				continue
			}

			if endDateStr.Valid {
				olympiad.EndDate = endDateStr.String
			} else {
				start, _ := time.Parse("2006-01-02", olympiad.StartDate)
				olympiad.EndDate = start.AddDate(0, 0, 1).Format("2006-01-02")
			}

			olympiad.Participants = currentParticipants
			endDate, err := time.Parse("2006-01-02", olympiad.EndDate)
			if err != nil {
				olympiad.Expired = false
			} else {
				olympiad.Expired = currentTime.After(endDate)
			}

			if _, exists := schoolMap[schoolID]; !exists {
				schoolMap[schoolID] = &SchoolWithOlympiads{
					SchoolID:   schoolID,
					SchoolName: schoolName,
					Olympiads:  []OlympiadData{},
				}
			}

			schoolMap[schoolID].Olympiads = append(schoolMap[schoolID].Olympiads, olympiad)
		}

		if err := rows.Err(); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing olympiad data"})
			return
		}

		var response []SchoolWithOlympiads
		for _, school := range schoolMap {
			response = append(response, *school)
		}

		if len(response) == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No olympiads found for the given subject"})
			return
		}

		log.Printf("Found olympiads from %d schools for subject %s", len(response), subjectName)
		utils.ResponseJSON(w, response)
	}
}
func (c *SubjectOlympiadController) GetAllSubjectOlympiadsSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Extract schoolID from URL path parameter
		vars := mux.Vars(r)
		schoolID := vars["school_id"]

		subject := r.URL.Query().Get("subject")

		// Updated query to include all the necessary fields
		query := `
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
				COALESCE(s.school_name, '') as school_name
			FROM 
				subject_olympiads so
			LEFT JOIN 
				users u ON so.creator_id = u.id
			LEFT JOIN 
				Schools s ON so.school_id = s.school_id
		`

		var rows *sql.Rows
		var params []interface{}
		var conditions []string

		if subject != "" {
			conditions = append(conditions, "so.subject_name = ?")
			params = append(params, subject)
		}

		if schoolID != "" {
			conditions = append(conditions, "so.school_id = ?")
			params = append(params, schoolID)
		}

		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
			rows, err = db.Query(query, params...)
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
			if err := rows.Scan(
				&o.ID,
				&o.SubjectName,
				&o.StartDate,
				&o.EndDate,
				&o.Description,
				&o.SchoolID,
				&o.Level,
				&o.Limit,
				&o.CreatorID,
				&o.CreatorFirstName,
				&o.CreatorLastName,
				&o.SchoolName,
			); err != nil {
				log.Println("Scan error:", err)
				continue
			}

			// Check if olympiad is expired
			currentTime := time.Now()
			endDate, err := time.Parse("2006-01-02", o.EndDate)
			if err != nil {
				log.Printf("Error parsing end date '%s': %v", o.EndDate, err)
				o.Expired = false
			} else {
				o.Expired = currentTime.After(endDate)
			}

			olympiads = append(olympiads, o)
		}

		utils.ResponseJSON(w, olympiads)
	}
}
func (c *SubjectOlympiadController) GetRegisteredStudentsByMonth(db *sql.DB) http.HandlerFunc {
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
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check superadmin access
		if userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can access this endpoint"})
			return
		}

		// Step 4: Query to count registered students by month
		query := `
			SELECT 
				DATE_FORMAT(so.date, '%Y-%m') as month,
				COUNT(DISTINCT reg.student_id) as registered_students
			FROM 
				subject_olympiads so
			LEFT JOIN 
				olympiad_registrations reg ON so.subject_olympiad_id = reg.subject_olympiad_id 
				AND reg.status = 'registered'
			GROUP BY 
				DATE_FORMAT(so.date, '%Y-%m')
			ORDER BY 
				month DESC
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error querying registered students by month:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching registration data"})
			return
		}
		defer rows.Close()

		// Step 5: Prepare response structure
		type MonthlyRegistration struct {
			Month              string `json:"month"`
			RegisteredStudents int    `json:"registered_students"`
		}

		var registrations []MonthlyRegistration
		for rows.Next() {
			var reg MonthlyRegistration
			err := rows.Scan(&reg.Month, &reg.RegisteredStudents)
			if err != nil {
				log.Println("Error scanning registration data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing registration data"})
				return
			}
			registrations = append(registrations, reg)
		}

		// Check for errors from iterating over rows
		if err = rows.Err(); err != nil {
			log.Println("Error iterating over rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing registration data"})
			return
		}

		// Step 6: Return the result
		log.Printf("Successfully retrieved registration data for superadmin (userID: %d)", userID)
		utils.ResponseJSON(w, registrations)
	}
}
func (c *SubjectOlympiadController) GetOlympiadParticipants(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		olympiadIDStr := vars["subject_olympiad_id"]
		olympiadID, err := strconv.Atoi(olympiadIDStr)
		if err != nil || olympiadID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid subject_olympiad_id"})
			return
		}

		query := `
			SELECT 
				so.subject_olympiad_id,
				s.school_name,
				so.date AS start_date,
				so.end_date,
				so.level,
				so.limit_participants,
				so.subject_name,
				(
					SELECT COUNT(*) 
					FROM olympiad_registrations r 
					WHERE r.subject_olympiad_id = so.subject_olympiad_id AND r.status = 'accepted'
				) AS participants
			FROM subject_olympiads so
			LEFT JOIN Schools s ON s.school_id = so.school_id
			WHERE so.subject_olympiad_id = ?
		`

		var result struct {
			ID           int    `json:"subject_olympiad_id"`
			SchoolName   string `json:"school_name"`
			StartDate    string `json:"start_date"`
			EndDate      string `json:"end_date"`
			Level        string `json:"level"`
			Limit        int    `json:"limit_participants"`
			SubjectName  string `json:"subject_name"`
			Participants int    `json:"participants"`
		}

		err = db.QueryRow(query, olympiadID).Scan(
			&result.ID,
			&result.SchoolName,
			&result.StartDate,
			&result.EndDate,
			&result.Level,
			&result.Limit,
			&result.SubjectName,
			&result.Participants,
		)

		if err == sql.ErrNoRows {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad not found"})
			return
		} else if err != nil {
			log.Println("Database error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}

		utils.ResponseJSON(w, result)
	}
}
func (c *SubjectOlympiadController) GetSchoolOlympiadStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка токена
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Получение school_id из URL параметров
		vars := mux.Vars(r)
		schoolID := vars["school_id"]

		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
			return
		}

		// Запрос для подсчета количества олимпиад, созданных школой
		query := `
			SELECT COUNT(*) as olympiad_count
			FROM subject_olympiads
			WHERE school_id = ?
		`

		var olympiadCount int
		err = db.QueryRow(query, schoolID).Scan(&olympiadCount)
		if err != nil {
			log.Println("Query error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get olympiad stats"})
			return
		}

		// Формирование ответа
		stats := map[string]interface{}{
			"school_id":      schoolID,
			"olympiad_count": olympiadCount,
		}

		utils.ResponseJSON(w, stats)
	}
}
