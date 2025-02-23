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
		// Parse the incoming request body
		var untScore models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&untScore); err != nil {
			log.Println("Error decoding request body:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Log the parsed data for debugging purposes
		log.Printf("Parsed UNTScore: %+v", untScore)

		// SQL query to insert UNT score
		query := `
			INSERT INTO UNT_Score (
				year, 
				unt_type_id, 
				student_id, 
				first_subject_score, 
				second_subject_score, 
				history_kazakhstan, 
				math_literacy, 
				reading_literacy
			) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		// Executing the SQL query
		_, err := db.Exec(query, untScore.Year, untScore.UNTTypeID, untScore.StudentID, untScore.FirstSubjectScore, untScore.SecondSubjectScore, untScore.HistoryKazakhstan, untScore.MathLiteracy, untScore.ReadingLiteracy)
		if err != nil {
			log.Println("Error inserting UNT score:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
			return
		}

		// Return success response
		utils.ResponseJSON(w, "UNT Score created successfully")
	}
}
func (sc UNTScoreController) GetUNTScores(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT unt_score_id, year, unt_type_id, student_id, score FROM UNT_Score")
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get UNT Scores"})
            return
        }
        defer rows.Close()

        var scores []models.UNTScore
        for rows.Next() {
            var score models.UNTScore
            if err := rows.Scan(&score.ID, &score.Year, &score.UNTTypeID, &score.StudentID, &score.TotalScore); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Scores"})
                return
            }
            scores = append(scores, score)
        }

        utils.ResponseJSON(w, scores)
    }
}
func (sc UNTScoreController) CalculateUNTScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var untScore models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&untScore); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Переменные для расчета
		var firstTypeScore, secondTypeScore, totalScore int
		var firstTypeMaxScore = 140
		var secondTypeMaxScore = 30

		// Рассчитываем для First_Type
		if untScore.UNTTypeID == 1 { // First_Type
			firstTypeScore = untScore.FirstSubjectScore + untScore.SecondSubjectScore +
				untScore.HistoryKazakhstan + untScore.MathLiteracy + untScore.ReadingLiteracy
			// Рассчитываем процент
			firstTypePercent := (float64(firstTypeScore) / float64(firstTypeMaxScore)) * 100
			totalScore = int(firstTypePercent) // Сохраняем в totalScore

		} else if untScore.UNTTypeID == 2 { // Second_Type
			secondTypeScore = untScore.HistoryKazakhstan + untScore.ReadingLiteracy
			// Рассчитываем процент
			secondTypePercent := (float64(secondTypeScore) / float64(secondTypeMaxScore)) * 100
			totalScore = int(secondTypePercent) // Сохраняем в totalScore
		}

		// Вставляем данные в таблицу UNT_Score
		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, score, first_subject_score, second_subject_score, 
										history_kazakhstan, math_literacy, reading_literacy, total_score) 
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, untScore.Year, untScore.UNTTypeID, untScore.StudentID, totalScore,
			untScore.FirstSubjectScore, untScore.SecondSubjectScore, untScore.HistoryKazakhstan,
			untScore.MathLiteracy, untScore.ReadingLiteracy, totalScore)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
			return
		}

		// Ответ с успешным сохранением
		utils.ResponseJSON(w, "UNT score created successfully")
	}
}
