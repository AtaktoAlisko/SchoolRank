package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"ranking-school/models"
	"ranking-school/utils"

	"github.com/gorilla/mux"
)

type EventsRegistrationController struct{}

func (ec *EventsRegistrationController) RegisterForEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Println("Token verification failed:", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		var student models.Student
		err = db.QueryRow("SELECT student_id, school_id, grade FROM student WHERE student_id = ?", userID).Scan(
			&student.ID, &student.SchoolID, &student.Grade)
		if err == sql.ErrNoRows {
			log.Println("No student found for userID:", userID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Student not found"})
			return
		}
		if err != nil {
			log.Println("Error querying student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
			return
		}

		if student.SchoolID == 0 {
			student.SchoolID = 1
			_, err = db.Exec("UPDATE student SET school_id = ? WHERE student_id = ?", student.SchoolID, student.ID)
			if err != nil {
				log.Println("Error assigning default school:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to assign default school"})
				return
			}
		}

		var request struct {
			EventID int `json:"event_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Println("Invalid request body:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}
		defer r.Body.Close()

		if request.EventID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid event ID"})
			return
		}

		var event models.Event
		var endDateStr sql.NullString
		var limit sql.NullInt64
		err = db.QueryRow(`
            SELECT id, event_name, start_date, end_date, limit_count, school_id, grade
            FROM Events WHERE id = ?`, request.EventID).Scan(
			&event.ID, &event.EventName, &event.StartDate, &endDateStr, &limit, &event.SchoolID, &event.Grade)
		if err == sql.ErrNoRows {
			log.Println("Event not found for eventID:", request.EventID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Event not found"})
			return
		}
		if err != nil {
			log.Println("Error querying event:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching event details"})
			return
		}

		if endDateStr.Valid {
			event.EndDate = endDateStr.String
		} else {
			start, _ := time.Parse("2006-01-02", event.StartDate)
			event.EndDate = start.AddDate(0, 0, 1).Format("2006-01-02")
		}
		if limit.Valid {
			event.Limit = int(limit.Int64)
		} else {
			event.Limit = 100
		}

		parsedEndDate, err := time.Parse("2006-01-02", event.EndDate)
		if err != nil {
			log.Println("Error parsing event end date:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Invalid event end date"})
			return
		}
		if time.Now().After(parsedEndDate) {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration period has ended"})
			return
		}

		if event.Grade > 0 && student.Grade != event.Grade {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "This event is only for students in grade " + strconv.Itoa(event.Grade)})
			return
		}

		var currentCount int
		err = db.QueryRow(`
            SELECT COUNT(*) FROM EventRegistrations
            WHERE event_id = ? AND status = 'registered'`, request.EventID).Scan(&currentCount)
		if err != nil {
			log.Println("Error counting registrations:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}

		if event.Limit > 0 && currentCount >= event.Limit {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Registration limit reached"})
			return
		}

		var exists int
		err = db.QueryRow(`
            SELECT COUNT(*) FROM EventRegistrations
            WHERE student_id = ? AND event_id = ? AND status = 'registered'`,
			student.ID, request.EventID).Scan(&exists)
		if err != nil {
			log.Println("Error checking existing registration:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		if exists > 0 {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Already registered"})
			return
		}

		now := time.Now()
		result, err := db.Exec(`
            INSERT INTO EventRegistrations
            (student_id, event_id, registration_date, status, school_id)
            VALUES (?, ?, ?, 'registered', ?)`,
			student.ID, request.EventID, now.Format("2006-01-02 15:04:05"), student.SchoolID)
		if err != nil {
			log.Println("Error inserting registration:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Registration failed"})
			return
		}

		regID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error getting registration ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Registration successful but failed to retrieve ID"})
			return
		}

		var registration models.EventRegistration
		var regDateStr string
		var eventEnd sql.NullString
		var schoolName sql.NullString

		err = db.QueryRow(`
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
			&registration.Status,
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
			log.Println("Error fetching registration details:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching registration details"})
			return
		}

		registration.RegistrationDate, err = time.Parse("2006-01-02 15:04:05", regDateStr)
		if err != nil {
			log.Println("Error parsing registration date:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error parsing registration date"})
			return
		}
		registration.EventEndDate = ""
		if eventEnd.Valid {
			registration.EventEndDate = eventEnd.String
		}
		if schoolName.Valid {
			registration.SchoolName = schoolName.String
		}

		utils.ResponseJSON(w, registration)
	}
}
func (ec *EventsRegistrationController) UpdateEventRegistrationStatus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		idStr := mux.Vars(r)["id"]
		regID, err := strconv.Atoi(idStr)
		if err != nil || regID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid body"})
			return
		}

		validStatuses := map[string]bool{
			"canceled":  true,
			"completed": true,
		}
		if !validStatuses[body.Status] {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status"})
			return
		}

		if role == "schooladmin" {
			var regSchoolID int
			err = db.QueryRow("SELECT school_id FROM EventRegistrations WHERE event_registration_id = ?", regID).Scan(&regSchoolID)
			if err != nil {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
				return
			}
			if !userSchoolID.Valid || regSchoolID != int(userSchoolID.Int64) {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only update registrations for your school"})
				return
			}
		}

		_, err = db.Exec("UPDATE EventRegistrations SET status = ? WHERE event_registration_id = ?", body.Status, regID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update status"})
			return
		}

		var registration models.EventRegistration
		var regDateStr string
		var eventEnd sql.NullString
		var schoolName sql.NullString
		var status sql.NullString

		err = db.QueryRow(`
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
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch updated registration"})
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
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		var schoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &schoolID)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		var rows *sql.Rows
		if role == "schooladmin" {
			if !schoolID.Valid {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "No school assigned"})
				return
			}
			rows, err = db.Query(`
				SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
					   r.school_id,
					   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
					   sc.school_name,
					   e.event_name, e.start_date, e.end_date
				FROM EventRegistrations r
				JOIN student s ON r.student_id = s.student_id
				JOIN Schools sc ON r.school_id = sc.school_id
				JOIN Events e ON r.event_id = e.id
				WHERE r.school_id = ?`, schoolID.Int64)
		} else {
			rows, err = db.Query(`
				SELECT r.event_registration_id, r.student_id, r.event_id, r.registration_date, r.status,
					   r.school_id,
					   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
					   sc.school_name,
					   e.event_name, e.start_date, e.end_date
				FROM EventRegistrations r
				JOIN student s ON r.student_id = s.student_id
				JOIN Schools sc ON r.school_id = sc.school_id
				JOIN Events e ON r.event_id = e.id`)
		}

		if err != nil {
			log.Println("Error querying event registrations:", err)
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
				&schoolName,
				&reg.EventName,
				&reg.EventStartDate,
				&endDate,
			)
			if err != nil {
				log.Println("Error scanning registration:", err)
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

			registrations = append(registrations, reg)
		}

		utils.ResponseJSON(w, registrations)
	}
}
func (ec *EventsRegistrationController) DeleteEventRegistrationByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var role string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil || (role != "superadmin" && role != "schooladmin") {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		idStr := mux.Vars(r)["id"]
		regID, err := strconv.Atoi(idStr)
		if err != nil || regID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		// Проверка schooladmin
		if role == "schooladmin" {
			var regSchoolID int
			err = db.QueryRow("SELECT school_id FROM EventRegistrations WHERE event_registration_id = ?", regID).Scan(&regSchoolID)
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
				return
			}
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking registration"})
				return
			}
			if !userSchoolID.Valid || regSchoolID != int(userSchoolID.Int64) {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only delete registrations for your school"})
				return
			}
		}

		// Удаление
		res, err := db.Exec("DELETE FROM EventRegistrations WHERE event_registration_id = ?", regID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete registration"})
			return
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "Registration deleted successfully"})
	}
}
