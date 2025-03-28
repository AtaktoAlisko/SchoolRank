package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"

	"github.com/gorilla/mux"
)

type ReviewController struct{}

func (rc *ReviewController) CreateReview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var review models.Review
		if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
			log.Println("Error decoding request body:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Запрос на добавление отзыва
		query := `INSERT INTO Reviews (school_id, user_id, rating, comment) VALUES (?, ?, ?, ?)`
		_, err := db.Exec(query, review.SchoolID, review.UserID, review.Rating, review.Comment)
		if err != nil {
			log.Println("Error inserting review:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create review"})
			return
		}

		utils.ResponseJSON(w, "Review created successfully")
	}
}

func (rc *ReviewController) GetReviewsBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем school_id из параметров запроса
		schoolID := mux.Vars(r)["school_id"]

		// Запрос на получение всех отзывов для школы
		query := `SELECT user_id, rating, comment, created_at FROM Reviews WHERE school_id = ?`
		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get reviews"})
			return
		}
		defer rows.Close()

		var reviews []models.Review
		for rows.Next() {
			var review models.Review
			err := rows.Scan(&review.UserID, &review.Rating, &review.Comment, &review.CreatedAt)
			if err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse reviews"})
				return
			}
			reviews = append(reviews, review)
		}

		utils.ResponseJSON(w, reviews)
	}
}

func (rc *ReviewController) GetAverageRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем school_id из параметров запроса
		schoolID := mux.Vars(r)["school_id"]

		// Запрос на получение среднего рейтинга по школе
		query := `SELECT AVG(rating) FROM Reviews WHERE school_id = ?`
		var averageRating float64
		err := db.QueryRow(query, schoolID).Scan(&averageRating)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get average rating"})
			return
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"average_rating": averageRating,
		})
	}
}
