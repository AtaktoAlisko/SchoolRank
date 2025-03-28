package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"time"
)

type RepublicanOlympiadController struct{}

// CreateRepublicanOlympiad - создаёт запись о республиканской олимпиаде
func (roc *RepublicanOlympiadController) CreateRepublicanOlympiad(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Проверка токена и получение userID
        userID, err := utils.VerifyToken(r)
        if err != nil {
            // Если токен неверный/просрочен — 401 Unauthorized
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
            return
        }

        // 2. Проверяем роль пользователя
        var role string
        err = db.QueryRow("SELECT role FROM Users WHERE id = ?", userID).Scan(&role)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
            return
        }

        // Если роль не "director", то запрещаем выполнение
        if role != "director" {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You are not authorized to perform this action"})
            return
        }

        // 3. Декодируем тело запроса
        var olympiad models.RepublicanOlympiad
        if err := json.NewDecoder(r.Body).Decode(&olympiad); err != nil {
            log.Printf("Error decoding request body: %v", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body format"})
            return
        }

        // 4. Валидация входных данных
        if olympiad.StudentID == 0 || olympiad.RepublicanOlympiadPlace == 0 {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student or place"})
            return
        }

        // 5. Проверка, что студент относится к школе директора
        var studentSchoolID int
        err = db.QueryRow("SELECT school_id FROM Student WHERE student_id = ?", olympiad.StudentID).Scan(&studentSchoolID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student school"})
            return
        }

        // Сравниваем, соответствует ли школа студента школе директора
        var directorSchoolID int
        err = db.QueryRow("SELECT school_id FROM Users WHERE id = ?", userID).Scan(&directorSchoolID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching director school"})
            return
        }

        if studentSchoolID != directorSchoolID {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "This student does not belong to your school"})
            return
        }

        // 6. Если дата не задана, ставим текущую
        if olympiad.CompetitionDate.IsZero() {
            olympiad.CompetitionDate = time.Now() // Устанавливаем текущую дату
        }

        // 7. Присваиваем баллы в зависимости от места
        switch olympiad.RepublicanOlympiadPlace {
        case 1:
            olympiad.Score = 50
        case 2:
            olympiad.Score = 30
        case 3:
            olympiad.Score = 20
        default:
            olympiad.Score = 0 // Для всех других мест 0 баллов
        }

        // 8. Вставка данных в таблицу RepublicanOlympiad
        query := `INSERT INTO RepublicanOlympiad (student_id, republican_olympiad_place, score, competition_date, school_id) 
                  VALUES (?, ?, ?, ?, ?)`
        _, err = db.Exec(query, olympiad.StudentID, olympiad.RepublicanOlympiadPlace, olympiad.Score, olympiad.CompetitionDate, directorSchoolID)
        if err != nil {
            log.Printf("Error inserting into RepublicanOlympiad: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create Republican Olympiad record"})
            return
        }

        // 9. Ответ успешного выполнения
        utils.ResponseJSON(w, map[string]string{"message": "Republican Olympiad record created successfully"})
    }
}
// GetRepublicanOlympiad - получение данных о республиканской олимпиаде
func (roc *RepublicanOlympiadController) GetRepublicanOlympiad(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Получаем данные о республиканской олимпиаде
        query := `SELECT id, student_id, republican_olympiad_place, score, competition_date 
                  FROM RepublicanOlympiad`
        
        rows, err := db.Query(query)
        if err != nil {
            log.Printf("Error fetching Republican Olympiad records: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch Republican Olympiad records"})
            return
        }
        defer rows.Close()

        var olympiadRecords []models.RepublicanOlympiad
        for rows.Next() {
            var olympiad models.RepublicanOlympiad
            var competitionDate string // Используем string для хранения значения

            // Сканируем данные
            err := rows.Scan(&olympiad.ID, &olympiad.StudentID, &olympiad.RepublicanOlympiadPlace, &olympiad.Score, &competitionDate)
            if err != nil {
                log.Printf("Error scanning row: %v", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning Republican Olympiad record"})
                return
            }

            // Проверка и конвертация строки в время
            if competitionDate != "" {
                olympiad.CompetitionDate, err = time.Parse("2006-01-02 15:04:05", competitionDate)
                if err != nil {
                    log.Printf("Error parsing competition_date: %v", err)
                    utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Invalid competition_date format"})
                    return
                }
            } else {
                olympiad.CompetitionDate = time.Time{} // Если пустое значение, присваиваем zero time
            }

            olympiadRecords = append(olympiadRecords, olympiad)
        }

        if len(olympiadRecords) == 0 {
            utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No records found"})
            return
        }

        // 2. Отправляем данные в формате JSON
        utils.ResponseJSON(w, olympiadRecords)
    }
}
func (roc *RepublicanOlympiadController) DeleteRepublicanOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверка токена и получение userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			// Если токен неверный/просрочен — 401 Unauthorized
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// 2. Проверка роли пользователя (директор)
		var role string
		err = db.QueryRow("SELECT role FROM Users WHERE id = ?", userID).Scan(&role)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
			return
		}

		// Если роль не "director", то запрещаем выполнение
		if role != "director" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You are not authorized to perform this action"})
			return
		}

		// 3. Получаем ID олимпиады для удаления
		olympiadID := r.URL.Query().Get("id")
		if olympiadID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Olympiad ID is required"})
			return
		}

		// 4. Получаем student_id и school_id студента
		var studentSchoolID int
		var studentID int
		err = db.QueryRow("SELECT student_id, school_id FROM RepublicanOlympiad WHERE id = ?", olympiadID).Scan(&studentID, &studentSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching olympiad data"})
			return
		}

		// 5. Получаем school_id директора
		var directorSchoolID int
		err = db.QueryRow("SELECT school_id FROM Users WHERE id = ?", userID).Scan(&directorSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching director school"})
			return
		}

		// 6. Проверка, что студент принадлежит школе директора
		if studentSchoolID != directorSchoolID {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "This student does not belong to your school"})
			return
		}

		// 7. Удаление записи о республиканской олимпиаде
		query := `DELETE FROM RepublicanOlympiad WHERE id = ?`
		_, err = db.Exec(query, olympiadID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete Republican Olympiad record"})
			return
		}

		// 8. Ответ успешного выполнения
		utils.ResponseJSON(w, map[string]string{"message": "Republican Olympiad record deleted successfully"})
	}
}
func (roc *RepublicanOlympiadController) GetAverageRepublicanOlympiadScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем ID школы (например, из query параметров)
		schoolID := r.URL.Query().Get("school_id")
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Запрос для получения всех записей о республиканской олимпиаде для этой школы
		query := `
			SELECT republican_olympiad_place 
			FROM RepublicanOlympiad
			WHERE school_id = ?
		`

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Printf("Error fetching Republican Olympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch Republican Olympiad records"})
			return
		}
		defer rows.Close()

		var totalScore int
		var prizeWinnersCount int

		// Пройдем по всем записям и вычислим баллы для 1, 2, и 3 мест
		for rows.Next() {
			var place int
			err := rows.Scan(&place)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Republican Olympiad records"})
				return
			}

			// Присваиваем баллы в зависимости от места
			if place == 1 {
				totalScore += 50
				prizeWinnersCount++
			} else if place == 2 {
				totalScore += 30
				prizeWinnersCount++
			} else if place == 3 {
				totalScore += 20
				prizeWinnersCount++
			}
		}

		// Проверка, есть ли призовые места
		if prizeWinnersCount == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No prize-winning students found"})
			return
		}

		// Рассчитываем average_republican_olympiad_score по формуле
		averageScore := float64(totalScore) / (50 * float64(prizeWinnersCount))

		// Рассчитываем rating для республиканской олимпиады
		republicanOlympiadRating := averageScore * 0.5

		// Возвращаем результат
		utils.ResponseJSON(w, map[string]interface{}{
			"average_republican_olympiad_score": averageScore,
			"republican_olympiad_rating":        republicanOlympiadRating,
			"total_score":                       totalScore,
			"prize_winners_count":               prizeWinnersCount,
		})
	}
}


