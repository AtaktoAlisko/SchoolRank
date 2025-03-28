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

type RegionalOlympiadController struct{}

// CreateRegionalOlympiad - создает запись о региональной олимпиаде
func (roc *RegionalOlympiadController) CreateRegionalOlympiad(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Проверка токена и получение userID
        userID, err := utils.VerifyToken(r)
        if err != nil {
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

        // 3. Декодируем тело запроса
        var olympiad models.RegionalOlympiad
        if err := json.NewDecoder(r.Body).Decode(&olympiad); err != nil {
            log.Printf("Error decoding request body: %v", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
            return
        }

        // 4. Проверка, если дата не задана, то ставим текущую
        if olympiad.CompetitionDate.IsZero() {
            olympiad.CompetitionDate = time.Now() // Устанавливаем текущую дату
        }

        // 5. Присваиваем баллы в зависимости от места
        switch olympiad.RegionalOlympiadPlace {
        case 1:
            olympiad.Score = 50
        case 2:
            olympiad.Score = 30
        case 3:
            olympiad.Score = 20
        default:
            olympiad.Score = 0 // Для всех других мест 0 баллов
        }

        // 6. Получаем school_id директора
        var directorSchoolID int
        err = db.QueryRow("SELECT school_id FROM Users WHERE id = ?", userID).Scan(&directorSchoolID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching director school"})
            return
        }

        // 7. Вставка новой записи о региональной олимпиаде с указанием school_id
        query := `INSERT INTO Regional_Olympiad (student_id, regional_olympiad_place, score, competition_date, school_id) 
                  VALUES (?, ?, ?, ?, ?)`
        _, err = db.Exec(query, olympiad.StudentID, olympiad.RegionalOlympiadPlace, olympiad.Score, olympiad.CompetitionDate, directorSchoolID)
        if err != nil {
            log.Printf("Error inserting into RegionalOlympiad: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create Regional Olympiad record"})
            return
        }

        // 8. Ответ успешного выполнения
        utils.ResponseJSON(w, map[string]string{"message": "Regional Olympiad record created successfully"})
    }
}
// GetRegionalOlympiad - получение данных о региональной олимпиаде
func (roc *RegionalOlympiadController) GetRegionalOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем данные о региональной олимпиаде
		query := `SELECT id, student_id, regional_olympiad_place, score, competition_date 
				  FROM Regional_Olympiad`
		
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Error fetching RegionalOlympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch Regional Olympiad records"})
			return
		}
		defer rows.Close()

		var olympiadRecords []models.RegionalOlympiad
		for rows.Next() {
			var olympiad models.RegionalOlympiad
			var competitionDate string // Используем string для хранения значения

			// Сканируем данные
			err := rows.Scan(&olympiad.ID, &olympiad.StudentID, &olympiad.RegionalOlympiadPlace, &olympiad.Score, &competitionDate)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning Regional Olympiad record"})
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
// GetAverageRegionalOlympiadScore - получение среднего балла для региональной олимпиады
func (roc *RegionalOlympiadController) GetAverageRegionalOlympiadScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем ID школы (например, из query параметров)
		schoolID := r.URL.Query().Get("school_id")
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Запрос для получения всех записей о региональной олимпиаде для этой школы
		query := `
			SELECT regional_olympiad_place 
			FROM Regional_Olympiad
			WHERE school_id = ?
		`

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Printf("Error fetching Regional Olympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch Regional Olympiad records"})
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
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Regional Olympiad records"})
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

		// Рассчитываем average_region_olympiad_score по формуле
		averageScore := float64(totalScore) / (50 * float64(prizeWinnersCount))

		// Рассчитываем рейтинг для региональной олимпиады
		regionOlympiadRating := averageScore * 0.3

		// Возвращаем результат
		utils.ResponseJSON(w, map[string]interface{}{
			"average_region_olympiad_score": averageScore,
			"region_olympiad_rating":        regionOlympiadRating,
			"total_score":                   totalScore,
			"prize_winners_count":           prizeWinnersCount,
		})
	}
}
func (roc *RegionalOlympiadController) DeleteRegionalOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверка токена и получение userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
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

		// 4. Получаем student_id из записи о региональной олимпиаде
		var studentSchoolID *int // Use a pointer to handle NULL
		var studentID int
		err = db.QueryRow("SELECT student_id FROM Regional_Olympiad WHERE id = ?", olympiadID).Scan(&studentID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching olympiad data"})
			return
		}

		// Получаем school_id студента
		err = db.QueryRow("SELECT school_id FROM Student WHERE student_id = ?", studentID).Scan(&studentSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student school"})
			return
		}

		// Проверка на случай, если school_id не найден (NULL)
		if studentSchoolID == nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student's school not found"})
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
		if *studentSchoolID != directorSchoolID {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "This student does not belong to your school"})
			return
		}

		// 7. Удаление записи о региональной олимпиаде
		query := `DELETE FROM Regional_Olympiad WHERE id = ?`
		_, err = db.Exec(query, olympiadID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete Regional Olympiad record"})
			return
		}

		// 8. Ответ успешного выполнения
		utils.ResponseJSON(w, map[string]string{"message": "Regional Olympiad record deleted successfully"})
	}
}

