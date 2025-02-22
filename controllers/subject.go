package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

// Controller for subjects
type SubjectController struct{}

// Get all First_Subject
// Get all First_Subject
func (sc SubjectController) GetFirstSubjects(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT first_subject_id, subject, score FROM First_Subject")
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get First Subjects"})
			return
		}
		defer rows.Close()

		var subjects []models.FirstSubject
		for rows.Next() {
			var subject models.FirstSubject
			if err := rows.Scan(&subject.ID, &subject.Subject, &subject.Score); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Subjects"})
				return
			}
			subjects = append(subjects, subject)
		}

		utils.ResponseJSON(w, subjects)
	}
}


// Get all Second_Subject
// Get all Second_Subject
func (sc SubjectController) GetSecondSubjects(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT second_subject_id, subject, score FROM Second_Subject")
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get Second Subjects"})
			return
		}
		defer rows.Close()

		var subjects []models.SecondSubject
		for rows.Next() {
			var subject models.SecondSubject
			if err := rows.Scan(&subject.ID, &subject.Subject, &subject.Score); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Subjects"})
				return
			}
			subjects = append(subjects, subject)
		}

		utils.ResponseJSON(w, subjects)
	}
}

func (sc SubjectController) CreateFirstSubject(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var subject models.FirstSubject
		if err := json.NewDecoder(r.Body).Decode(&subject); err != nil {
			log.Println("JSON Error:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Вставляем новый предмет в таблицу
		_, err := db.Exec("INSERT INTO First_Subject (subject, score) VALUES (?, ?)", subject.Subject, subject.Score)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create First Subject"})
			return
		}

		utils.ResponseJSON(w, models.Message{Message: "First Subject created successfully"})
	}
}
// Add a new Second_Subject
func (sc SubjectController) CreateSecondSubject(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var subject models.SecondSubject
		if err := json.NewDecoder(r.Body).Decode(&subject); err != nil {
			log.Println("JSON Error:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Вставляем новый предмет в таблицу
		_, err := db.Exec("INSERT INTO Second_Subject (subject, score) VALUES (?, ?)", subject.Subject, subject.Score)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create Second Subject"})
			return
		}

		utils.ResponseJSON(w, models.Message{Message: "Second Subject created successfully"})
	}
}
