package controllers

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"strconv"

	"ranking-school/models"
	"ranking-school/utils"

	"github.com/gorilla/mux"
)

type SchoolRatingController struct{}

// GetSchoolCompleteRating возвращает полный рейтинг школы, включающий все компоненты
func (src *SchoolRatingController) GetSchoolCompleteRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Проверяем метод запроса
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		// Step 2: Получаем userID из токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 3: Проверяем роль пользователя и его school_id
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		// Step 4: Получаем school_id из URL параметров
		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		schoolID, err := strconv.ParseInt(schoolIDStr, 10, 64)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный ID школы"})
			return
		}

		// Step 5: Проверяем права доступа на основе роли
		if userRole == "schooladmin" {
			if !userSchoolID.Valid || userSchoolID.Int64 != schoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Вы не можете просматривать данные других школ"})
				return
			}
		} else if userRole != "admin" && userRole != "moderator" && userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для просмотра рейтинга школы"})
			return
		}

		// Step 6: Проверяем существование школы
		var schoolExists bool
		var schoolName string
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?), (SELECT school_name FROM Schools WHERE school_id = ?)", schoolID, schoolID).Scan(&schoolExists, &schoolName)
		if err != nil || !schoolExists {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Школа не найдена"})
			return
		}

		// Step 7: Получаем UNT рейтинг (unt_rank)
		untRank, err := src.getUNTRank(db, schoolID)
		if err != nil {
			log.Printf("Ошибка при получении UNT рейтинга: %v", err)
			untRank = 0.0 // Устанавливаем 0, если нет данных
		}

		// Step 8: Получаем счет событий (score)
		eventScore, err := src.getEventScore(db, schoolID)
		if err != nil {
			log.Printf("Ошибка при получении счета событий: %v", err)
			eventScore = 0.0
		}

		// Step 9: Получаем очки участников (points)
		participantPoints, err := src.getParticipantPoints(db, schoolID)
		if err != nil {
			log.Printf("Ошибка при получении очков участников: %v", err)
			participantPoints = 0.0
		}

		// Step 10: Получаем рейтинг отзывов (average_rating_rank)
		averageRatingRank, err := src.getAverageRatingRank(db, schoolID)
		if err != nil {
			log.Printf("Ошибка при получении рейтинга отзывов: %v", err)
			averageRatingRank = 0.0
		}

		// Step 11: Получаем олимпиадный рейтинг (olympiad_rank)
		olympiadRank, err := src.getOlympiadRank(db, schoolID)
		if err != nil {
			log.Printf("Ошибка при получении олимпиадного рейтинга: %v", err)
			olympiadRank = 0.0
		}

		// Step 12: Вычисляем общий рейтинг
		totalRating := eventScore + participantPoints + averageRatingRank + untRank + olympiadRank

		// Step 13: Формируем ответ
		response := map[string]interface{}{
			"school_id":           schoolID,
			"school_name":         schoolName,
			"unt_rank":            math.Round(untRank*100) / 100,
			"event_score":         math.Round(eventScore*100) / 100,
			"participant_points":  math.Round(participantPoints*100) / 100,
			"average_rating_rank": math.Round(averageRatingRank*100) / 100,
			"olympiad_rank":       math.Round(olympiadRank*100) / 100,
			"total_rating":        math.Round(totalRating*100) / 100,
			"rating_components": map[string]interface{}{
				"unt_contribution":          math.Round(untRank*100) / 100,
				"events_contribution":       math.Round(eventScore*100) / 100,
				"participants_contribution": math.Round(participantPoints*100) / 100,
				"reviews_contribution":      math.Round(averageRatingRank*100) / 100,
				"olympiad_contribution":     math.Round(olympiadRank*100) / 100,
			},
		}

		utils.ResponseJSON(w, response)
	}
}

// getUNTRank получает UNT рейтинг школы
func (src *SchoolRatingController) getUNTRank(db *sql.DB, schoolID int64) (float64, error) {
	query := `
        SELECT 
            exam_type,
            AVG(total_score) as average_score,
            COUNT(*) as student_count
        FROM 
            UNT_Exams 
        WHERE 
            school_id = ? 
            AND exam_type IN ('regular', 'creative')
        GROUP BY 
            exam_type`

	rows, err := db.Query(query, schoolID)
	if err != nil {
		return 0.0, err
	}
	defer rows.Close()

	var regularAverage float64
	var regularCount int
	var creativeAverage float64
	var creativeCount int

	for rows.Next() {
		var examType string
		var avgScore float64
		var studentCount int
		if err := rows.Scan(&examType, &avgScore, &studentCount); err != nil {
			continue
		}
		if examType == "regular" {
			regularAverage = avgScore
			regularCount = studentCount
		} else if examType == "creative" {
			creativeAverage = avgScore
			creativeCount = studentCount
		}
	}

	if regularCount == 0 && creativeCount == 0 {
		return 0.0, nil
	}

	// Нормализуем баллы к 100-балльной шкале
	normalizedRegular := 0.0
	if regularCount > 0 {
		normalizedRegular = (regularAverage / 140.0) * 100.0
	}

	normalizedCreative := 0.0
	if creativeCount > 0 {
		normalizedCreative = (creativeAverage / 120.0) * 100.0
	}

	// Вычисляем взвешенное среднее
	totalStudents := regularCount + creativeCount
	var combinedAverage float64
	if totalStudents > 0 {
		combinedAverage = (normalizedRegular*float64(regularCount) + normalizedCreative*float64(creativeCount)) / float64(totalStudents)
	}

	// Вычисляем UNT рейтинг
	untRank := (25.0 / 100.0) * combinedAverage
	return untRank, nil
}

// getEventScore получает счет событий школы
func (src *SchoolRatingController) getEventScore(db *sql.DB, schoolID int64) (float64, error) {
	// Получаем количество событий для школы
	var eventCount int
	err := db.QueryRow(`
        SELECT COUNT(e.id) as event_count
        FROM Schools s
        LEFT JOIN Events e ON s.school_id = e.school_id
        WHERE s.school_id = ?
    `, schoolID).Scan(&eventCount)
	if err != nil {
		return 0.0, err
	}

	// Получаем максимальное количество событий среди всех школ
	var maxEventCount int
	err = db.QueryRow(`
        SELECT COALESCE(MAX(event_count), 0)
        FROM (
            SELECT COUNT(id) as event_count
            FROM Events
            GROUP BY school_id
        ) as counts
    `).Scan(&maxEventCount)
	if err != nil {
		return 0.0, err
	}

	// Вычисляем счет
	var score float64
	if maxEventCount > 0 {
		score = (float64(eventCount) / float64(maxEventCount)) * 10
	}

	return score, nil
}

// getParticipantPoints получает очки участников школы
func (src *SchoolRatingController) getParticipantPoints(db *sql.DB, schoolID int64) (float64, error) {
	// Получаем общее количество участников по всем школам
	var maxParticipants int
	countQuery := `
        SELECT COUNT(r.event_registration_id)
        FROM Schools s
        LEFT JOIN EventRegistrations r ON s.school_id = r.school_id
        WHERE r.status IN ('registered', 'accepted', 'completed')`
	err := db.QueryRow(countQuery).Scan(&maxParticipants)
	if err != nil {
		return 0.0, err
	}

	// Получаем количество участников для конкретной школы
	var participantCount int
	query := `
        SELECT COUNT(r.event_registration_id) AS participant_count
        FROM Schools s
        LEFT JOIN EventRegistrations r ON s.school_id = r.school_id
        WHERE s.school_id = ? AND r.status IN ('registered', 'accepted', 'completed')`
	err = db.QueryRow(query, schoolID).Scan(&participantCount)
	if err != nil {
		return 0.0, err
	}

	// Вычисляем очки
	const maxPoints = 30.0
	var points float64
	if maxParticipants > 0 {
		points = (float64(participantCount) / float64(maxParticipants)) * maxPoints
	}

	return points, nil
}

// getAverageRatingRank получает рейтинг отзывов школы
func (src *SchoolRatingController) getAverageRatingRank(db *sql.DB, schoolID int64) (float64, error) {
	query := `SELECT AVG(rating) FROM Reviews WHERE school_id = ?`
	var averageRating float64
	err := db.QueryRow(query, schoolID).Scan(&averageRating)
	if err != nil {
		// Если нет отзывов, возвращаем 0
		if err == sql.ErrNoRows {
			return 0.0, nil
		}
		return 0.0, err
	}

	// Вычисляем average_rating_rank как средний рейтинг, умноженный на 2
	averageRatingRank := averageRating * 2
	return averageRatingRank, nil
}

// getOlympiadRank получает олимпиадный рейтинг школы
func (src *SchoolRatingController) getOlympiadRank(db *sql.DB, schoolID int64) (float64, error) {
	// Используем существующую функцию calculateOlympiadRatingByLevel
	cityRating := calculateOlympiadRatingByLevel(db, int(schoolID), "city", 0.2)
	regionRating := calculateOlympiadRatingByLevel(db, int(schoolID), "region", 0.3)
	republicanRating := calculateOlympiadRatingByLevel(db, int(schoolID), "republican", 0.5)

	// Общий олимпиадный рейтинг
	totalOlympiadRating := cityRating + regionRating + republicanRating

	// Олимпиадный ранк (25 * общий олимпиадный рейтинг)
	olympiadRank := 25.0 * totalOlympiadRating

	return olympiadRank, nil
}
