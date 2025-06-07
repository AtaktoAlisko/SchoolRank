package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"ranking-school/models"
	"ranking-school/utils"

	"github.com/gorilla/mux"
)

type OlympiadRegistrationController struct {
	DB *sql.DB
}

// func (c *OlympiadRegistrationController) RegisterStudent(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodPost {
// 			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
// 			return
// 		}

// 		userID, err := utils.VerifyToken(r)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
// 			return
// 		}

// 		var student models.Student
// 		err = db.QueryRow("SELECT student_id, role, school_id FROM student WHERE student_id = ?", userID).Scan(
// 			&student.ID, &student.Role, &student.SchoolID)
// 		if err != nil || student.Role != "student" {
// 			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only students can register for olympiads"})
// 			return
// 		}

// 		if student.SchoolID == 0 {
// 			student.SchoolID = 1
// 			_, err = db.Exec("UPDATE student SET school_id = ? WHERE student_id = ?", student.SchoolID, student.ID)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to assign default school"})
// 				return
// 			}
// 		}

// 		var request struct {
// 			SubjectOlympiadID int `json:"subject_olympiad_id"`
// 		}
// 		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request format"})
// 			return
// 		}
// 		defer r.Body.Close()

// 		if request.SubjectOlympiadID <= 0 {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid subject olympiad ID"})
// 			return
// 		}

// 		var olympiad models.SubjectOlympiad
// 		var endDateStr sql.NullString
// 		var limit sql.NullInt64
// 		var creatorSchoolID int
// 		var participants int
// 		err = db.QueryRow(`
// 		SELECT so.id, so.subject_name, so.date, so.end_date, so.limit_participants, so.school_id, so.participants
// 		FROM subject_olympiads so
// 		WHERE so.id = ?`, request.SubjectOlympiadID).Scan(
// 			&olympiad.ID, &olympiad.SubjectName, &olympiad.StartDate, &endDateStr, &limit, &creatorSchoolID, &participants)

// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Subject olympiad not found"})
// 			return
// 		}

// 		if endDateStr.Valid {
// 			olympiad.EndDate = endDateStr.String
// 		} else {
// 			start, _ := time.Parse("2006-01-02", olympiad.StartDate)
// 			olympiad.EndDate = start.AddDate(0, 0, 1).Format("2006-01-02")
// 		}

// 		if limit.Valid {
// 			olympiad.Limit = int(limit.Int64)
// 		} else {
// 			olympiad.Limit = 100
// 		}

// 		parsedEndDate, _ := time.Parse("2006-01-02", olympiad.EndDate)
// 		if time.Now().After(parsedEndDate) {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration period has ended"})
// 			return
// 		}

// 		var exists int
// 		err = db.QueryRow(`
// 			SELECT COUNT(*) FROM olympiad_registrations
// 			WHERE student_id = ? AND subject_olympiad_id = ? AND status = 'registered'`,
// 			student.ID, request.SubjectOlympiadID).Scan(&exists)
// 		if err != nil || exists > 0 {
// 			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Already registered"})
// 			return
// 		}

// 		const maxStudentsFromCreatorSchool = 2
// 		if student.SchoolID == creatorSchoolID {
// 			var schoolStudentsCount int
// 			err = db.QueryRow(`
// 				SELECT COUNT(*) FROM olympiad_registrations r
// 				JOIN student s ON r.student_id = s.student_id
// 				WHERE r.subject_olympiad_id = ?
// 				AND s.school_id = ?
// 				AND r.status = 'registered'`,
// 				request.SubjectOlympiadID, creatorSchoolID).Scan(&schoolStudentsCount)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
// 				return
// 			}

// 			if schoolStudentsCount >= maxStudentsFromCreatorSchool {
// 				utils.RespondWithError(w, http.StatusConflict,
// 					models.Error{Message: "Maximum limit for your school has been reached"})
// 				return
// 			}
// 		}

// 		now := time.Now()
// 		result, err := db.Exec(`
// 			INSERT INTO olympiad_registrations
// 			(student_id, subject_olympiad_id, registration_date, status, school_id)
// 			VALUES (?, ?, ?, 'registered', ?)`,
// 			student.ID, request.SubjectOlympiadID, now.Format("2006-01-02 15:04:05"), student.SchoolID)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Registration failed"})
// 			return
// 		}

// 		regID, _ := result.LastInsertId()

// 		// Увеличиваем participants на 1
// 		_, err = db.Exec("UPDATE subject_olympiads SET current_participants = current_participants + 1 WHERE id = ?", request.SubjectOlympiadID)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update participants count"})
// 			return
// 		}

// 		// Проверяем не превышен ли лимит после инкремента
// 		var updatedParticipants int
// 		err = db.QueryRow("SELECT current_participants FROM subject_olympiads WHERE id = ?", request.SubjectOlympiadID).Scan(&updatedParticipants)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to re-check participants count"})
// 			return
// 		}
// 		if olympiad.Limit > 0 && updatedParticipants > olympiad.Limit {
// 			// откатываем регистрацию
// 			_, _ = db.Exec("DELETE FROM olympiad_registrations WHERE olympiads_registrations_id = ?", regID)
// 			_, _ = db.Exec("UPDATE subject_olympiads SET participants = participants - 1 WHERE id = ?", request.SubjectOlympiadID)

// 			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Registration limit exceeded, your slot was revoked"})
// 			return
// 		}

// 		var registration models.OlympiadRegistration
// 		var regDateStr string
// 		var olympiadEnd sql.NullString
// 		var olympiadLevel sql.NullString

// 		err = db.QueryRow(`
// 			SELECT r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
// 				   r.school_id,
// 				   CONCAT(s.first_name, ' ', s.last_name) AS student_name,
// 				   s.grade, s.letter,
// 				   o.subject_name, o.date, o.end_date, o.level
// 			FROM olympiad_registrations r
// 			JOIN student s ON r.student_id = s.student_id
// JOIN subject_olympiads o ON r.subject_olympiad_id = o.id
// 			WHERE r.olympiads_registrations_id = ?`, regID).Scan(
// 			&registration.OlympiadsRegistrationsID,
// 			&registration.StudentID,
// 			&registration.SubjectOlympiadID,
// 			&regDateStr,
// 			&registration.Status,
// 			&registration.SchoolID,
// 			&registration.StudentName,
// 			&registration.StudentGrade,
// 			&registration.StudentLetter,
// 			&registration.OlympiadName,
// 			&registration.OlympiadStartDate,
// 			&olympiadEnd,
// 			&olympiadLevel,
// 		)

// 		if err == nil {
// 			registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
// 			if olympiadEnd.Valid {
// 				registration.OlympiadEndDate = olympiadEnd.String
// 			}
// 			if olympiadLevel.Valid {
// 				registration.Level = olympiadLevel.String
// 			}
// 		} else {
// 			registration = models.OlympiadRegistration{
// 				OlympiadsRegistrationsID: int(regID),
// 				StudentID:                student.ID,
// 				SubjectOlympiadID:        request.SubjectOlympiadID,
// 				Status:                   "registered",
// 				SchoolID:                 student.SchoolID,
// 			}
// 		}

// 		utils.ResponseJSON(w, registration)
// 	}
// }

// func (c *OlympiadRegistrationController) RegisterStudent(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodPost {
// 			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
// 			return
// 		}

// 		userID, err := utils.VerifyToken(r)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
// 			return
// 		}

// 		var student models.Student
// 		err = db.QueryRow("SELECT student_id, role, school_id FROM student WHERE student_id = ?", userID).Scan(
// 			&student.ID, &student.Role, &student.SchoolID)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only students can register for olympiads"})
// 			return
// 		}

// 		if student.SchoolID == 0 {
// 			student.SchoolID = 1
// 			_, err = db.Exec("UPDATE student SET school_id = ? WHERE student_id = ?", student.SchoolID, student.ID)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to assign default school"})
// 				return
// 			}
// 		}

// 		var request struct {
// 			SubjectOlympiadID int `json:"subject_olympiad_id"`
// 		}
// 		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request format"})
// 			return
// 		}
// 		defer r.Body.Close()

// 		if request.SubjectOlympiadID <= 0 {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid subject olympiad ID"})
// 			return
// 		}

// 		var olympiad models.SubjectOlympiad
// 		var endDateStr sql.NullString
// 		var limit sql.NullInt64
// 		var creatorSchoolID int
// 		var currentParticipants int

// 		err = db.QueryRow(`
// 			SELECT so.subject_olympiad_id, so.subject_name, so.date, so.end_date,
// 			so.limit_participants, so.school_id, so.current_participants
// 			FROM subject_olympiads so
// 			WHERE so.subject_olympiad_id = ?`, request.SubjectOlympiadID).Scan(
// 			&olympiad.ID, &olympiad.SubjectName, &olympiad.StartDate, &endDateStr, &limit, &creatorSchoolID, &currentParticipants)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Subject olympiad not found"})
// 			return
// 		}

// 		if endDateStr.Valid {
// 			olympiad.EndDate = endDateStr.String
// 		} else {
// 			start, _ := time.Parse("2006-01-02", olympiad.StartDate)
// 			olympiad.EndDate = start.AddDate(0, 0, 1).Format("2006-01-02")
// 		}

// 		if limit.Valid {
// 			olympiad.Limit = int(limit.Int64)
// 		} else {
// 			olympiad.Limit = 100
// 		}

// 		parsedEndDate, _ := time.Parse("2006-01-02", olympiad.EndDate)
// 		if time.Now().After(parsedEndDate) {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration period has ended"})
// 			return
// 		}

// 		var exists int
// 		err = db.QueryRow(`
// 			SELECT COUNT(*) FROM olympiad_registrations
// 			WHERE student_id = ? AND subject_olympiad_id = ? AND status = 'registered'`,
// 			student.ID, request.SubjectOlympiadID).Scan(&exists)
// 		if err != nil || exists > 0 {
// 			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Already registered"})
// 			return
// 		}

// 		if olympiad.Limit > 0 && currentParticipants >= olympiad.Limit {
// 			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Registration limit reached"})
// 			return
// 		}

// 		const maxStudentsFromCreatorSchool = 2
// 		if student.SchoolID == creatorSchoolID {
// 			var schoolStudentsCount int
// 			err = db.QueryRow(`
// 				SELECT COUNT(*) FROM olympiad_registrations r
// 				JOIN student s ON r.student_id = s.student_id
// 				WHERE r.subject_olympiad_id = ? AND s.school_id = ? AND r.status = 'registered'`,
// 				request.SubjectOlympiadID, creatorSchoolID).Scan(&schoolStudentsCount)
// 			if err != nil {
// 				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
// 				return
// 			}

// 			if schoolStudentsCount >= maxStudentsFromCreatorSchool {
// 				utils.RespondWithError(w, http.StatusConflict,
// 					models.Error{Message: "Maximum limit for your school has been reached"})
// 				return
// 			}
// 		}

// 		now := time.Now()
// 		result, err := db.Exec(`
// 			INSERT INTO olympiad_registrations
// 			(student_id, subject_olympiad_id, registration_date, status, school_id)
// 			VALUES (?, ?, ?, 'registered', ?)`,
// 			student.ID, request.SubjectOlympiadID, now.Format("2006-01-02 15:04:05"), student.SchoolID)
// 		if err != nil {
// 			log.Printf("Failed to insert registration: %v", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Registration failed"})
// 			return
// 		}

// 		regID, _ := result.LastInsertId()

// 		var registration models.OlympiadRegistration
// 		var regDateStr string
// 		var olympiadEnd sql.NullString
// 		var olympiadLevel sql.NullString

// 		err = db.QueryRow(`
// 			SELECT r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
// 					   r.school_id,
// 					   CONCAT(s.first_name, ' ', s.last_name) AS student_name,
// 					   s.grade, s.letter,
// 					   o.subject_name, o.date, o.end_date, o.level
// 			FROM olympiad_registrations r
// 			JOIN student s ON r.student_id = s.student_id
// 			JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
// 			WHERE r.olympiads_registrations_id = ?`, regID).Scan(
// 			&registration.OlympiadsRegistrationsID,
// 			&registration.StudentID,
// 			&registration.SubjectOlympiadID,
// 			&regDateStr,
// 			&registration.Status,
// 			&registration.SchoolID,
// 			&registration.StudentName,
// 			&registration.StudentGrade,
// 			&registration.StudentLetter,
// 			&registration.OlympiadName,
// 			&registration.OlympiadStartDate,
// 			&olympiadEnd,
// 			&olympiadLevel,
// 		)
// 		if err != nil {
// 			utils.ResponseJSON(w, models.OlympiadRegistration{
// 				OlympiadsRegistrationsID: int(regID),
// 				StudentID:                student.ID,
// 				SubjectOlympiadID:        request.SubjectOlympiadID,
// 				Status:                   "registered",
// 				SchoolID:                 student.SchoolID,
// 			})
// 			return
// 		}

// 		registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
// 		registration.OlympiadEndDate = ""
// 		if olympiadEnd.Valid {
// 			registration.OlympiadEndDate = olympiadEnd.String
// 		}
// 		if olympiadLevel.Valid {
// 			registration.Level = olympiadLevel.String
// 		} else {
// 			registration.Level = ""
// 		}

// 		utils.ResponseJSON(w, registration)
// 	}
// }

func (c *OlympiadRegistrationController) RegisterStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		var student models.Student
		err = db.QueryRow("SELECT student_id, role, school_id, grade FROM student WHERE student_id = ?", userID).Scan(
			&student.ID, &student.Role, &student.SchoolID, &student.Grade)
		if err != nil {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only students can register for olympiads"})
			return
		}

		if student.SchoolID == 0 {
			student.SchoolID = 1
			_, err = db.Exec("UPDATE student SET school_id = ? WHERE student_id = ?", student.SchoolID, student.ID)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to assign default school"})
				return
			}
		}

		var request struct {
			SubjectOlympiadID int `json:"subject_olympiad_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request format"})
			return
		}
		defer r.Body.Close()

		if request.SubjectOlympiadID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid subject olympiad ID"})
			return
		}

		var olympiad models.SubjectOlympiad
		var endDateStr sql.NullString
		var limit sql.NullInt64
		var creatorSchoolID int
		var currentParticipants int
		var olympiadGrade int

		err = db.QueryRow(`
			SELECT so.subject_olympiad_id, so.subject_name, so.date, so.end_date,
			so.limit_participants, so.school_id, so.current_participants, so.grade
			FROM subject_olympiads so
			WHERE so.subject_olympiad_id = ?`, request.SubjectOlympiadID).Scan(
			&olympiad.ID, &olympiad.SubjectName, &olympiad.StartDate, &endDateStr,
			&limit, &creatorSchoolID, &currentParticipants, &olympiadGrade)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Subject olympiad not found"})
			return
		}

		if endDateStr.Valid {
			olympiad.EndDate = endDateStr.String
		} else {
			start, _ := time.Parse("2006-01-02", olympiad.StartDate)
			olympiad.EndDate = start.AddDate(0, 0, 1).Format("2006-01-02")
		}

		if limit.Valid {
			olympiad.Limit = int(limit.Int64)
		} else {
			olympiad.Limit = 100
		}

		parsedEndDate, _ := time.Parse("2006-01-02", olympiad.EndDate)
		if time.Now().After(parsedEndDate) {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Registration period has ended"})
			return
		}

		// Проверка на соответствие класса ученика классу олимпиады
		if student.Grade != olympiadGrade {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "You do not meet the required grade for this olympiad"})
			return
		}

		var exists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM olympiad_registrations
			WHERE student_id = ? AND subject_olympiad_id = ? AND status = 'registered'`,
			student.ID, request.SubjectOlympiadID).Scan(&exists)
		if err != nil || exists > 0 {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Already registered"})
			return
		}

		if olympiad.Limit > 0 && currentParticipants >= olympiad.Limit {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Registration limit reached"})
			return
		}

		const maxStudentsFromCreatorSchool = 2
		if student.SchoolID == creatorSchoolID {
			var schoolStudentsCount int
			err = db.QueryRow(`
				SELECT COUNT(*) FROM olympiad_registrations r
				JOIN student s ON r.student_id = s.student_id
				WHERE r.subject_olympiad_id = ? AND s.school_id = ? AND r.status = 'registered'`,
				request.SubjectOlympiadID, creatorSchoolID).Scan(&schoolStudentsCount)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
				return
			}

			if schoolStudentsCount >= maxStudentsFromCreatorSchool {
				utils.RespondWithError(w, http.StatusConflict,
					models.Error{Message: "Maximum limit for your school has been reached"})
				return
			}
		}

		now := time.Now()
		result, err := db.Exec(`
			INSERT INTO olympiad_registrations
			(student_id, subject_olympiad_id, registration_date, status, school_id)
			VALUES (?, ?, ?, 'registered', ?)`,
			student.ID, request.SubjectOlympiadID, now.Format("2006-01-02 15:04:05"), student.SchoolID)
		if err != nil {
			log.Printf("Failed to insert registration: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Registration failed"})
			return
		}

		regID, _ := result.LastInsertId()

		var registration models.OlympiadRegistration
		var regDateStr string
		var olympiadEnd sql.NullString
		var olympiadLevel sql.NullString

		err = db.QueryRow(`
			SELECT r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
					   r.school_id,
					   CONCAT(s.first_name, ' ', s.last_name) AS student_name,
					   s.grade, s.letter,
					   o.subject_name, o.date, o.end_date, o.level
			FROM olympiad_registrations r
			JOIN student s ON r.student_id = s.student_id
			JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
			WHERE r.olympiads_registrations_id = ?`, regID).Scan(
			&registration.OlympiadsRegistrationsID,
			&registration.StudentID,
			&registration.SubjectOlympiadID,
			&regDateStr,
			&registration.Status,
			&registration.SchoolID,
			&registration.StudentName,
			&registration.StudentGrade,
			&registration.StudentLetter,
			&registration.OlympiadName,
			&registration.OlympiadStartDate,
			&olympiadEnd,
			&olympiadLevel,
		)
		if err != nil {
			utils.ResponseJSON(w, models.OlympiadRegistration{
				OlympiadsRegistrationsID: int(regID),
				StudentID:                student.ID,
				SubjectOlympiadID:        request.SubjectOlympiadID,
				Status:                   "registered",
				SchoolID:                 student.SchoolID,
			})
			return
		}

		registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
		registration.OlympiadEndDate = ""
		if olympiadEnd.Valid {
			registration.OlympiadEndDate = olympiadEnd.String
		}
		if olympiadLevel.Valid {
			registration.Level = olympiadLevel.String
		} else {
			registration.Level = ""
		}

		utils.ResponseJSON(w, registration)
	}
}

func (c *OlympiadRegistrationController) GetOlympiadRegistrations(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		var role string
		var schoolID sql.NullInt64

		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &schoolID)
		if err != nil {
			log.Printf("User with ID %d not found", userID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "User not found"})
			return
		}

		// Filters
		studentID := r.URL.Query().Get("student_id")
		subjectOlympiadID := r.URL.Query().Get("subject_olympiad_id")
		status := r.URL.Query().Get("status")

		query := `
            SELECT r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
                   r.school_id, r.document_url,
                   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
                   sch.school_name,
                   o.subject_name, o.date, o.end_date, o.level,
                   r.score, r.olympiad_place
            FROM olympiad_registrations r
            JOIN student s ON r.student_id = s.student_id
            JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
            JOIN Schools sch ON r.school_id = sch.school_id
            WHERE 1=1`

		var params []interface{}

		switch role {
		case "superadmin":
			if studentID != "" {
				query += " AND r.student_id = ?"
				params = append(params, studentID)
			}
		case "schooladmin":
			if !schoolID.Valid {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Missing school ID for schooladmin"})
				return
			}
			query += " AND (r.school_id = ? OR o.created_by = ?)"
			params = append(params, schoolID.Int64, userID)
			if studentID != "" {
				query += " AND r.student_id = ?"
				params = append(params, studentID)
			}
		case "student":
			query += " AND r.student_id = ?"
			params = append(params, userID)
		default:
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied for role: " + role})
			return
		}

		if subjectOlympiadID != "" {
			query += " AND r.subject_olympiad_id = ?"
			params = append(params, subjectOlympiadID)
		}
		if status != "" {
			query += " AND r.status = ?"
			params = append(params, status)
		}

		query += " ORDER BY r.registration_date DESC"

		rows, err := db.Query(query, params...)
		if err != nil {
			log.Printf("Query error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Query execution failed"})
			return
		}
		defer rows.Close()

		var registrations []models.OlympiadRegistration
		for rows.Next() {
			var reg models.OlympiadRegistration
			var regDateStr string
			var olympiadEnd sql.NullString
			var level sql.NullString
			var score sql.NullInt64
			var olympiadPlace sql.NullInt64
			var documentURL sql.NullString

			err := rows.Scan(
				&reg.OlympiadsRegistrationsID,
				&reg.StudentID,
				&reg.SubjectOlympiadID,
				&regDateStr,
				&reg.Status,
				&reg.SchoolID,
				&documentURL,
				&reg.StudentFirstName,
				&reg.StudentLastName,
				&reg.StudentPatronymic,
				&reg.StudentGrade,
				&reg.StudentLetter,
				&reg.SchoolName,
				&reg.OlympiadName,
				&reg.OlympiadStartDate,
				&olympiadEnd,
				&level,
				&score,
				&olympiadPlace,
			)
			if err != nil {
				log.Printf("Scan error: %v", err)
				continue
			}

			reg.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
			if olympiadEnd.Valid {
				reg.OlympiadEndDate = olympiadEnd.String
			}
			if level.Valid {
				reg.Level = level.String
			}
			if score.Valid {
				reg.Score = int(score.Int64)
			}
			if olympiadPlace.Valid {
				reg.OlympiadPlace = int(olympiadPlace.Int64)
			}
			if documentURL.Valid {
				reg.DocumentURL = documentURL.String
			}

			registrations = append(registrations, reg)
		}

		utils.ResponseJSON(w, registrations)
	}
}
func (c *OlympiadRegistrationController) UpdateRegistrationStatus(db *sql.DB) http.HandlerFunc {
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
			"registered": true,
			"accepted":   true,
			"rejected":   true,
			"pending":    true,
			"canceled":   true,
			"completed":  true,
		}
		if !validStatuses[body.Status] {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid status"})
			return
		}

		// Проверка школы для schooladmin
		// if role == "schooladmin" {
		// 	var regSchoolID int
		// 	err = db.QueryRow("SELECT school_id FROM olympiad_registrations WHERE olympiads_registrations_id = ?", regID).Scan(&regSchoolID)
		// 	if err != nil {
		// 		utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
		// 		return
		// 	}
		// 	if !userSchoolID.Valid || regSchoolID != int(userSchoolID.Int64) {
		// 		utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only update registrations for your school"})
		// 		return
		// 	}
		// }

		// Обновление
		_, err = db.Exec("UPDATE olympiad_registrations SET status = ? WHERE olympiads_registrations_id = ?", body.Status, regID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update status"})
			return
		}

		// Получаем обновлённую запись
		var registration models.OlympiadRegistration
		var regDateStr string
		var olympiadEndDate sql.NullString
		var status sql.NullString

		err = db.QueryRow(`
			SELECT r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
				   r.school_id,
				   CONCAT(s.first_name, ' ', s.last_name) AS student_name,
				   s.grade AS student_grade,
				   s.letter AS student_letter,
				   o.subject_name AS olympiad_name,
				   o.date AS olympiad_start_date,
				   o.end_date AS olympiad_end_date
			FROM olympiad_registrations r
			JOIN student s ON r.student_id = s.student_id
			JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
			WHERE r.olympiads_registrations_id = ?`, regID).Scan(
			&registration.OlympiadsRegistrationsID,
			&registration.StudentID,
			&registration.SubjectOlympiadID,
			&regDateStr,
			&status,
			&registration.SchoolID,
			&registration.StudentName,
			&registration.StudentGrade,
			&registration.StudentLetter,
			&registration.OlympiadName,
			&registration.OlympiadStartDate,
			&olympiadEndDate,
		)

		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch updated registration"})
			return
		}

		registration.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
		if status.Valid {
			registration.Status = status.String
		} else {
			registration.Status = ""
		}
		if olympiadEndDate.Valid {
			registration.OlympiadEndDate = olympiadEndDate.String
		} else {
			registration.OlympiadEndDate = ""
		}

		utils.ResponseJSON(w, registration)
	}
}
func (c *OlympiadRegistrationController) AssignPlaceToRegistration(db *sql.DB) http.HandlerFunc {
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

		// Parse multipart form data (up to 10 MB)
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Failed to parse form data"})
			return
		}

		// Get place from form data
		placeStr := r.FormValue("place")
		place, err := strconv.Atoi(placeStr)
		if err != nil || place < 1 || place > 3 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Place must be 1, 2, or 3"})
			return
		}

		// Handle file upload
		file, fileHeader, err := r.FormFile("document")
		var documentURL string
		if err == nil {
			defer file.Close()

			// Validate file type
			allowedTypes := map[string]bool{
				"image/jpeg":      true,
				"image/png":       true,
				"image/jpg":       true,
				"application/pdf": true,
			}
			contentType := fileHeader.Header.Get("Content-Type")
			if !allowedTypes[contentType] {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid file type. Only JPEG, PNG, and PDF files are allowed"})
				return
			}

			// Generate file name
			fileExt := filepath.Ext(fileHeader.Filename)
			fileName := fmt.Sprintf("olympiad_doc_%d_%s%s", regID, time.Now().Format("20060102150405"), fileExt)

			// Upload file to S3
			documentURL, err = utils.UploadFileToS3(file, fileName, "olympiaddoc")
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: fmt.Sprintf("Failed to upload document to S3: %v", err)})
				return
			}
		} else if err != http.ErrMissingFile {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading document file"})
			return
		}

		// Fetch registration details
		var schoolID, subjectOlympiadID int
		var status string
		var createdBy sql.NullInt64
		err = db.QueryRow(`
            SELECT r.school_id, r.status, r.subject_olympiad_id, o.created_by
            FROM olympiad_registrations r
            JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
            WHERE r.olympiads_registrations_id = ?
        `, regID).Scan(&schoolID, &status, &subjectOlympiadID, &createdBy)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}

		if status != "accepted" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Cannot assign place unless registration is accepted"})
			return
		}

		// Check access for schooladmin
		if role == "schooladmin" {
			hasAccess := false
			if userSchoolID.Valid && schoolID == int(userSchoolID.Int64) {
				hasAccess = true
			} else if createdBy.Valid && int(createdBy.Int64) == userID {
				hasAccess = true
			}
			if !hasAccess {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only assign places for your school or olympiads you created"})
				return
			}
		}

		// Check if place is already assigned
		var existingPlace sql.NullInt64
		err = db.QueryRow(`
            SELECT olympiad_place FROM olympiad_registrations WHERE olympiads_registrations_id = ?
        `, regID).Scan(&existingPlace)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to check existing place"})
			return
		}
		if existingPlace.Valid {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Place already assigned"})
			return
		}

		// Calculate score based on place
		score := 0
		switch place {
		case 1:
			score = 50
		case 2:
			score = 30
		case 3:
			score = 20
		}

		// Update database
		var query string
		var args []interface{}
		if documentURL != "" {
			query = `
                UPDATE olympiad_registrations 
                SET olympiad_place = ?, score = ?, document_url = ? 
                WHERE olympiads_registrations_id = ?
            `
			args = []interface{}{place, score, documentURL, regID}
		} else {
			query = `
                UPDATE olympiad_registrations 
                SET olympiad_place = ?, score = ? 
                WHERE olympiads_registrations_id = ?
            `
			args = []interface{}{place, score, regID}
		}

		_, err = db.Exec(query, args...)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to assign place"})
			return
		}

		// Prepare response
		response := map[string]interface{}{
			"message": "Place assigned successfully",
			"place":   place,
			"score":   score,
		}
		if documentURL != "" {
			response["document_url"] = documentURL
		}

		utils.ResponseJSON(w, response)
	}
}
func (c *OlympiadRegistrationController) DeleteRegistration(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Авторизация
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Проверка роли (только schooladmin или superadmin)
		var role string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if err != nil || (role != "schooladmin" && role != "superadmin") {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied"})
			return
		}

		// Получение ID
		idStr := mux.Vars(r)["id"]
		regID, err := strconv.Atoi(idStr)
		if err != nil || regID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		// Удаление
		result, err := db.Exec("DELETE FROM olympiad_registrations WHERE olympiads_registrations_id = ?", regID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete registration"})
			return
		}

		affected, _ := result.RowsAffected()
		if affected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "Registration deleted successfully"})
	}
}
func (c *OlympiadRegistrationController) GetTotalOlympiadRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Authentication
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Get user role and school ID
		var role string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil {
			log.Printf("User with ID %d not found in users table", userID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "User not found"})
			return
		}

		log.Printf("Authenticated user: ID=%d, role=%s, school_id=%v", userID, role, userSchoolID)

		// Handle school ID based on role
		var schoolID int
		if role == "superadmin" {
			// For superadmin, get school_id from URL parameters
			schoolIDStr := r.URL.Query().Get("school_id")
			if schoolIDStr == "" {
				schoolIDStr = mux.Vars(r)["school_id"] // Check if it's in route params
			}

			if schoolIDStr == "" {
				// Default school ID 1 for superadmin if not specified
				schoolID = 1
			} else {
				parsedID, err := strconv.Atoi(schoolIDStr)
				if err != nil || parsedID <= 0 {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
					return
				}
				schoolID = parsedID
			}
		} else if role == "schooladmin" {
			// For schooladmin, use their associated school
			if !userSchoolID.Valid || userSchoolID.Int64 <= 0 {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "School not assigned to this administrator"})
				return
			}
			schoolID = int(userSchoolID.Int64)
		} else {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied for role: " + role})
			return
		}

		// Calculate ratings by level
		cityRating := calculateOlympiadRatingByLevel(db, schoolID, "city", 0.2)
		regionRating := calculateOlympiadRatingByLevel(db, schoolID, "region", 0.3)
		republicanRating := calculateOlympiadRatingByLevel(db, schoolID, "republican", 0.5)
		totalRating := cityRating + regionRating + republicanRating

		// Get additional school info
		var schoolName string
		var schoolCity sql.NullString
		err = db.QueryRow("SELECT name, city FROM school WHERE school_id = ?", schoolID).Scan(&schoolName, &schoolCity)
		if err == nil {
			utils.ResponseJSON(w, map[string]interface{}{
				"school_id":                  schoolID,
				"school_name":                schoolName,
				"school_city":                schoolCity.String,
				"city_olympiad_rating":       cityRating,
				"region_olympiad_rating":     regionRating,
				"republican_olympiad_rating": republicanRating,
				"total_olympiad_rating":      totalRating,
			})
		} else {
			// Fallback if school name/city couldn't be retrieved
			utils.ResponseJSON(w, map[string]interface{}{
				"school_id":                  schoolID,
				"city_olympiad_rating":       cityRating,
				"region_olympiad_rating":     regionRating,
				"republican_olympiad_rating": republicanRating,
				"total_olympiad_rating":      totalRating,
			})
		}
	}
}
func (c *OlympiadRegistrationController) GetTotalOlympiadRankBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Authentication
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Get user role and school ID
		var role string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&role, &userSchoolID)
		if err != nil {
			log.Printf("User with ID %d not found in users table", userID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "User not found"})
			return
		}

		log.Printf("Authenticated user: ID=%d, role=%s, school_id=%v", userID, role, userSchoolID)

		// Handle school ID based on role
		var schoolID int
		if role == "superadmin" {
			// For superadmin, get school_id from URL parameters
			schoolIDStr := r.URL.Query().Get("school_id")
			if schoolIDStr == "" {
				schoolIDStr = mux.Vars(r)["school_id"] // Check if it's in route params
			}

			if schoolIDStr == "" {
				// Default school ID 1 for superadmin if not specified
				schoolID = 1
			} else {
				parsedID, err := strconv.Atoi(schoolIDStr)
				if err != nil || parsedID <= 0 {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
					return
				}
				schoolID = parsedID
			}
		} else if role == "schooladmin" {
			// For schooladmin, use their associated school
			if !userSchoolID.Valid || userSchoolID.Int64 <= 0 {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "School not assigned to this administrator"})
				return
			}
			schoolID = int(userSchoolID.Int64)
		} else {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Access denied for role: " + role})
			return
		}

		// Calculate ratings by level
		cityRating := calculateOlympiadRatingByLevel(db, schoolID, "city", 0.2)
		regionRating := calculateOlympiadRatingByLevel(db, schoolID, "region", 0.3)
		republicanRating := calculateOlympiadRatingByLevel(db, schoolID, "republican", 0.5)
		totalRating := cityRating + regionRating + republicanRating

		// Calculate Olympiad rank
		olympiadRank := 25.0 * totalRating

		// Get additional school info
		var schoolName string
		var schoolCity sql.NullString
		err = db.QueryRow("SELECT name, city FROM school WHERE school_id = ?", schoolID).Scan(&schoolName, &schoolCity)
		if err == nil {
			utils.ResponseJSON(w, map[string]interface{}{
				"school_id":                  schoolID,
				"school_name":                schoolName,
				"school_city":                schoolCity.String,
				"city_olympiad_rating":       cityRating,
				"region_olympiad_rating":     regionRating,
				"republican_olympiad_rating": republicanRating,
				"total_olympiad_rating":      totalRating,
				"OlympiadRank":               olympiadRank,
			})
		} else {
			// Fallback if school name/city couldn't be retrieved
			utils.ResponseJSON(w, map[string]interface{}{
				"school_id":                  schoolID,
				"city_olympiad_rating":       cityRating,
				"region_olympiad_rating":     regionRating,
				"republican_olympiad_rating": republicanRating,
				"total_olympiad_rating":      totalRating,
				"OlympiadRank":               olympiadRank,
			})
		}
	}
}
func calculateOlympiadRatingByLevel(db *sql.DB, schoolID int, level string, weight float64) float64 {
	query := `
		SELECT o.olympiad_place, o.score
		FROM olympiad_registrations o
		JOIN subject_olympiads s ON s.subject_olympiad_id = o.subject_olympiad_id
		WHERE o.school_id = ? AND s.level = ? AND (o.status = 'completed' OR o.status = 'accepted')
		AND o.olympiad_place IS NOT NULL
	`

	rows, err := db.Query(query, schoolID, level)
	if err != nil {
		log.Printf("Error fetching %s olympiad records for school %d: %v", level, schoolID, err)
		return 0
	}
	defer rows.Close()

	var totalScore float64
	var count int

	for rows.Next() {
		var place sql.NullInt64
		var score sql.NullInt64
		if err := rows.Scan(&place, &score); err != nil {
			log.Printf("Error scanning %s olympiad row: %v", level, err)
			continue
		}

		// Skip records with NULL place
		if !place.Valid {
			continue
		}

		// Use the calculated score if available
		if score.Valid && score.Int64 > 0 {
			totalScore += float64(score.Int64)
		} else {
			// Otherwise calculate based on place
			switch place.Int64 {
			case 1:
				totalScore += 50
			case 2:
				totalScore += 30
			case 3:
				totalScore += 20
			}
		}
		count++
	}

	if count == 0 {
		return 0
	}

	// Calculate average score as a percentage of maximum possible (50 points per participant)
	average := totalScore / (50 * float64(count))
	weightedScore := average * weight

	// Round to 2 decimal places for better readability
	weightedScore = math.Round(weightedScore*100) / 100

	return weightedScore
}
func (c *OlympiadRegistrationController) GetOverallOlympiadParticipationCount(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Авторизация
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Подсчёт всех регистраций
		var total int
		err = db.QueryRow("SELECT COUNT(*) FROM olympiad_registrations").Scan(&total)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to count registrations"})
			return
		}

		utils.ResponseJSON(w, map[string]int{
			"overall_total_participations": total,
		})
	}
}

func (c *OlympiadRegistrationController) GetRegistrationsByMonth(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Авторизация
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// SQL-запрос для группировки регистраций по месяцам
		query := `
            SELECT 
                MONTHNAME(registration_date) AS month_name, 
                COUNT(*) AS registration_count
            FROM olympiad_registrations
            GROUP BY MONTH(registration_date), month_name
            ORDER BY MONTH(registration_date)
        `

		rows, err := db.Query(query)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch registrations by month"})
			return
		}
		defer rows.Close()

		// Создаем map для хранения результатов
		registrationsByMonth := make(map[string]int)

		// Инициализируем все месяцы с нулевым значением
		monthMap := map[string]string{
			"January":   "Januarycount",
			"February":  "Februarycount",
			"March":     "Marchcount",
			"April":     "Aprilcount",
			"May":       "Maycount",
			"June":      "Junecount",
			"July":      "Julycount",
			"August":    "Augustcount",
			"September": "Septembercount",
			"October":   "Octobercount",
			"November":  "Novembercount",
			"December":  "Decembercount",
		}
		for _, key := range monthMap {
			registrationsByMonth[key] = 0
		}

		// Заполняем map данными из запроса
		for rows.Next() {
			var monthName string
			var count int
			if err := rows.Scan(&monthName, &count); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			// Приводим к формату, ожидаемому фронтендом
			if formattedKey, ok := monthMap[monthName]; ok {
				registrationsByMonth[formattedKey] = count
			}
		}

		// Возвращаем результат
		utils.ResponseJSON(w, registrationsByMonth)
	}
}
func (c *OlympiadRegistrationController) GetOlympiadRegistrationByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid registration ID"})
			return
		}

		var reg models.OlympiadRegistration
		var regDateStr string
		var olympiadEnd sql.NullString
		var level sql.NullString
		var score sql.NullInt64
		var olympiadPlace sql.NullInt64
		var documentURL sql.NullString

		err = db.QueryRow(`
			SELECT r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
				   r.school_id, r.document_url,
				   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
				   sch.school_name,
				   o.subject_name, o.date, o.end_date, o.level,
				   r.score, r.olympiad_place
			FROM olympiad_registrations r
			JOIN student s ON r.student_id = s.student_id
			JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
			JOIN Schools sch ON r.school_id = sch.school_id
			WHERE r.olympiads_registrations_id = ?`, id).Scan(
			&reg.OlympiadsRegistrationsID,
			&reg.StudentID,
			&reg.SubjectOlympiadID,
			&regDateStr,
			&reg.Status,
			&reg.SchoolID,
			&documentURL,
			&reg.StudentFirstName,
			&reg.StudentLastName,
			&reg.StudentPatronymic,
			&reg.StudentGrade,
			&reg.StudentLetter,
			&reg.SchoolName,
			&reg.OlympiadName,
			&reg.OlympiadStartDate,
			&olympiadEnd,
			&level,
			&score,
			&olympiadPlace,
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Registration not found"})
			return
		}

		reg.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
		if olympiadEnd.Valid {
			reg.OlympiadEndDate = olympiadEnd.String
		}
		if level.Valid {
			reg.Level = level.String
		}
		if score.Valid {
			reg.Score = int(score.Int64)
		}
		if olympiadPlace.Valid {
			reg.OlympiadPlace = int(olympiadPlace.Int64)
		}
		if documentURL.Valid {
			reg.DocumentURL = documentURL.String
		}

		utils.ResponseJSON(w, reg)
	}
}
func (c *OlympiadRegistrationController) GetOlympiadRegistrationsBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify JWT token
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Extract school_id from URL parameters
		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Query registrations for the given school_id
		rows, err := db.Query(`
			SELECT r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
				   r.school_id, r.document_url,
				   s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
				   sch.school_name,
				   o.subject_name, o.date, o.end_date, o.level,
				   r.score, r.olympiad_place
			FROM olympiad_registrations r
			JOIN student s ON r.student_id = s.student_id
			JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
			JOIN Schools sch ON r.school_id = sch.school_id
			WHERE r.school_id = ?`, schoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error querying registrations"})
			return
		}
		defer rows.Close()

		// Collect all registrations
		var registrations []models.OlympiadRegistration
		for rows.Next() {
			var reg models.OlympiadRegistration
			var regDateStr string
			var olympiadEnd sql.NullString
			var level sql.NullString
			var score sql.NullInt64
			var olympiadPlace sql.NullInt64
			var documentURL sql.NullString

			err := rows.Scan(
				&reg.OlympiadsRegistrationsID,
				&reg.StudentID,
				&reg.SubjectOlympiadID,
				&regDateStr,
				&reg.Status,
				&reg.SchoolID,
				&documentURL,
				&reg.StudentFirstName,
				&reg.StudentLastName,
				&reg.StudentPatronymic,
				&reg.StudentGrade,
				&reg.StudentLetter,
				&reg.SchoolName,
				&reg.OlympiadName,
				&reg.OlympiadStartDate,
				&olympiadEnd,
				&level,
				&score,
				&olympiadPlace,
			)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning registration"})
				return
			}

			// Parse registration date
			reg.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)

			// Handle nullable fields
			if olympiadEnd.Valid {
				reg.OlympiadEndDate = olympiadEnd.String
			}
			if level.Valid {
				reg.Level = level.String
			}
			if score.Valid {
				reg.Score = int(score.Int64)
			}
			if olympiadPlace.Valid {
				reg.OlympiadPlace = int(olympiadPlace.Int64)
			}
			if documentURL.Valid {
				reg.DocumentURL = documentURL.String
			}

			registrations = append(registrations, reg)
		}

		// Check if any registrations were found
		if len(registrations) == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No registrations found for this school"})
			return
		}

		// Return the list of registrations
		utils.ResponseJSON(w, registrations)
	}
}
func (c *OlympiadRegistrationController) GetOlympiadPrizeStatsBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify JWT token
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Extract school_id from URL parameters
		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Query to count places by school_id
		query := `
			SELECT 
				COALESCE(SUM(CASE WHEN olympiad_place = 1 THEN 1 ELSE 0 END), 0) as first_place,
				COALESCE(SUM(CASE WHEN olympiad_place = 2 THEN 1 ELSE 0 END), 0) as second_place,
				COALESCE(SUM(CASE WHEN olympiad_place = 3 THEN 1 ELSE 0 END), 0) as third_place
			FROM olympiad_registrations 
			WHERE school_id = ? AND olympiad_place IS NOT NULL
		`

		var firstPlace, secondPlace, thirdPlace int
		err = db.QueryRow(query, schoolID).Scan(&firstPlace, &secondPlace, &thirdPlace)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error querying prize statistics"})
			return
		}

		// Prepare response
		response := map[string]int{
			"first_place":  firstPlace,
			"second_place": secondPlace,
			"third_place":  thirdPlace,
		}

		utils.ResponseJSON(w, response)
	}
}

func (c *OlympiadRegistrationController) GetOlympiadRegistrationsByOlympID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		vars := mux.Vars(r)
		subjectOlympiadIDStr := vars["olympiad_id"]
		subjectOlympiadID, err := strconv.Atoi(subjectOlympiadIDStr)
		if err != nil || subjectOlympiadID <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID"})
			return
		}

		rows, err := db.Query(`
			SELECT 
				r.olympiads_registrations_id, r.student_id, r.subject_olympiad_id, r.registration_date, r.status,
				r.school_id,
				s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
				sch.school_name,
				o.subject_name, o.date, o.end_date, o.level
			FROM olympiad_registrations r
			JOIN student s ON r.student_id = s.student_id
			JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
			JOIN Schools sch ON r.school_id = sch.school_id
			WHERE r.subject_olympiad_id = ?
		`, subjectOlympiadID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}
		defer rows.Close()

		var registrations []models.OlympiadRegistration
		for rows.Next() {
			var reg models.OlympiadRegistration
			var regDateStr string
			var olympiadEnd sql.NullString
			var level sql.NullString

			err := rows.Scan(
				&reg.OlympiadsRegistrationsID,
				&reg.StudentID,
				&reg.SubjectOlympiadID,
				&regDateStr,
				&reg.Status,
				&reg.SchoolID,
				&reg.StudentFirstName,
				&reg.StudentLastName,
				&reg.StudentPatronymic,
				&reg.StudentGrade,
				&reg.StudentLetter,
				&reg.SchoolName,
				&reg.OlympiadName,
				&reg.OlympiadStartDate,
				&olympiadEnd,
				&level,
			)
			if err != nil {
				log.Println("Scan error:", err)
				continue
			}

			reg.RegistrationDate, _ = time.Parse("2006-01-02 15:04:05", regDateStr)
			if olympiadEnd.Valid {
				reg.OlympiadEndDate = olympiadEnd.String
			}
			if level.Valid {
				reg.Level = level.String
			}

			registrations = append(registrations, reg)
		}

		utils.ResponseJSON(w, registrations)
	}
}
