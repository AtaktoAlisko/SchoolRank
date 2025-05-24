package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"ranking-school/models"
	"ranking-school/utils"

	"github.com/gorilla/mux"
)

type EventsRegistrationController struct{}

func (ec *EventsRegistrationController) RegisterForEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Check user role
		var role string
		var studentSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &studentSchoolID)
		if err != nil || role != "student" {
			log.Printf("Access denied for user %d: invalid role or error %v", userID, err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only students can register for events"})
			return
		}

		// Parse request body
		var body struct {
			EventID int `json:"event_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			log.Printf("Invalid request body for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid body"})
			return
		}
		defer r.Body.Close()

		eventID := body.EventID
		if eventID <= 0 {
			log.Printf("Invalid event ID %d for user %d", eventID, userID)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID"})
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer tx.Rollback()

		// Fetch event details
		var eventSchoolID int
		var limitCount int
		err = tx.QueryRow("SELECT school_id, limit_count FROM Events WHERE id = ?", eventID).Scan(&eventSchoolID, &limitCount)
		if err == sql.ErrNoRows {
			log.Printf("Event %d not found for user %d", eventID, userID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
			return
		}
		if err != nil {
			log.Printf("Error fetching event %d: %v", eventID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking event"})
			return
		}

		// Calculate limits
		halfLimit := limitCount / 2
		otherHalfLimit := limitCount - halfLimit

		// Count current registrations
		var sameSchoolCount, otherSchoolCount int
		err = tx.QueryRow(`
            SELECT 
                COUNT(*) FILTER (WHERE r.school_id = ?) AS same_school,
                COUNT(*) FILTER (WHERE r.school_id != ?) AS other_school
            FROM EventRegistrations r
            WHERE r.event_id = ? AND r.status = 'registered'`,
			eventSchoolID, eventSchoolID, eventID).Scan(&sameSchoolCount, &otherSchoolCount)
		if err != nil {
			log.Printf("Error counting registrations for event %d: %v", eventID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking registration limits"})
			return
		}

		// Check registration limits based on student's school
		if studentSchoolID.Valid && int(studentSchoolID.Int64) == eventSchoolID {
			if sameSchoolCount >= halfLimit {
				log.Printf("Registration limit reached for event %d's school (%d/%d)", eventID, sameSchoolCount, halfLimit)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration limit reached for this school's students"})
				return
			}
		} else {
			if otherSchoolCount >= otherHalfLimit {
				log.Printf("Registration limit reached for other schools for event %d (%d/%d)", eventID, otherSchoolCount, otherHalfLimit)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration limit reached for students from other schools"})
				return
			}
		}

		// Insert new registration
		currentTime := time.Now().Format("2006-01-02 15:04:05") // ~2025-05-24 15:22:00Z
		_, err = tx.Exec(`
            INSERT INTO EventRegistrations (event_registration_id, student_id, event_id, registration_date, status, school_id)
            VALUES (?, ?, ?, ?, 'registered', ?)`,
			nil, userID, eventID, currentTime, studentSchoolID.Int64)
		if err != nil {
			log.Printf("Error inserting registration for user %d, event %d: %v", userID, eventID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to register for event"})
			return
		}

		// Fetch the newly created registration
		var registration models.EventRegistration
		var regDateStr string
		var eventEnd sql.NullString
		var schoolName sql.NullString
		var status sql.NullString

		err = tx.QueryRow(`
            SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
                   r.school_id,
                   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
                   sc.school_name,
                   e.event_name, e.start_date, e.end_date
            FROM EventRegistrations r
            JOIN student s ON r.student_id = s.student_id
            JOIN Schools sc ON r.school_id = sc.school_id
            JOIN Events e ON r.event_id = e.id
            WHERE r.student_id = ? AND r.event_id = ? AND r.registration_date = ?`,
			userID, eventID, currentTime).Scan(
			&registration.EventRegistrationID,
			&registration.StudentID,
			&registration.EventID,
			&regDateStr,
			&status,
			&registration.SchoolID,
			&registration.StudentFirstName,
			&registration.StudentLastName,
			&registration.StudentPatronymic,
			&registration.StudentGrade,
			&registration.StudentLetter,
			&schoolName,
			&registration.EventName,
			&registration.EventStartDate,
			&eventEnd,
		)
		if err != nil {
			log.Printf("Error fetching new registration for user %d, event %d: %v", userID, eventID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch registration"})
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction for user %d, event %d: %v", userID, eventID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to complete registration"})
			return
		}

		// Parse and set fields
		registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
		if status.Valid {
			registration.Status = status.String
		}
		if schoolName.Valid {
			registration.SchoolName = schoolName.String
		}
		if eventEnd.Valid {
			registration.EventEndDate = eventEnd.String
		}

		utils.ResponseJSON(w, registration)
	}
}
func (ec *EventsRegistrationController) UpdateEventRegistrationStatus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			log.Printf("Access denied for user %d: invalid role or error %v", userID, err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		idStr := mux.Vars(r)["id"]
		regID, err := strconv.Atoi(idStr)
		if err != nil || regID <= 0 {
			log.Printf("Invalid registration ID %s: %v", idStr, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			log.Printf("Invalid request body for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid body"})
			return
		}
		defer r.Body.Close()

		validStatuses := map[string]bool{
			"canceled":  true,
			"completed": true,
		}
		if !validStatuses[body.Status] {
			log.Printf("Invalid status %s for registration %d", body.Status, regID)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer tx.Rollback()

		if role == "schooladmin" {
			var regSchoolID int
			err = tx.QueryRow("SELECT school_id FROM EventRegistrations WHERE event_registration_id = ?", regID).Scan(&regSchoolID)
			if err != nil {
				log.Printf("Error fetching school ID for registration %d: %v", regID, err)
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
				return
			}
			if !userSchoolID.Valid || regSchoolID != int(userSchoolID.Int64) {
				log.Printf("School mismatch for user %d, registration %d: user school %v, reg school %d", userID, regID, userSchoolID, regSchoolID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only update registrations for your school"})
				return
			}
		}

		_, err = tx.Exec("UPDATE EventRegistrations SET status = ? WHERE event_registration_id = ?", body.Status, regID)
		if err != nil {
			log.Printf("Error updating status for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update status"})
			return
		}

		var registration models.EventRegistration
		var regDateStr string
		var eventEnd sql.NullString
		var schoolName sql.NullString
		var status sql.NullString

		err = tx.QueryRow(`
            SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
                   r.school_id,
                   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
                   sc.school_name,
                   e.event_name, e.start_date, e.end_date
            FROM EventRegistrations r
            JOIN student s ON r.student_id = s.student_id
            JOIN Schools sc ON r.school_id = sc.school_id
            JOIN Events e ON r.event_id = e.id
            WHERE r.event_registration_id = ?`, regID).Scan(
			&registration.EventRegistrationID,
			&registration.StudentID,
			&registration.EventID,
			&regDateStr,
			&status,
			&registration.SchoolID,
			&registration.StudentFirstName,
			&registration.StudentLastName,
			&registration.StudentPatronymic,
			&registration.StudentGrade,
			&registration.StudentLetter,
			&schoolName,
			&registration.EventName,
			&registration.EventStartDate,
			&eventEnd,
		)
		if err != nil {
			log.Printf("Error fetching updated registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch updated registration"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update registration"})
			return
		}

		registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
		if status.Valid {
			registration.Status = status.String
		}
		if schoolName.Valid {
			registration.SchoolName = schoolName.String
		}
		if eventEnd.Valid {
			registration.EventEndDate = eventEnd.String
		}

		utils.ResponseJSON(w, registration)
	}
}
func (ec *EventsRegistrationController) GetEventRegistrations(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		var schoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &schoolID)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			log.Printf("Access denied for user %d: invalid role or error %v", userID, err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		var rows *sql.Rows
		if role == "schooladmin" {
			if !schoolID.Valid {
				log.Printf("No school assigned to schooladmin %d", userID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "No school assigned"})
				return
			}
			rows, err = db.Query(`
                SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
                       r.school_id,
                       s.first_name, s.last_name, s.patronymic, s.grade, s.letter, s.role,
                       sc.school_name,
                       e.event_name, e.start_date, e.end_date, e.category
                FROM EventRegistrations r
                JOIN student s ON r.student_id = s.student_id
                JOIN Schools sc ON r.school_id = sc.school_id
                JOIN Events e ON r.event_id = e.id
                WHERE r.status = 'registered' AND r.school_id = ?`, schoolID.Int64)
		} else {
			rows, err = db.Query(`
                SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
                       r.school_id,
                       s.first_name, s.last_name, s.patronymic, s.grade, s.letter, s.role,
                       sc.school_name,
                       e.event_name, e.start_date, e.end_date, e.category
                FROM EventRegistrations r
                JOIN student s ON r.student_id = s.student_id
                JOIN Schools sc ON r.school_id = sc.school_id
                JOIN Events e ON r.event_id = e.id
                WHERE r.status = 'registered'`)
		}

		if err != nil {
			log.Printf("Error querying event registrations: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer rows.Close()

		var registrations []models.EventRegistration
		for rows.Next() {
			var reg models.EventRegistration
			var regDateStr string
			var endDate sql.NullString
			var schoolName sql.NullString
			var status sql.NullString
			var studentRole sql.NullString
			var eventCategory sql.NullString

			err := rows.Scan(
				&reg.EventRegistrationID,
				&reg.StudentID,
				&reg.EventID,
				&regDateStr,
				&status,
				&reg.SchoolID,
				&reg.StudentFirstName,
				&reg.StudentLastName,
				&reg.StudentPatronymic,
				&reg.StudentGrade,
				&reg.StudentLetter,
				&studentRole,
				&schoolName,
				&reg.EventName,
				&reg.EventStartDate,
				&endDate,
				&eventCategory,
			)
			if err != nil {
				log.Printf("Error scanning registration: %v", err)
				continue
			}

			// Log raw scanned values for debugging
			log.Printf("Scanned: regDateStr=%s, status=%v, schoolName=%v, endDate=%v, studentRole=%v, eventCategory=%v",
				regDateStr, status, schoolName, endDate, studentRole, eventCategory)

			reg.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
			if status.Valid {
				reg.Status = status.String
			}
			if endDate.Valid {
				reg.EventEndDate = endDate.String
			}
			if schoolName.Valid {
				reg.SchoolName = schoolName.String
			}
			if studentRole.Valid {
				reg.StudentRole = studentRole.String
			}
			if eventCategory.Valid {
				reg.EventCategory = eventCategory.String
			}

			registrations = append(registrations, reg)
		}
		log.Printf("Rows found: %d", len(registrations))

		utils.ResponseJSON(w, registrations)
	}
}
func (ec *EventsRegistrationController) DeleteEventRegistrationByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		idStr := mux.Vars(r)["id"]
		regID, err := strconv.Atoi(idStr)
		if err != nil || regID <= 0 {
			log.Printf("Invalid registration ID %s: %v", idStr, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer tx.Rollback()

		var role string
		var userSchoolID sql.NullInt64
		err = tx.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil {
			log.Printf("Error fetching user role for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		if role != "schooladmin" && role != "superadmin" {
			log.Printf("Access denied for user %d with role %s", userID, role)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		if role == "schooladmin" {
			var regSchoolID int
			err = tx.QueryRow("SELECT school_id FROM EventRegistrations WHERE event_registration_id = ?", regID).Scan(&regSchoolID)
			if err == sql.ErrNoRows {
				log.Printf("Registration %d not found", regID)
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
				return
			}
			if err != nil {
				log.Printf("Error fetching registration %d: %v", regID, err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking registration"})
				return
			}
			if !userSchoolID.Valid || regSchoolID != int(userSchoolID.Int64) {
				log.Printf("School mismatch for schooladmin %d, registration %d: user school %v, reg school %d", userID, regID, userSchoolID, regSchoolID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only delete registrations for your school"})
				return
			}
		}

		res, err := tx.Exec("DELETE FROM EventRegistrations WHERE event_registration_id = ?", regID)
		if err != nil {
			log.Printf("Error deleting registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete registration"})
			return
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			log.Printf("No rows affected when deleting registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete registration"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "Registration deleted successfully"})
	}
}
func (ec *EventsRegistrationController) DeleteMyEventRegistration(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		idStr := mux.Vars(r)["id"]
		regID, err := strconv.Atoi(idStr)
		if err != nil || regID <= 0 {
			log.Printf("Invalid registration ID %s: %v", idStr, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer tx.Rollback()

		var role string
		err = tx.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if err != nil {
			log.Printf("Error fetching user role for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}
		if role != "student" {
			log.Printf("Access denied for user %d with role %s", userID, role)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only students can delete their own registrations"})
			return
		}

		var regStudentID int
		err = tx.QueryRow("SELECT student_id FROM EventRegistrations WHERE event_registration_id = ?", regID).Scan(&regStudentID)
		if err == sql.ErrNoRows {
			log.Printf("Registration %d not found", regID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}
		if err != nil {
			log.Printf("Error fetching registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking registration"})
			return
		}
		if regStudentID != userID {
			log.Printf("Student %d attempted to delete registration %d belonging to student %d", userID, regID, regStudentID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only delete your own registration"})
			return
		}

		res, err := tx.Exec("DELETE FROM EventRegistrations WHERE event_registration_id = ?", regID)
		if err != nil {
			log.Printf("Error deleting registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete registration"})
			return
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			log.Printf("No rows affected when deleting registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete registration"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "Registration deleted successfully"})
	}
}
func (ec *EventsRegistrationController) ApproveOrCancelEventRegistration(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			log.Printf("Access denied for user %d: invalid role or error %v", userID, err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		idStr := mux.Vars(r)["id"]
		regID, err := strconv.Atoi(idStr)
		if err != nil || regID <= 0 {
			log.Printf("Invalid registration ID %s: %v", idStr, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			log.Printf("Invalid request body for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid body"})
			return
		}
		defer r.Body.Close()

		validStatuses := map[string]bool{
			"accepted": true,
			"canceled": true,
		}
		if !validStatuses[body.Status] {
			log.Printf("Invalid status %s for registration %d", body.Status, regID)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status, must be 'accepted' or 'canceled'"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer tx.Rollback()

		var eventSchoolID int
		var eventID int
		err = tx.QueryRow(`
            SELECT e.school_id, r.event_id 
            FROM EventRegistrations r
            JOIN Events e ON r.event_id = e.id
            WHERE r.event_registration_id = ?`, regID).Scan(&eventSchoolID, &eventID)
		if err == sql.ErrNoRows {
			log.Printf("Registration %d not found", regID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}
		if err != nil {
			log.Printf("Error fetching event details for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking registration"})
			return
		}

		if role == "schooladmin" {
			if !userSchoolID.Valid {
				log.Printf("No school assigned to schooladmin %d", userID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "No school assigned"})
				return
			}
			if eventSchoolID != int(userSchoolID.Int64) {
				log.Printf("School mismatch for schooladmin %d, event %d: user school %v, event school %d", userID, eventID, userSchoolID, eventSchoolID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only approve/cancel registrations for your school's events"})
				return
			}
		}

		_, err = tx.Exec("UPDATE EventRegistrations SET status = ? WHERE event_registration_id = ?", body.Status, regID)
		if err != nil {
			log.Printf("Error updating status for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update status"})
			return
		}

		var registration models.EventRegistration
		var regDateStr string
		var eventEnd sql.NullString
		var schoolName sql.NullString
		var status sql.NullString

		err = tx.QueryRow(`
            SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
                   r.school_id,
                   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
                   sc.school_name,
                   e.event_name, e.start_date, e.end_date
            FROM EventRegistrations r
            JOIN student s ON r.student_id = s.student_id
            JOIN Schools sc ON r.school_id = sc.school_id
            JOIN Events e ON r.event_id = e.id
            WHERE r.event_registration_id = ?`, regID).Scan(
			&registration.EventRegistrationID,
			&registration.StudentID,
			&registration.EventID,
			&regDateStr,
			&status,
			&registration.SchoolID,
			&registration.StudentFirstName,
			&registration.StudentLastName,
			&registration.StudentPatronymic,
			&registration.StudentGrade,
			&registration.StudentLetter,
			&schoolName,
			&registration.EventName,
			&registration.EventStartDate,
			&eventEnd,
		)
		if err != nil {
			log.Printf("Error fetching updated registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch updated registration"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction for registration %d: %v", regID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update registration"})
			return
		}

		registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
		if status.Valid {
			registration.Status = status.String
		}
		if schoolName.Valid {
			registration.SchoolName = schoolName.String
		}
		if eventEnd.Valid {
			registration.EventEndDate = eventEnd.String
		}

		utils.ResponseJSON(w, registration)
	}
}

func (ec *EventsRegistrationController) GetSchoolRanking(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if err != nil || (role != "schooladmin" && role != "superadmin" && role != "student") {
			log.Printf("Access denied for user %d: invalid role or error %v", userID, err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		// Fetch total registered participants per school
		rows, err := db.Query(`
            SELECT s.school_id, s.school_name, COUNT(r.event_registration_id) as participant_count
            FROM Schools s
            LEFT JOIN EventRegistrations r ON r.school_id = s.school_id
            WHERE r.status = 'registered'
            GROUP BY s.school_id, s.school_name`)
		if err != nil {
			log.Printf("Error querying school rankings: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer rows.Close()

		type SchoolRanking struct {
			SchoolID         int    `json:"school_id"`
			SchoolName       string `json:"school_name"`
			ParticipantCount int    `json:"participant_count"`
		}

		var rankings []SchoolRanking
		for rows.Next() {
			var ranking SchoolRanking
			err := rows.Scan(&ranking.SchoolID, &ranking.SchoolName, &ranking.ParticipantCount)
			if err != nil {
				log.Printf("Error scanning school ranking: %v", err)
				continue
			}
			rankings = append(rankings, ranking)
		}

		// Sort rankings by participant count in descending order
		sort.Slice(rankings, func(i, j int) bool {
			return rankings[i].ParticipantCount > rankings[j].ParticipantCount
		})

		utils.ResponseJSON(w, rankings)
	}
}
