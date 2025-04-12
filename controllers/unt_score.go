package controllers

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"

	"github.com/gorilla/mux"
)

type UNTScoreController struct{}



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
func (usc *UNTScoreController) GetAverageRatingBySchool(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Извлекаем school_id из параметров URL
        vars := mux.Vars(r)
        schoolID, err := strconv.Atoi(vars["school_id"])
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
            return
        }

        // Запрос для получения среднего балла по школе
        query := `
        SELECT 
            AVG(CASE WHEN ft.first_subject_score IS NOT NULL THEN ft.first_subject_score ELSE 0 END) AS avg_first_subject_score,
            AVG(CASE WHEN ft.second_subject_score IS NOT NULL THEN ft.second_subject_score ELSE 0 END) AS avg_second_subject_score,
            AVG(CASE WHEN ft.history_of_kazakhstan IS NOT NULL THEN ft.history_of_kazakhstan ELSE 0 END) AS avg_history_of_kazakhstan,
            AVG(CASE WHEN ft.mathematical_literacy IS NOT NULL THEN ft.mathematical_literacy ELSE 0 END) AS avg_mathematical_literacy,
            AVG(CASE WHEN ft.reading_literacy IS NOT NULL THEN ft.reading_literacy ELSE 0 END) AS avg_reading_literacy,
            AVG(CASE WHEN ft.first_subject_score IS NOT NULL AND ft.second_subject_score IS NOT NULL AND 
                     ft.history_of_kazakhstan IS NOT NULL AND ft.mathematical_literacy IS NOT NULL AND 
                     ft.reading_literacy IS NOT NULL 
                     THEN (ft.first_subject_score + ft.second_subject_score + ft.history_of_kazakhstan + 
                           ft.mathematical_literacy + ft.reading_literacy) ELSE 0 END) AS avg_total_score
        FROM First_Type ft
        WHERE ft.school_id = ?`

        row := db.QueryRow(query, schoolID)

        var avgFirstSubjectScore, avgSecondSubjectScore, avgHistoryOfKazakhstan, avgMathematicalLiteracy, avgReadingLiteracy, avgTotalScore float64

        err = row.Scan(&avgFirstSubjectScore, &avgSecondSubjectScore, &avgHistoryOfKazakhstan, &avgMathematicalLiteracy, &avgReadingLiteracy, &avgTotalScore)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate average rating"})
            return
        }

        // Response with average score
        result := map[string]float64{
            "avg_first_subject_score":      avgFirstSubjectScore,
            "avg_second_subject_score":     avgSecondSubjectScore,
            "avg_history_of_kazakhstan":    avgHistoryOfKazakhstan,
            "avg_mathematical_literacy":    avgMathematicalLiteracy,
            "avg_reading_literacy":         avgReadingLiteracy,
            "avg_total_score":              avgTotalScore,
        }

        utils.ResponseJSON(w, result)
    }
}
func (c *UNTScoreController) GetAverageRatingSecondBySchool(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Извлекаем school_id из параметров URL
        vars := mux.Vars(r)
        schoolID, err := strconv.Atoi(vars["school_id"])
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
            return
        }

        // Запрос для получения всех оценок по конкретной школе
        query := `
        SELECT 
            history_of_kazakhstan_creative,
            reading_literacy_creative,
            creative_exam1,
            creative_exam2
        FROM Second_Type
        WHERE school_id = ?`

        rows, err := db.Query(query, schoolID)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get Second Types by School"})
            return
        }
        defer rows.Close()

        var totalScore float64
        var studentCount int

        for rows.Next() {
            var historyOfKazakhstanCreative, readingLiteracyCreative, creativeExam1, creativeExam2 sql.NullInt64

            // Считываем данные для каждого экзамена
            if err := rows.Scan(&historyOfKazakhstanCreative, &readingLiteracyCreative, &creativeExam1, &creativeExam2); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Types"})
                return
            }

            // Вычисляем сумму оценок для каждого студента
            totalScore += float64(historyOfKazakhstanCreative.Int64 + readingLiteracyCreative.Int64 + creativeExam1.Int64 + creativeExam2.Int64)
            studentCount++
        }

        if studentCount == 0 {
            utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No students found for this school"})
            return
        }

        // Рассчитываем средний балл
        averageRating := totalScore / float64(studentCount)

        // Возвращаем результат в формате JSON
        utils.ResponseJSON(w, map[string]interface{}{
            "average_rating": averageRating,
        })
    }
}
func (usc *UNTScoreController) GetCombinedAverageRating(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {

        // Извлекаем school_id из query параметра URL
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
            return
        }

        // 1. Получаем avg_total_score для первого типа
        queryFirstType := `
        SELECT 
            AVG(CASE WHEN ft.first_subject_score IS NOT NULL THEN ft.first_subject_score ELSE 0 END) AS avg_first_subject_score,
            AVG(CASE WHEN ft.second_subject_score IS NOT NULL THEN ft.second_subject_score ELSE 0 END) AS avg_second_subject_score,
            AVG(CASE WHEN ft.history_of_kazakhstan IS NOT NULL THEN ft.history_of_kazakhstan ELSE 0 END) AS avg_history_of_kazakhstan,
            AVG(CASE WHEN ft.mathematical_literacy IS NOT NULL THEN ft.mathematical_literacy ELSE 0 END) AS avg_mathematical_literacy,
            AVG(CASE WHEN ft.reading_literacy IS NOT NULL THEN ft.reading_literacy ELSE 0 END) AS avg_reading_literacy,
            AVG(CASE WHEN ft.first_subject_score IS NOT NULL AND ft.second_subject_score IS NOT NULL AND 
                     ft.history_of_kazakhstan IS NOT NULL AND ft.mathematical_literacy IS NOT NULL AND 
                     ft.reading_literacy IS NOT NULL 
                     THEN (ft.first_subject_score + ft.second_subject_score + ft.history_of_kazakhstan + 
                           ft.mathematical_literacy + ft.reading_literacy) ELSE 0 END) AS avg_total_score
        FROM First_Type ft
        WHERE ft.school_id = ?`

        rowFirstType := db.QueryRow(queryFirstType, schoolID)

        var avgFirstSubjectScore, avgSecondSubjectScore, avgHistoryOfKazakhstan, avgMathematicalLiteracy, avgReadingLiteracy, avgTotalScore float64
        err := rowFirstType.Scan(&avgFirstSubjectScore, &avgSecondSubjectScore, &avgHistoryOfKazakhstan, &avgMathematicalLiteracy, &avgReadingLiteracy, &avgTotalScore)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get total score for first type"})
            return
        }

        // 2. Получаем average_rating для второго типа
        querySecondType := `
        SELECT 
            history_of_kazakhstan_creative,
            reading_literacy_creative,
            creative_exam1,
            creative_exam2
        FROM Second_Type
        WHERE school_id = ?`

        rowsSecondType, err := db.Query(querySecondType, schoolID)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get Second Types by School"})
            return
        }
        defer rowsSecondType.Close()

        var totalScoreSecondType float64
        var studentCountSecondType int

        for rowsSecondType.Next() {
            var historyOfKazakhstanCreative, readingLiteracyCreative, creativeExam1, creativeExam2 sql.NullInt64

            if err := rowsSecondType.Scan(&historyOfKazakhstanCreative, &readingLiteracyCreative, &creativeExam1, &creativeExam2); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Types"})
                return
            }

            totalScoreSecondType += float64(historyOfKazakhstanCreative.Int64 + readingLiteracyCreative.Int64 + creativeExam1.Int64 + creativeExam2.Int64)
            studentCountSecondType++
        }

        if studentCountSecondType == 0 {
            utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No students found for this school"})
            return
        }

        averageRatingSecondType := totalScoreSecondType / float64(studentCountSecondType)

        // 3. Рассчитываем комбинированный рейтинг
        combinedAverageRating := (((avgTotalScore*100)/140) + ((averageRatingSecondType*100)/120)) / 2
        roundedCombinedRating := customRound(combinedAverageRating) // Применяем округление


        // Возвращаем результат в формате JSON
        utils.ResponseJSON(w, map[string]interface{}{
            "avg_total_score_first_type": avgTotalScore,
            "avg_total_score_second_type": averageRatingSecondType,
            "combined_average_rating": roundedCombinedRating,
        })
    }
}
// Функция округления с вашими правилами
func customRound(value float64) float64 {
    if value - math.Floor(value) >= 0.5 {
        return math.Ceil(value) // Округляем в большую сторону
    }
    return math.Floor(value) // Оставляем без изменений
}









