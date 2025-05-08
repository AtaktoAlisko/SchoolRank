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

// Метод для создания олимпиады по предмету
func (c *SubjectOlympiadController) CreateSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var error models.Error

		// Step 1: Verify user token and get user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			error.Message = "Invalid token."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Step 2: Get user role from DB
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check access permission
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create an olympiad"})
			return
		}

		// Step 4: Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10MB limit
		if err != nil {
			error.Message = "Error parsing form data"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		// Step 5: Determine school_id
		var schoolID int
		if userRole == "schooladmin" {
			// Get school_id from users table
			err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", userID).Scan(&schoolID)
			if err != nil {
				log.Println("Error fetching school_id for schooladmin:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Cannot retrieve school_id for schooladmin"})
				return
			}
		} else if userRole == "superadmin" {
			// Get school_id from form
			schoolIDStr := r.FormValue("school_id")
			if schoolIDStr == "" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required for superadmin"})
				return
			}
			schoolID, err = strconv.Atoi(schoolIDStr)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id format"})
				log.Printf("Error converting school_id: %v", err)
				return
			}
		}

		// Step 6: Parse other form fields
		limit, err := strconv.Atoi(r.FormValue("limit"))
		if err != nil || limit <= 0 {
			error.Message = "Invalid participant limit format or value"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Error converting limit: %v", err)
			return
		}

		olympiad := models.SubjectOlympiad{
			OlympiadName: r.FormValue("subject_name"),
			OlympiadType: r.FormValue("olympiad_name"),
			StartDate:    r.FormValue("date"),
			EndDate:      r.FormValue("end_date"),
			Description:  r.FormValue("description"),
			City:         r.FormValue("city"),
			SchoolID:     schoolID,
			Level:        r.FormValue("level"),
			Limit:        limit,
		}

		// Step 7: Validate required fields
		if olympiad.OlympiadName == "" || olympiad.OlympiadType == "" || olympiad.StartDate == "" ||
			olympiad.City == "" || olympiad.Level == "" {
			error.Message = "Subject name, olympiad name, start date, city, and level are required fields."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Missing required fields: %v", olympiad)
			return
		}

		// Step 8: File upload
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

		olympiad.PhotoURL = photoURL

		// Step 9: Insert into DB
		query := `INSERT INTO subject_olympiads (subject_name, olympiad_name, date, end_date, description, 
                 photo_url, city, school_id, level, limit_participants) 
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = db.Exec(query,
			olympiad.OlympiadName,
			olympiad.OlympiadType,
			olympiad.StartDate,
			olympiad.EndDate,
			olympiad.Description,
			olympiad.PhotoURL,
			olympiad.City,
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

// DeleteSubjectOlympiad handles the deletion of a subject olympiad
func (c *SubjectOlympiadController) DeleteSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var error models.Error

		// Step 1: Verify user token and get user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			error.Message = "Invalid token."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Step 2: Get user role from DB
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Get olympiad ID from URL params
		params := mux.Vars(r)
		olympiadID, err := strconv.Atoi(params["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID"})
			return
		}

		// Step 4: Check if olympiad exists and get school_id
		var olympiadSchoolID int
		err = db.QueryRow("SELECT school_id FROM subject_olympiads WHERE id = ?", olympiadID).Scan(&olympiadSchoolID)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad not found"})
			} else {
				log.Println("Error fetching olympiad:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching olympiad"})
			}
			return
		}

		// Step 5: Check access permission
		if userRole == "schooladmin" {
			// Verify if the user belongs to the same school as the olympiad
			var userSchoolID int
			err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", userID).Scan(&userSchoolID)
			if err != nil {
				log.Println("Error fetching school_id for user:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error verifying access"})
				return
			}

			if userSchoolID != olympiadSchoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You don't have permission to delete this olympiad"})
				return
			}
		} else if userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You don't have permission to delete olympiads"})
			return
		}

		// Step 6: Delete the olympiad
		_, err = db.Exec("DELETE FROM subject_olympiads WHERE id = ?", olympiadID)
		if err != nil {
			log.Println("Error deleting olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete olympiad"})
			return
		}

		// Step 7: Return success response
		utils.ResponseJSON(w, map[string]string{"message": "Olympiad deleted successfully"})
	}
}

// UpdateSubjectOlympiad handles the updating of a subject olympiad
func (c *SubjectOlympiadController) UpdateSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var error models.Error

		// Step 1: Verify user token and get user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			error.Message = "Invalid token."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Step 2: Get user role from DB
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Get olympiad ID from URL params
		params := mux.Vars(r)
		olympiadID, err := strconv.Atoi(params["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID"})
			return
		}

		// Step 4: Check if olympiad exists and get school_id
		var olympiadSchoolID int
		err = db.QueryRow("SELECT school_id FROM subject_olympiads WHERE id = ?", olympiadID).Scan(&olympiadSchoolID)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad not found"})
			} else {
				log.Println("Error fetching olympiad:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching olympiad"})
			}
			return
		}

		// Step 5: Check access permission
		if userRole == "schooladmin" {
			// Verify if the user belongs to the same school as the olympiad
			var userSchoolID int
			err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", userID).Scan(&userSchoolID)
			if err != nil {
				log.Println("Error fetching school_id for user:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error verifying access"})
				return
			}

			if userSchoolID != olympiadSchoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You don't have permission to update this olympiad"})
				return
			}
		} else if userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You don't have permission to update olympiads"})
			return
		}

		// Step 6: Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10MB limit
		if err != nil {
			error.Message = "Error parsing form data"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		// Step 7: Parse form fields
		olympiad := models.SubjectOlympiad{
			ID:           olympiadID,
			OlympiadName: r.FormValue("subject_name"),
			OlympiadType: r.FormValue("olympiad_name"),
			StartDate:    r.FormValue("date"),
			EndDate:      r.FormValue("end_date"),
			Description:  r.FormValue("description"),
			City:         r.FormValue("city"),
			SchoolID:     olympiadSchoolID, // Keep the original school_id
			Level:        r.FormValue("level"),
		}

		// Handle limit if provided
		limitStr := r.FormValue("limit")
		if limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				error.Message = "Invalid participant limit format or value"
				utils.RespondWithError(w, http.StatusBadRequest, error)
				log.Printf("Error converting limit: %v", err)
				return
			}
			olympiad.Limit = limit
		} else {
			// Get the current limit value from DB
			err = db.QueryRow("SELECT limit_participants FROM subject_olympiads WHERE id = ?", olympiadID).Scan(&olympiad.Limit)
			if err != nil {
				log.Println("Error fetching current limit:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving current olympiad data"})
				return
			}
		}

		// Step 8: Validate required fields
		if olympiad.OlympiadName == "" || olympiad.OlympiadType == "" || olympiad.StartDate == "" ||
			olympiad.City == "" || olympiad.Level == "" {
			error.Message = "Subject name, olympiad name, start date, city, and level are required fields."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Missing required fields: %v", olympiad)
			return
		}

		// Step 9: Handle file upload if provided
		file, _, err := r.FormFile("photo_url")
		var photoURL string

		if err == nil {
			// A new file was uploaded
			defer file.Close()
			uniqueFileName := fmt.Sprintf("olympiad-%d-%d.jpg", olympiadID, time.Now().Unix())
			photoURL, err = utils.UploadFileToS3(file, uniqueFileName, "schoolphoto")
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload photo"})
				log.Println("Error uploading file:", err)
				return
			}
			olympiad.PhotoURL = photoURL
		} else if err != http.ErrMissingFile {
			// There was an error that wasn't simply missing file
			log.Println("Error reading file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
			return
		} else {
			// No new file, keep the existing photo URL
			err = db.QueryRow("SELECT photo_url FROM subject_olympiads WHERE id = ?", olympiadID).Scan(&olympiad.PhotoURL)
			if err != nil {
				log.Println("Error fetching current photo URL:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving current photo"})
				return
			}
		}

		// Step 10: Update olympiad in DB
		query := `UPDATE subject_olympiads 
				 SET subject_name = ?, olympiad_name = ?, date = ?, end_date = ?, 
				 description = ?, photo_url = ?, city = ?, level = ?, limit_participants = ? 
				 WHERE id = ?`

		_, err = db.Exec(query,
			olympiad.OlympiadName,
			olympiad.OlympiadType,
			olympiad.StartDate,
			olympiad.EndDate,
			olympiad.Description,
			olympiad.PhotoURL,
			olympiad.City,
			olympiad.Level,
			olympiad.Limit,
			olympiad.ID)

		if err != nil {
			log.Println("Error updating olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update olympiad"})
			return
		}

		utils.ResponseJSON(w, olympiad)
	}
}

// GetSubjectOlympiad returns a specific subject olympiad by ID
func (c *SubjectOlympiadController) GetSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var error models.Error

		// Step 1: Verify user token
		_, err := utils.VerifyToken(r)
		if err != nil {
			error.Message = "Invalid token."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Step 2: Get olympiad ID from URL params
		params := mux.Vars(r)
		olympiadID, err := strconv.Atoi(params["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID"})
			return
		}

		// Step 3: Query the olympiad
		var olympiad models.SubjectOlympiad
		var createdAt, updatedAt string
		query := `SELECT id, subject_name, olympiad_name, date, end_date, description, 
				 photo_url, city, school_id, level, limit_participants, created_at, updated_at
				 FROM subject_olympiads WHERE id = ?`

		err = db.QueryRow(query, olympiadID).Scan(
			&olympiad.ID,
			&olympiad.OlympiadName,
			&olympiad.OlympiadType,
			&olympiad.StartDate,
			&olympiad.EndDate,
			&olympiad.Description,
			&olympiad.PhotoURL,
			&olympiad.City,
			&olympiad.SchoolID,
			&olympiad.Level,
			&olympiad.Limit,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad not found"})
			} else {
				log.Println("Error fetching olympiad:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiad"})
			}
			return
		}

		// Step 4: Get school name for the olympiad (we'll add it to the response separately)
		var schoolName string
		err = db.QueryRow("SELECT name FROM schools WHERE id = ?", olympiad.SchoolID).Scan(&schoolName)
		if err != nil && err != sql.ErrNoRows {
			log.Println("Error fetching school name:", err)
			// Continue without school name rather than failing the whole request
		}

		// Create response with school name
		type OlympiadResponse struct {
			models.SubjectOlympiad
			SchoolName string `json:"school_name"`
		}

		response := OlympiadResponse{
			SubjectOlympiad: olympiad,
			SchoolName:      schoolName,
		}

		// Step 5: Return the olympiad data with school name
		utils.ResponseJSON(w, response)
	}
}

// GetAllSubjectOlympiads returns a list of all subject olympiads
func (c *SubjectOlympiadController) GetAllSubjectOlympiads(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var error models.Error

		// Step 1: Verify user token
		_, err := utils.VerifyToken(r)
		if err != nil {
			error.Message = "Invalid token."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Step 2: Parse query parameters for filtering
		query := "SELECT id, subject_name, olympiad_name, date, end_date, description, photo_url, city, school_id, level, limit_participants, created_at, updated_at FROM subject_olympiads WHERE 1=1"
		args := []interface{}{}

		// Filter by school_id if provided
		schoolID := r.URL.Query().Get("school_id")
		if schoolID != "" {
			query += " AND school_id = ?"
			schoolIDInt, err := strconv.Atoi(schoolID)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
				return
			}
			args = append(args, schoolIDInt)
		}

		// Filter by subject_name if provided
		subjectName := r.URL.Query().Get("subject_name")
		if subjectName != "" {
			query += " AND subject_name LIKE ?"
			args = append(args, "%"+subjectName+"%")
		}

		// Filter by olympiad_name if provided
		olympiadName := r.URL.Query().Get("olympiad_name")
		if olympiadName != "" {
			query += " AND olympiad_name LIKE ?"
			args = append(args, "%"+olympiadName+"%")
		}

		// Filter by city if provided
		city := r.URL.Query().Get("city")
		if city != "" {
			query += " AND city LIKE ?"
			args = append(args, "%"+city+"%")
		}

		// Filter by level if provided
		level := r.URL.Query().Get("level")
		if level != "" {
			query += " AND level = ?"
			args = append(args, level)
		}

		// Add ordering
		query += " ORDER BY created_at DESC"

		// Step 3: Execute the query
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println("Error querying olympiads:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiads"})
			return
		}
		defer rows.Close()

		// Step 4: Scan results into a slice
		olympiads := []models.SubjectOlympiad{}
		for rows.Next() {
			var olympiad models.SubjectOlympiad
			var createdAt, updatedAt string
			if err := rows.Scan(
				&olympiad.ID,
				&olympiad.OlympiadName,
				&olympiad.OlympiadType,
				&olympiad.StartDate,
				&olympiad.EndDate,
				&olympiad.Description,
				&olympiad.PhotoURL,
				&olympiad.City,
				&olympiad.SchoolID,
				&olympiad.Level,
				&olympiad.Limit,
				&createdAt,
				&updatedAt,
			); err != nil {
				log.Println("Error scanning olympiad row:", err)
				continue // Skip this row and continue with the next
			}
			olympiads = append(olympiads, olympiad)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating olympiad rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing olympiads"})
			return
		}

		// Step 5: Create response with school names
		type OlympiadResponse struct {
			models.SubjectOlympiad
			SchoolName string `json:"school_name"`
		}

		responses := make([]OlympiadResponse, len(olympiads))

		for i := range olympiads {
			var schoolName string
			err = db.QueryRow("SELECT name FROM schools WHERE id = ?", olympiads[i].SchoolID).Scan(&schoolName)
			if err != nil && err != sql.ErrNoRows {
				log.Println("Error fetching school name:", err)
				// Continue without school name rather than failing the whole request
			}

			responses[i] = OlympiadResponse{
				SubjectOlympiad: olympiads[i],
				SchoolName:      schoolName,
			}
		}

		// Step 6: Return the olympiads with school names
		utils.ResponseJSON(w, responses)
	}
}
