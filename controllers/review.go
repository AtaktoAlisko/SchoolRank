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

		// Validate required fields
		if review.SchoolID <= 0 || review.UserID <= 0 || review.Rating < 1 || review.Rating > 5 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid review data: school_id and user_id must be positive, rating must be between 1 and 5"})
			return
		}

		// Check if user exists
		// Check if user exists
		var userExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", review.UserID).Scan(&userExists)
		if err != nil {
			log.Println("Error checking if user exists:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking if user exists"})
			return
		}

		if !userExists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "User does not exist"})
			return
		}

		// Check if school exists
		var schoolExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", review.SchoolID).Scan(&schoolExists)
		if err != nil {
			log.Println("Error checking if school exists:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking if school exists"})
			return
		}

		if !schoolExists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School does not exist"})
			return
		}

		// Check if user already submitted a review for this school
		var reviewExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Reviews WHERE school_id = ? AND user_id = ?)",
			review.SchoolID, review.UserID).Scan(&reviewExists)
		if err != nil {
			log.Println("Error checking for existing review:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking for existing review"})
			return
		}

		if reviewExists {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "User has already submitted a review for this school"})
			return
		}

		// Insert review with proper error handling
		result, err := db.Exec(
			"INSERT INTO Reviews (school_id, user_id, rating, comment) VALUES (?, ?, ?, ?)",
			review.SchoolID, review.UserID, review.Rating, review.Comment,
		)
		if err != nil {
			log.Println("Error inserting review:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create review"})
			return
		}

		// Get the inserted ID
		reviewID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error getting last insert ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Review created but failed to retrieve ID"})
			return
		}

		// Create response with the new review ID
		response := map[string]interface{}{
			"message":   "Review created successfully",
			"review_id": reviewID,
		}

		utils.ResponseJSON(w, response)
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
