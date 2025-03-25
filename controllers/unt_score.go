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

        // Handle default zero values if necessary
        if untScore.FirstSubjectScore == 0 {
            untScore.FirstSubjectScore = 0
        }
        if untScore.SecondSubjectScore == 0 {
            untScore.SecondSubjectScore = 0
        }
        if untScore.HistoryKazakhstan == 0 {
            untScore.HistoryKazakhstan = 0
        }
        if untScore.MathematicalLiteracy == 0 {
            untScore.MathematicalLiteracy = 0
        }
        if untScore.ReadingLiteracy == 0 {
            untScore.ReadingLiteracy = 0
        }

        // Log to verify values are correctly parsed
        log.Printf("FirstSubjectScore: %d, SecondSubjectScore: %d, HistoryKazakhstan: %d, MathematicalLiteracy: %d, ReadingLiteracy: %d\n",
            untScore.FirstSubjectScore, untScore.SecondSubjectScore, untScore.HistoryKazakhstan, untScore.MathematicalLiteracy, untScore.ReadingLiteracy)

        // Calculate total score based on subject scores directly in Go
        totalScore := untScore.FirstSubjectScore + untScore.SecondSubjectScore + untScore.HistoryKazakhstan + untScore.MathematicalLiteracy + untScore.ReadingLiteracy

        // Log to verify total score calculation
        log.Printf("Calculated Total Score: %d\n", totalScore)

        // Check if UNT_Type and Student exists
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
        query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, total_score) VALUES (?, ?, ?, ?)`
        _, err := db.Exec(query, untScore.Year, untScore.UNTTypeID, untScore.StudentID, totalScore)
        if err != nil {
            log.Println("Error inserting UNT score:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
            return
        }

        utils.ResponseJSON(w, "UNT Score created successfully")
    }
}
func (usc *UNTScoreController) GetUNTScores(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        query := `
        SELECT 
            us.unt_score_id, 
            us.year, 
            COALESCE(us.unt_type_id, 0) AS unt_type_id, 
            us.student_id, 
            fs.subject AS first_subject_name, 
            COALESCE(fs.score, 0) AS first_subject_score,
            ss.subject AS second_subject_name, 
            COALESCE(ss.score, 0) AS second_subject_score,
            COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan,
            COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy,
            COALESCE(ft.reading_literacy, 0) AS reading_literacy,
            s.first_name, 
            s.last_name, 
            s.iin
        FROM UNT_Score us
        LEFT JOIN UNT_Type ut ON us.unt_type_id = ut.unt_type_id
        LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
        LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
        LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
        LEFT JOIN Student s ON us.student_id = s.student_id
        `
        log.Println("Executing query: ", query)

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
            var untTypeID sql.NullInt64
            var firstSubjectScore, secondSubjectScore sql.NullInt64
            var historyKazakhstan, mathematicalLiteracy, readingLiteracy sql.NullInt64
            var firstSubjectName, secondSubjectName sql.NullString
            var firstName, lastName, iin sql.NullString

            err := rows.Scan(
                &score.ID, &score.Year, &untTypeID, &score.StudentID,
                &firstSubjectName, &firstSubjectScore,
                &secondSubjectName, &secondSubjectScore,
                &historyKazakhstan, &mathematicalLiteracy, &readingLiteracy,
                &firstName, &lastName, &iin,
            )
            if err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Scores"})
                return
            }

            // Заполнение значений
            if untTypeID.Valid {
                score.UNTTypeID = int(untTypeID.Int64)
            } else {
                score.UNTTypeID = 0
            }
            if firstSubjectScore.Valid {
                score.FirstSubjectScore = int(firstSubjectScore.Int64)
            }
            if secondSubjectScore.Valid {
                score.SecondSubjectScore = int(secondSubjectScore.Int64)
            }
            if historyKazakhstan.Valid {
                score.HistoryKazakhstan = int(historyKazakhstan.Int64)
            }
            if mathematicalLiteracy.Valid {
                score.MathematicalLiteracy = int(mathematicalLiteracy.Int64)
            }
            if readingLiteracy.Valid {
                score.ReadingLiteracy = int(readingLiteracy.Int64)
            }
            if firstSubjectName.Valid {
                score.FirstSubjectName = firstSubjectName.String
            }
            if secondSubjectName.Valid {
                score.SecondSubjectName = secondSubjectName.String
            }
            if firstName.Valid {
                score.FirstName = firstName.String
            }
            if lastName.Valid {
                score.LastName = lastName.String
            }
            if iin.Valid {
                score.IIN = iin.String
            }

            // Расчет total_score
            totalScore := score.FirstSubjectScore + score.SecondSubjectScore + score.HistoryKazakhstan + score.MathematicalLiteracy + score.ReadingLiteracy
            score.TotalScore = totalScore

            // Установка рейтинга
            score.Rating = float64(totalScore)

            // Обновление рейтинга в базе данных
            _, err = db.Exec("UPDATE UNT_Score SET rating = ? WHERE unt_score_id = ?", score.Rating, score.ID)
            if err != nil {
                log.Println("Error updating rating:", err)
            }

            scores = append(scores, score)
        }

        utils.ResponseJSON(w, scores)
    }
}
func (usc UNTScoreController) GetAverageScoreForSchool(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
            return
        }

        // Query to retrieve total scores for students in this school
        query := `
            SELECT total_score
            FROM UNT_Score us
            JOIN Student s ON us.student_id = s.student_id
            WHERE s.school_id = ?`
        
        rows, err := db.Query(query, schoolID)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get student scores"})
            return
        }
        defer rows.Close()

        var totalScoreSum, studentCount int
        for rows.Next() {
            var score int
            err := rows.Scan(&score)
            if err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse scores"})
                return
            }
            totalScoreSum += score
            studentCount++
        }

        // Debugging log to ensure calculation
        log.Printf("Total Score Sum: %d, Student Count: %d", totalScoreSum, studentCount)

        if studentCount == 0 {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "No students found for this school"})
            return
        }

        // Calculate average rating
        averageRating := float64(totalScoreSum) / float64(studentCount)
        log.Printf("Calculated Average Rating: %f", averageRating)

        // Send response with average rating
        utils.ResponseJSON(w, map[string]float64{"average_rating": averageRating})
    }
}
func (usc UNTScoreController) CalculateSchoolAverageRating(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
            return
        }

        // Получаем все total_score для учеников этой школы
        query := `
            SELECT 
                (COALESCE(fs.score, 0) + COALESCE(ss.score, 0) +
                 COALESCE(ft.history_of_kazakhstan, 0) + 
                 COALESCE(ft.mathematical_literacy, 0) + 
                 COALESCE(ft.reading_literacy, 0)) AS total_score
            FROM UNT_Score us
            JOIN Student s ON us.student_id = s.student_id
            LEFT JOIN UNT_Type ut ON us.unt_type_id = ut.unt_type_id
            LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
            LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
            LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
            WHERE s.school_id = ?
        `

        rows, err := db.Query(query, schoolID)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get UNT Scores"})
            return
        }
        defer rows.Close()

        var totalScoreSum, studentCount int
        for rows.Next() {
            var totalScore int
            if err := rows.Scan(&totalScore); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Scores"})
                return
            }
            totalScoreSum += totalScore
            studentCount++
        }

        if studentCount == 0 {
            utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No students found for this school"})
            return
        }

        averageRating := float64(totalScoreSum) / float64(studentCount)

        utils.ResponseJSON(w, map[string]interface{}{
            "average_rating": averageRating,
        })
    }
}
