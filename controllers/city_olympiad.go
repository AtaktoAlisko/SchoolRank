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

type CityOlympiadController struct{}

func (coc CityOlympiadController) CreateCityOlympiad(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Проверяем токен и получаем userID
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
            return
        }

        // 2. Получаем school_id из данных пользователя
        var schoolID int
        err = db.QueryRow("SELECT school_id FROM Users WHERE id = ?", userID).Scan(&schoolID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user school"})
            return
        }

        // 3. Декодируем тело запроса
        var cityOlympiad models.CityOlympiad
        if err := json.NewDecoder(r.Body).Decode(&cityOlympiad); err != nil {
            log.Printf("Error decoding request body: %v", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
            return
        }

        // 4. Присваиваем баллы в зависимости от места
        switch cityOlympiad.CityOlympiadPlace {
        case 1:
            cityOlympiad.Score = 50
        case 2:
            cityOlympiad.Score = 30
        case 3:
            cityOlympiad.Score = 20
        default:
            cityOlympiad.Score = 0 // Для всех других мест 0 баллов
        }

        // 5. Вставка новой записи с добавлением school_id
        query := `INSERT INTO City_Olympiad (student_id, city_olympiad_place, score, competition_date, school_id) 
                  VALUES (?, ?, ?, ?, ?)`
        _, err = db.Exec(query, cityOlympiad.StudentID, cityOlympiad.CityOlympiadPlace, cityOlympiad.Score, cityOlympiad.CompetitionDate, schoolID)
        if err != nil {
            log.Printf("Error inserting CityOlympiad: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create City Olympiad record"})
            return
        }

        utils.ResponseJSON(w, "City Olympiad record created successfully")
    }
}

func (coc CityOlympiadController) GetCityOlympiad(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var cityOlympiads []models.CityOlympiad

        rows, err := db.Query("SELECT id, student_id, city_olympiad_place, score, competition_date FROM City_Olympiad")
        if err != nil {
            log.Printf("Error fetching CityOlympiad records: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch City Olympiad records"})
            return
        }
        defer rows.Close()

        for rows.Next() {
            var cityOlympiad models.CityOlympiad
            var competitionDate []byte // Используем []byte для хранения значения

            err := rows.Scan(&cityOlympiad.ID, &cityOlympiad.StudentID, &cityOlympiad.CityOlympiadPlace, &cityOlympiad.Score, &competitionDate)
            if err != nil {
                log.Printf("Error scanning CityOlympiad: %v", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse City Olympiad records"})
                return
            }

            // Преобразуем байты в строку и затем в time.Time
            if len(competitionDate) > 0 {
                parsedDate, err := time.Parse("2006-01-02 15:04:05", string(competitionDate))
                if err != nil {
                    log.Printf("Error parsing competition_date: %v", err)
                    utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Invalid competition_date format"})
                    return
                }
                cityOlympiad.CompetitionDate = parsedDate
            } else {
                cityOlympiad.CompetitionDate = time.Time{} // Если пустое значение, присваиваем zero time
            }

            cityOlympiads = append(cityOlympiads, cityOlympiad)
        }

        utils.ResponseJSON(w, cityOlympiads)
    }
}

func (coc CityOlympiadController) GetAverageCityOlympiadScore(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Получаем ID школы (например, из query параметров)
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
            return
        }

        // Запрос для получения всех записей о городской олимпиаде для этой школы
        query := `
            SELECT city_olympiad_place 
            FROM City_Olympiad
            WHERE school_id = ?
        `

        rows, err := db.Query(query, schoolID)
        if err != nil {
            log.Printf("Error fetching City Olympiad records: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch City Olympiad records"})
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
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse City Olympiad records"})
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

        // Рассчитываем average_city_olympiad_score по формуле
        averageScore := float64(totalScore) / (50 * float64(prizeWinnersCount))

        // Вычисляем city_olympiad_rating (например, по 20 баллов за олимпиады)
        cityOlympiadRating := averageScore * 0.2

        // Возвращаем результат
        utils.ResponseJSON(w, map[string]interface{}{
            "average_city_olympiad_score": averageScore,
            "total_score":                 totalScore,
            "prize_winners_count":         prizeWinnersCount,
            "city_olympiad_rating":        cityOlympiadRating,
        })
    }
}

func (coc CityOlympiadController) DeleteCityOlympiad(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Получаем ID пользователя (директора) из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
            return
        }

        // Получаем ID олимпиады из параметров запроса
        olympiadID := r.URL.Query().Get("id")
        if olympiadID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Olympiad ID is required"})
            return
        }

        // Получаем school_id пользователя (директора)
        var directorSchoolID int
        err = db.QueryRow("SELECT school_id FROM Users WHERE id = ?", userID).Scan(&directorSchoolID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user school"})
            return
        }

        // Проверяем, есть ли такая олимпиада и принадлежит ли она данной школе
        var olympiadSchoolID int
        err = db.QueryRow("SELECT school_id FROM City_Olympiad WHERE id = ?", olympiadID).Scan(&olympiadSchoolID)
        if err != nil {
            if err == sql.ErrNoRows {
                utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad not found"})
                return
            }
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking olympiad school"})
            return
        }

        // Если школа директора не совпадает с school_id олимпиады, запрещаем удаление
        if olympiadSchoolID != directorSchoolID {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You are not authorized to delete this olympiad"})
            return
        }

        // Удаляем запись о городской олимпиаде
        _, err = db.Exec("DELETE FROM City_Olympiad WHERE id = ?", olympiadID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete City Olympiad"})
            return
        }

        // Возвращаем успешный ответ
        utils.ResponseJSON(w, "City Olympiad record deleted successfully")
    }
}

