package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type HistoryController struct{}

func (hc *HistoryController) GetMyHistory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE student_id = ?)", userID).Scan(&exists)
		if err != nil || !exists {
			log.Printf("ERROR: student_id = %d not found in student table", userID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			return
		}

		olympRows, err := db.Query(`
			SELECT 
				o.subject_name, 
				o.level, 
				o.date, 
				o.end_date, 
				r.status, 
				r.score, 
				r.olympiad_place,
				sc.school_name
			FROM olympiad_registrations r
			JOIN subject_olympiads o ON r.subject_olympiad_id = o.subject_olympiad_id
			JOIN Schools sc ON o.school_id = sc.school_id
			WHERE r.student_id = ?
			ORDER BY r.registration_date DESC
		`, userID)
		if err != nil {
			log.Printf("ERROR: olympiad query error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Olympiad query error"})
			return
		}
		defer olympRows.Close()

		var olympiads []models.MyHistoryOlympiad
		for olympRows.Next() {
			var o models.MyHistoryOlympiad
			var score sql.NullInt64
			var place sql.NullInt64
			var schoolName sql.NullString

			err := olympRows.Scan(
				&o.Subject,
				&o.Level,
				&o.StartDate,
				&o.EndDate,
				&o.Status,
				&score,
				&place,
				&schoolName,
			)
			if err != nil {
				log.Printf("ERROR: olympiad scan error: %v", err)
				continue
			}
			if score.Valid {
				o.Score = int(score.Int64)
			}
			if place.Valid {
				o.Place = int(place.Int64)
			}
			if schoolName.Valid {
				o.SchoolName = schoolName.String
			}
			olympiads = append(olympiads, o)
		}

		eventRows, err := db.Query(`
			SELECT 
				e.event_name, 
				e.category, 
				e.start_date, 
				e.end_date, 
				r.status,
				sc.school_name
			FROM EventRegistrations r
			JOIN Events e ON r.event_id = e.id
			JOIN Schools sc ON e.school_id = sc.school_id
			WHERE r.student_id = ?
			ORDER BY r.registration_date DESC
		`, userID)
		if err != nil {
			log.Printf("ERROR: event query error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Event query error"})
			return
		}
		defer eventRows.Close()

		var events []models.MyHistoryEvent
		for eventRows.Next() {
			var e models.MyHistoryEvent
			var schoolName sql.NullString

			err := eventRows.Scan(
				&e.Name,
				&e.Category,
				&e.StartDate,
				&e.EndDate,
				&e.Status,
				&schoolName,
			)
			if err != nil {
				log.Printf("ERROR: event scan error: %v", err)
				continue
			}
			if schoolName.Valid {
				e.SchoolName = schoolName.String
			}
			events = append(events, e)
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"events":    events,
			"olympiads": olympiads,
		})
	}
}
func (hc *HistoryController) GetMyAchievements(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE student_id = ?)", userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			return
		}

		// -------- Олимпиады --------
		olympRows, err := db.Query(`
			SELECT 
				olympiad_name,
				level,
				date,
				score,
				olympiad_place,
				s.school_name,
				document_url
			FROM Olympiads o
			JOIN Schools s ON o.school_id = s.school_id
			WHERE student_id = ?
			ORDER BY date DESC
		`, userID)
		if err != nil {
			log.Printf("ERROR: Olympiads query: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Olympiad query error"})
			return
		}
		defer olympRows.Close()

		var olympiads []models.MyHistoryOlympiad
		for olympRows.Next() {
			var o models.MyHistoryOlympiad
			var score sql.NullInt64
			var place sql.NullInt64
			var schoolName sql.NullString
			var document sql.NullString

			err := olympRows.Scan(
				&o.Subject,
				&o.Level,
				&o.StartDate,
				&score,
				&place,
				&schoolName,
				&document,
			)
			if err != nil {
				log.Printf("ERROR: scan Olympiads: %v", err)
				continue
			}
			if score.Valid {
				o.Score = int(score.Int64)
			}
			if place.Valid {
				o.Place = int(place.Int64)
			}
			if schoolName.Valid {
				o.SchoolName = schoolName.String
			}
			if document.Valid {
				o.DocumentURL = document.String
			}
			olympiads = append(olympiads, o)
		}

		// -------- Ивенты --------
		eventRows, err := db.Query(`
			SELECT 
				e.events_name,
				e.category,
				e.date,
				e.date,
				e.role,
				s.school_name,
				e.document
			FROM events_participants e
			JOIN Schools s ON e.school_id = s.school_id
			WHERE e.student_id = ?
			ORDER BY e.date DESC
		`, userID)
		if err != nil {
			log.Printf("ERROR: Events query: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Event query error"})
			return
		}
		defer eventRows.Close()

		var events []models.MyHistoryEvent
		for eventRows.Next() {
			var e models.MyHistoryEvent
			var schoolName sql.NullString
			var document sql.NullString

			err := eventRows.Scan(
				&e.Name,
				&e.Category,
				&e.StartDate,
				&e.EndDate,
				&e.Status,
				&schoolName,
				&document,
			)
			if err != nil {
				log.Printf("ERROR: scan Events: %v", err)
				continue
			}
			if schoolName.Valid {
				e.SchoolName = schoolName.String
			}
			if document.Valid {
				e.DocumentURL = document.String
			}
			events = append(events, e)
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"my_achievements": map[string]interface{}{
				"events":    events,
				"olympiads": olympiads,
			},
		})
	}
}
