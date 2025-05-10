package controllers

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ranking-school/models"
	"ranking-school/utils"
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

		// Get other form fields
		event.EventName = r.FormValue("event_name")
		event.Description = r.FormValue("description")
		event.DateTime = r.FormValue("date_time")
		event.Category = r.FormValue("category")
		event.Location = r.FormValue("location")
		event.Status = r.FormValue("status")

		// Validate required fields
		if event.EventName == "" || event.DateTime == "" || event.Location == "" || event.Status == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Missing required fields: event_name, date_time, location, and status are required",
			})
			return
		}

		// Validate status value
		if event.Status != "Upcoming" && event.Status != "Completed" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Invalid status value: must be 'Upcoming' or 'Completed'",
			})
			return
		}

		// Handle file upload for photo
		var photoFileName string
		file, handler, err := r.FormFile("photo")
		if err != nil && err != http.ErrMissingFile {
			log.Println("Error retrieving photo file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error processing photo upload"})
			return
		}

		if file != nil {
			defer file.Close()

			// Create unique filename
			fileExt := filepath.Ext(handler.Filename)
			photoFileName = fmt.Sprintf("event_%d%s", time.Now().UnixNano(), fileExt)

			// Create uploads directory if it doesn't exist
			uploadDir := "./uploads/events"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Println("Error creating uploads directory:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error saving photo"})
				return
			}

			// Create the file
			dst, err := os.Create(filepath.Join(uploadDir, photoFileName))
			if err != nil {
				log.Println("Error creating destination file:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error saving photo"})
				return
			}
			defer dst.Close()

			// Copy the uploaded file to the destination file
			if _, err := io.Copy(dst, file); err != nil {
				log.Println("Error copying file:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error saving photo"})
				return
			}

			// Set the photo path in the event object
			event.Photo = filepath.Join("uploads/events", photoFileName)
		}

		// If role is schooladmin, verify the school_id
		if role == "schooladmin" {
			// Check if user is associated with the school
			// Assuming school administrators are stored in the Users table with school_id
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
		}

		// For schooladmin, school_id is required
		if role == "schooladmin" && event.SchoolID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "School ID is required for school administrators",
			})
			return
		}

		// Check if school exists
		if event.SchoolID > 0 {
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

		// Set the current user as creator
		event.UserID = userID

		// Set timestamps
		now := time.Now().Format("2006-01-02 15:04:05")

		// Insert event with proper error handling
		// FIXED: Changed 'id' to 'user_id' in the column list to match the struct field
		result, err := db.Exec(
			`INSERT INTO Events (
				school_id, user_id, event_name, description, photo, 
				date_time, category, location, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			event.SchoolID, event.UserID, event.EventName, event.Description, event.Photo,
			event.DateTime, event.Category, event.Location, event.Status,
			now, now,
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
		eventID := query.Get("id")
		schoolID := query.Get("school_id")
		status := query.Get("status")
		category := query.Get("category")
		dateFrom := query.Get("date_from")
		dateTo := query.Get("date_to")
		limit := query.Get("limit")
		offset := query.Get("offset")

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
			utils.ResponseJSON(w, event)
			return
		}

		// Строим запрос для получения списка событий
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString(`
			SELECT e.id, e.school_id, e.user_id, e.event_name, e.description, 
			e.photo, e.date_time, e.category, e.location, e.status, 
			e.created_at, e.updated_at, u.email AS created_by
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

		if status != "" {
			// Проверяем корректность статуса
			if status != "Upcoming" && status != "Completed" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status value"})
				return
			}
			queryBuilder.WriteString(" AND e.status = ?")
			args = append(args, status)
		}

		if category != "" {
			queryBuilder.WriteString(" AND e.category = ?")
			args = append(args, category)
		}

		// Фильтр по датам
		if dateFrom != "" {
			// Проверяем формат даты
			_, err := time.Parse("2006-01-02", dateFrom)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_from format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.date_time >= ?")
			args = append(args, dateFrom)
		}

		if dateTo != "" {
			// Проверяем формат даты
			_, err := time.Parse("2006-01-02", dateTo)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_to format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.date_time <= ?")
			args = append(args, dateTo)
		}

		// Добавляем сортировку по дате - сначала показываем ближайшие события
		queryBuilder.WriteString(" ORDER BY e.date_time ASC")

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

		// Выполняем запрос
		rows, err := db.Query(queryBuilder.String(), args...)
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
				&event.Photo, &event.DateTime, &event.Category, &event.Location, &event.Status,
				&event.CreatedAt, &event.UpdatedAt, &event.CreatedBy,
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

		// Проверяем, нашлись ли события
		if len(events) == 0 {
			// Отправляем пустой массив, а не ошибку
			utils.ResponseJSON(w, []models.Event{})
			return
		}

		// Отправляем результат
		utils.ResponseJSON(w, events)
	}
}

// Вспомогательная функция для получения события по ID
func getEventByID(db *sql.DB, eventID int) (*models.Event, error) {
	query := `
		SELECT e.id, e.school_id, e.user_id, e.event_name, e.description, 
		e.photo, e.date_time, e.category, e.location, e.status, 
		e.created_at, e.updated_at, u.email AS created_by
		FROM Events e
		LEFT JOIN users u ON e.user_id = u.id
		WHERE e.id = ?
	`

	var event models.Event
	err := db.QueryRow(query, eventID).Scan(
		&event.ID, &event.SchoolID, &event.UserID, &event.EventName, &event.Description,
		&event.Photo, &event.DateTime, &event.Category, &event.Location, &event.Status,
		&event.CreatedAt, &event.UpdatedAt, &event.CreatedBy,
	)

	if err != nil {
		return nil, err
	}

	return &event, nil
}

// // UpdateEvent allows modification of an existing event
// func (ec *EventController) UpdateEvent(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Check if user is authenticated using VerifyToken
// 		userID, err := utils.VerifyToken(r)
// 		if err != nil {
// 			log.Println("Authentication error:", err)
// 			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Authentication required"})
// 			return
// 		}

// 		// Get user details to determine role
// 		user, err := utils.GetUserByID(db, userID)
// 		if err != nil {
// 			log.Println("Error retrieving user details:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving user information"})
// 			return
// 		}

// 		role := user.Role

// 		// Decode request body
// 		var event models.Event
// 		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
// 			log.Println("Error decoding request body:", err)
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
// 			return
// 		}

// 		if event.ID <= 0 {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Event ID is required for updates"})
// 			return
// 		}

// 		// Validate status value if provided
// 		if event.Status != "" && event.Status != "Upcoming" && event.Status != "Completed" {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
// 				Message: "Invalid status value: must be 'Upcoming' or 'Completed'",
// 			})
// 			return
// 		}

// 		// Check permissions based on role
// 		if role == "schooladmin" {
// 			// School admin can only update events they created or for their school
// 			var creatorID int
// 			var eventSchoolID int
// 			err := db.QueryRow("SELECT user_id, school_id FROM Events WHERE id = ?", event.ID).Scan(&creatorID, &eventSchoolID)

// 			if err != nil {
// 				if err == sql.ErrNoRows {
// 					utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
// 				} else {
// 					log.Println("Error fetching event details:", err)
// 					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving event information"})
// 				}
// 				return
// 			}

// 			// Check if school admin is associated with this school
// 			var isAssociated bool
// 			err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM school_admins WHERE user_id = ? AND school_id = ?)",
// 				userID, eventSchoolID).Scan(&isAssociated)

// 			if err != nil {
// 				log.Println("Error checking school association:", err)
// 				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error validating permissions"})
// 				return
// 			}

// 			if !isAssociated && creatorID != userID {
// 				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You don't have permission to update this event"})
// 				return
// 			}
// 		} else if role != "superadmin" {
// 			// Neither schooladmin nor superadmin
// 			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Insufficient permissions"})
// 			return
// 		}

// 		// Update the event
// 		_, err = db.Exec(
// 			`UPDATE Events SET
// 				event_name = COALESCE(NULLIF(?, ''), event_name),
// 				description = COALESCE(NULLIF(?, ''), description),
// 				photo = COALESCE(NULLIF(?, ''), photo),
// 				date_time = COALESCE(NULLIF(?, ''), date_time),
// 				category = COALESCE(NULLIF(?, ''), category),
// 				location = COALESCE(NULLIF(?, ''), location),
// 				organizer = COALESCE(NULLIF(?, ''), organizer),
// 				status = COALESCE(NULLIF(?, ''), status),
// 				updated_at = ?
// 			WHERE id = ?`,
// 			event.EventName, event.Description, event.Photo, event.DateTime,
// 			event.Category, event.Location, event.Organizer, event.Status,
// 			time.Now().Format("2006-01-02 15:04:05"), event.ID,
// 		)

// 		if err != nil {
// 			log.Println("Error updating event:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update event"})
// 			return
// 		}

// 		// Create response
// 		response := map[string]interface{}{
// 			"message":  "Event updated successfully",
// 			"event_id": event.ID,
// 		}

// 		utils.ResponseJSON(w, response)
// 	}
// }
