package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ranking-school/models"
	"ranking-school/utils"

	"github.com/gorilla/mux"
)

type EventController struct{}

func (ec *EventController) AddEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user is authenticated using VerifyToken
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Println("Authentication error:", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Authentication required"})
			return
		}

		// Get user details to determine role
		user, err := utils.GetUserByID(db, userID)
		if err != nil {
			log.Println("Error retrieving user details:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving user information"})
			return
		}

		role := user.Role

		// Ensure user has appropriate role
		if role != "schooladmin" && role != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Insufficient permissions"})
			return
		}

		// Parse multipart form with 10MB max memory
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Println("Error parsing multipart form:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid form data"})
			return
		}

		// Create event object from form data
		var event models.Event

		// Parse school_id
		schoolIDStr := r.FormValue("school_id")
		if schoolIDStr != "" {
			schoolID, err := strconv.ParseInt(schoolIDStr, 10, 64)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
				return
			}
			event.SchoolID = int(schoolID)
		}

		// Get standard form fields
		event.EventName = r.FormValue("event_name")
		event.Description = r.FormValue("description")
		event.Category = r.FormValue("category")
		event.Location = r.FormValue("location")
		event.StartDate = r.FormValue("start_date")
		event.EndDate = r.FormValue("end_date")

		// Parse grade
		gradeStr := r.FormValue("grade")
		if gradeStr != "" {
			grade, err := strconv.Atoi(gradeStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade format"})
				return
			}
			event.Grade = grade
		}

		// Parse limit
		limitStr := r.FormValue("limit")
		if limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid limit format"})
				return
			}
			event.Limit = limit
		}

		// Validate required fields
		if event.EventName == "" || event.Location == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Missing required fields: event_name and location are required",
			})
			return
		}

		// Validate date fields
		if event.StartDate == "" || event.EndDate == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Missing required date fields: start_date and end_date are required",
			})
			return
		}

		// Validate date formats
		dateFormats := []string{"2006-01-02", "2006-01-02 15:04:05"}
		startDateValid := false
		endDateValid := false

		for _, format := range dateFormats {
			_, err := time.Parse(format, event.StartDate)
			if err == nil {
				startDateValid = true
			}

			_, err = time.Parse(format, event.EndDate)
			if err == nil {
				endDateValid = true
			}
		}

		if !startDateValid {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Invalid start_date format. Use YYYY-MM-DD or YYYY-MM-DD HH:MM:SS",
			})
			return
		}

		if !endDateValid {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Invalid end_date format. Use YYYY-MM-DD or YYYY-MM-DD HH:MM:SS",
			})
			return
		}

		// Handle file upload for photo
		file, handler, err := r.FormFile("photo")
		if err != nil && err != http.ErrMissingFile {
			log.Println("Error retrieving photo file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error processing photo upload"})
			return
		}

		if file != nil {
			defer file.Close()

			// Create unique filename for S3
			fileExt := filepath.Ext(handler.Filename)
			photoFileName := fmt.Sprintf("event_%d%s", time.Now().UnixNano(), fileExt)

			// Upload file to S3 using the "schoolphoto" case
			photoURL, err := utils.UploadFileToS3(file, photoFileName, "schoolphoto")
			if err != nil {
				log.Println("Error uploading photo to S3:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error saving photo to cloud storage"})
				return
			}

			// Set the S3 URL in the event object
			event.Photo = photoURL
		}

		// Validate school access based on role
		if role == "schooladmin" {
			// For schooladmin: check if they can add events for this school
			if event.SchoolID <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{
					Message: "School ID is required for school administrators",
				})
				return
			}

			// Check if user is associated with the school
			var isUserAssociatedWithSchool bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND school_id = ? AND role = 'schooladmin')",
				userID, event.SchoolID).Scan(&isUserAssociatedWithSchool)

			if err != nil {
				log.Println("Error checking school association:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{
					Message: "Error validating school association",
				})
				return
			}

			if !isUserAssociatedWithSchool {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{
					Message: "You are not authorized to add events for this school",
				})
				return
			}
		} else if role == "superadmin" {
			// For superadmin: can add events for any school, but school_id should be valid if provided
			if event.SchoolID > 0 {
				// Check if school exists
				var schoolExists bool
				err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)",
					event.SchoolID).Scan(&schoolExists)

				if err != nil {
					log.Println("Error checking if school exists:", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{
						Message: "Error checking if school exists",
					})
					return
				}

				if !schoolExists {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{
						Message: "School does not exist",
					})
					return
				}
			}
		}

		// Set the current user as creator
		event.UserID = userID

		// Get the username of the creator
		var createdBy string
		err = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&createdBy)
		if err != nil {
			log.Println("Error fetching username:", err)
			createdBy = ""
		}
		event.CreatedBy = createdBy

		// Set timestamps
		now := time.Now().Format("2006-01-02 15:04:05")
		event.CreatedAt = now
		event.UpdatedAt = now

		// Insert event with proper error handling
		result, err := db.Exec(
			`INSERT INTO Events (
                school_id, user_id, event_name, description, photo, 
                start_date, end_date, category, location, 
                grade, limit_count, created_at, updated_at, created_by
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			event.SchoolID, event.UserID, event.EventName, event.Description, event.Photo,
			event.StartDate, event.EndDate, event.Category, event.Location,
			event.Grade, event.Limit, now, now, event.CreatedBy,
		)

		if err != nil {
			log.Println("Error inserting event:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{
				Message: "Failed to create event",
			})
			return
		}

		// Get the inserted ID
		eventID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error getting last insert ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{
				Message: "Event created but failed to retrieve ID",
			})
			return
		}

		// Create response with the new event ID
		response := map[string]interface{}{
			"message":  "Event created successfully",
			"event_id": eventID,
		}

		utils.ResponseJSON(w, response)
	}
}
func (ec *EventController) GetEvents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что используется метод GET
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Получаем параметры запроса
		query := r.URL.Query()

		// Создаем map для хранения всех параметров
		params := make(map[string]string)

		// Собираем основные параметры
		params["id"] = query.Get("id")
		params["school_id"] = query.Get("school_id")
		params["grade"] = query.Get("grade")
		params["category"] = query.Get("category")
		params["date_from"] = query.Get("date_from")
		params["date_to"] = query.Get("date_to")
		params["limit"] = query.Get("limit")
		params["offset"] = query.Get("offset")

		// Добавляем все остальные параметры
		for key, values := range query {
			if _, exists := params[key]; !exists && len(values) > 0 {
				params[key] = values[0]
			}
		}

		// Логируем все параметры
		log.Println("GetEvents called with parameters:", params)

		// Если запрошен debug режим, возвращаем все параметры
		if query.Get("debug") == "true" {
			utils.ResponseJSON(w, map[string]interface{}{
				"message":    "Debug mode: showing all parameters",
				"parameters": params,
			})
			return
		}

		// Переменные для основных параметров
		eventID := params["id"]
		schoolID := params["school_id"]
		grade := params["grade"]
		category := params["category"]
		dateFrom := params["date_from"]
		dateTo := params["date_to"]
		limit := params["limit"]
		offset := params["offset"]

		// Если указан конкретный ID события - получаем только его
		if eventID != "" {
			id, err := strconv.Atoi(eventID)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID format"})
				return
			}
			event, err := getEventByID(db, id)
			if err != nil {
				if err == sql.ErrNoRows {
					utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
				} else {
					log.Println("Error fetching event by ID:", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching event"})
				}
				return
			}

			// Формируем ответ без parameters_used
			response := map[string]interface{}{
				"event": event,
			}
			utils.ResponseJSON(w, response)
			return
		}

		// Строим запрос для получения списка событий
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString(`
            SELECT e.id, e.school_id, e.user_id, e.event_name, e.description, 
            e.photo, e.start_date, e.end_date, e.category, e.location, 
            e.grade, e.limit_count, e.created_at, e.updated_at, u.email AS created_by
            FROM Events e
            LEFT JOIN users u ON e.user_id = u.id
            WHERE 1=1
        `)

		var args []interface{}

		// Добавляем фильтры, если они указаны
		if schoolID != "" {
			schoolIDInt, err := strconv.Atoi(schoolID)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
				return
			}
			queryBuilder.WriteString(" AND e.school_id = ?")
			args = append(args, schoolIDInt)
		}

		if grade != "" {
			gradeInt, err := strconv.Atoi(grade)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade format"})
				return
			}
			queryBuilder.WriteString(" AND e.grade = ?")
			args = append(args, gradeInt)
		}

		if category != "" {
			queryBuilder.WriteString(" AND e.category = ?")
			args = append(args, category)
		}

		// Фильтр по датам (start_date)
		if dateFrom != "" {
			_, err := time.Parse("2006-01-02", dateFrom)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_from format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.start_date >= ?")
			args = append(args, dateFrom)
		}

		if dateTo != "" {
			_, err := time.Parse("2006-01-02", dateTo)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_to format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.end_date <= ?")
			args = append(args, dateTo)
		}

		// Добавляем сортировку по start_date
		queryBuilder.WriteString(" ORDER BY e.start_date ASC")

		// Добавляем пагинацию
		if limit != "" {
			limitInt, err := strconv.Atoi(limit)
			if err != nil || limitInt <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid limit parameter"})
				return
			}
			queryBuilder.WriteString(" LIMIT ?")
			args = append(args, limitInt)

			if offset != "" {
				offsetInt, err := strconv.Atoi(offset)
				if err != nil || offsetInt < 0 {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid offset parameter"})
					return
				}
				queryBuilder.WriteString(" OFFSET ?")
				args = append(args, offsetInt)
			}
		}

		// Логируем финальный SQL запрос
		finalQuery := queryBuilder.String()
		log.Printf("Executing SQL query: %s with args: %v", finalQuery, args)

		// Выполняем запрос
		rows, err := db.Query(finalQuery, args...)
		if err != nil {
			log.Println("Error executing events query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch events"})
			return
		}
		defer rows.Close()

		// Собираем результаты
		var events []models.Event
		for rows.Next() {
			var event models.Event
			err := rows.Scan(
				&event.ID, &event.SchoolID, &event.UserID, &event.EventName, &event.Description,
				&event.Photo, &event.StartDate, &event.EndDate, &event.Category,
				&event.Location, &event.Grade, &event.Limit, &event.CreatedAt, &event.UpdatedAt,
				&event.CreatedBy,
			)
			if err != nil {
				log.Println("Error scanning event row:", err)
				continue
			}
			events = append(events, event)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating event rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing events data"})
			return
		}

		// Подготавливаем ответ без parameters_used
		response := map[string]interface{}{
			"events":      events,
			"total_count": len(events),
		}

		if len(events) == 0 {
			response["message"] = "No events found for the specified criteria"
		}

		utils.ResponseJSON(w, response)
	}
}
func (ec *EventController) UpdateEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем аутентификацию
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Println("Authentication error:", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Authentication required"})
			return
		}

		// Получаем роль пользователя
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user information"})
			return
		}

		// Проверяем, что пользователь имеет роль schooladmin или superadmin
		if userRole != "schooladmin" && userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Insufficient permissions"})
			return
		}

		// Получаем ID события из URL
		vars := mux.Vars(r)
		eventIDStr := vars["id"]
		if eventIDStr == "" {
			eventIDStr = vars["event_id"]
		}

		eventID, err := strconv.Atoi(eventIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID format"})
			return
		}

		// Проверяем, существует ли событие
		var existingEvent models.Event
		var eventSchoolID int
		var eventCreatorID int
		var currentPhotoURL string

		err = db.QueryRow("SELECT id, school_id, user_id, photo FROM Events WHERE id = ?", eventID).Scan(
			&existingEvent.ID, &eventSchoolID, &eventCreatorID, &currentPhotoURL,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
			} else {
				log.Println("Error fetching event:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking occasion existence"})
			}
			return
		}

		// Проверяем права доступа для schooladmin
		if userRole == "schooladmin" {
			if !userSchoolID.Valid || int(userSchoolID.Int64) != eventSchoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You don't have permission to update this event"})
				return
			}
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Println("Error parsing form:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid form data"})
			return
		}

		// Структура для обновления
		var updatedEvent models.Event
		updatedEvent.ID = eventID

		// Обновляем поля, которые пришли в запросе
		updateFields := make(map[string]interface{})
		if eventName := r.FormValue("event_name"); eventName != "" {
			updateFields["event_name"] = eventName
			updatedEvent.EventName = eventName
		}

		if description := r.FormValue("description"); description != "" {
			updateFields["description"] = description
			updatedEvent.Description = description
		}

		if startDate := r.FormValue("start_date"); startDate != "" {
			dateFormats := []string{"2006-01-02", "2006-01-02 15:04:05"}
			validFormat := false

			for _, format := range dateFormats {
				_, err := time.Parse(format, startDate)
				if err == nil {
					validFormat = true
					break
				}
			}

			if !validFormat {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid start_date format. Use YYYY-MM-DD or YYYY-MM-DD HH:MM:SS"})
				return
			}
			updateFields["start_date"] = startDate
			updatedEvent.StartDate = startDate
		}

		if endDate := r.FormValue("end_date"); endDate != "" {
			dateFormats := []string{"2006-01-02", "2006-01-02 15:04:05"}
			validFormat := false

			for _, format := range dateFormats {
				_, err := time.Parse(format, endDate)
				if err == nil {
					validFormat = true
					break
				}
			}

			if !validFormat {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid end_date format. Use YYYY-MM-DD or YYYY-MM-DD HH:MM:SS"})
				return
			}
			updateFields["end_date"] = endDate
			updatedEvent.EndDate = endDate
		}

		if gradeStr := r.FormValue("grade"); gradeStr != "" {
			grade, err := strconv.Atoi(gradeStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade format"})
				return
			}
			updateFields["grade"] = grade
			updatedEvent.Grade = grade
		}

		if limitStr := r.FormValue("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid limit format"})
				return
			}
			updateFields["limit_count"] = limit
			updatedEvent.Limit = limit
		}

		if category := r.FormValue("category"); category != "" {
			updateFields["category"] = category
			updatedEvent.Category = category
		}

		if location := r.FormValue("location"); location != "" {
			updateFields["location"] = location
			updatedEvent.Location = location
		}

		// Обработка фото
		file, handler, err := r.FormFile("photo")
		if err == nil {
			defer file.Close()

			fileExt := filepath.Ext(handler.Filename)
			photoFileName := fmt.Sprintf("event_%d_%d%s", eventID, time.Now().UnixNano(), fileExt)

			photoURL, err := utils.UploadFileToS3(file, photoFileName, "schoolphoto")
			if err != nil {
				log.Println("Error uploading photo to S3:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error saving photo to cloud storage"})
				return
			}

			updateFields["photo"] = photoURL
			updatedEvent.Photo = photoURL

			if currentPhotoURL != "" && strings.Contains(currentPhotoURL, "amazonaws.com") {
				err = utils.DeleteFileFromS3(currentPhotoURL)
				if err != nil {
					log.Println("Error deleting old photo from S3:", err)
				}
			}
		}

		// Добавляем поле updated_at
		now := time.Now().Format("2006-01-02 15:04:05")
		updateFields["updated_at"] = now

		// Если нет полей для обновления
		if len(updateFields) == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "No fields to update"})
			return
		}

		// Строим SQL запрос
		query := "UPDATE Events SET "
		args := make([]interface{}, 0, len(updateFields)+1)

		i := 0
		for field, value := range updateFields {
			if i > 0 {
				query += ", "
			}
			query += field + " = ?"
			args = append(args, value)
			i++
		}

		query += " WHERE id = ?"
		args = append(args, eventID)

		// Выполняем запрос
		result, err := db.Exec(query, args...)
		if err != nil {
			log.Println("Error updating event:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update event"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No event was updated"})
			return
		}

		// Получаем обновленное событие
		var updated models.Event
		err = db.QueryRow(`
            SELECT id, school_id, user_id, event_name, description, 
            photo, start_date, end_date, category, location, 
            grade, limit_count as limit, created_at, updated_at, created_by
            FROM Events 
            WHERE id = ?
        `, eventID).Scan(
			&updated.ID, &updated.SchoolID, &updated.UserID, &updated.EventName, &updated.Description,
			&updated.Photo, &updated.StartDate, &updated.EndDate,
			&updated.Category, &updated.Location, &updated.Grade, &updated.Limit,
			&updated.CreatedAt, &updated.UpdatedAt, &updated.CreatedBy,
		)

		if err != nil {
			log.Println("Error fetching updated event:", err)
			utils.ResponseJSON(w, map[string]interface{}{
				"message":  "Event updated successfully",
				"event_id": eventID,
			})
			return
		}

		// Отправляем ответ
		utils.ResponseJSON(w, map[string]interface{}{
			"message": "Event updated successfully",
			"event":   updated,
		})
	}
}
func getEventByID(db *sql.DB, id int) (models.Event, error) {
	var event models.Event

	query := `
        SELECT e.id, e.school_id, e.user_id, e.event_name, e.description, 
        e.photo, e.start_date, e.end_date, e.category, e.location, 
        e.grade, e.limit_count, e.created_at, e.updated_at, u.email AS created_by
        FROM Events e
        LEFT JOIN users u ON e.user_id = u.id
        WHERE e.id = ?
    `

	err := db.QueryRow(query, id).Scan(
		&event.ID, &event.SchoolID, &event.UserID, &event.EventName, &event.Description,
		&event.Photo, &event.StartDate, &event.EndDate, &event.Category, &event.Location,
		&event.Grade, &event.Limit, &event.CreatedAt, &event.UpdatedAt, &event.CreatedBy,
	)

	if err != nil {
		return models.Event{}, err
	}

	return event, nil
}
func (c *EventController) DeleteEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка токена
		_, err := utils.VerifyToken(r)
		if err != nil {
			log.Println("Invalid token:", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
			return
		}

		// Получение ID события из URL
		vars := mux.Vars(r)
		eventIDStr, ok := vars["event_id"]
		if !ok {
			log.Println("Missing event ID")
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Event ID is required"})
			return
		}

		eventID, err := strconv.Atoi(eventIDStr)
		if err != nil {
			log.Println("Invalid event ID format:", eventIDStr)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID format"})
			return
		}

		// Удаление события
		res, err := db.Exec("DELETE FROM Events WHERE id = ?", eventID)
		if err != nil {
			log.Printf("Error deleting event with ID %d: %v", eventID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete event"})
			return
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("No event found with ID %d", eventID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
			return
		}

		log.Printf("Event with ID %d deleted successfully", eventID)
		utils.ResponseJSON(w, map[string]string{"message": "Event deleted successfully"})
	}
}
func (ec *EventController) GetEventsBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Println("Authentication error:", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Authentication required"})
			return
		}

		// Get user details to determine role
		user, err := utils.GetUserByID(db, userID)
		if err != nil {
			log.Println("Error retrieving user details:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving user information"})
			return
		}

		// Get school_id from URL parameters
		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
			return
		}

		// Validate school exists
		var schoolExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", schoolID).Scan(&schoolExists)
		if err != nil {
			log.Println("Error checking if school exists:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking school existence"})
			return
		}
		if !schoolExists {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
			return
		}

		// Check permissions based on role
		if user.Role == "schooladmin" {
			// Verify user is associated with the school
			var isAssociated bool
			err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND school_id = ? AND role = 'schooladmin')",
				userID, schoolID).Scan(&isAssociated)
			if err != nil {
				log.Println("Error checking school association:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error validating school association"})
				return
			}
			if !isAssociated {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Not authorized to view events for this school"})
				return
			}
		} else if user.Role != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Insufficient permissions"})
			return
		}

		// Get query parameters for filtering
		query := r.URL.Query()
		params := map[string]string{
			"grade":     query.Get("grade"),
			"category":  query.Get("category"),
			"date_from": query.Get("date_from"),
			"date_to":   query.Get("date_to"),
			"limit":     query.Get("limit"),
			"offset":    query.Get("offset"),
		}

		// Build the SQL query
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString(`
            SELECT e.id, e.school_id, e.user_id, e.event_name, e.description, 
            e.photo, e.start_date, e.end_date, e.category, e.location, 
            e.grade, e.limit_count, e.created_at, e.updated_at, e.created_by
            FROM Events e
            WHERE e.school_id = ?
        `)

		args := []interface{}{schoolID}

		// Add filters
		if params["grade"] != "" {
			gradeInt, err := strconv.Atoi(params["grade"])
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade format"})
				return
			}
			queryBuilder.WriteString(" AND e.grade = ?")
			args = append(args, gradeInt)
		}

		if params["category"] != "" {
			queryBuilder.WriteString(" AND e.category = ?")
			args = append(args, params["category"])
		}

		if params["date_from"] != "" {
			_, err := time.Parse("2006-01-02", params["date_from"])
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_from format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.start_date >= ?")
			args = append(args, params["date_from"])
		}

		if params["date_to"] != "" {
			_, err := time.Parse("2006-01-02", params["date_to"])
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_to format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.end_date <= ?")
			args = append(args, params["date_to"])
		}

		// Add sorting
		queryBuilder.WriteString(" ORDER BY e.start_date ASC")

		// Add pagination
		if params["limit"] != "" {
			limitInt, err := strconv.Atoi(params["limit"])
			if err != nil || limitInt <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid limit parameter"})
				return
			}
			queryBuilder.WriteString(" LIMIT ?")
			args = append(args, limitInt)

			if params["offset"] != "" {
				offsetInt, err := strconv.Atoi(params["offset"])
				if err != nil || offsetInt < 0 {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid offset parameter"})
					return
				}
				queryBuilder.WriteString(" OFFSET ?")
				args = append(args, offsetInt)
			}
		}

		// Execute query
		rows, err := db.Query(queryBuilder.String(), args...)
		if err != nil {
			log.Println("Error executing events query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch events"})
			return
		}
		defer rows.Close()

		// Collect results
		var events []models.Event
		for rows.Next() {
			var event models.Event
			err := rows.Scan(
				&event.ID, &event.SchoolID, &event.UserID, &event.EventName, &event.Description,
				&event.Photo, &event.StartDate, &event.EndDate, &event.Category,
				&event.Location, &event.Grade, &event.Limit, &event.CreatedAt, &event.UpdatedAt,
				&event.CreatedBy,
			)
			if err != nil {
				log.Println("Error scanning event row:", err)
				continue
			}
			events = append(events, event)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating event rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing events data"})
			return
		}

		// Prepare response
		response := map[string]interface{}{
			"events":      events,
			"total_count": len(events),
			"school_id":   schoolID,
		}

		if len(events) == 0 {
			response["message"] = "No events found for this school"
		}

		utils.ResponseJSON(w, response)
	}
}
func (ec *EventController) CountEvents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Get query parameters
		query := r.URL.Query()

		// Create parameters map
		params := make(map[string]string)

		// Collect filter parameters
		params["school_id"] = query.Get("school_id")
		params["status"] = query.Get("status")
		params["category"] = query.Get("category")
		params["date_from"] = query.Get("date_from")
		params["date_to"] = query.Get("date_to")

		log.Println("CountEvents called with parameters:", params)

		// Build query for counting events
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString("SELECT COUNT(*) FROM Events WHERE 1=1")

		var args []interface{}

		// Add filters if specified
		if schoolID := params["school_id"]; schoolID != "" {
			schoolIDInt, err := strconv.Atoi(schoolID)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
				return
			}
			queryBuilder.WriteString(" AND school_id = ?")
			args = append(args, schoolIDInt)
		}

		if status := params["status"]; status != "" {
			// Validate status value
			if status != "Upcoming" && status != "Completed" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status value: must be 'Upcoming' or 'Completed'"})
				return
			}
			queryBuilder.WriteString(" AND status = ?")
			args = append(args, status)
		}

		if category := params["category"]; category != "" {
			queryBuilder.WriteString(" AND category = ?")
			args = append(args, category)
		}

		// Filter by date range
		if dateFrom := params["date_from"]; dateFrom != "" {
			// Validate date format
			_, err := time.Parse("2006-01-02", dateFrom)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_from format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND date_time >= ?")
			args = append(args, dateFrom)
		}

		if dateTo := params["date_to"]; dateTo != "" {
			// Validate date format
			_, err := time.Parse("2006-01-02", dateTo)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_to format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND date_time <= ?")
			args = append(args, dateTo)
		}

		// Log the final SQL query for debugging
		finalQuery := queryBuilder.String()
		log.Printf("Executing count query: %s with args: %v", finalQuery, args)

		// Execute the query
		var totalCount int
		err := db.QueryRow(finalQuery, args...).Scan(&totalCount)
		if err != nil {
			log.Println("Error executing count query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to count events"})
			return
		}

		// Prepare response with count data and parameters used
		response := map[string]interface{}{
			"total_events": totalCount,
		}

		// Send the result
		utils.ResponseJSON(w, response)
	}
}
