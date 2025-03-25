package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type OlympiadController struct{}

func (oc OlympiadController) CreateOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var olympiad models.Olympiad
		if err := json.NewDecoder(r.Body).Decode(&olympiad); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		query := `INSERT INTO olympiads (school_id, student_id, level, place, year) VALUES (?, ?, ?, ?, ?)`
		_, err := db.Exec(query, olympiad.SchoolID, olympiad.StudentID, olympiad.Level, olympiad.Place, olympiad.Year)
		if err != nil {
			log.Println("Insert Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create olympiad record"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "Olympiad record created"})
	}
}
func (oc OlympiadController) CalculateOlympiadPoints(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schoolID := r.URL.Query().Get("school_id")
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		query := `
			SELECT level, place
			FROM olympiads
			WHERE school_id = ?
		`

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch olympiad records"})
			return
		}
		defer rows.Close()

		var totalPoints float64
		var recordCount int

		for rows.Next() {
			var level string
			var place int
			if err := rows.Scan(&level, &place); err != nil {
				log.Println("Scan Error:", err)
				continue
			}

			var levelWeight float64
			switch level {
			case "city":
				levelWeight = 0.5
			case "region":
				levelWeight = 0.3
			case "republic":
				levelWeight = 0.2
			default:
				continue // если уровень неизвестный — пропускаем
			}

			var placePoints int
			switch place {
			case 1:
				placePoints = 3
			case 2:
				placePoints = 2
			case 3:
				placePoints = 1
			default:
				continue // места вне 1-3 не считаем
			}

			points := float64(placePoints) * levelWeight
			totalPoints += points
			recordCount++
		}

		if recordCount == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No valid olympiad records found"})
			return
		}

		// Максимум, который можно получить в этом разделе рейтинга — 15
		// Ограничим, чтобы не вышло за пределы
		if totalPoints > 15 {
			totalPoints = 15
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"school_id":       schoolID,
			"olympiad_points": totalPoints,
			"max_points":      15,
		})
	}
}
func (oc OlympiadController) CalculateOlympiadRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schoolID := r.URL.Query().Get("school_id")
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		query := `
			SELECT level, place
			FROM olympiads
			WHERE school_id = ?`

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get olympiad data"})
			return
		}
		defer rows.Close()

		var totalPoints, maxEarnedPoints float64
		for rows.Next() {
			var level string
			var place int

			if err := rows.Scan(&level, &place); err != nil {
				log.Println("Scan Error:", err)
				continue
			}

			var levelWeight float64
			switch level {
			case "city":
				levelWeight = 0.5
			case "region":
				levelWeight = 0.3
			case "republic":
				levelWeight = 0.2
			default:
				levelWeight = 0 // не засчитываем
			}

			var placePoints int
			switch place {
			case 1:
				placePoints = 3
			case 2:
				placePoints = 2
			case 3:
				placePoints = 1
			default:
				placePoints = 0
			}

			// Считаем реальный и возможный балл
			totalPoints += float64(placePoints) * levelWeight
			maxEarnedPoints += 3.0 * levelWeight
		}

		if maxEarnedPoints == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No olympiad results found for this school"})
			return
		}

		// Нормализация на 15-балльную шкалу
		normalized := (totalPoints / maxEarnedPoints) * 15
		if normalized > 15 {
			normalized = 15 // ограничим максимум
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"school_id":      schoolID,
			"olympiad_points": normalized,
			"max_points":     15,
		})
	}
}

