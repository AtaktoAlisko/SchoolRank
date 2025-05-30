package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type ReviewController struct{}

func (rc *ReviewController) CreateReview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get user ID
		tokenUserID, err := utils.VerifyToken(r) // Returns userID directly, not claims
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		var review models.Review
		if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
			log.Println("Error decoding request body:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Validate that token user ID matches review user ID
		if tokenUserID != review.UserID {
			log.Println("Token user ID doesn't match review user ID")
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only submit reviews as yourself"})
			return
		}

		// Validate required fields
		if review.SchoolID <= 0 || review.UserID <= 0 || review.Rating < 1 || review.Rating > 5 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid review data: school_id and user_id must be positive, rating must be between 1 and 5"})
			return
		}

		// Check if user exists
		var userExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", review.UserID).Scan(&userExists)
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

		// Insert review with Likes and CreatedAt
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		result, err := db.Exec(
			"INSERT INTO Reviews (school_id, user_id, rating, comment, likes, created_at) VALUES (?, ?, ?, ?, 0, ?)",
			review.SchoolID, review.UserID, review.Rating, review.Comment, currentTime,
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

		// Create response
		response := map[string]interface{}{
			"message":   "Review created successfully",
			"review_id": reviewID,
		}

		utils.ResponseJSON(w, response)
	}
}
func (rc *ReviewController) CreateLike(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var like models.Like
		if err := json.NewDecoder(r.Body).Decode(&like); err != nil {
			log.Println("Error decoding request body:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		userID, err := utils.VerifyToken(r) // Предполагается, что VerifyToken возвращает ID пользователя
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Check if review exists
		var reviewExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Reviews WHERE id = ?)", like.ReviewID).Scan(&reviewExists)
		if err != nil || !reviewExists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Review does not exist"})
			return
		}

		// Check if user already liked the review
		var likeExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Likes WHERE review_id = ? AND user_id = ?)",
			like.ReviewID, userID).Scan(&likeExists)
		if err != nil {
			log.Println("Error checking for existing like:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking for existing like"})
			return
		}

		if likeExists {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "User has already liked this review"})
			return
		}

		// Insert new like
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		_, err = db.Exec(
			"INSERT INTO Likes (review_id, user_id, created_at) VALUES (?, ?, ?)",
			like.ReviewID, userID, currentTime,
		)
		if err != nil {
			log.Println("Error inserting like:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create like"})
			return
		}

		// Increment likes in Reviews table
		_, err = db.Exec("UPDATE Reviews SET likes = likes + 1 WHERE id = ?", like.ReviewID)
		if err != nil {
			log.Println("Error updating likes:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update likes count"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "Like created successfully"})
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

		// Запрос на получение среднего рейтинга по школе с обработкой NULL
		query := `SELECT COALESCE(AVG(rating), 0) FROM Reviews WHERE school_id = ?`
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
func (rc *ReviewController) GetAllReviews(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if a school_id parameter was provided
		schoolIDParam := r.URL.Query().Get("school_id")

		var reviewsQuery string
		var args []interface{}

		// Modify query based on whether school_id filter is provided
		if schoolIDParam != "" {
			schoolID, err := strconv.Atoi(schoolIDParam)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school_id parameter"})
				return
			}
			reviewsQuery = `SELECT id, school_id, user_id, rating, comment, created_at FROM Reviews WHERE school_id = ?`
			args = append(args, schoolID)
		} else {
			reviewsQuery = `SELECT id, school_id, user_id, rating, comment, created_at FROM Reviews`
		}

		// Execute the query with any arguments
		var rows *sql.Rows
		var err error
		if len(args) > 0 {
			rows, err = db.Query(reviewsQuery, args...)
		} else {
			rows, err = db.Query(reviewsQuery)
		}

		if err != nil {
			log.Println("SQL Error (Reviews):", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get reviews"})
			return
		}
		defer rows.Close()

		// Create a slice to hold the reviews
		var reviews []models.Review

		// Map to store school IDs that we need to fetch names for
		schoolIDs := make(map[int]bool)

		for rows.Next() {
			var review models.Review
			// Scan the row into the Review struct (without school_name yet)
			err := rows.Scan(&review.ID, &review.SchoolID, &review.UserID, &review.Rating, &review.Comment, &review.CreatedAt)
			if err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse reviews"})
				return
			}

			// Keep track of school IDs
			schoolIDs[review.SchoolID] = true

			// Set empty school name for now
			review.SchoolName = ""

			// Append the review to the slice
			reviews = append(reviews, review)
		}

		// Check for errors after processing all rows
		if err := rows.Err(); err != nil {
			log.Println("Rows Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing reviews"})
			return
		}

		// Step 2: Get school names for all the school IDs we have
		if len(schoolIDs) > 0 {
			// Create a slice of school IDs for the query
			var ids []int
			for id := range schoolIDs {
				ids = append(ids, id)
			}

			// Create placeholders for the SQL query
			placeholders := make([]string, len(ids))
			args := make([]interface{}, len(ids))
			for i, id := range ids {
				placeholders[i] = "?"
				args[i] = id
			}

			// Construct the query with placeholders
			schoolsQuery := fmt.Sprintf(`SELECT school_id, school_name FROM Schools WHERE school_id IN (%s)`, strings.Join(placeholders, ","))

			// Execute the query
			schoolRows, err := db.Query(schoolsQuery, args...)
			if err != nil {
				log.Println("SQL Error (Schools):", err)
				// Continue without school names
			} else {
				defer schoolRows.Close()

				// Create a map of school IDs to names
				schoolNames := make(map[int]string)
				for schoolRows.Next() {
					var schoolID int
					var schoolName string
					err := schoolRows.Scan(&schoolID, &schoolName)
					if err != nil {
						log.Println("Scan Error (Schools):", err)
						continue
					}
					// Store the school name
					schoolNames[schoolID] = schoolName
				}

				// Update the reviews with school names
				for i := range reviews {
					if name, ok := schoolNames[reviews[i].SchoolID]; ok {
						reviews[i].SchoolName = name
					}
				}
			}
		}

		// Return the list of reviews with school names
		utils.ResponseJSON(w, reviews)
	}
}

func (rc *ReviewController) DeleteReview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get user ID
		tokenUserID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Get the review ID from URL parameters
		vars := mux.Vars(r)
		reviewID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid review ID"})
			return
		}

		// Step 2: Check if the review exists and get its user_id
		var reviewUserID int
		var exists bool
		err = db.QueryRow("SELECT user_id, TRUE FROM Reviews WHERE id = ?", reviewID).Scan(&reviewUserID, &exists)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Review not found"})
				return
			}
			log.Println("Error checking review:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking review"})
			return
		}

		// Step 3: Get user role to check for admin privileges
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", tokenUserID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 4: Check if the user is authorized to delete the review
		// Allow if user is the review owner OR a superadmin OR a schooladmin
		if tokenUserID != reviewUserID && userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to delete this review"})
			return
		}

		// For schooladmin, we need to check if the review belongs to their school
		if userRole == "schooladmin" {
			var schoolID int
			err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", tokenUserID).Scan(&schoolID)
			if err != nil {
				log.Println("Error fetching school ID for admin:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking admin permissions"})
				return
			}

			var reviewSchoolID int
			err = db.QueryRow("SELECT school_id FROM Reviews WHERE id = ?", reviewID).Scan(&reviewSchoolID)
			if err != nil {
				log.Println("Error fetching review school ID:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking review details"})
				return
			}

			if schoolID != reviewSchoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to delete reviews for this school"})
				return
			}
		}

		// Step 5: Delete the review
		result, err := db.Exec("DELETE FROM Reviews WHERE id = ?", reviewID)
		if err != nil {
			log.Println("Error deleting review:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete review"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Println("Error getting rows affected:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error confirming deletion"})
			return
		}

		if rowsAffected == 0 {
			// This shouldn't happen since we already checked if the review exists
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Review not found"})
			return
		}

		// Step 6: Return success response
		response := map[string]interface{}{
			"message": "Review deleted successfully",
		}

		utils.ResponseJSON(w, response)
	}
}
func (rc *ReviewController) GetAverageRatingRank(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем school_id из параметров запроса
		schoolID := mux.Vars(r)["school_id"]

		// Запрос на получение среднего рейтинга по школе
		query := `SELECT AVG(rating) FROM Reviews WHERE school_id = ?`
		var averageRating float64
		err := db.QueryRow(query, schoolID).Scan(&averageRating)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get average rating rank"})
			return
		}

		// Вычисляем average_rating_rank как средний рейтинг, умноженный на 2
		averageRatingRank := averageRating * 2

		utils.ResponseJSON(w, map[string]interface{}{
			"average_rating_rank": averageRatingRank,
		})
	}
}

type ReviewResponse struct {
	*models.Review
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	AvatarURL *string `json:"avatar_url"` // Removed omitempty to ensure field is always present
}

func (rc *ReviewController) GetReviewBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Get the school ID from the URL parameters
		vars := mux.Vars(r)
		schoolID := vars["school_id"]

		// Convert the school ID to integer
		schoolIDInt, err := strconv.Atoi(schoolID)
		if err != nil {
			log.Println("Invalid school ID:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Step 2: Get reviews for the specific school, including user details
		reviewsQuery := `
			SELECT 
				r.id, 
				r.school_id, 
				r.user_id, 
				r.rating, 
				r.comment, 
				r.created_at,
				s.school_name,
				(SELECT COUNT(*) FROM Likes l WHERE l.review_id = r.id) as likes,
				u.first_name,
				u.last_name,
				u.avatar_url
			FROM Reviews r
			JOIN Schools s ON r.school_id = s.school_id
			JOIN users u ON r.user_id = u.id
			WHERE r.school_id = ?
		`
		rows, err := db.Query(reviewsQuery, schoolIDInt)
		if err != nil {
			log.Println("SQL Error (Reviews):", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get reviews"})
			return
		}
		defer rows.Close()

		// Create a slice to hold the review responses
		var reviews []ReviewResponse

		// Step 3: Scan reviews into the slice
		for rows.Next() {
			var review ReviewResponse
			review.Review = &models.Review{} // Initialize the embedded Review struct
			var firstName, lastName, avatarURL sql.NullString

			// Scan the row into the ReviewResponse struct
			err := rows.Scan(
				&review.ID,
				&review.SchoolID,
				&review.UserID,
				&review.Rating,
				&review.Comment,
				&review.CreatedAt,
				&review.SchoolName,
				&review.Likes,
				&firstName,
				&lastName,
				&avatarURL,
			)
			if err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse reviews"})
				return
			}

			// Handle nullable first_name and last_name
			if firstName.Valid {
				review.FirstName = &firstName.String
			}
			if lastName.Valid {
				review.LastName = &lastName.String
			}

			// Handle nullable avatar_url
			if avatarURL.Valid && avatarURL.String != "" {
				review.AvatarURL = &avatarURL.String
			} else {
				review.AvatarURL = nil
			}

			// Append the review to the slice
			reviews = append(reviews, review)
		}

		// Check for errors after processing all rows
		if err := rows.Err(); err != nil {
			log.Println("Rows Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing reviews"})
			return
		}

		// Step 4: Return the list of reviews with school names and user details
		utils.ResponseJSON(w, reviews)
	}
}
