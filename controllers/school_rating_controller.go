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

type SchoolRatingController struct{}

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
		// Step 11.1: Получаем баллы за активность олимпиад
		olympiadActivityScore, err := src.getOlympiadActivityScore(db, schoolID)
		if err != nil {
			log.Printf("Ошибка при получении активности олимпиад: %v", err)
			olympiadActivityScore = 0.0
		}

		// Step 12: Вычисляем общий рейтинг
		totalRating := eventScore + participantPoints + averageRatingRank + untRank + olympiadRank + olympiadActivityScore

		// Step 13: Формируем ответ
		response := map[string]interface{}{
			"school_id":             schoolID,
			"school_name":           schoolName,
			"unt_rank":              math.Round(untRank*100) / 100,
			"event_rank":            math.Round(eventScore*100) / 100,
			"event_participants":    math.Round(participantPoints*100) / 100,
			"average_rating_rank":   math.Round(averageRatingRank*100) / 100,
			"olympiad_rank":         math.Round(olympiadRank*100) / 100,
			"olympiad_participants": math.Round(olympiadActivityScore*100) / 100,
			"total_rating":          math.Round(totalRating*100) / 100,
		}

		utils.ResponseJSON(w, response)
	}
}
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
	untRank := (30.0 / 100.0) * combinedAverage
	return untRank, nil
}

// func (src *SchoolRatingController) getEventScore(db *sql.DB, schoolID int64) (float64, error) {
// 	// Получаем количество событий для школы
// 	var eventCount int
// 	err := db.QueryRow(`
//         SELECT COUNT(e.id) as event_count
//         FROM Schools s
//         LEFT JOIN Events e ON s.school_id = e.school_id
//         WHERE s.school_id = ?
//     `, schoolID).Scan(&eventCount)
// 	if err != nil {
// 		return 0.0, err
// 	}

// 	// Получаем максимальное количество событий среди всех школ
// 	var maxEventCount int
// 	err = db.QueryRow(`
//         SELECT COALESCE(MAX(event_count), 0)
//         FROM (
//             SELECT COUNT(id) as event_count
//             FROM Events
//             GROUP BY school_id
//         ) as counts
//     `).Scan(&maxEventCount)
// 	if err != nil {
// 		return 0.0, err
// 	}

// 	// Вычисляем счет
// 	var score float64
// 	if maxEventCount > 0 {
// 		score = (float64(eventCount) / float64(maxEventCount)) * 10
// 	}

// 	return score, nil
// }

func (src *SchoolRatingController) getEventScore(db *sql.DB, schoolID int64) (float64, error) {
	// Считаем количество валидных мероприятий конкретной школы
	var schoolValidEventCount int
	querySchool := `
		SELECT COUNT(e.id)
		FROM Events e
		WHERE e.school_id = ?
		AND e.end_date < CURRENT_DATE
		AND (
			SELECT COUNT(*) FROM events_participants ep WHERE ep.events_name = e.event_name
		) >= 0.05 * (
			SELECT COUNT(*) FROM EventRegistrations r WHERE r.event_id = e.id
		)
	`
	err := db.QueryRow(querySchool, schoolID).Scan(&schoolValidEventCount)
	if err != nil {
		return 0.0, err
	}

	// Считаем максимальное количество таких мероприятий по всем школам
	var maxValidEventCount int
	queryMax := `
		SELECT MAX(valid_event_count) FROM (
			SELECT e.school_id, COUNT(e.id) AS valid_event_count
			FROM Events e
			WHERE e.end_date < CURRENT_DATE
			AND (
				SELECT COUNT(*) FROM events_participants ep WHERE ep.events_name = e.event_name
			) >= 0.05 * (
				SELECT COUNT(*) FROM EventRegistrations r WHERE r.event_id = e.id
			)
			GROUP BY e.school_id
		) AS sub
	`
	err = db.QueryRow(queryMax).Scan(&maxValidEventCount)
	if err != nil {
		return 0.0, err
	}

	// Вычисляем баллы
	var score float64
	if maxValidEventCount > 0 {
		score = (float64(schoolValidEventCount) / float64(maxValidEventCount)) * 10
	}

	return score, nil
}

// func (src *SchoolRatingController) getOlympiadActivityScore(db *sql.DB, schoolID int64) (float64, error) {
// 	// Кол-во валидных олимпиад конкретной школы
// 	var schoolValidOlympCount int
// 	querySchool := `
// 		SELECT COUNT(o.subject_olympiad_id)
// 		FROM subject_olympiads o
// 		WHERE o.school_id = ?
// 		AND o.end_date < CURRENT_DATE
// 		AND (
// 			SELECT COUNT(*)
// 			FROM olympiad_registrations r
// 			WHERE r.subject_olympiad_id = o.subject_olympiad_id
// 			AND r.status = 'completed'
// 		) >= 0.05 * (
// 			SELECT COUNT(*)
// 			FROM olympiad_registrations r
// 			WHERE r.subject_olympiad_id = o.subject_olympiad_id
// 		)
// 	`
// 	err := db.QueryRow(querySchool, schoolID).Scan(&schoolValidOlympCount)
// 	if err != nil {
// 		return 0.0, err
// 	}

// 	// Максимум валидных олимпиад по всем школам
// 	var maxValidOlympCount int
// 	queryMax := `
// 	SELECT COALESCE(MAX(valid_count), 0) FROM (
// 			SELECT o.school_id, COUNT(o.subject_olympiad_id) AS valid_count
// 			FROM subject_olympiads o
// 			WHERE o.end_date < CURRENT_DATE
// 			AND (
// 				SELECT COUNT(*)
// 				FROM olympiad_registrations r
// 				WHERE r.subject_olympiad_id = o.subject_olympiad_id
// 				AND r.status = 'completed'
// 			) >= 0.05 * (
// 				SELECT COUNT(*)
// 				FROM olympiad_registrations r
// 				WHERE r.subject_olympiad_id = o.subject_olympiad_id
// 			)
// 			GROUP BY o.school_id
// 		) AS sub
// 	`
// 	err = db.QueryRow(queryMax).Scan(&maxValidOlympCount)
// 	if err != nil {
// 		return 0.0, err
// 	}

// 	// Вычисляем итоговый балл
// 	var score float64
// 	if maxValidOlympCount > 0 {
// 		score = (float64(schoolValidOlympCount) / float64(maxValidOlympCount)) * 10
// 	}

// 	return score, nil
// }

func (src *SchoolRatingController) getOlympiadActivityScore(db *sql.DB, schoolID int64) (float64, error) {
	// ✅ 1. Считаем количество валидных олимпиад конкретной школы
	var schoolValidOlympCount int
	querySchool := `
		SELECT COUNT(o.subject_olympiad_id)
		FROM subject_olympiads o
		WHERE o.school_id = ?
		AND o.end_date < CURRENT_DATE
		AND (
			SELECT COUNT(*) 
			FROM olympiad_registrations r 
			WHERE r.subject_olympiad_id = o.subject_olympiad_id 
			AND r.status = 'completed'
		) >= 0.05 * (
			SELECT COUNT(*) 
			FROM olympiad_registrations r 
			WHERE r.subject_olympiad_id = o.subject_olympiad_id
		)
	`
	err := db.QueryRow(querySchool, schoolID).Scan(&schoolValidOlympCount)
	if err != nil {
		return 0.0, err
	}

	// ✅ 2. Считаем максимальное количество валидных олимпиад среди всех школ
	var maxValidOlympCount int
	queryMax := `
		SELECT MAX(valid_count) FROM (
			SELECT o.school_id, COUNT(o.subject_olympiad_id) AS valid_count
			FROM subject_olympiads o
			WHERE o.end_date < CURRENT_DATE
			AND (
				SELECT COUNT(*) 
				FROM olympiad_registrations r 
				WHERE r.subject_olympiad_id = o.subject_olympiad_id 
				AND r.status = 'completed'
			) >= 0.05 * (
				SELECT COUNT(*) 
				FROM olympiad_registrations r 
				WHERE r.subject_olympiad_id = o.subject_olympiad_id
			)
			GROUP BY o.school_id
		) AS sub
	`
	err = db.QueryRow(queryMax).Scan(&maxValidOlympCount)
	if err != nil {
		return 0.0, err
	}

	// ✅ 3. Вычисляем итоговый балл по формуле
	var score float64
	if maxValidOlympCount > 0 {
		score = (float64(schoolValidOlympCount) / float64(maxValidOlympCount)) * 20
	}

	return score, nil
}

func (src *SchoolRatingController) getParticipantPoints(db *sql.DB, schoolID int64) (float64, error) {
	// Получаем общее количество участников по всем школам
	var maxParticipants int
	countQuery := `
		SELECT COUNT(ep.id)
		FROM Schools s
		LEFT JOIN Events e ON s.school_id = e.school_id
		LEFT JOIN events_participants ep ON ep.events_name = e.event_name`
	err := db.QueryRow(countQuery).Scan(&maxParticipants)
	if err != nil {
		log.Println("Ошибка при подсчете всех участников:", err)
		return 0.0, err
	}

	// Получаем количество участников для конкретной школы
	var participantCount int
	query := `
		SELECT COUNT(ep.id) AS participant_count
		FROM Schools s
		LEFT JOIN Events e ON s.school_id = e.school_id
		LEFT JOIN events_participants ep ON ep.events_name = e.event_name
		WHERE s.school_id = ?`
	err = db.QueryRow(query, schoolID).Scan(&participantCount)
	if err != nil {
		log.Println("Ошибка при подсчете участников школы:", err)
		return 0.0, err
	}

	// Вычисляем очки
	const maxPoints = 20.0
	var points float64
	if maxParticipants > 0 {
		points = (float64(participantCount) / float64(maxParticipants)) * maxPoints
	}

	return points, nil
}

func (src *SchoolRatingController) getAverageRatingRank(db *sql.DB, schoolID int64) (float64, error) {
	query := `SELECT COALESCE(AVG(rating), 0) FROM Reviews WHERE school_id = ?`
	var averageRating float64
	err := db.QueryRow(query, schoolID).Scan(&averageRating)
	if err != nil {
		// Log the error and return 0.0 with the error for upstream handling
		return 0.0, err
	}

	// Calculate average_rating_rank as the average rating multiplied by 2
	averageRatingRank := averageRating * 2
	return averageRatingRank, nil
}
func (src *SchoolRatingController) getOlympiadRank(db *sql.DB, schoolID int64) (float64, error) {
	// Используем существующую функцию calculateOlympiadRatingByLevel
	cityRating := calculateOlympiadRatingByLevel(db, int(schoolID), "city", 0.2)
	regionRating := calculateOlympiadRatingByLevel(db, int(schoolID), "region", 0.3)
	republicanRating := calculateOlympiadRatingByLevel(db, int(schoolID), "republican", 0.5)

	// Общий олимпиадный рейтинг
	totalOlympiadRating := cityRating + regionRating + republicanRating

	// Олимпиадный ранк (25 * общий олимпиадный рейтинг)
	olympiadRank := 20.0 * totalOlympiadRating

	return olympiadRank, nil
}
