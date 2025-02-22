package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

// Controller для UNT Score
type UNTScoreController struct{}

// Создание записи UNT Score
func (uc UNTScoreController) CreateUNTScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var untScore models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&untScore); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		query := "INSERT INTO unt_score (year, unt_type_id, student_id, score) VALUES (?, ?, ?, ?)"
		result, err := db.Exec(query, untScore.Year, untScore.UNTTypeID, untScore.StudentID, untScore.Score)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
			return
		}

		id, _ := result.LastInsertId()
		untScore.UNTScoreID = int(id)

		utils.ResponseJSON(w, untScore)
	}
}

// Получение всех результатов UNT Score
func (uc UNTScoreController) GetUNTScores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT unt_score_id, year, unt_type_id, student_id, score FROM unt_score")
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get UNT scores"})
			return
		}
		defer rows.Close()

		var untScores []models.UNTScore
		for rows.Next() {
			var untScore models.UNTScore
			if err := rows.Scan(&untScore.UNTScoreID, &untScore.Year, &untScore.UNTTypeID, &untScore.StudentID, &untScore.Score); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT scores"})
				return
			}
			untScores = append(untScores, untScore)
		}

		utils.ResponseJSON(w, untScores)
	}
}
