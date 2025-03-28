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
func (usc *UNTScoreController) GetUNTScores(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        query := `
            SELECT 
                us.unt_score_id, 
                us.year, 
                COALESCE(us.unt_type_id, 0) AS unt_type_id, 
                us.student_id, 
                -- Для первого типа: только предметы
                CASE WHEN us.unt_type_id = 1 THEN fs.subject ELSE NULL END AS first_subject_name,
                CASE WHEN us.unt_type_id = 1 THEN COALESCE(fs.score, 0) ELSE NULL END AS first_subject_score,
                CASE WHEN us.unt_type_id = 1 THEN ss.subject ELSE NULL END AS second_subject_name,
                CASE WHEN us.unt_type_id = 1 THEN COALESCE(ss.score, 0) ELSE NULL END AS second_subject_score,
                -- Для второго типа: только творческие экзамены
                CASE WHEN us.unt_type_id = 2 THEN 'Creative History' ELSE NULL END AS first_subject_name,
                CASE WHEN us.unt_type_id = 2 THEN COALESCE(st.history_of_kazakhstan_creative, 0) ELSE NULL END AS first_subject_score,
                CASE WHEN us.unt_type_id = 2 THEN 'Creative Literacy' ELSE NULL END AS second_subject_name,
                CASE WHEN us.unt_type_id = 2 THEN COALESCE(st.reading_literacy_creative, 0) ELSE NULL END AS second_subject_score,
                -- Общие параметры
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
            LEFT JOIN Second_Type st ON ut.second_type_id = st.second_type_id
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
            var firstSubjectName, secondSubjectName, firstName, lastName, iin sql.NullString

            err := rows.Scan(
                &score.ID, 
                &score.Year, 
                &untTypeID, 
                &score.StudentID,
                &firstSubjectName, 
                &firstSubjectScore,
                &secondSubjectName, 
                &secondSubjectScore,
                &historyKazakhstan, 
                &mathematicalLiteracy, 
                &readingLiteracy,
                &firstName, 
                &lastName, 
                &iin,
            )
            if err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Scores"})
                return
            }

            // Заполнение значений
            if untTypeID.Valid {
                score.UNTTypeID = int(untTypeID.Int64)
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

            // Для первого типа рассчитываем total_score
            if score.UNTTypeID == 1 {
                totalScore := score.FirstSubjectScore + score.SecondSubjectScore + score.HistoryKazakhstan + score.MathematicalLiteracy + score.ReadingLiteracy
                score.TotalScore = totalScore
            }

            // Для второго типа рассчитываем total_score_creative
            if score.UNTTypeID == 2 {
                totalScoreCreative := score.HistoryKazakhstan + score.ReadingLiteracy
                score.TotalScoreCreative = totalScoreCreative
            }

            // Добавление score в список
            scores = append(scores, score)
        }

        // Возвращаем данные в формате JSON
        utils.ResponseJSON(w, scores)
    }
}
func (usc *UNTScoreController) GetTotalScoreForSchool(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
            return
        }

        // Запрос для подсчета общего балла по всем студентам школы для первого типа экзаменов
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

        // Запрос для подсчета общего балла по всем студентам школы для второго типа экзаменов
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

        // Объединяем результаты и отправляем их как ответ
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






