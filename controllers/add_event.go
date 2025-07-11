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

		// Parse limit and limit_participants
		limitStr := r.FormValue("limit")
		if limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid limit format"})
				return
			}
			event.Limit = limit
		}
		limitParticipantsStr := r.FormValue("limit_participants")
		if limitParticipantsStr != "" {
			limitParticipants, err := strconv.Atoi(limitParticipantsStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid limit_participants format"})
				return
			}
			event.LimitParticipants = limitParticipants
		}

		// Parse participants (default to 0 if not provided)
		participantsStr := r.FormValue("participants")
		if participantsStr != "" {
			participants, err := strconv.Atoi(participantsStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid participants format"})
				return
			}
			if participants < 0 {
				participants = 0
			}
			event.Participants = participants
		} else {
			event.Participants = 0 // Default to 0 if not provided
		}

		// Parse and validate category (previously type)
		event.Category = strings.TrimSpace(r.FormValue("category"))
		allowedCategories := []string{"Science", "Humanities", "Sport", "Creative"}
		validCategory := false
		for _, c := range allowedCategories {
			if event.Category == c {
				validCategory = true
				break
			}
		}
		if event.Category == "" || !validCategory {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Invalid or missing category. Allowed values are: Science, Humanities, Sport, Creative",
			})
			return
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
                start_date, end_date, location, 
                grade, limit_count, participants, limit_participants, created_at, updated_at, created_by, category
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			event.SchoolID, event.UserID, event.EventName, event.Description, event.Photo,
			event.StartDate, event.EndDate, event.Location,
			event.Grade, event.Limit, event.Participants, event.LimitParticipants, now, now, event.CreatedBy, event.Category,
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

// func (ec *EventController) GetEvents(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Check that GET method is used
// 		if r.Method != http.MethodGet {
// 			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
// 			return
// 		}

// 		// Get query parameters
// 		query := r.URL.Query()

// 		// Create map to store all parameters
// 		params := make(map[string]string)

// 		// Collect main parameters
// 		params["id"] = query.Get("id")
// 		params["school_id"] = query.Get("school_id")
// 		params["grade"] = query.Get("grade")
// 		params["date_from"] = query.Get("date_from")
// 		params["date_to"] = query.Get("date_to")
// 		params["limit"] = query.Get("limit")
// 		params["offset"] = query.Get("offset")
// 		params["category"] = query.Get("category")
// 		params["location"] = query.Get("location")

// 		// Add all other parameters
// 		for key, values := range query {
// 			if _, exists := params[key]; !exists && len(values) > 0 {
// 				params[key] = values[0]
// 			}
// 		}

// 		// Log all parameters
// 		log.Println("GetEvents called with parameters:", params)

// 		// If debug mode is requested, return all parameters
// 		if query.Get("debug") == "true" {
// 			utils.ResponseJSON(w, map[string]interface{}{
// 				"message":    "Debug mode: showing all parameters",
// 				"parameters": params,
// 			})
// 			return
// 		}

// 		// Variables for main parameters
// 		eventID := params["id"]
// 		schoolID := params["school_id"]
// 		grade := params["grade"]
// 		dateFrom := params["date_from"]
// 		dateTo := params["date_to"]
// 		limit := params["limit"]
// 		offset := params["offset"]
// 		eventCategory := params["category"]
// 		location := params["location"]

// 		// If a specific event ID is provided, fetch only that event
// 		if eventID != "" {
// 			id, err := strconv.Atoi(eventID)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID format"})
// 				return
// 			}
// 			event, err := getEventByID(db, id)
// 			if err != nil {
// 				if err == sql.ErrNoRows {
// 					utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
// 				} else {
// 					log.Println("Error fetching event by ID:", err)
// 					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching event"})
// 				}
// 				return
// 			}

// 			// Form response without parameters_used
// 			response := map[string]interface{}{
// 				"event": event,
// 			}
// 			utils.ResponseJSON(w, response)
// 			return
// 		}

// 		// Build query for fetching list of events
// 		queryBuilder := strings.Builder{}
// 		queryBuilder.WriteString(`
//             SELECT e.id, e.school_id, e.user_id, e.event_name, e.description,
//             e.photo, e.start_date, e.end_date, e.location,
//             e.grade, e.limit_count as ` + "`limit`" + `, e.participants, e.limit_participants, e.created_at, e.updated_at,
//             u.email AS created_by, e.category, s.school_name
//             FROM Events e
//             LEFT JOIN users u ON e.user_id = u.id
//             LEFT JOIN Schools s ON e.school_id = s.school_id
//             WHERE 1=1
//         `)

// 		var args []interface{}

// 		// Add filters if provided
// 		if schoolID != "" {
// 			schoolIDInt, err := strconv.Atoi(schoolID)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
// 				return
// 			}
// 			queryBuilder.WriteString(" AND e.school_id = ?")
// 			args = append(args, schoolIDInt)
// 		}

// 		if grade != "" {
// 			gradeInt, err := strconv.Atoi(grade)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade format"})
// 				return
// 			}
// 			queryBuilder.WriteString(" AND e.grade = ?")
// 			args = append(args, gradeInt)
// 		}

// 		// Filter by category
// 		if eventCategory != "" {
// 			allowedCategories := []string{"Science", "Humanities", "Sport", "Creative"}
// 			validCategory := false
// 			for _, c := range allowedCategories {
// 				if eventCategory == c {
// 					validCategory = true
// 					break
// 				}
// 			}
// 			if !validCategory {
// 				utils.RespondWithError(w, http.StatusBadRequest, models.Error{
// 					Message: "Invalid category. Allowed values are: Science, Humanities, Sport, Creative",
// 				})
// 				return
// 			}
// 			queryBuilder.WriteString(" AND e.category = ?")
// 			args = append(args, eventCategory)
// 		}

// 		// Filter by location (case-insensitive partial match)
// 		if location != "" {
// 			queryBuilder.WriteString(" AND LOWER(e.location) LIKE ?")
// 			args = append(args, "%"+strings.ToLower(location)+"%")
// 		}

// 		// Filter by dates (start_date)
// 		if dateFrom != "" {
// 			_, err := time.Parse("2006-01-02", dateFrom)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_from format. Use YYYY-MM-DD"})
// 				return
// 			}
// 			queryBuilder.WriteString(" AND e.start_date >= ?")
// 			args = append(args, dateFrom)
// 		}

// 		if dateTo != "" {
// 			_, err := time.Parse("2006-01-02", dateTo)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_to format. Use YYYY-MM-DD"})
// 				return
// 			}
// 			queryBuilder.WriteString(" AND e.end_date <= ?")
// 			args = append(args, dateTo)
// 		}

// 		// Add sorting by start_date
// 		queryBuilder.WriteString(" ORDER BY e.start_date ASC")

// 		// Add pagination
// 		if limit != "" {
// 			limitInt, err := strconv.Atoi(limit)
// 			if err != nil || limitInt <= 0 {
// 				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid limit parameter"})
// 				return
// 			}
// 			queryBuilder.WriteString(" LIMIT ?")
// 			args = append(args, limitInt)

// 			if offset != "" {
// 				offsetInt, err := strconv.Atoi(offset)
// 				if err != nil || offsetInt < 0 {
// 					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid offset parameter"})
// 					return
// 				}
// 				queryBuilder.WriteString(" OFFSET ?")
// 				args = append(args, offsetInt)
// 			}
// 		}

// 		// Log final SQL query
// 		finalQuery := queryBuilder.String()
// 		log.Printf("Executing SQL query: %s with args: %v", finalQuery, args)

// 		// Execute query
// 		rows, err := db.Query(finalQuery, args...)
// 		if err != nil {
// 			log.Println("Error executing events query:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch events"})
// 			return
// 		}
// 		defer rows.Close()

// 		// Collect results
// 		var events []models.Event
// 		for rows.Next() {
// 			var event models.Event
// 			err := rows.Scan(
// 				&event.ID, &event.SchoolID, &event.UserID, &event.EventName, &event.Description,
// 				&event.Photo, &event.StartDate, &event.EndDate, &event.Location,
// 				&event.Grade, &event.Limit, &event.Participants, &event.LimitParticipants,
// 				&event.CreatedAt, &event.UpdatedAt, &event.CreatedBy, &event.Category, &event.SchoolName,
// 			)
// 			if err != nil {
// 				log.Println("Error scanning event row:", err)
// 				continue
// 			}
// 			events = append(events, event)
// 		}

// 		if err = rows.Err(); err != nil {
// 			log.Println("Error iterating event rows:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing events data"})
// 			return
// 		}

// 		// Prepare response
// 		response := map[string]interface{}{
// 			"events":      events,
// 			"total_count": len(events),
// 		}

// 		if len(events) == 0 {
// 			response["message"] = "No events found for the specified criteria"
// 		}

// 		utils.ResponseJSON(w, response)
// 	}
// }

func (ec *EventController) GetEvents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that GET method is used
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Get query parameters
		query := r.URL.Query()

		// Create map to store all parameters
		params := make(map[string]string)

		// Collect main parameters
		params["id"] = query.Get("id")
		params["school_id"] = query.Get("school_id")
		params["grade"] = query.Get("grade")
		params["date_from"] = query.Get("date_from")
		params["date_to"] = query.Get("date_to")
		params["limit"] = query.Get("limit")
		params["offset"] = query.Get("offset")
		params["category"] = query.Get("category")
		params["location"] = query.Get("location")

		// Add all other parameters
		for key, values := range query {
			if _, exists := params[key]; !exists && len(values) > 0 {
				params[key] = values[0]
			}
		}

		// Log all parameters
		log.Println("GetEvents called with parameters:", params)

		// If debug mode is requested, return all parameters
		if query.Get("debug") == "true" {
			utils.ResponseJSON(w, map[string]interface{}{
				"message":    "Debug mode: showing all parameters",
				"parameters": params,
			})
			return
		}

		// Variables for main parameters
		eventID := params["id"]
		schoolID := params["school_id"]
		grade := params["grade"]
		dateFrom := params["date_from"]
		dateTo := params["date_to"]
		limit := params["limit"]
		offset := params["offset"]
		eventCategory := params["category"]
		location := params["location"]

		// If a specific event ID is provided, fetch only that event
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

			// Fetch participants count directly from EventRegistrations table
			var participantsCount int
			err = db.QueryRow("SELECT COUNT(*) FROM EventRegistrations WHERE event_id = ?", event.ID).Scan(&participantsCount)
			if err != nil {
				log.Println("Error counting participants:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error counting participants"})
				return
			}

			// Form response with participants count
			response := map[string]interface{}{
				"event":        event,
				"participants": participantsCount, // Return the participants count
			}
			utils.ResponseJSON(w, response)
			return
		}

		// Build query for fetching list of events
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString(`
            SELECT e.id, e.school_id, e.user_id, e.event_name, e.description, 
            e.photo, e.start_date, e.end_date, e.location, 
            e.grade, e.limit_count as ` + "`limit`" + `, e.participants, e.limit_participants, e.created_at, e.updated_at, 
            u.email AS created_by, e.category, s.school_name
            FROM Events e
            LEFT JOIN users u ON e.user_id = u.id
            LEFT JOIN Schools s ON e.school_id = s.school_id
            WHERE 1=1
        `)

		var args []interface{}

		// Add filters if provided
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

		// Filter by category
		if eventCategory != "" {
			allowedCategories := []string{"Science", "Humanities", "Sport", "Creative"}
			validCategory := false
			for _, c := range allowedCategories {
				if eventCategory == c {
					validCategory = true
					break
				}
			}
			if !validCategory {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{
					Message: "Invalid category. Allowed values are: Science, Humanities, Sport, Creative",
				})
				return
			}
			queryBuilder.WriteString(" AND e.category = ?")
			args = append(args, eventCategory)
		}

		// Filter by location (case-insensitive partial match)
		if location != "" {
			queryBuilder.WriteString(" AND LOWER(e.location) LIKE ?")
			args = append(args, "%"+strings.ToLower(location)+"%")
		}

		// Filter by dates (start_date)
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

		// Add sorting by start_date
		queryBuilder.WriteString(" ORDER BY e.start_date ASC")

		// Add pagination
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

		// Log final SQL query
		finalQuery := queryBuilder.String()
		log.Printf("Executing SQL query: %s with args: %v", finalQuery, args)

		// Execute query
		rows, err := db.Query(finalQuery, args...)
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
				&event.Photo, &event.StartDate, &event.EndDate, &event.Location,
				&event.Grade, &event.Limit, &event.Participants, &event.LimitParticipants,
				&event.CreatedAt, &event.UpdatedAt, &event.CreatedBy, &event.Category, &event.SchoolName,
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

		err = db.QueryRow("SELECT id, school_id, user_id, photo, category FROM Events WHERE id = ?", eventID).Scan(
			&existingEvent.ID, &eventSchoolID, &eventCreatorID, &currentPhotoURL, &existingEvent.Category,
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

		if location := r.FormValue("location"); location != "" {
			updateFields["location"] = location
			updatedEvent.Location = location
		}

		// Обработка категории (previously type)
		if eventCategory := r.FormValue("category"); eventCategory != "" {
			allowedCategories := []string{"Science", "Humanities", "Sport", "Creative"}
			validCategory := false
			for _, c := range allowedCategories {
				if eventCategory == c {
					validCategory = true
					break
				}
			}
			if !validCategory {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{
					Message: "Invalid category. Allowed values are: Science, Humanities, Sport, Creative",
				})
				return
			}
			updateFields["category"] = eventCategory
			updatedEvent.Category = eventCategory
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
            photo, start_date, end_date, location, 
            grade, limit_count as limit, participants, limit_participants, created_at, updated_at, created_by, category,
            FROM Events 
            WHERE id = ?
        `, eventID).Scan(
			&updated.ID, &updated.SchoolID, &updated.UserID, &updated.EventName, &updated.Description,
			&updated.Photo, &updated.StartDate, &updated.EndDate, &updated.Location,
			&updated.Grade, &updated.Limit, &updated.Participants, &updated.LimitParticipants,
			&updated.CreatedAt, &updated.UpdatedAt, &updated.CreatedBy, &updated.Category,
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
		// Verify user authentication
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Println("Authentication error:", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Authentication required"})
			return
		}

		// Retrieve user details
		user, err := utils.GetUserByID(db, userID)
		if err != nil {
			log.Println("Error retrieving user details:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving user information"})
			return
		}

		// Extract school_id from URL parameters
		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
			return
		}

		// Check if school exists
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

		// Authorization checks
		if user.Role == "schooladmin" {
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

		// Parse query parameters
		query := r.URL.Query()
		params := map[string]string{
			"grade":     query.Get("grade"),
			"date_from": query.Get("date_from"),
			"date_to":   query.Get("date_to"),
			"limit":     query.Get("limit"),
			"offset":    query.Get("offset"),
			"category":  query.Get("category"),
		}

		// Build SQL query
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString(`
			SELECT e.id, e.school_id, e.user_id, e.event_name, e.description,
			e.photo, e.start_date, e.end_date, e.location,
			e.grade, e.limit_count as ` + "`limit`" + `, e.participants, e.limit_participants,
			e.created_at, e.updated_at, e.created_by, e.category, s.school_name
			FROM Events e
			LEFT JOIN users u ON e.user_id = u.id
			LEFT JOIN Schools s ON e.school_id = s.school_id
			WHERE e.school_id = ?
		`)

		args := []interface{}{schoolID}

		// Handle grade filter
		if params["grade"] != "" {
			gradeInt, err := strconv.Atoi(params["grade"])
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade format"})
				return
			}
			queryBuilder.WriteString(" AND e.grade = ?")
			args = append(args, gradeInt)
		}

		// Handle date_from filter
		if params["date_from"] != "" {
			_, err := time.Parse("2006-01-02", params["date_from"])
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_from format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.start_date >= ?")
			args = append(args, params["date_from"])
		}

		// Handle date_to filter
		if params["date_to"] != "" {
			_, err := time.Parse("2006-01-02", params["date_to"])
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_to format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND e.end_date <= ?")
			args = append(args, params["date_to"])
		}

		// Handle category filter
		if params["category"] != "" {
			allowedCategories := []string{"Science", "Humanities", "Sport", "Creative"}
			validCategory := false
			for _, c := range allowedCategories {
				if params["category"] == c {
					validCategory = true
					break
				}
			}
			if !validCategory {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{
					Message: "Invalid category. Allowed values are: Science, Humanities, Sport, Creative",
				})
				return
			}
			queryBuilder.WriteString(" AND e.category = ?")
			args = append(args, params["category"])
		}

		// Order by start_date
		queryBuilder.WriteString(" ORDER BY e.start_date ASC")

		// Handle limit and offset
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

		// Log the query for debugging
		// log.Printf("Executing query: %s\nWith args: %v\n", queryBuilder.String(), args)

		// Execute query
		rows, err := db.Query(queryBuilder.String(), args...)
		if err != nil {
			log.Println("Error executing events query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch events"})
			return
		}
		defer rows.Close()

		// Scan query results
		var events []models.Event
		for rows.Next() {
			var event models.Event
			err := rows.Scan(
				&event.ID, &event.SchoolID, &event.UserID, &event.EventName, &event.Description,
				&event.Photo, &event.StartDate, &event.EndDate, &event.Location,
				&event.Grade, &event.Limit, &event.Participants, &event.LimitParticipants,
				&event.CreatedAt, &event.UpdatedAt, &event.CreatedBy, &event.Category, &event.SchoolName,
			)
			if err != nil {
				log.Println("Error scanning event row:", err)
				continue
			}
			events = append(events, event)
		}

		// Check for errors during row iteration
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

		// Send JSON response
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

		// Handle status filter (using start_date and end_date to determine status)
		if status := params["status"]; status != "" {
			// Validate status value
			if status != "Upcoming" && status != "Completed" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status value: must be 'Upcoming' or 'Completed'"})
				return
			}
			// Use current timestamp to determine event status
			currentTime := time.Now().Format("2006-01-02 15:04:05")
			if status == "Upcoming" {
				queryBuilder.WriteString(" AND end_date > ?")
				args = append(args, currentTime)
			} else if status == "Completed" {
				queryBuilder.WriteString(" AND end_date <= ?")
				args = append(args, currentTime)
			}
		}

		// Filter by date range
		if dateFrom := params["date_from"]; dateFrom != "" {
			// Validate date format
			_, err := time.Parse("2006-01-02", dateFrom)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_from format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND start_date >= ?")
			args = append(args, dateFrom)
		}

		if dateTo := params["date_to"]; dateTo != "" {
			// Validate date format
			_, err := time.Parse("2006-01-02", dateTo)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date_to format. Use YYYY-MM-DD"})
				return
			}
			queryBuilder.WriteString(" AND end_date <= ?")
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
func (ec *EventController) GetEventsBySchoolAndType(db *sql.DB) http.HandlerFunc {
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
			"date_from": query.Get("date_from"),
			"date_to":   query.Get("date_to"),
			"limit":     query.Get("limit"),
			"offset":    query.Get("offset"),
			"category":  query.Get("category"), // Renamed from type
		}

		// Build the SQL query
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString(`
            SELECT e.id, e.school_id, e.user_id, e.event_name, e.description, 
            e.photo, e.participants, e.limit_count as limit, e.limit_participants, e.start_date, e.end_date, 
            e.location, e.grade, e.created_at, e.updated_at, u.email AS created_by, e.category, s.school_name
            FROM Events e
            LEFT JOIN users u ON e.user_id = u.id
            LEFT JOIN Schools s ON e.school_id = s.school_id
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

		// Add category filter (previously type)
		if params["category"] != "" {
			allowedCategories := []string{"Science", "Humanities", "Sport", "Creative"}
			validCategory := false
			for _, c := range allowedCategories {
				if params["category"] == c {
					validCategory = true
					break
				}
			}
			if !validCategory {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{
					Message: "Invalid category. Allowed values are: Science, Humanities, Sport, Creative",
				})
				return
			}
			queryBuilder.WriteString(" AND e.category = ?")
			args = append(args, params["category"])
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
				&event.Photo, &event.Participants, &event.Limit, &event.LimitParticipants,
				&event.StartDate, &event.EndDate, &event.Location, &event.Grade,
				&event.CreatedAt, &event.UpdatedAt, &event.CreatedBy, &event.Category, &event.SchoolName,
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
func getEventByID(db *sql.DB, id int) (models.Event, error) {
	var event models.Event
	err := db.QueryRow(`
        SELECT id, school_id, user_id, event_name, description, 
               photo, start_date, end_date, location, 
               grade, limit_count as limit, participants, limit_participants, created_at, updated_at, 
               u.email AS created_by, category, s.school_name
        FROM Events e
        LEFT JOIN users u ON e.user_id = u.id
        LEFT JOIN Schools s ON e.school_id = s.school_id
        WHERE e.id = ?
    `, id).Scan(
		&event.ID, &event.SchoolID, &event.UserID, &event.EventName, &event.Description,
		&event.Photo, &event.StartDate, &event.EndDate, &event.Location,
		&event.Grade, &event.Limit, &event.Participants, &event.LimitParticipants,
		&event.CreatedAt, &event.UpdatedAt, &event.CreatedBy, &event.Category, &event.SchoolName,
	)
	if err != nil {
		return event, err
	}
	return event, nil
}
func (ec *EventController) GetEventsByCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that GET method is used
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Get category from URL path
		vars := mux.Vars(r)
		category := vars["category"]

		if category == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Category parameter is required"})
			return
		}

		// Validate category
		allowedCategories := []string{"Science", "Humanities", "Sport", "Creative"}
		validCategory := false
		for _, c := range allowedCategories {
			if category == c {
				validCategory = true
				break
			}
		}

		if !validCategory {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{
				Message: "Invalid category. Allowed values are: Science, Humanities, Sport, Creative",
			})
			return
		}

		// Get optional query parameters for additional filtering
		query := r.URL.Query()
		schoolID := query.Get("school_id")
		id := query.Get("id")
		limit := query.Get("limit")
		offset := query.Get("offset")

		log.Printf("GetEventsByCategory called with category: %s, school_id: %s, id: %s", category, schoolID, id)

		// Build SQL query - selecting only the required fields, including e.id
		queryBuilder := strings.Builder{}
		queryBuilder.WriteString(`
            SELECT e.id, e.school_id, s.school_name, e.event_name, e.photo as photo_url, 
                   (SELECT COUNT(*) FROM EventRegistrations r WHERE r.event_id = e.id) as participants,
                   e.limit_count as ` + "`limit`" + `,
                   e.start_date, e.end_date, e.location,     e.grade
            FROM Events e
            LEFT JOIN Schools s ON e.school_id = s.school_id
            WHERE e.category = ?
        `)

		var args []interface{}
		args = append(args, category)

		// Add school_id filter if provided
		if schoolID != "" {
			schoolIDInt, err := strconv.Atoi(schoolID)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
				return
			}
			queryBuilder.WriteString(" AND e.school_id = ?")
			args = append(args, schoolIDInt)
		}

		// Add id filter if provided
		if id != "" {
			idInt, err := strconv.Atoi(id)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid id format"})
				return
			}
			queryBuilder.WriteString(" AND e.id = ?")
			args = append(args, idInt)
		}

		// Add sorting by school_id and start_date
		queryBuilder.WriteString(" ORDER BY e.school_id, e.start_date ASC")

		// Add pagination if provided
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

		// Log final SQL query
		finalQuery := queryBuilder.String()
		// log.Printf("Executing SQL query: %s with args: %v", finalQuery, args)

		// Execute query
		rows, err := db.Query(finalQuery, args...)
		if err != nil {
			log.Println("Error executing events by category query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch events"})
			return
		}
		defer rows.Close()

		// Define structs for the response
		type Event struct {
			EventID      int64  `json:"event_id"` // Added event_id
			EventName    string `json:"event_name"`
			PhotoURL     string `json:"photo_url"`
			Participants int    `json:"participants"`
			Limit        int    `json:"limit"`
			StartDate    string `json:"start_date"`
			EndDate      string `json:"end_date"`
			Location     string `json:"location"`
			Grade        string `json:"grade"`
		}

		type School struct {
			SchoolID   int     `json:"school_id"`
			SchoolName string  `json:"school_name"`
			Events     []Event `json:"events"`
		}

		// Collect results, grouping by school
		schoolMap := make(map[int]*School)
		for rows.Next() {
			var eventID int64 // Added to store e.id
			var schoolID int
			var schoolName string
			var event Event
			err := rows.Scan(
				&eventID, &schoolID, &schoolName, &event.EventName, &event.PhotoURL,
				&event.Participants, &event.Limit,
				&event.StartDate, &event.EndDate, &event.Location, &event.Grade,
			)
			if err != nil {
				log.Println("Error scanning event row:", err)
				continue
			}

			// Assign eventID to event struct
			event.EventID = eventID

			// If school doesn't exist in map, create it
			if _, exists := schoolMap[schoolID]; !exists {
				schoolMap[schoolID] = &School{
					SchoolID:   schoolID,
					SchoolName: schoolName,
					Events:     []Event{},
				}
			}

			// Append event to school's events list
			schoolMap[schoolID].Events = append(schoolMap[schoolID].Events, event)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating event rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing events data"})
			return
		}

		// Convert school map to slice
		schools := make([]School, 0, len(schoolMap))
		for _, school := range schoolMap {
			schools = append(schools, *school)
		}

		// Prepare response
		response := map[string]interface{}{
			"category": category,
			"schools":  schools,
		}

		if len(schools) == 0 {
			response["message"] = fmt.Sprintf("No events found for category '%s'", category)
		}

		utils.ResponseJSON(w, response)
	}
}
func (ec *EventController) GetEventByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that GET method is used
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Get event ID from URL path
		vars := mux.Vars(r)
		eventIDStr := vars["id"]
		if eventIDStr == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Event ID is required"})
			return
		}

		eventID, err := strconv.Atoi(eventIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID format"})
			return
		}

		log.Printf("GetEventByID called with event_id: %d", eventID)

		// Build SQL query - selecting all fields from Event struct except created_by
		query := `
            SELECT e.id, e.school_id, COALESCE(s.school_name, '') as school_name, e.user_id, 
                   e.event_name, COALESCE(e.description, '') as description,
                   COALESCE(e.photo, '') as photo_url, 
                   (SELECT COUNT(*) FROM EventRegistrations r WHERE r.event_id = e.id) as participants,
                   e.limit_count as ` + "`limit`" + `,
                   e.start_date, e.end_date, COALESCE(e.location, '') as location, 
                   COALESCE(e.grade, 0) as grade,
                   COALESCE(e.created_at, '') as created_at, 
                   COALESCE(e.updated_at, '') as updated_at, 
                   COALESCE(e.category, '') as category
            FROM Events e
            LEFT JOIN Schools s ON e.school_id = s.school_id
            WHERE e.id = ?
        `

		// Execute query
		row := db.QueryRow(query, eventID)

		// Define struct for the response
		type Event struct {
			ID           int    `json:"event_id"`
			SchoolID     int    `json:"school_id"`
			SchoolName   string `json:"school_name"`
			UserID       int    `json:"user_id"`
			EventName    string `json:"event_name"`
			Description  string `json:"description"`
			PhotoURL     string `json:"photo_url"`
			Participants int    `json:"participants"`
			Limit        int    `json:"limit"`
			StartDate    string `json:"start_date"`
			EndDate      string `json:"end_date"`
			Location     string `json:"location"`
			Grade        int    `json:"grade"`
			CreatedAt    string `json:"created_at"`
			UpdatedAt    string `json:"updated_at"`
			Category     string `json:"category"`
		}

		// Scan the result
		var event Event
		err = row.Scan(
			&event.ID, &event.SchoolID, &event.SchoolName, &event.UserID, &event.EventName, &event.Description,
			&event.PhotoURL, &event.Participants, &event.Limit,
			&event.StartDate, &event.EndDate, &event.Location, &event.Grade,
			&event.CreatedAt, &event.UpdatedAt, &event.Category,
		)
		if err == sql.ErrNoRows {
			log.Printf("Event with ID %d not found", eventID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
			return
		}
		if err != nil {
			log.Println("Error scanning event row:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching event"})
			return
		}

		// Prepare response
		utils.ResponseJSON(w, event)
	}
}
func (ec *EventController) GetEventCountsBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that GET method is used
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Build SQL query to count events per school
		query := `
            SELECT s.school_id, s.school_name, COUNT(e.id) as event_count
            FROM Schools s
            LEFT JOIN Events e ON s.school_id = e.school_id
            GROUP BY s.school_id, s.school_name
            ORDER BY s.school_id ASC
        `

		// Execute query
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error executing event count query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event counts"})
			return
		}
		defer rows.Close()

		// Define struct for response
		type SchoolEventCount struct {
			SchoolID   int    `json:"school_id"`
			SchoolName string `json:"school_name"`
			EventCount int    `json:"event_count"`
		}

		// Collect results
		var schoolEventCounts []SchoolEventCount
		for rows.Next() {
			var sec SchoolEventCount
			err := rows.Scan(&sec.SchoolID, &sec.SchoolName, &sec.EventCount)
			if err != nil {
				log.Println("Error scanning event count row:", err)
				continue
			}
			schoolEventCounts = append(schoolEventCounts, sec)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating event count rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing event count data"})
			return
		}

		// Prepare response
		response := map[string]interface{}{
			"schools":       schoolEventCounts,
			"total_schools": len(schoolEventCounts),
		}

		if len(schoolEventCounts) == 0 {
			response["message"] = "No schools found with events"
		}

		utils.ResponseJSON(w, response)
	}
}
func (ec *EventController) GetEventCountsAndScoresBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that GET method is used
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Build SQL query to count events per school
		query := `
            SELECT s.school_id, s.school_name, COUNT(e.id) as event_count
            FROM Schools s
            LEFT JOIN Events e ON s.school_id = e.school_id
            GROUP BY s.school_id, s.school_name
            ORDER BY s.school_id ASC
        `

		// Execute query
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error executing event count query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event counts"})
			return
		}
		defer rows.Close()

		// Define struct for response
		type SchoolEventScore struct {
			SchoolID   int     `json:"school_id"`
			SchoolName string  `json:"school_name"`
			EventCount int     `json:"event_count"`
			Score      float64 `json:"score"`
		}

		// Collect results and find maximum event count
		var schoolEventScores []SchoolEventScore
		maxEventCount := 0
		for rows.Next() {
			var ses SchoolEventScore
			err := rows.Scan(&ses.SchoolID, &ses.SchoolName, &ses.EventCount)
			if err != nil {
				log.Println("Error scanning event count row:", err)
				continue
			}
			if ses.EventCount > maxEventCount {
				maxEventCount = ses.EventCount
			}
			schoolEventScores = append(schoolEventScores, ses)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating event count rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing event count data"})
			return
		}

		// Calculate scores (max event count = 10 points, others scaled proportionally)
		for i := range schoolEventScores {
			if maxEventCount == 0 {
				schoolEventScores[i].Score = 0
			} else {
				schoolEventScores[i].Score = (float64(schoolEventScores[i].EventCount) / float64(maxEventCount)) * 10
			}
		}

		// Prepare response
		response := map[string]interface{}{
			"schools":       schoolEventScores,
			"total_schools": len(schoolEventScores),
		}

		if len(schoolEventScores) == 0 {
			response["message"] = "No schools found with events"
		}

		utils.ResponseJSON(w, response)
	}
}

// func (ec *EventController) GetEventCountAndScoreBySchoolID(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Check that GET method is used
// 		if r.Method != http.MethodGet {
// 			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
// 			return
// 		}

// 		// Extract school_id from URL parameters
// 		vars := mux.Vars(r)
// 		schoolIDStr := vars["school_id"]
// 		schoolID, err := strconv.Atoi(schoolIDStr)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
// 			return
// 		}

// 		// Check if school exists
// 		var schoolExists bool
// 		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", schoolID).Scan(&schoolExists)
// 		if err != nil {
// 			log.Println("Error checking if school exists:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking school existence"})
// 			return
// 		}
// 		if !schoolExists {
// 			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
// 			return
// 		}

// 		// Get the event count for the specific school
// 		var schoolName string
// 		var eventCount int
// 		err = db.QueryRow(`
// 			SELECT s.school_name, COUNT(e.id) as event_count
// 			FROM Schools s
// 			LEFT JOIN Events e ON s.school_id = e.school_id
// 			WHERE s.school_id = ?
// 			GROUP BY s.school_id, s.school_name
// 		`, schoolID).Scan(&schoolName, &eventCount)
// 		if err != nil {
// 			log.Println("Error fetching event count for school:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event count"})
// 			return
// 		}

// 		// Get the maximum event count across all schools
// 		var maxEventCount int
// 		err = db.QueryRow(`
// 			SELECT COALESCE(MAX(event_count), 0)
// 			FROM (
// 				SELECT COUNT(id) as event_count
// 				FROM Events
// 				GROUP BY school_id
// 			) as counts
// 		`).Scan(&maxEventCount)
// 		if err != nil {
// 			log.Println("Error fetching maximum event count:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate score"})
// 			return
// 		}

// 		// Calculate score
// 		var score float64
// 		if maxEventCount == 0 {
// 			score = 0
// 		} else {
// 			score = (float64(eventCount) / float64(maxEventCount)) * 10
// 		}

// 		// Get monthly event counts based on end_date
// 		monthlyQuery := `
// 			SELECT
// 				CASE
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 1 THEN 'january'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 2 THEN 'february'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 3 THEN 'march'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 4 THEN 'april'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 5 THEN 'may'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 6 THEN 'june'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 7 THEN 'july'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 8 THEN 'august'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 9 THEN 'september'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 10 THEN 'october'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 11 THEN 'november'
// 					WHEN MONTH(STR_TO_DATE(end_date, '%Y-%m-%d')) = 12 THEN 'december'
// 				END as month_name,
// 				COUNT(*) as count
// 			FROM Events
// 			WHERE school_id = ? AND end_date IS NOT NULL AND end_date != ''
// 			GROUP BY MONTH(STR_TO_DATE(end_date, '%Y-%m-%d'))
// 		`

// 		rows, err := db.Query(monthlyQuery, schoolID)
// 		if err != nil {
// 			log.Println("Error fetching monthly event counts:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch monthly event counts"})
// 			return
// 		}
// 		defer rows.Close()

// 		// Initialize all months with 0
// 		monthlyEvents := map[string]int{
// 			"january":   0,
// 			"february":  0,
// 			"march":     0,
// 			"april":     0,
// 			"may":       0,
// 			"june":      0,
// 			"july":      0,
// 			"august":    0,
// 			"september": 0,
// 			"october":   0,
// 			"november":  0,
// 			"december":  0,
// 		}

// 		// Fill in actual counts
// 		for rows.Next() {
// 			var monthName string
// 			var count int
// 			err := rows.Scan(&monthName, &count)
// 			if err != nil {
// 				log.Println("Error scanning monthly event row:", err)
// 				continue
// 			}
// 			monthlyEvents[monthName] = count
// 		}

// 		if err = rows.Err(); err != nil {
// 			log.Println("Error iterating monthly event rows:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing monthly event data"})
// 			return
// 		}

// 		// Define struct for response
// 		type SchoolEventScore struct {
// 			SchoolID      int            `json:"school_id"`
// 			SchoolName    string         `json:"school_name"`
// 			EventCount    int            `json:"event_count"`
// 			Score         float64        `json:"score"`
// 			MonthlyEvents map[string]int `json:"monthly_events"`
// 		}

// 		// Prepare response
// 		response := map[string]interface{}{
// 			"school": SchoolEventScore{
// 				SchoolID:      schoolID,
// 				SchoolName:    schoolName,
// 				EventCount:    eventCount,
// 				Score:         score,
// 				MonthlyEvents: monthlyEvents,
// 			},
// 		}

// 		utils.ResponseJSON(w, response)
// 	}
// }

func (ec *EventController) GetEventCountAndScoreBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
			return
		}

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

		yearStr := r.URL.Query().Get("year")
		year := time.Now().Year()
		if yearStr != "" {
			parsedYear, err := strconv.Atoi(yearStr)
			if err == nil {
				year = parsedYear
			}
		}

		today := time.Now()
		cutoffDate := time.Date(today.Year(), today.Month(), 5, 0, 0, 0, 0, today.Location())
		if today.Before(cutoffDate) {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Data for the current month is not yet available"})
			return
		}

		var schoolName string
		var eventCount int
		err = db.QueryRow(`
			SELECT s.school_name, COUNT(e.id) as event_count
			FROM Schools s
			LEFT JOIN Events e ON s.school_id = e.school_id AND YEAR(e.end_date) = ?
			WHERE s.school_id = ?
			GROUP BY s.school_id, s.school_name
		`, year, schoolID).Scan(&schoolName, &eventCount)
		if err != nil {
			log.Println("Error fetching event count for school:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch event count"})
			return
		}

		var maxEventCount int
		err = db.QueryRow(`
			SELECT COALESCE(MAX(event_count), 0)
			FROM (
				SELECT COUNT(id) as event_count
				FROM Events
				WHERE YEAR(end_date) = ?
				GROUP BY school_id
			) as counts
		`, year).Scan(&maxEventCount)
		if err != nil {
			log.Println("Error fetching maximum event count:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate score"})
			return
		}

		var score float64
		if maxEventCount == 0 {
			score = 0
		} else {
			score = (float64(eventCount) / float64(maxEventCount)) * 10
		}

		monthlyQuery := `
			SELECT 
				CASE 
					WHEN MONTH(end_date) = 1 THEN 'january'
					WHEN MONTH(end_date) = 2 THEN 'february'
					WHEN MONTH(end_date) = 3 THEN 'march'
					WHEN MONTH(end_date) = 4 THEN 'april'
					WHEN MONTH(end_date) = 5 THEN 'may'
					WHEN MONTH(end_date) = 6 THEN 'june'
					WHEN MONTH(end_date) = 7 THEN 'july'
					WHEN MONTH(end_date) = 8 THEN 'august'
					WHEN MONTH(end_date) = 9 THEN 'september'
					WHEN MONTH(end_date) = 10 THEN 'october'
					WHEN MONTH(end_date) = 11 THEN 'november'
					WHEN MONTH(end_date) = 12 THEN 'december'
				END as month_name,
				COUNT(*) as count
			FROM Events 
			WHERE school_id = ? AND end_date IS NOT NULL AND end_date != '' AND YEAR(end_date) = ?
			GROUP BY MONTH(end_date)
		`

		rows, err := db.Query(monthlyQuery, schoolID, year)
		if err != nil {
			log.Println("Error fetching monthly event counts:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch monthly event counts"})
			return
		}
		defer rows.Close()

		monthlyEvents := map[string]int{
			"january": 0, "february": 0, "march": 0, "april": 0, "may": 0, "june": 0,
			"july": 0, "august": 0, "september": 0, "october": 0, "november": 0, "december": 0,
		}

		for rows.Next() {
			var monthName string
			var count int
			if err := rows.Scan(&monthName, &count); err != nil {
				log.Println("Error scanning monthly event row:", err)
				continue
			}
			monthlyEvents[monthName] = count
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating monthly event rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing monthly event data"})
			return
		}

		type SchoolEventScore struct {
			SchoolID      int            `json:"school_id"`
			SchoolName    string         `json:"school_name"`
			EventCount    int            `json:"event_count"`
			Score         float64        `json:"score"`
			MonthlyEvents map[string]int `json:"monthly_events"`
			Year          int            `json:"year"`
		}

		response := SchoolEventScore{
			SchoolID:      schoolID,
			SchoolName:    schoolName,
			EventCount:    eventCount,
			Score:         score,
			MonthlyEvents: monthlyEvents,
			Year:          year,
		}

		utils.ResponseJSON(w, response)
	}
}
