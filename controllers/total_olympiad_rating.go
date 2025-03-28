package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type TotalOlympiadRatingController struct{}

// Функция для расчета рейтинга для городской олимпиады
func calculateCityOlympiadRating(db *sql.DB, schoolID string) float64 {
	// Запрос для получения всех записей о городской олимпиаде для этой школы
	query := `
		SELECT city_olympiad_place 
		FROM City_Olympiad
		WHERE school_id = ?
	`

	rows, err := db.Query(query, schoolID)
	if err != nil {
		log.Printf("Error fetching City Olympiad records: %v", err)
		return 0
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
			return 0
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
		return 0
	}

	// Рассчитываем average_city_olympiad_score
	averageScore := float64(totalScore) / (50 * float64(prizeWinnersCount))

	// Рассчитываем city_olympiad_rating (например, по 20 баллов за олимпиады)
	cityOlympiadRating := averageScore * 0.2

	return cityOlympiadRating
}

// Функция для расчета рейтинга для областной олимпиады
func calculateRegionalOlympiadRating(db *sql.DB, schoolID string) float64 {
	// Подобный запрос для областной олимпиады
	query := `
		SELECT regional_olympiad_place 
		FROM Regional_Olympiad
		WHERE school_id = ?
	`

	rows, err := db.Query(query, schoolID)
	if err != nil {
		log.Printf("Error fetching Regional Olympiad records: %v", err)
		return 0
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
			return 0
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
		return 0
	}

	// Рассчитываем average_region_olympiad_score
	averageScore := float64(totalScore) / (50 * float64(prizeWinnersCount))

	// Рассчитываем region_olympiad_rating (например, по 30 баллов за олимпиады)
	regionOlympiadRating := averageScore * 0.3

	return regionOlympiadRating
}

// Функция для расчета рейтинга для республиканской олимпиады
func calculateRepublicanOlympiadRating(db *sql.DB, schoolID string) float64 {
	query := `
		SELECT republican_olympiad_place 
		FROM RepublicanOlympiad
		WHERE school_id = ?
	`

	rows, err := db.Query(query, schoolID)
	if err != nil {
		log.Printf("Error fetching Republican Olympiad records: %v", err)
		return 0
	}
	defer rows.Close()

	var totalScore int
	var prizeWinnersCount int

	// Loop through all records and calculate points for 1st, 2nd, and 3rd places
	for rows.Next() {
		var place int
		err := rows.Scan(&place)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			return 0
		}

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

	// Check if there are any prize-winning students
	if prizeWinnersCount == 0 {
		log.Println("No prize-winning students found")
		return 0
	}

	// Calculate the average score
	averageScore := float64(totalScore) / (50 * float64(prizeWinnersCount))

	// Calculate the rating for Republican Olympiad
	republicanOlympiadRating := averageScore * 0.5

	// Save the rating to the database
	saveRatingQuery := `
		UPDATE Schools 
		SET republican_olympiad_rating = ? 
		WHERE school_id = ?
	`
	_, err = db.Exec(saveRatingQuery, republicanOlympiadRating, schoolID)
	if err != nil {
		log.Printf("Error saving republican olympiad rating: %v", err)
		return 0
	}

	// Return the calculated rating
	return republicanOlympiadRating
}

// Функция для суммирования всех рейтингов
func (toc *TotalOlympiadRatingController) GetTotalOlympiadRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем school_id из параметров
		schoolID := r.URL.Query().Get("school_id")
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Получаем рейтинги для каждой олимпиады
		cityOlympiadRating := calculateCityOlympiadRating(db, schoolID)
		regionOlympiadRating := calculateRegionalOlympiadRating(db, schoolID)
		republicanOlympiadRating := calculateRepublicanOlympiadRating(db, schoolID)

		// Суммируем рейтинги
		totalOlympiadRating := cityOlympiadRating + regionOlympiadRating + republicanOlympiadRating

		// Возвращаем результат
		utils.ResponseJSON(w, map[string]interface{}{
			"city_olympiad_rating":    cityOlympiadRating,
			"region_olympiad_rating":  regionOlympiadRating,
			"republican_olympiad_rating": republicanOlympiadRating,
			"total_olympiad_rating":   totalOlympiadRating,
		})
	}
}
