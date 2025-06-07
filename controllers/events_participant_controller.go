package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type EventsParticipantController struct{}

func (c *EventsParticipantController) AddEventsParticipant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the token
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

		// Step 3: Check access permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to add event participants"})
			return
		}

		// Step 4: Parse the form
		err = r.ParseMultipartForm(10 << 20) // 10MB limit
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error parsing form data"})
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		// Step 5: Parse required fields
		schoolIDStr := r.FormValue("school_id")
		studentIDStr := r.FormValue("student_id")
		eventsName := r.FormValue("events_name")
		category := r.FormValue("category")
		role := r.FormValue("role")
		dateStr := r.FormValue("date")

		// Validate required fields
		if schoolIDStr == "" || studentIDStr == "" || eventsName == "" || category == "" || role == "" || dateStr == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "All fields (school_id, student_id, events_name, category, role, date) are required",
			})
			return
		}

		// Parse optional fields, allowing empty values (NULL in DB)
		grade := sql.NullString{String: r.FormValue("grade"), Valid: r.FormValue("grade") != ""}
		letter := sql.NullString{String: r.FormValue("letter"), Valid: r.FormValue("letter") != ""}

		// Convert string values to integers
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil || schoolID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
			return
		}

		studentID, err := strconv.Atoi(studentIDStr)
		if err != nil || studentID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student_id format"})
			return
		}

		// Validate and parse date
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date format, expected YYYY-MM-DD"})
			return
		}
		date := parsedDate.Format("2006-01-02")

		// Step 6: Upload document if provided
		var documentURL string
		file, header, err := r.FormFile("document")
		if err == nil {
			defer file.Close()

			// Validate file extension
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".pdf" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Only JPG, JPEG, PNG, or PDF files are allowed"})
				return
			}

			uniqueFileName := fmt.Sprintf("event-doc-%d-%d%s", studentID, time.Now().Unix(), ext)
			documentURL, err = utils.UploadFileToS3(file, uniqueFileName, "olympiaddoc")
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload document"})
				log.Println("Error uploading file:", err)
				return
			}
		}

		// Step 7: Check if the student exists and belongs to the specified school
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM student WHERE student_id = ? AND school_id = ?", studentID, schoolID).Scan(&count)
		if err != nil {
			log.Println("Error checking student existence:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error while verifying student"})
			return
		}

		if count == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student not found or does not belong to the specified school"})
			return
		}

		// Step 8: Insert the event participant into the database
		query := `INSERT INTO events_participants 
            (school_id, grade, letter, student_id, events_name, document, category, role, date, creator_id) 
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := db.Exec(query,
			schoolID,
			grade,
			letter,
			studentID,
			eventsName,
			documentURL,
			category,
			role,
			date,
			userID)

		if err != nil {
			log.Println("Error inserting event participant:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to add event participant"})
			return
		}

		// Get the ID of the created event participant
		participantID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error getting last insert ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve participant ID"})
			return
		}

		// Step 9: Retrieve complete participant data for response
		var participant models.EventsParticipant

		query = `SELECT ep.id, ep.school_id, ep.grade, ep.letter, ep.student_id, ep.events_name, 
                ep.document, ep.category, ep.role, ep.date, 
                s.student_id, s.first_name as student_name, s.last_name as student_lastname,
                sch.school_name as school_name,
                c.id as creator_id, c.first_name as creator_first_name, c.last_name as creator_last_name
            FROM events_participants ep
            JOIN student s ON ep.student_id = s.student_id
            JOIN Schools sch ON ep.school_id = sch.id
            JOIN users c ON ep.creator_id = c.id
            WHERE ep.id = ?`

		err = db.QueryRow(query, participantID).Scan(
			&participant.ID,
			&participant.SchoolID,
			&participant.Grade,
			&participant.Letter,
			&participant.StudentID,
			&participant.EventsName,
			&participant.Document,
			&participant.Category,
			&participant.Role,
			&participant.Date,
			&participant.StudentID,
			&participant.StudentName,
			&participant.StudentLastName,
			&participant.SchoolName,
			&participant.CreatorID,
			&participant.CreatorFirstName,
			&participant.CreatorLastName,
		)

		if err != nil {
			log.Println("Error fetching created participant:", err)
			// If unable to fetch complete data, return basic information
			utils.ResponseJSON(w, models.EventsParticipant{
				ID:         int(participantID),
				SchoolID:   schoolID,
				Grade:      grade.String,  // Fixed: use grade.String instead of Grade
				Letter:     letter.String, // Fixed: use letter.String instead of Letter
				StudentID:  studentID,
				EventsName: eventsName,
				Document:   documentURL,
				Category:   category,
				Role:       role,
				Date:       date,
				CreatorID:  userID,
			})
			return
		}

		utils.ResponseJSON(w, participant)
	}
}

// func (c *EventsParticipantController) DeleteEventsParticipant(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Step 1: Verify the token
// 		userID, err := utils.VerifyToken(r)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
// 			return
// 		}

// 		// Step 2: Get user role
// 		var userRole string
// 		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
// 		if err != nil {
// 			log.Println("Error fetching user role:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
// 			return
// 		}

// 		// Step 3: Check access permissions
// 		if userRole != "superadmin" && userRole != "schooladmin" {
// 			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to delete event participants"})
// 			return
// 		}

// 		// Step 4: Get the events_id from the URL
// 		vars := mux.Vars(r)
// 		eventsID, ok := vars["events_id"]
// 		if !ok {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing events_id parameter"})
// 			return
// 		}

// 		// Step 5: Delete the event participant
// 		result, err := db.Exec("DELETE FROM events_participants WHERE id = ?", eventsID)
// 		if err != nil {
// 			log.Println("Error deleting event participant:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete event participant"})
// 			return
// 		}

// 		// Check if any row was affected
// 		rowsAffected, err := result.RowsAffected()
// 		if err != nil {
// 			log.Println("Error getting rows affected:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to confirm deletion"})
// 			return
// 		}

// 		if rowsAffected == 0 {
// 			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event participant not found"})
// 			return
// 		}

// 		// Return success response
// 		utils.ResponseJSON(w, map[string]string{"message": "Event participant deleted successfully"})
// 	}
// }

func (c *EventsParticipantController) DeleteEventsParticipant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Проверка токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token."})
			return
		}

		// Шаг 2: Получение роли
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to delete event participants"})
			return
		}

		// Шаг 3: Получение events_id
		vars := mux.Vars(r)
		eventsID, ok := vars["events_id"]
		if !ok {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing events_id parameter"})
			return
		}

		// Шаг 4: Получение student_id и event_name ДО удаления
		var studentID int
		var eventName string
		err = db.QueryRow(`
			SELECT ep.student_id, e.event_name
			FROM events_participants ep
			JOIN Events e ON ep.id = e.id
			WHERE ep.id = ?`, eventsID).Scan(&studentID, &eventName)
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event participant not found"})
			return
		}
		if err != nil {
			log.Println("Error fetching participant data:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch participant info"})
			return
		}

		// Шаг 5: Удаление участника
		result, err := db.Exec("DELETE FROM events_participants WHERE id = ?", eventsID)
		if err != nil {
			log.Println("Error deleting event participant:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete event participant"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Println("Error checking delete result:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to confirm deletion"})
			return
		}
		if rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event participant not found"})
			return
		}

		// Шаг 6: Обновление статуса на "registered" в EventRegistrations
		_, err = db.Exec(`
			UPDATE EventRegistrations r
			JOIN Events e ON r.event_id = e.id
			SET r.status = 'registered'
			WHERE r.student_id = ? AND e.event_name = ?`, studentID, eventName)
		if err != nil {
			log.Println("Error updating registration status:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update registration status"})
			return
		}

		// Шаг 7: Успешный ответ
		utils.ResponseJSON(w, map[string]string{"message": "Participant deleted and status set to 'registered'"})
	}
}

func (c *EventsParticipantController) UpdateEventsParticipant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the token
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

		// Step 3: Check access permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to update event participants"})
			return
		}

		// Step 4: Get and validate events_id
		vars := mux.Vars(r)
		eventsID, ok := vars["events_id"]
		if !ok {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing events_id parameter"})
			return
		}
		eventsIDInt, err := strconv.Atoi(eventsID)
		if err != nil || eventsIDInt <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid events_id format"})
			return
		}

		// Step 5: Parse the form
		err = r.ParseMultipartForm(10 << 20) // 10MB limit
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error parsing form data"})
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		// Step 6: Parse fields
		schoolIDStr := r.FormValue("school_id")
		studentIDStr := r.FormValue("student_id")
		eventsName := strings.TrimSpace(r.FormValue("events_name"))
		category := strings.TrimSpace(r.FormValue("category"))
		role := strings.TrimSpace(r.FormValue("role"))
		dateStr := r.FormValue("date")
		grade := r.FormValue("grade")
		letter := r.FormValue("letter")

		// Step 7: Prepare SQL update query and parameters
		var updateFields []string
		var params []interface{}

		if schoolIDStr != "" {
			schoolID, err := strconv.Atoi(schoolIDStr)
			if err != nil || schoolID <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
				return
			}
			var exists bool
			err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", schoolID).Scan(&exists)
			if err != nil || !exists {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id: school does not exist"})
				return
			}
			updateFields = append(updateFields, "school_id = ?")
			params = append(params, schoolID)
		}

		if studentIDStr != "" {
			studentID, err := strconv.Atoi(studentIDStr)
			if err != nil || studentID <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student_id format"})
				return
			}
			var exists bool
			err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE student_id = ?)", studentID).Scan(&exists)
			if err != nil || !exists {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student_id: student does not exist"})
				return
			}
			updateFields = append(updateFields, "student_id = ?")
			params = append(params, studentID)
		}

		if eventsName != "" {
			if len(eventsName) > 255 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "events_name is too long"})
				return
			}
			updateFields = append(updateFields, "events_name = ?")
			params = append(params, eventsName)
		}

		if category != "" {
			updateFields = append(updateFields, "category = ?")
			params = append(params, category)
		}

		if role != "" {
			updateFields = append(updateFields, "role = ?")
			params = append(params, role)
		}

		if dateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date format, expected YYYY-MM-DD"})
				return
			}
			date := parsedDate.Format("2006-01-02")
			updateFields = append(updateFields, "date = ?")
			params = append(params, date)
		}

		if grade != "" {
			updateFields = append(updateFields, "grade = ?")
			params = append(params, grade)
		}

		if letter != "" {
			updateFields = append(updateFields, "letter = ?")
			params = append(params, letter)
		}

		// Step 8: Handle file upload
		handleFileUpload := func(studentIDStr, eventsID string) (string, error) {
			file, header, err := r.FormFile("document")
			if err != nil {
				return "", nil // No file provided
			}
			defer file.Close()

			ext := strings.ToLower(filepath.Ext(header.Filename))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".pdf" {
				return "", fmt.Errorf("invalid file extension: %s", ext)
			}

			var studentID int
			if studentIDStr != "" {
				studentID, err = strconv.Atoi(studentIDStr)
				if err != nil {
					return "", fmt.Errorf("invalid student_id: %v", err)
				}
			} else {
				err := db.QueryRow("SELECT student_id FROM events_participants WHERE id = ?", eventsID).Scan(&studentID)
				if err != nil {
					return "", fmt.Errorf("failed to fetch student_id: %v", err)
				}
			}

			uniqueFileName := fmt.Sprintf("event-doc-%d-%d%s", studentID, time.Now().Unix(), ext)
			documentURL, err := utils.UploadFileToS3(file, uniqueFileName, "olympiaddoc")
			if err != nil {
				return "", fmt.Errorf("failed to upload file: %v", err)
			}
			return documentURL, nil
		}

		documentURL, err := handleFileUpload(studentIDStr, eventsID)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: err.Error()})
			return
		}
		if documentURL != "" {
			updateFields = append(updateFields, "document = ?")
			params = append(params, documentURL)
		}

		// Step 9: Check if there are fields to update
		if len(updateFields) == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "No fields provided for update"})
			return
		}

		// Step 10: Start a transaction
		tx, err := db.Begin()
		if err != nil {
			log.Println("Error starting transaction:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Step 11: Update the event participant
		query := fmt.Sprintf("UPDATE events_participants SET %s, modifier_id = ? WHERE id = ?",
			strings.Join(updateFields, ", "))
		params = append(params, userID, eventsIDInt)

		result, err := tx.Exec(query, params...)
		if err != nil {
			log.Printf("Error updating event participant: query=%s, params=%v, err=%v", query, params, err)
			if strings.Contains(err.Error(), "foreign key") {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid reference in provided data (e.g., school_id or student_id)"})
			} else if strings.Contains(err.Error(), "unique constraint") {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Duplicate entry detected"})
			} else {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update event participant"})
			}
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Println("Error getting rows affected:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to confirm update"})
			return
		}

		if rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event participant not found or no changes made"})
			return
		}

		// Step 12: Commit transaction
		if err := tx.Commit(); err != nil {
			log.Println("Error committing transaction:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to commit update"})
			return
		}

		// Step 13: Retrieve updated participant data
		var participant models.EventsParticipant
		querySelect := `SELECT ep.id, ep.school_id, ep.grade, ep.letter, ep.student_id, ep.events_name, 
                              ep.document, ep.category, ep.role, ep.date, 
                              s.student_id, s.first_name as student_name, s.last_name as student_lastname,
                              sch.school_name as school_name,
                              c.id as creator_id, c.first_name as creator_first_name, c.last_name as creator_last_name
                       FROM events_participants ep
                       JOIN student s ON ep.student_id = s.student_id
                       JOIN Schools sch ON ep.school_id = sch.school_id
                       JOIN users c ON ep.creator_id = c.id
                       WHERE ep.id = ?`

		err = db.QueryRow(querySelect, eventsIDInt).Scan(
			&participant.ID,
			&participant.SchoolID,
			&participant.Grade,
			&participant.Letter,
			&participant.StudentID,
			&participant.EventsName,
			&participant.Document,
			&participant.Category,
			&participant.Role,
			&participant.Date,
			&participant.StudentID,
			&participant.StudentName,
			&participant.StudentLastName,
			&participant.SchoolName,
			&participant.CreatorID,
			&participant.CreatorFirstName,
			&participant.CreatorLastName,
		)

		if err != nil {
			log.Printf("Error fetching updated participant for events_id %d: %v", eventsIDInt, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Event participant updated, but failed to retrieve updated details"})
			return
		}

		// Step 14: Return updated participant data
		log.Printf("Successfully updated and retrieved participant ID %d for user ID %d", eventsIDInt, userID)
		utils.ResponseJSON(w, participant)
	}
}
func (c *EventsParticipantController) GetEventsParticipant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Printf("Error fetching user role for user ID %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check access permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			log.Printf("Access denied for user ID %d with role %s", userID, userRole)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view event participants"})
			return
		}

		// Step 4: Build query with optional filters
		query := `SELECT ep.id, ep.school_id, ep.grade, ep.letter, ep.student_id, ep.events_name, 
                        ep.document, ep.category, ep.role, ep.date, 
                        s.student_id, s.first_name as student_name, s.last_name as student_lastname,
                        sch.school_name as school_name,
                        c.id as creator_id, c.first_name as creator_first_name, c.last_name as creator_last_name
                 FROM events_participants ep
                 JOIN student s ON ep.student_id = s.student_id
                 JOIN Schools sch ON ep.school_id = sch.school_id
                 JOIN users c ON ep.creator_id = c.id`

		var args []interface{}
		conditions := []string{}

		// Optional filter by creator_id (e.g., ?creator_id=112)
		if creatorIDStr := r.URL.Query().Get("creator_id"); creatorIDStr != "" {
			creatorID, err := strconv.Atoi(creatorIDStr)
			if err != nil || creatorID <= 0 {
				log.Printf("Invalid creator_id format: %s, error: %v", creatorIDStr, err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid creator_id format"})
				return
			}
			conditions = append(conditions, "ep.creator_id = ?")
			args = append(args, creatorID)
			log.Printf("Filtering by creator_id: %d", creatorID)
		}

		// Optional filter by school_id
		if schoolIDStr := r.URL.Query().Get("school_id"); schoolIDStr != "" {
			schoolID, err := strconv.Atoi(schoolIDStr)
			if err != nil || schoolID <= 0 {
				log.Printf("Invalid school_id format: %s, error: %v", schoolIDStr, err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
				return
			}
			conditions = append(conditions, "ep.school_id = ?")
			args = append(args, schoolID)
			log.Printf("Filtering by school_id: %d", schoolID)
		}

		// Append conditions to query
		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
		}

		// Step 5: Execute query
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Error querying participants: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event participants"})
			return
		}
		defer rows.Close()

		// Step 6: Collect participants
		var participants []models.EventsParticipant
		for rows.Next() {
			var p models.EventsParticipant
			err := rows.Scan(
				&p.ID,
				&p.SchoolID,
				&p.Grade,
				&p.Letter,
				&p.StudentID,
				&p.EventsName,
				&p.Document,
				&p.Category,
				&p.Role,
				&p.Date,
				&p.StudentID,
				&p.StudentName,
				&p.StudentLastName,
				&p.SchoolName,
				&p.CreatorID,
				&p.CreatorFirstName,
				&p.CreatorLastName,
			)
			if err != nil {
				log.Printf("Error scanning participant row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to process participant data"})
				return
			}
			participants = append(participants, p)
		}

		// Check for errors during iteration
		if err = rows.Err(); err != nil {
			log.Printf("Error during row iteration: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event participants"})
			return
		}

		// Step 7: Return participants
		if len(participants) == 0 {
			log.Printf("No participants found for user ID %d with query: %s", userID, query)
			utils.ResponseJSON(w, []models.EventsParticipant{})
			return
		}

		log.Printf("Successfully retrieved %d participants for user ID %d", len(participants), userID)
		utils.ResponseJSON(w, participants)
	}
}
func (c *EventsParticipantController) GetSingleEventsParticipant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Printf("Error fetching user role for user ID %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check access permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			log.Printf("Access denied for user ID %d with role %s", userID, userRole)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view event participants"})
			return
		}

		// Step 4: Get and validate events_id from URL
		vars := mux.Vars(r)
		eventsID, ok := vars["events_id"]
		if !ok {
			log.Printf("Missing events_id parameter in request URL: %s", r.URL.String())
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing events_id parameter"})
			return
		}
		eventsIDInt, err := strconv.Atoi(eventsID)
		if err != nil || eventsIDInt <= 0 {
			log.Printf("Invalid events_id format: %s, error: %v", eventsID, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid events_id format"})
			return
		}

		// Step 5: Retrieve participant data
		var participant models.EventsParticipant
		query := `SELECT ep.id, ep.school_id, ep.grade, ep.letter, ep.student_id, ep.events_name, 
                        ep.document, ep.category, ep.role, ep.date, 
                        s.student_id, s.first_name as student_name, s.last_name as student_lastname,
                        sch.school_name as school_name,
                        c.id as creator_id, c.first_name as creator_first_name, c.last_name as creator_last_name
                 FROM events_participants ep
                 JOIN student s ON ep.student_id = s.student_id
                 JOIN Schools sch ON ep.school_id = sch.school_id
                 JOIN users c ON ep.creator_id = c.id
                 WHERE ep.id = ?`

		err = db.QueryRow(query, eventsIDInt).Scan(
			&participant.ID,
			&participant.SchoolID,
			&participant.Grade,
			&participant.Letter,
			&participant.StudentID,
			&participant.EventsName,
			&participant.Document,
			&participant.Category,
			&participant.Role,
			&participant.Date,
			&participant.StudentID,
			&participant.StudentName,
			&participant.StudentLastName,
			&participant.SchoolName,
			&participant.CreatorID,
			&participant.CreatorFirstName,
			&participant.CreatorLastName,
		)

		if err == sql.ErrNoRows {
			log.Printf("No participant found for events_id %d", eventsIDInt)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event participant not found"})
			return
		}
		if err != nil {
			log.Printf("Error fetching participant for events_id %d: %v", eventsIDInt, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event participant"})
			return
		}

		// Step 6: Return participant data
		log.Printf("Successfully retrieved participant ID %d for user ID %d", eventsIDInt, userID)
		utils.ResponseJSON(w, participant)
	}
}
func (c *EventsParticipantController) GetEventsParticipantBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Printf("Error fetching user role for user ID %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check access permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			log.Printf("Access denied for user ID %d with role %s", userID, userRole)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view event participants"})
			return
		}

		// Step 4: Get and validate school_id from URL
		vars := mux.Vars(r)
		schoolID, ok := vars["school_id"]
		if !ok {
			log.Printf("Missing school_id parameter in request URL: %s", r.URL.String())
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing school_id parameter"})
			return
		}
		schoolIDInt, err := strconv.Atoi(schoolID)
		if err != nil || schoolIDInt <= 0 {
			log.Printf("Invalid school_id format: %s, error: %v", schoolID, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
			return
		}

		// Step 5: Build query for participants by school_id
		query := `SELECT ep.id, ep.school_id, ep.grade, ep.letter, ep.student_id, ep.events_name, 
				ep.document, ep.category, ep.role, ep.date, 
				s.student_id, s.first_name as student_name, s.last_name as student_lastname,
				sch.school_name as school_name,
				c.id as creator_id, c.first_name as creator_first_name, c.last_name as creator_last_name
			FROM events_participants ep
			JOIN student s ON ep.student_id = s.student_id
			JOIN Schools sch ON ep.school_id = sch.school_id
			JOIN users c ON ep.creator_id = c.id
			WHERE ep.school_id = ?`

		// Step 6: Execute query
		rows, err := db.Query(query, schoolIDInt)
		if err != nil {
			log.Printf("Error querying participants by school_id %d: %v", schoolIDInt, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event participants"})
			return
		}
		defer rows.Close()

		// Step 7: Collect participants
		var participants []models.EventsParticipant
		for rows.Next() {
			var p models.EventsParticipant
			err := rows.Scan(
				&p.ID,
				&p.SchoolID,
				&p.Grade,
				&p.Letter,
				&p.StudentID,
				&p.EventsName,
				&p.Document,
				&p.Category,
				&p.Role,
				&p.Date,
				&p.StudentID,
				&p.StudentName,
				&p.StudentLastName,
				&p.SchoolName,
				&p.CreatorID,
				&p.CreatorFirstName,
				&p.CreatorLastName,
			)
			if err != nil {
				log.Printf("Error scanning participant row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to process participant data"})
				return
			}
			participants = append(participants, p)
		}

		// Check for errors during iteration
		if err = rows.Err(); err != nil {
			log.Printf("Error during row iteration: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event participants"})
			return
		}

		// Step 8: Return participants
		if len(participants) == 0 {
			log.Printf("No participants found for school_id %d", schoolIDInt)
			utils.ResponseJSON(w, []models.EventsParticipant{})
			return
		}

		log.Printf("Successfully retrieved %d participants for school_id %d", len(participants), schoolIDInt)
		utils.ResponseJSON(w, participants)
	}
}
func (c *EventsParticipantController) CountOlympiadParticipants(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the token
		userID, err := utils.VerifyToken(r) // userID and err are defined here
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole) // err redefined in this scope
		if err != nil {
			log.Printf("Error fetching user role for user ID %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check access permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			log.Printf("Access denied for user ID %d with role %s", userID, userRole)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view olympiad statistics"})
			return
		}

		// Step 4: Build query with optional filters
		query := `SELECT COUNT(ep.student_id) AS total_participants`
		countByEventQuery := `SELECT 
            ep.events_name, 
            COUNT(ep.student_id) AS participant_count 
        FROM events_participants ep`

		var args []interface{}
		conditions := []string{}

		// Optional filter by school_id
		if schoolIDStr := r.URL.Query().Get("school_id"); schoolIDStr != "" {
			schoolID, err := strconv.Atoi(schoolIDStr) // err redefined in this scope
			if err != nil || schoolID <= 0 {
				log.Printf("Invalid school_id format: %s, error: %v", schoolIDStr, err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
				return
			}
			conditions = append(conditions, "ep.school_id = ?")
			args = append(args, schoolID)
			log.Printf("Filtering by school_id: %d", schoolID)
		}

		// If user is schooladmin, restrict to their school
		if userRole == "schooladmin" {
			var schoolID int
			err = db.QueryRow("SELECT school_id FROM school_admins WHERE user_id = ?", userID).Scan(&schoolID) // err redefined
			if err != nil {
				log.Printf("Error fetching school_id for schooladmin user ID %d: %v", userID, err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school details"})
				return
			}
			conditions = append(conditions, "ep.school_id = ?")
			args = append(args, schoolID)
			log.Printf("Restricting schooladmin to school_id: %d", schoolID)
		}

		// Optional filter by category
		if category := r.URL.Query().Get("category"); category != "" {
			conditions = append(conditions, "ep.category = ?")
			args = append(args, category)
			log.Printf("Filtering by category: %s", category)
		}

		// Optional filter by event name
		if eventsName := r.URL.Query().Get("events_name"); eventsName != "" {
			conditions = append(conditions, "ep.events_name = ?")
			args = append(args, eventsName)
			log.Printf("Filtering by events_name: %s", eventsName)
		}

		// Optional filter by date range
		if startDate := r.URL.Query().Get("start_date"); startDate != "" {
			conditions = append(conditions, "ep.date >= ?")
			args = append(args, startDate)
		}

		if endDate := r.URL.Query().Get("end_date"); endDate != "" {
			conditions = append(conditions, "ep.date <= ?")
			args = append(args, endDate)
		}

		// Complete the queries with FROM clause and conditions
		query += " FROM events_participants ep"
		countByEventQuery += " FROM events_participants ep"

		// Append conditions to queries
		if len(conditions) > 0 {
			whereClause := " WHERE " + strings.Join(conditions, " AND ")
			query += whereClause
			countByEventQuery += whereClause
		}

		// Complete the group by for the second query
		countByEventQuery += " GROUP BY ep.events_name ORDER BY participant_count DESC"

		// Response structure
		type OlympiadStats struct {
			TotalParticipants int `json:"total_participants"`
			EventBreakdown    []struct {
				EventName        string `json:"event_name"`
				ParticipantCount int    `json:"participant_count"`
			} `json:"event_breakdown,omitempty"`
		}

		// Initialize response
		stats := OlympiadStats{}

		// Step 5: Execute the total count query
		err = db.QueryRow(query, args...).Scan(&stats.TotalParticipants) // err redefined
		if err != nil {
			log.Printf("Error counting participants: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to count olympiad participants"})
			return
		}

		// Check if we should include breakdown by event
		includeBreakdown := r.URL.Query().Get("include_breakdown")
		if includeBreakdown == "true" {
			// Execute the breakdown query
			rows, err := db.Query(countByEventQuery, args...) // err redefined
			if err != nil {
				log.Printf("Error querying event breakdown: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event breakdown"})
				return
			}
			defer rows.Close()

			// Process event breakdown results
			for rows.Next() {
				var eventName string
				var count int
				err := rows.Scan(&eventName, &count) // err redefined
				if err != nil {
					log.Printf("Error scanning event breakdown row: %v", err)
					continue
				}
				stats.EventBreakdown = append(stats.EventBreakdown, struct {
					EventName        string `json:"event_name"`
					ParticipantCount int    `json:"participant_count"`
				}{
					EventName:        eventName,
					ParticipantCount: count,
				})
			}

			// Check for errors during iteration
			if err = rows.Err(); err != nil {
				log.Printf("Error during row iteration for event breakdown: %v", err)
				// Continue with the total count result anyway
			}
		}

		// Return the statistics
		log.Printf("Successfully counted olympiad participants: %d for user ID %d", stats.TotalParticipants, userID)
		utils.ResponseJSON(w, stats)
	}
}
func (c *EventsParticipantController) CountOlympiadParticipantsBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Step 2: Get user role and school_id (if schooladmin)
		var userRole string
		var userSchoolID sql.NullInt64 // Use NullInt64 to handle NULL school_id
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Printf("Error fetching user details for user ID %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check access permissions
		if userRole != "superadmin" && userRole != "schooladmin" {
			log.Printf("Access denied for user ID %d with role %s", userID, userRole)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view olympiad statistics"})
			return
		}

		// Step 4: Extract school_id from URL
		vars := mux.Vars(r)
		schoolIDParam := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDParam)
		if err != nil || schoolID <= 0 {
			log.Printf("Invalid school_id format: %s, error: %v", schoolIDParam, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
			return
		}

		// Step 5: Restrict schooladmin to their school
		if userRole == "schooladmin" {
			if !userSchoolID.Valid {
				log.Printf("No school_id associated with schooladmin user ID %d", userID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "No school assigned to this admin"})
				return
			}
			if int(userSchoolID.Int64) != schoolID {
				log.Printf("Schooladmin user ID %d attempted to access school ID %d, but is assigned to school ID %d", userID, schoolID, userSchoolID.Int64)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view this school's data"})
				return
			}
		}

		// Step 6: Build queries with mandatory school_id filter
		query := `SELECT COUNT(ep.student_id) AS total_participants`
		eventCountQuery := `SELECT COUNT(DISTINCT ep.events_name) AS event_count`
		countByEventQuery := `SELECT 
            ep.events_name, 
            COUNT(ep.student_id) AS participant_count 
        FROM events_participants ep`

		var args []interface{}
		conditions := []string{"ep.school_id = ?"} // Mandatory school_id filter
		args = append(args, schoolID)

		// Optional filter by category
		if category := r.URL.Query().Get("category"); category != "" {
			conditions = append(conditions, "ep.category = ?")
			args = append(args, category)
			log.Printf("Filtering by category: %s", category)
		}

		// Optional filter by event name
		if eventsName := r.URL.Query().Get("events_name"); eventsName != "" {
			conditions = append(conditions, "ep.events_name = ?")
			args = append(args, eventsName)
			log.Printf("Filtering by events_name: %s", eventsName)
		}

		// Optional filter by date range
		if startDate := r.URL.Query().Get("start_date"); startDate != "" {
			conditions = append(conditions, "ep.date >= ?")
			args = append(args, startDate)
			log.Printf("Filtering by start_date: %s", startDate)
		}

		if endDate := r.URL.Query().Get("end_date"); endDate != "" {
			conditions = append(conditions, "ep.date <= ?")
			args = append(args, endDate)
			log.Printf("Filtering by end_date: %s", endDate)
		}

		// Complete the queries with FROM clause and conditions
		query += " FROM events_participants ep"
		eventCountQuery += " FROM events_participants ep"
		countByEventQuery += " FROM events_participants ep"

		// Append conditions to queries
		if len(conditions) > 0 {
			whereClause := " WHERE " + strings.Join(conditions, " AND ")
			query += whereClause
			eventCountQuery += whereClause
			countByEventQuery += whereClause
		}

		// Complete the group by for the event breakdown query
		countByEventQuery += " GROUP BY ep.events_name ORDER BY participant_count DESC"

		// Response structure
		type OlympiadStats struct {
			TotalEvents       int `json:"total_events"`
			TotalParticipants int `json:"total_participants"`
			EventBreakdown    []struct {
				EventName        string `json:"event_name"`
				ParticipantCount int    `json:"participant_count"`
			} `json:"event_breakdown,omitempty"`
		}

		// Initialize response
		stats := OlympiadStats{}

		// Step 7: Execute the event count query
		err = db.QueryRow(eventCountQuery, args...).Scan(&stats.TotalEvents)
		if err != nil {
			log.Printf("Error counting events for school ID %d: %v", schoolID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to count events"})
			return
		}

		// Step 8: Execute the participant count query
		err = db.QueryRow(query, args...).Scan(&stats.TotalParticipants)
		if err != nil {
			log.Printf("Error counting participants for school ID %d: %v", schoolID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to count participants"})
			return
		}

		// Step 9: Check if we should include breakdown by event
		includeBreakdown := r.URL.Query().Get("include_breakdown")
		if includeBreakdown == "true" {
			rows, err := db.Query(countByEventQuery, args...)
			if err != nil {
				log.Printf("Error querying event breakdown for school ID %d: %v", schoolID, err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event breakdown"})
				return
			}
			defer rows.Close()

			for rows.Next() {
				var eventName string
				var count int
				err := rows.Scan(&eventName, &count)
				if err != nil {
					log.Printf("Error scanning event breakdown row: %v", err)
					continue
				}
				stats.EventBreakdown = append(stats.EventBreakdown, struct {
					EventName        string `json:"event_name"`
					ParticipantCount int    `json:"participant_count"`
				}{
					EventName:        eventName,
					ParticipantCount: count,
				})
			}

			if err = rows.Err(); err != nil {
				log.Printf("Error during row iteration for event breakdown: %v", err)
			}
		}

		// Step 10: Return the statistics
		log.Printf("Successfully counted %d events and %d participants for school ID %d, user ID %d", stats.TotalEvents, stats.TotalParticipants, schoolID, userID)
		utils.ResponseJSON(w, stats)
	}
}

func (c *EventsParticipantController) GetParticipantByEventNameAndStudentID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		vars := mux.Vars(r)
		eventsName := vars["events_name"]
		studentID, err2 := strconv.Atoi(vars["student_id"])
		if err2 != nil || eventsName == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid parameters"})
			return
		}

		query := `
			SELECT ep.id, ep.school_id, ep.grade, ep.letter, ep.student_id, ep.events_name,
			       ep.document, ep.category, ep.role, ep.date,
			       s.first_name as student_name, s.last_name as student_lastname,
			       sch.school_name,
			       c.id as creator_id, c.first_name as creator_first_name, c.last_name as creator_last_name
			FROM events_participants ep
			JOIN student s ON ep.student_id = s.student_id
			JOIN Schools sch ON ep.school_id = sch.school_id
			JOIN users c ON ep.creator_id = c.id
			WHERE ep.events_name = ? AND ep.student_id = ?
		`

		var participant models.EventsParticipant
		err = db.QueryRow(query, eventsName, studentID).Scan(
			&participant.ID,
			&participant.SchoolID,
			&participant.Grade,
			&participant.Letter,
			&participant.StudentID,
			&participant.EventsName,
			&participant.Document,
			&participant.Category,
			&participant.Role,
			&participant.Date,
			&participant.StudentName,
			&participant.StudentLastName,
			&participant.SchoolName,
			&participant.CreatorID,
			&participant.CreatorFirstName,
			&participant.CreatorLastName,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Participant not found"})
			} else {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			}
			return
		}

		utils.ResponseJSON(w, participant)
	}
}
