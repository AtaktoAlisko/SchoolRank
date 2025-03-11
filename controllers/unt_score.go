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

        // Calculate total score based on subject scores
        var totalScore int
        if untScore.UNTTypeID != 0 {
            err := db.QueryRow(`
                SELECT 
                    COALESCE(SUM(fs.score), 0) + 
                    COALESCE(MAX(ft.history_of_kazakhstan), 0) + 
                    COALESCE(MAX(ft.mathematical_literacy), 0) +  
                    COALESCE(MAX(ft.reading_literacy), 0) 
                FROM UNT_Score us
                LEFT JOIN UNT_Type ut ON us.unt_type_id = ut.unt_type_id
                LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
                LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
                LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
                LEFT JOIN Second_Type st ON ut.second_type_id = st.second_type_id
                WHERE us.unt_type_id = ?
            `, untScore.UNTTypeID).Scan(&totalScore)

            if err != nil {
                log.Println("Error calculating total score:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate total score"})
                return
            }
        } else {
            totalScore = untScore.FirstSubjectScore + untScore.SecondSubjectScore + untScore.HistoryKazakhstan + untScore.MathematicalLiteracy + untScore.ReadingLiteracy
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
func (sc UNTScoreController) GetUNTScores(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        query := `
        SELECT 
            us.unt_score_id, 
            us.year, 
            us.unt_type_id, 
            us.student_id, 
            us.total_score,
            fs.subject AS first_subject_name, 
            fs.score AS first_subject_score,
            ss.subject AS second_subject_name, 
            ss.score AS second_subject_score,
            COALESCE(st.history_of_kazakhstan, ft.history_of_kazakhstan, 0) AS history_of_kazakhstan,
            COALESCE(st.mathematical_literacy, ft.mathematical_literacy, 0) AS mathematical_literacy,
            COALESCE(st.reading_literacy, ft.reading_literacy, 0) AS reading_literacy,
            s.first_name, 
            s.last_name, 
            s.iin
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

            // Using sql.Null* for fields that may have NULL values
            var untTypeID, studentID, totalScore sql.NullInt64
            var firstSubjectScore, secondSubjectScore sql.NullInt64
            var historyKazakhstan, mathematicalLiteracy, readingLiteracy sql.NullInt64
            var firstSubjectName, secondSubjectName sql.NullString
            var firstName, lastName, iin sql.NullString

            // Scanning the row into appropriate variables
            if err := rows.Scan(
                &score.ID, &score.Year, &untTypeID, &studentID, &totalScore,
                &firstSubjectName, &firstSubjectScore,
                &secondSubjectName, &secondSubjectScore,
                &historyKazakhstan, &mathematicalLiteracy, &readingLiteracy,
                &firstName, &lastName, &iin,
            ); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Scores"})
                return
            }

            // Handling sql.Null* types to set actual values
            if untTypeID.Valid {
                score.UNTTypeID = int(untTypeID.Int64)
            }

            if studentID.Valid {
                score.StudentID = int(studentID.Int64)
            }

            if totalScore.Valid {
                score.TotalScore = int(totalScore.Int64)
            }

            if firstSubjectScore.Valid {
                score.FirstSubjectScore = int(firstSubjectScore.Int64)
            }

            if secondSubjectScore.Valid {
                score.SecondSubjectScore = int(secondSubjectScore.Int64)
            }

            if historyKazakhstan.Valid {
                score.HistoryKazakhstan = int(historyKazakhstan.Int64)
            } else {
                score.HistoryKazakhstan = 0 // default value
            }

            if mathematicalLiteracy.Valid {
                score.MathematicalLiteracy = int(mathematicalLiteracy.Int64)
            } else {
                score.MathematicalLiteracy = 0 // default value
            }

            if readingLiteracy.Valid {
                score.ReadingLiteracy = int(readingLiteracy.Int64)
            } else {
                score.ReadingLiteracy = 0 // default value
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

            // Add to the result slice
            scores = append(scores, score)
        }

        // Send back the result as JSON
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
func (usc UNTScoreController) CalculateStudentRatings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Запрос для получения всех студентов и их баллов
		query := `
			SELECT us.student_id, us.year, us.total_score, ut.first_type_id, ut.second_type_id,
			       fs.subject AS first_subject_name, fs.score AS first_subject_score,
			       ss.subject AS second_subject_name, ss.score AS second_subject_score,
			       st.history_of_kazakhstan, st.reading_literacy
			FROM UNT_Score us
			JOIN UNT_Type ut ON us.unt_type_id = ut.unt_type_id
			LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
			LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
			LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
			LEFT JOIN Second_Type st ON ut.second_type_id = st.second_type_id
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get student data"})
			return
		}
		defer rows.Close()

		// Массив для хранения всех студентов и их баллов
		var students []models.StudentRating
		var totalMaxScore int
		var totalStudentScore int

		for rows.Next() {
			var rating models.StudentRating
			var firstSubjectScore, secondSubjectScore, historyKazakhstan, readingLiteracy int
			var firstSubjectName, secondSubjectName string

			// Сканируем данные о студенте и его баллы
			err := rows.Scan(
				&rating.StudentID, &rating.Year, &rating.TotalScore,
				&rating.FirstTypeID, &rating.SecondTypeID,
				&firstSubjectName, &firstSubjectScore,
				&secondSubjectName, &secondSubjectScore,
				&historyKazakhstan, &readingLiteracy,
			)
			if err != nil {
				log.Println("Error scanning data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to scan student data"})
				return
			}

			// Рассчитываем процент для каждого студента
			rating.FirstSubjectName = firstSubjectName
			rating.SecondSubjectName = secondSubjectName
			rating.FirstSubjectScore = firstSubjectScore
			rating.SecondSubjectScore = secondSubjectScore
			rating.HistoryKazakhstan = historyKazakhstan
			rating.ReadingLiteracy = readingLiteracy

			// В данном случае используем максимальный возможный балл для 1 типа (140) и 2 типа (30) экзамена
			maxScore := 140
			if rating.SecondTypeID != nil {
				maxScore = 30
			}

			rating.Rating = float64(rating.TotalScore) / float64(maxScore) * 100

			// Добавляем информацию о студенте в список
			students = append(students, rating)
			totalMaxScore += maxScore
			totalStudentScore += rating.TotalScore
		}

		// Рассчитываем общий рейтинг для класса
		classAverage := (float64(totalStudentScore) / float64(totalMaxScore)) * 100

		// Добавляем общий рейтинг в ответ
		utils.ResponseJSON(w, struct {
			AverageRating float64         `json:"average_rating"`
			Students      []models.StudentRating `json:"students"`
		}{
			AverageRating: classAverage,
			Students:      students,
		})
	}
}
