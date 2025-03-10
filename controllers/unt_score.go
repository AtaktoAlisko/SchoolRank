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

        // Заменяем 0 на значения по умолчанию, если не заданы
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

        // Проверка существования UNT_Type и Student
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

        // Подтягиваем баллы из связанных таблиц только если UNT_Type задан
        var totalScore int
        if untScore.UNTTypeID != 0 {
            err := db.QueryRow(`
                SELECT COALESCE(SUM(score), 0) + 
                    COALESCE(MAX(ft.history_of_kazakhstan), 0) + 
                    COALESCE(MAX(ft.mathematical_literacy), 0) +  
                    COALESCE(MAX(ft.reading_literacy), 0) 
                FROM (
                    SELECT fs.score, ft.history_of_kazakhstan, ft.mathematical_literacy, ft.reading_literacy
                    FROM First_Subject fs
                    JOIN First_Type ft ON fs.first_subject_id = ft.first_subject_id
                    JOIN UNT_Type ut ON ft.first_type_id = ut.first_type_id
                    WHERE ut.unt_type_id = ?
                    UNION ALL
                    SELECT ss.score, ft.history_of_kazakhstan, ft.mathematical_literacy, ft.reading_literacy
                    FROM Second_Subject ss
                    JOIN First_Type ft ON ss.second_subject_id = ft.second_subject_id
                    JOIN UNT_Type ut ON ft.first_type_id = ut.first_type_id
                    WHERE ut.unt_type_id = ?
                ) AS scores
            `, untScore.UNTTypeID, untScore.UNTTypeID).Scan(&totalScore)

            if err != nil {
                log.Println("Error calculating total score:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate total score"})
                return
            }
        } else {
            // Если UNT_Type отсутствует, просто считаем баллы на основе переданных значений
            totalScore = untScore.FirstSubjectScore + untScore.SecondSubjectScore + untScore.HistoryKazakhstan + untScore.MathematicalLiteracy + untScore.ReadingLiteracy
        }

        // Вставляем данные в таблицу
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

// Получить все UNT Scores
func (sc UNTScoreController) GetUNTScores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
		SELECT us.unt_score_id, us.year, us.unt_type_id, us.student_id, us.total_score,
		       fs.subject AS first_subject_name, fs.score AS first_subject_score,
		       ss.subject AS second_subject_name, ss.score AS second_subject_score,
		       st.history_of_kazakhstan, st.reading_literacy,
		       s.first_name, s.last_name, s.iin
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

			// Используем sql.NullInt64 для полей, которые могут быть NULL
			var untTypeID, studentID, totalScore sql.NullInt64
			var firstSubjectScore, secondSubjectScore sql.NullInt64
			var historyKazakhstan, readingLiteracy sql.NullInt64
			var firstSubjectName, secondSubjectName sql.NullString
			var firstName, lastName, iin sql.NullString

			// Сканы для полей с возможными NULL значениями
			if err := rows.Scan(
				&score.ID, &score.Year, &untTypeID, &studentID, &totalScore,
				&firstSubjectName, &firstSubjectScore,
				&secondSubjectName, &secondSubjectScore,
				&historyKazakhstan, &readingLiteracy,
				&firstName, &lastName, &iin,
			); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Scores"})
				return
			}

			// Преобразуем sql.NullInt64 в обычные значения
			if untTypeID.Valid {
				score.UNTTypeID = int(untTypeID.Int64)
			} else {
				score.UNTTypeID = 0 // или другое значение по умолчанию
			}

			if studentID.Valid {
				score.StudentID = int(studentID.Int64)
			} else {
				score.StudentID = 0 // или другое значение по умолчанию
			}

			// Обработка total_score, которое может быть NULL
			if totalScore.Valid {
				score.TotalScore = int(totalScore.Int64)
			} else {
				score.TotalScore = 0 // или другое значение по умолчанию
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

			// Добавляем результат в срез
			scores = append(scores, score)
		}

		// Отправляем результат в формате JSON
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
