package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ranking-school/models"
	"ranking-school/utils"

	"github.com/gorilla/mux"
)

type EventsRegistrationController struct{}

// 		currentTime := time.Now().Format("2006-01-02 15:04:05")
// 		var schoolIDForInsert interface{}
// 		if studentSchoolID.Valid {
// 			schoolIDForInsert = studentSchoolID.Int64
// 		} else {
// 			schoolIDForInsert = nil
// 		}

// 		result, err := tx.Exec(`
// 			INSERT INTO EventRegistrations (student_id, event_id, registration_date, status, school_id)
// 			VALUES (?, ?, ?, 'registered', ?)`,
// 			userID, eventID, currentTime, schoolIDForInsert)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to register for event"})
// 			return
// 		}

// 		_, err = tx.Exec("UPDATE Events SET participants = participants + 1 WHERE id = ?", eventID)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update participants count"})
// 			return
// 		}

// 		registrationID, _ := result.LastInsertId()

// 		var registration models.EventRegistration
// 		var regDateStr string
// 		var eventEnd sql.NullString
// 		var schoolName sql.NullString
// 		var status sql.NullString

// 		err = tx.QueryRow(`
// 			SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
// 				   COALESCE(r.school_id, 0) as school_id,
// 				   COALESCE(s.first_name, '') as first_name,
// 				   COALESCE(s.last_name, '') as last_name,
// 				   COALESCE(s.patronymic, '') as patronymic,
// 				   COALESCE(s.grade, 0) as grade,
// 				   COALESCE(s.letter, '') as letter,
// 				   COALESCE(sc.school_name, '') as school_name,
// 				   COALESCE(e.event_name, '') as event_name,
// 				   COALESCE(e.start_date, '') as start_date,
// 				   e.end_date
// 			FROM EventRegistrations r
// 			LEFT JOIN student s ON r.student_id = s.student_id
// 			LEFT JOIN Schools sc ON r.school_id = sc.school_id
// 			LEFT JOIN Events e ON r.event_id = e.id
// 			WHERE r.student_id = ? AND r.event_id = ? AND r.registration_date = ?`,
// 			userID, eventID, currentTime).Scan(
// 			&registration.EventRegistrationID,
// 			&registration.StudentID,
// 			&registration.EventID,
// 			&regDateStr,
// 			&status,
// 			&registration.SchoolID,
// 			&registration.StudentFirstName,
// 			&registration.StudentLastName,
// 			&registration.StudentPatronymic,
// 			&registration.StudentGrade,
// 			&registration.StudentLetter,
// 			&schoolName,
// 			&registration.EventName,
// 			&registration.EventStartDate,
// 			&eventEnd,
// 		)

// 		if err != nil {
// 			registration = models.EventRegistration{
// 				EventRegistrationID: int(registrationID),
// 				StudentID:           userID,
// 				EventID:             eventID,
// 				Status:              "registered",
// 				Message:             "Successfully registered for event",
// 			}
// 		} else {
// 			registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
// 			if status.Valid {
// 				registration.Status = status.String
// 			}
// 			if schoolName.Valid {
// 				registration.SchoolName = schoolName.String
// 			}
// 			if eventEnd.Valid {
// 				registration.EventEndDate = eventEnd.String
// 			}
// 			registration.Message = "Successfully registered for event"
// 		}

// 		if err := tx.Commit(); err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to complete registration"})
// 			return
// 		}

// 		utils.ResponseJSON(w, registration)
// 	}
// }

func (ec *EventsRegistrationController) RegisterForEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		var studentSchoolID sql.NullInt64
		var studentGrade int
		err = db.QueryRow("SELECT role, school_id, grade FROM student WHERE student_id = ?", userID).Scan(&role, &studentSchoolID, &studentGrade)
		if err != nil || role != "student" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only students can register for events"})
			return
		}

		var body struct {
			EventID int `json:"event_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid body"})
			return
		}
		defer r.Body.Close()

		eventID := body.EventID
		if eventID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer tx.Rollback()

		var existingRegistration int
		err = tx.QueryRow("SELECT COUNT(*) FROM EventRegistrations WHERE student_id = ? AND event_id = ? AND status = 'registered'", userID, eventID).Scan(&existingRegistration)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		if existingRegistration > 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Already registered for this event"})
			return
		}

		var eventGrade, limitCount, participants int
		var eventSchoolID int
		var eventName, eventStartDate string

		err = tx.QueryRow("SELECT school_id, grade, limit_count, event_name, participants, start_date FROM Events WHERE id = ?", eventID).
			Scan(&eventSchoolID, &eventGrade, &limitCount, &eventName, &participants, &eventStartDate)
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
			return
		}
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking event"})
			return
		}

		parsedStartDate, err := time.Parse("2006-01-02", eventStartDate)
		if err == nil && time.Now().After(parsedStartDate) {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration has already closed"})
			return
		}

		if studentGrade != eventGrade {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Your grade does not match the required grade for this event"})
			return
		}

		// Обработка school_id
		var schoolID int64
		if studentSchoolID.Valid {
			schoolID = studentSchoolID.Int64
		} else {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School is not set for this student"})
			return
		}

		// 🔒 Ограничение 2 участника только от школы-организатора
		if schoolID == int64(eventSchoolID) {
			var countFromCreatorSchool int
			err = tx.QueryRow(`
		SELECT COUNT(*) FROM EventRegistrations
		WHERE event_id = ? AND school_id = ?`,
				eventID, schoolID).Scan(&countFromCreatorSchool)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking school participant count"})
				return
			}
			if countFromCreatorSchool >= 2 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Only 2 participants allowed from the creator's school"})
				return
			}
		}

		if limitCount > 0 && participants >= limitCount {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration limit for this event has been reached"})
			return
		}

		currentTime := time.Now().Format("2006-01-02 15:04:05")

		result, err := tx.Exec(`
			INSERT INTO EventRegistrations (student_id, event_id, registration_date, status, school_id)
			VALUES (?, ?, ?, 'registered', ?)`,
			userID, eventID, currentTime, schoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to register for event"})
			return
		}

		_, err = tx.Exec("UPDATE Events SET participants = participants + 1 WHERE id = ?", eventID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update participants count"})
			return
		}

		registrationID, _ := result.LastInsertId()

		var registration models.EventRegistration
		var regDateStr string
		var eventEnd sql.NullString
		var schoolName sql.NullString
		var status sql.NullString

		err = tx.QueryRow(`
			SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
				   COALESCE(r.school_id, 0) as school_id,
				   COALESCE(s.first_name, '') as first_name, 
				   COALESCE(s.last_name, '') as last_name, 
				   COALESCE(s.patronymic, '') as patronymic, 
				   COALESCE(s.grade, 0) as grade, 
				   COALESCE(s.letter, '') as letter,
				   COALESCE(sc.school_name, '') as school_name,
				   COALESCE(e.event_name, '') as event_name, 
				   COALESCE(e.start_date, '') as start_date, 
				   e.end_date
			FROM EventRegistrations r
			LEFT JOIN student s ON r.student_id = s.student_id
			LEFT JOIN Schools sc ON r.school_id = sc.school_id
			LEFT JOIN Events e ON r.event_id = e.id
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
			registration = models.EventRegistration{
				EventRegistrationID: int(registrationID),
				StudentID:           userID,
				EventID:             eventID,
				Status:              "registered",
				Message:             "Successfully registered for event",
			}
		} else {
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
			registration.Message = "Successfully registered for event"
		}

		if err := tx.Commit(); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to complete registration"})
			return
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

		// if role == "schooladmin" {
		// 	var regSchoolID int
		// 	err = tx.QueryRow("SELECT school_id FROM EventRegistrations WHERE event_registration_id = ?", regID).Scan(&regSchoolID)
		// 	if err != nil {
		// 		log.Printf("Error fetching school ID for registration %d: %v", regID, err)
		// 		utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
		// 		return
		// 	}
		// 	if !userSchoolID.Valid || regSchoolID != int(userSchoolID.Int64) {
		// 		log.Printf("School mismatch for user %d, registration %d: user school %v, reg school %d", userID, regID, userSchoolID, regSchoolID)
		// 		utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only update registrations for your school"})
		// 		return
		// 	}
		// }

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
                WHERE r.school_id = ?`, schoolID.Int64) // УБРАЛ фильтр по статусу
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
                ORDER BY r.status DESC`) // УБРАЛ фильтр и добавил сортировку по статусу
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

			// Преобразование и установка дополнительных полей
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
func (ec *EventsRegistrationController) GetParticipantsBySchoolID(db *sql.DB) http.HandlerFunc {
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
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			log.Printf("Access denied for user %d: invalid role or error %v", userID, err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		// Get school_id from URL parameters
		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil || schoolID <= 0 {
			log.Printf("Invalid school ID %s: %v", schoolIDStr, err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// For schooladmin, ensure they can only access their own school's data
		if role == "schooladmin" {
			if !userSchoolID.Valid || int(userSchoolID.Int64) != schoolID {
				log.Printf("School mismatch for user %d: user school %v, requested school %d", userID, userSchoolID, schoolID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only access your school's participants"})
				return
			}
		}

		// Get status filter from query parameters (default to all if not provided)
		statusFilter := r.URL.Query().Get("status")
		if statusFilter != "" && statusFilter != "registered" && statusFilter != "accepted" && statusFilter != "canceled" {
			log.Printf("Invalid status filter %s for school %d", statusFilter, schoolID)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status filter. Use 'registered', 'accepted', or 'canceled'"})
			return
		}

		// Query registrations for the specified school_id with optional status filter
		query := `
            SELECT 
                r.event_registration_id,
                s.first_name,
                s.last_name,
                s.patronymic,
                s.grade,
                s.letter,
                s.role AS student_role,
                s.email,
                s.phone,
                s.iin,
                sc.school_name,
                e.event_name,
                e.category
            FROM EventRegistrations r
            JOIN student s ON r.student_id = s.student_id
            JOIN Schools sc ON r.school_id = sc.school_id
            JOIN Events e ON r.event_id = e.id
            WHERE r.school_id = ?`
		var args []interface{}
		args = append(args, schoolID)

		if statusFilter != "" {
			query += " AND r.status = ?"
			args = append(args, statusFilter)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Error querying participant registrations for school %d: %v", schoolID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer rows.Close()

		// Define a struct to hold the response data
		type Participant struct {
			EventRegistrationID int    `json:"event_registration_id"`
			FullName            string `json:"full_name"`
			StudentFirstName    string `json:"student_first_name"`
			StudentLastName     string `json:"student_last_name"`
			StudentPatronymic   string `json:"student_patronymic"`
			StudentGrade        string `json:"student_grade"`
			StudentLetter       string `json:"student_letter"`
			StudentRole         string `json:"student_role"`
			Email               string `json:"email"`
			Phone               string `json:"phone"`
			IIN                 string `json:"iin"`
			SchoolName          string `json:"school_name"`
			EventName           string `json:"event_name"`
			EventCategory       string `json:"event_category,omitempty"`
		}

		var participants []Participant
		for rows.Next() {
			var p Participant
			var firstName, lastName, patronymic, schoolName, eventName, eventCategory, grade, letter, studentRole, email, phone, iin sql.NullString

			err := rows.Scan(
				&p.EventRegistrationID,
				&firstName,
				&lastName,
				&patronymic,
				&grade,
				&letter,
				&studentRole,
				&email,
				&phone,
				&iin,
				&schoolName,
				&eventName,
				&eventCategory,
			)
			if err != nil {
				log.Printf("Error scanning participant: %v", err)
				continue
			}

			// Assign values, handling NULLs
			p.StudentFirstName = firstName.String
			p.StudentLastName = lastName.String
			p.StudentPatronymic = patronymic.String
			p.StudentGrade = grade.String
			p.StudentLetter = letter.String
			p.StudentRole = studentRole.String
			p.Email = email.String
			p.Phone = phone.String
			p.IIN = iin.String
			p.SchoolName = schoolName.String
			p.EventName = eventName.String
			if eventCategory.Valid {
				p.EventCategory = eventCategory.String
			}

			// Construct FullName
			fullNameParts := []string{}
			if firstName.Valid && firstName.String != "" {
				fullNameParts = append(fullNameParts, firstName.String)
			}
			if lastName.Valid && lastName.String != "" {
				fullNameParts = append(fullNameParts, lastName.String)
			}
			if patronymic.Valid && patronymic.String != "" {
				fullNameParts = append(fullNameParts, patronymic.String)
			}
			p.FullName = strings.Join(fullNameParts, " ")

			participants = append(participants, p)
		}

		// Query summary of participant counts by status
		summaryQuery := `
            SELECT status, COUNT(*) as count
            FROM EventRegistrations
            WHERE school_id = ?
            GROUP BY status`
		rows, err = db.Query(summaryQuery, schoolID)
		if err != nil {
			log.Printf("Error querying participant summary for school %d: %v", schoolID, err)
			// Continue without summary if query fails
		} else {
			defer rows.Close()
			type StatusCount struct {
				Status string `json:"status"`
				Count  int    `json:"count"`
			}
			var summary []StatusCount
			for rows.Next() {
				var s StatusCount
				if err := rows.Scan(&s.Status, &s.Count); err != nil {
					log.Printf("Error scanning status count: %v", err)
					continue
				}
				summary = append(summary, s)
			}
			// Add summary to response
			if len(summary) > 0 {
				for i := range participants {
					participants[i].FullName += " (Summary: " + strings.Join(func() []string {
						var result []string
						for _, s := range summary {
							result = append(result, fmt.Sprintf("%s: %d", s.Status, s.Count))
						}
						return result
					}(), ", ") + ")"
				}
			}
		}

		log.Printf("Participants found for school %d: %d", schoolID, len(participants))
		utils.ResponseJSON(w, participants)
	}
}
func (ec *EventsRegistrationController) GetEventParticipantsBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that GET method is used
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Build SQL query to count participants per school
		query := `
            SELECT s.school_id, s.school_name, COUNT(r.event_registration_id) as participant_count
            FROM Schools s
            LEFT JOIN EventRegistrations r ON s.school_id = r.school_id
            WHERE r.status = 'registered'
            GROUP BY s.school_id, s.school_name
            ORDER BY s.school_id ASC
        `

		// Execute query
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error executing participant count query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch participant counts"})
			return
		}
		defer rows.Close()

		// Define struct for response
		type SchoolParticipantCount struct {
			SchoolID         int    `json:"school_id"`
			SchoolName       string `json:"school_name"`
			ParticipantCount int    `json:"participant_count"`
		}

		// Collect results
		var schoolParticipantCounts []SchoolParticipantCount
		for rows.Next() {
			var spc SchoolParticipantCount
			err := rows.Scan(&spc.SchoolID, &spc.SchoolName, &spc.ParticipantCount)
			if err != nil {
				log.Println("Error scanning participant count row:", err)
				continue
			}
			schoolParticipantCounts = append(schoolParticipantCounts, spc)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating participant count rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing participant count data"})
			return
		}

		// Prepare response
		response := map[string]interface{}{
			"schools":       schoolParticipantCounts,
			"total_schools": len(schoolParticipantCounts),
		}

		if len(schoolParticipantCounts) == 0 {
			response["message"] = "No schools found with event participants"
		}

		utils.ResponseJSON(w, response)
	}
}
func (ec *EventsRegistrationController) GetEventParticipantsByAllSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that GET method is used
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Get status filter from query parameters
		statusFilter := r.URL.Query().Get("status")
		validStatuses := map[string]bool{
			"registered": true,
			"accepted":   true,
			"canceled":   true,
			"completed":  true, // Added to account for UpdateEventRegistrationStatus
		}
		if statusFilter != "" && !validStatuses[statusFilter] {
			log.Printf("Invalid status filter: %s", statusFilter)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status filter. Use 'registered', 'accepted', 'canceled', or 'completed'"})
			return
		}

		// Build SQL query to count participants per school
		query := `
            SELECT s.school_id, s.school_name, COALESCE(COUNT(r.event_registration_id), 0) as participant_count
            FROM Schools s
            LEFT JOIN EventRegistrations r ON s.school_id = r.school_id`
		args := []interface{}{}

		if statusFilter != "" {
			query += " AND r.status = ?"
			args = append(args, statusFilter)
		} else {
			query += " AND r.status IN ('registered', 'accepted', 'completed')"
		}

		query += `
            GROUP BY s.school_id, s.school_name
            ORDER BY s.school_id ASC`

		// Execute query
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println("Error executing participant count query:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch participant counts"})
			return
		}
		defer rows.Close()

		// Define struct for response
		type SchoolParticipantCount struct {
			SchoolID         int     `json:"school_id"`
			SchoolName       string  `json:"school_name"`
			ParticipantCount int     `json:"participant_count"`
			Points           float64 `json:"points"`
			Percentage       float64 `json:"percentage"`
		}

		// Collect results
		var schoolParticipantCounts []SchoolParticipantCount
		maxParticipants := 0
		for rows.Next() {
			var spc SchoolParticipantCount
			err := rows.Scan(&spc.SchoolID, &spc.SchoolName, &spc.ParticipantCount)
			if err != nil {
				log.Println("Error scanning participant count row:", err)
				continue
			}
			if spc.ParticipantCount > maxParticipants {
				maxParticipants = spc.ParticipantCount
			}
			schoolParticipantCounts = append(schoolParticipantCounts, spc)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error iterating participant count rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing participant count data"})
			return
		}

		// Calculate points and percentage
		const maxPoints = 30.0
		for i := range schoolParticipantCounts {
			if maxParticipants == 0 {
				schoolParticipantCounts[i].Points = 0
				schoolParticipantCounts[i].Percentage = 0
			} else {
				schoolParticipantCounts[i].Points = (float64(schoolParticipantCounts[i].ParticipantCount) / float64(maxParticipants)) * maxPoints
				schoolParticipantCounts[i].Percentage = (float64(schoolParticipantCounts[i].ParticipantCount) / float64(maxParticipants)) * 100
				// Round to 2 decimal places
				schoolParticipantCounts[i].Points = math.Round(schoolParticipantCounts[i].Points*100) / 100
				schoolParticipantCounts[i].Percentage = math.Round(schoolParticipantCounts[i].Percentage*100) / 100
			}
		}

		// Debug: Log registration status summary
		summaryQuery := `
            SELECT status, COUNT(*)
            FROM EventRegistrations
            GROUP BY status`
		summaryRows, err := db.Query(summaryQuery)
		if err == nil {
			defer summaryRows.Close()
			log.Printf("DEBUG: Registration status summary:")
			for summaryRows.Next() {
				var status string
				var count int
				if err := summaryRows.Scan(&status, &count); err == nil {
					log.Printf("DEBUG: Status %s: %d registrations", status, count)
				}
			}
		} else {
			log.Println("Error querying registration status summary:", err)
		}

		// Debug: Log schools with participants
		log.Printf("DEBUG: Found %d schools, max participants: %d", len(schoolParticipantCounts), maxParticipants)
		for _, spc := range schoolParticipantCounts {
			log.Printf("DEBUG: School %d (%s): %d participants, %.2f points, %.2f%%",
				spc.SchoolID, spc.SchoolName, spc.ParticipantCount, spc.Points, spc.Percentage)
		}

		// Debug: Check for NULL school_id registrations
		var nullSchoolCount int
		err = db.QueryRow("SELECT COUNT(*) FROM EventRegistrations WHERE school_id IS NULL").Scan(&nullSchoolCount)
		if err == nil {
			log.Printf("DEBUG: %d registrations with NULL school_id", nullSchoolCount)
		}

		// Prepare response
		response := map[string]interface{}{
			"schools":       schoolParticipantCounts,
			"total_schools": len(schoolParticipantCounts),
		}

		if len(schoolParticipantCounts) == 0 {
			response["message"] = "No schools found"
		}

		utils.ResponseJSON(w, response)
	}
}
func (ec *EventsRegistrationController) GetEventParticipantsBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Получение и проверка параметра status
		statusFilter := r.URL.Query().Get("status")
		validStatuses := map[string]bool{
			"registered": true,
			"accepted":   true,
			"canceled":   true,
			"completed":  true,
		}
		if statusFilter != "" && !validStatuses[statusFilter] {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status filter"})
			return
		}

		// Извлечение school_id из URL
		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
			return
		}

		// Проверка существования школы
		var schoolExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", schoolID).Scan(&schoolExists)
		if err != nil || !schoolExists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School not found"})
			return
		}

		// Получение общего количества участников по всем школам (для вычисления points и percentage)
		var maxParticipants int
		countQuery := `
            SELECT COUNT(r.event_registration_id)
            FROM Schools s
            LEFT JOIN EventRegistrations r ON s.school_id = r.school_id
            WHERE r.status IN ('registered', 'accepted', 'completed')`
		err = db.QueryRow(countQuery).Scan(&maxParticipants)
		if err != nil {
			log.Println("Error fetching max participants:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate max participants"})
			return
		}

		// Запрос информации по конкретной школе
		query := `
            SELECT 
                s.school_name, 
                COUNT(r.event_registration_id) AS participant_count
            FROM Schools s
            LEFT JOIN EventRegistrations r ON s.school_id = r.school_id
            WHERE s.school_id = ?`

		args := []interface{}{schoolID}
		if statusFilter != "" {
			query += " AND r.status = ?"
			args = append(args, statusFilter)
		} else {
			query += " AND r.status IN ('registered', 'accepted', 'completed')"
		}

		query += " GROUP BY s.school_name"

		var schoolName string
		var participantCount int
		err = db.QueryRow(query, args...).Scan(&schoolName, &participantCount)
		if err != nil {
			log.Println("Error scanning result:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch data"})
			return
		}

		// Вычисление points и percentage
		const maxPoints = 30.0
		var points float64
		var percentage float64
		if maxParticipants > 0 {
			points = (float64(participantCount) / float64(maxParticipants)) * maxPoints
			percentage = (float64(participantCount) / float64(maxParticipants)) * 100
			points = math.Round(points*100) / 100
			percentage = math.Round(percentage*100) / 100
		}

		// Ответная структура
		type SchoolParticipantCount struct {
			SchoolID         int     `json:"school_id"`
			SchoolName       string  `json:"school_name"`
			ParticipantCount int     `json:"participant_count"`
			Points           float64 `json:"points"`
			Percentage       float64 `json:"percentage"`
		}

		response := map[string]interface{}{
			"school": SchoolParticipantCount{
				SchoolID:         schoolID,
				SchoolName:       schoolName,
				ParticipantCount: participantCount,
				Points:           points,
				Percentage:       percentage,
			},
		}

		utils.ResponseJSON(w, response)
	}
}

func (ec *EventsRegistrationController) GetEventRegistrationsByEventID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			log.Printf("Access denied for user %d: %v", userID, err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		// Получение event_id из пути
		vars := mux.Vars(r)
		eventIDStr := vars["event_id"]
		eventID, err := strconv.Atoi(eventIDStr)
		if err != nil || eventID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event_id"})
			return
		}

		rows, err := db.Query(`
			SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
				   r.school_id,
				   s.first_name, s.last_name, s.patronymic, s.grade, s.letter, s.role,
				   sc.school_name,
				   e.event_name, e.start_date, e.end_date, e.category
			FROM EventRegistrations r
			JOIN student s ON r.student_id = s.student_id
			JOIN Schools sc ON r.school_id = sc.school_id
			JOIN Events e ON r.event_id = e.id
			WHERE r.event_id = ?`, eventID)

		if err != nil {
			log.Printf("Query error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer rows.Close()

		var registrations []models.EventRegistration
		for rows.Next() {
			var reg models.EventRegistration
			var regDateStr string
			var endDate, schoolName, status, studentRole, eventCategory sql.NullString

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
				log.Printf("Scan error: %v", err)
				continue
			}

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

		utils.ResponseJSON(w, registrations)
	}
}
