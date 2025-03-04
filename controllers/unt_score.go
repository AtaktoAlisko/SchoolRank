package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type UNTScoreController struct{}

func (usc UNTScoreController) CreateUNTScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var untScore models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&untScore); err != nil {
			log.Println("Error decoding request body:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Проверяем существование UNT_Type и Student
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM UNT_Type WHERE unt_type_id = ?)", untScore.UNTTypeID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "UNT Type ID does not exist"})
			return
		}

		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Student WHERE student_id = ?)", untScore.StudentID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID does not exist"})
			return
		}

		// Подтягиваем баллы из связанных таблиц
		var totalScore int
		err = db.QueryRow(`
			SELECT COALESCE(SUM(score), 0) FROM (
				SELECT fs.score FROM First_Subject fs
				JOIN First_Type ft ON fs.first_subject_id = ft.first_subject_id
				JOIN UNT_Type ut ON ft.first_type_id = ut.first_type_id
				WHERE ut.unt_type_id = ?
				UNION ALL
				SELECT ss.score FROM Second_Subject ss
				JOIN First_Type ft ON ss.second_subject_id = ft.second_subject_id
				JOIN UNT_Type ut ON ft.first_type_id = ut.first_type_id
				WHERE ut.unt_type_id = ?
				UNION ALL
				SELECT st.history_of_kazakhstan FROM Second_Type st
				JOIN UNT_Type ut ON st.second_type_id = ut.second_type_id
				WHERE ut.unt_type_id = ?
				UNION ALL
				SELECT st.reading_literacy FROM Second_Type st
				JOIN UNT_Type ut ON st.second_type_id = ut.second_type_id
				WHERE ut.unt_type_id = ?
			) AS scores`, untScore.UNTTypeID, untScore.UNTTypeID, untScore.UNTTypeID, untScore.UNTTypeID).Scan(&totalScore)
		if err != nil {
			log.Println("Error calculating total score:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate total score"})
			return
		}

		// Вставляем данные в таблицу
		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, score) VALUES (?, ?, ?, ?)`
		_, err = db.Exec(query, untScore.Year, untScore.UNTTypeID, untScore.StudentID, totalScore)
		if err != nil {
			log.Println("Error inserting UNT score:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
			return
		}

		utils.ResponseJSON(w, "UNT Score created successfully")
	}
}

func (sc UNTScoreController) GetUNTScores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT us.unt_score_id, us.year, us.unt_type_id, us.student_id, us.total_score,
			       fs.subject AS first_subject_name, fs.score AS first_subject_score,
			       ss.subject AS second_subject_name, ss.score AS second_subject_score,
			       st.history_of_kazakhstan, st.reading_literacy,
			       s.first_name, s.last_name, s.iin
			FROM UNT_Score us
			LEFT JOIN UNT_Type ut ON us.unt_type_id = ut.unt_type_id
			LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
			LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
			LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
			LEFT JOIN Second_Type st ON ut.second_type_id = st.second_type_id
			LEFT JOIN Student s ON us.student_id = s.student_id
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get UNT Scores"})
			return
		}
		defer rows.Close()

		var scores []models.UNTScore
		for rows.Next() {
			var score models.UNTScore
			if err := rows.Scan(
				&score.ID, &score.Year, &score.UNTTypeID, &score.StudentID, &score.TotalScore,
				&score.FirstSubjectName, &score.FirstSubjectScore,
				&score.SecondSubjectName, &score.SecondSubjectScore,
				&score.HistoryKazakhstan, &score.ReadingLiteracy,
				&score.FirstName, &score.LastName, &score.IIN,
			); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Scores"})
				return
			}
			scores = append(scores, score)
		}

		utils.ResponseJSON(w, scores)
	}
}

func (sc *UNTScoreController) GetUNTScoreByStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentID := r.URL.Query().Get("student_id")
		if studentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "student_id is required"})
			return
		}

		query := `
			SELECT us.unt_score_id, us.year, us.unt_type_id, us.student_id, us.total_score,
			       s.first_name, s.last_name, s.iin
			FROM UNT_Score us
			JOIN Student s ON us.student_id = s.student_id
			WHERE us.student_id = ?
		`

		var result models.UNTScore
		err := db.QueryRow(query, studentID).Scan(
			&result.ID, &result.Year, &result.UNTTypeID, &result.StudentID, &result.TotalScore,
			&result.FirstName, &result.LastName, &result.IIN,
		)

		if err != nil {
			log.Printf("Error retrieving UNT score: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve UNT score"})
			return
		}

		utils.ResponseJSON(w, result)
	}
}
