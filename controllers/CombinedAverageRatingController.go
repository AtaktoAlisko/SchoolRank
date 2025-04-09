package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type CombinedAverageRatingController struct{}

func (c *CombinedAverageRatingController) GetCombinedAverageRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract school_id from query parameters
		schoolID := r.URL.Query().Get("school_id")
		if schoolID == "" {
			// Use models.Error instead of map
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Query to get total score for first type
		queryFirstType := `
			SELECT 
				SUM(COALESCE(fs.score, 0) + COALESCE(ss.score, 0) + COALESCE(ft.history_of_kazakhstan, 0) + 
					COALESCE(ft.mathematical_literacy, 0) + COALESCE(ft.reading_literacy, 0)) AS total_score
			FROM UNT_Score us
			JOIN Student s ON us.student_id = s.student_id
			LEFT JOIN UNT_Type ut ON us.unt_type_id = ut.unt_type_id
			LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
			LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
			LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
			WHERE s.school_id = ? AND us.unt_type_id = 1
		`

		var totalScoreFirstType sql.NullFloat64
		err := db.QueryRow(queryFirstType, schoolID).Scan(&totalScoreFirstType)
		if err != nil {
			log.Println("SQL Error for first type:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get total score for first type"})
			return
		}

		// Query to get total score for second type
		querySecondType := `
			SELECT 
				SUM(COALESCE(st.history_of_kazakhstan_creative, 0) + COALESCE(st.reading_literacy_creative, 0) + 
					COALESCE(st.creative_exam1, 0) + COALESCE(st.creative_exam2, 0)) AS total_score_creative
			FROM UNT_Score us
			JOIN Student s ON us.student_id = s.student_id
			LEFT JOIN UNT_Type ut ON us.unt_type_id = ut.unt_type_id
			LEFT JOIN Second_Type st ON ut.second_type_id = st.second_type_id
			WHERE s.school_id = ? AND us.unt_type_id = 2
		`

		var totalScoreSecondType sql.NullFloat64
		err = db.QueryRow(querySecondType, schoolID).Scan(&totalScoreSecondType)
		if err != nil {
			log.Println("SQL Error for second type:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get total score for second type"})
			return
		}

		// Combine total score results
		var totalScore float64
		if totalScoreFirstType.Valid {
			totalScore += totalScoreFirstType.Float64
		}
		if totalScoreSecondType.Valid {
			totalScore += totalScoreSecondType.Float64
		}

		// Calculate average for both types
		var avgFirstType float64
		var avgSecondType float64

		if totalScoreFirstType.Valid {
			avgFirstType = totalScoreFirstType.Float64
		}

		if totalScoreSecondType.Valid {
			avgSecondType = totalScoreSecondType.Float64
		}

		// Combine both averages and calculate final combined average
		combinedAverageRating := (avgFirstType + avgSecondType) / 2

		// Respond with the combined average rating
		utils.ResponseJSON(w, map[string]interface{}{
			"avg_total_score_first_type": avgFirstType,
			"avg_total_score_second_type": avgSecondType,
			"combined_average_rating": combinedAverageRating,
		})
	}
}
