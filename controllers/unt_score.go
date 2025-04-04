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

// CreateUNTScore handles the creation of a UNT score entry.
func (usc *UNTScoreController) CreateUNTScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var untScore models.UNTScore

		// Decode incoming request
		if err := json.NewDecoder(r.Body).Decode(&untScore); err != nil {
			log.Println("Error decoding request body:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Calculate total score for the first type (profile exam subjects)
		totalScore := untScore.FirstSubjectScore + untScore.SecondSubjectScore + untScore.HistoryKazakhstan + untScore.MathematicalLiteracy + untScore.ReadingLiteracy

		// Calculate total score for the second type (creative exam and reading history)
		totalScoreCreative := untScore.CreativeExam1 + untScore.CreativeExam2 + untScore.HistoryKazakhstan + untScore.ReadingLiteracy

		// Calculate average rating for second type (creative exams)
		creativeExamPercent := 0.0
		if untScore.CreativeExam1 > 0 {
			creativeExamPercent += float64(untScore.CreativeExam1) / 120 * 100
		}
		if untScore.CreativeExam2 > 0 {
			creativeExamPercent += float64(untScore.CreativeExam2) / 140 * 100
		}
		averageRatingSecond := creativeExamPercent / 2 // Average of creative exams

		// Calculate total average rating combining both types
		averageRating := (creativeExamPercent + float64(totalScore)) / 2

		// Check if UNT_Type and Student exist
		var exists bool
		if untScore.UNTTypeID != 0 {
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM UNT_Type WHERE unt_type_id = ?)", untScore.UNTTypeID).Scan(&exists)
			if err != nil || !exists {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "UNT Type ID does not exist"})
				return
			}
		}

		if untScore.StudentID != 0 {
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM Student WHERE student_id = ?)", untScore.StudentID).Scan(&exists)
			if err != nil || !exists {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID does not exist"})
				return
			}
		}

		// Insert data into UNT_Score table
		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, total_score, total_score_creative, average_rating, average_rating_second) 
				  VALUES (?, ?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, untScore.Year, untScore.UNTTypeID, untScore.StudentID, totalScore, totalScoreCreative, averageRating, averageRatingSecond)
		if err != nil {
			log.Println("Error inserting UNT score:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
			return
		}

		utils.ResponseJSON(w, "UNT Score created successfully")
	}
}
func (usc *UNTScoreController) GetUNTScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем ID студента из параметров запроса
		studentID := r.URL.Query().Get("student_id")
		if studentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID is required"})
			return
		}

		// Проверяем, существует ли студент с данным ID
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM Student WHERE student_id = ?)", studentID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID does not exist"})
			return
		}

		// Извлекаем информацию о балле из базы данных
		var untScore models.UNTScore
		query := `SELECT year, unt_type_id, student_id, total_score, total_score_creative, average_rating, average_rating_second 
				  FROM UNT_Score WHERE student_id = ?`
		row := db.QueryRow(query, studentID)

		err = row.Scan(&untScore.Year, &untScore.UNTTypeID, &untScore.StudentID, 
			&untScore.TotalScore, &untScore.TotalScoreCreative, &untScore.AverageRating, &untScore.AverageRatingSecond)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "UNT score not found"})
			} else {
				log.Println("Error retrieving UNT score:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve UNT score"})
			}
			return
		}

		// Отправляем ответ с данными
		utils.ResponseJSON(w, untScore)
	}
}
// GetTotalScoreForSchool calculates the total UNT score for a specific school
func (usc *UNTScoreController) GetTotalScoreForSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schoolID := r.URL.Query().Get("school_id")
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Query for total score for first type
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
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get total score for first type"})
			return
		}

		// Query for total score for second type
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
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get total score for second type"})
			return
		}

		// Combine total score results and send the response
		var totalScore float64
		if totalScoreFirstType.Valid {
			totalScore += totalScoreFirstType.Float64
		}
		if totalScoreSecondType.Valid {
			totalScore += totalScoreSecondType.Float64
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"total_score_first_type": totalScoreFirstType.Float64,
			"total_score_second_type": totalScoreSecondType.Float64,
			"total_score": totalScore,
		})
	}
}
