package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"

	"github.com/gorilla/mux"
)

type OlympiadController struct {
}

func (oc *OlympiadController) CreateOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверяем токен и получаем userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// 2. Получаем school_id из данных пользователя (директора)
		var directorSchoolID int
		err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", userID).Scan(&directorSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user school"})
			return
		}

		// 3. Декодируем тело запроса
		var olympiad models.Olympiads
		if err := json.NewDecoder(r.Body).Decode(&olympiad); err != nil {
			log.Printf("Error decoding request body: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// 4. Проверяем, существует ли студент с таким student_id
		var studentSchoolID int
		err = db.QueryRow("SELECT school_id FROM student WHERE student_id = ?", olympiad.StudentID).Scan(&studentSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking student school"})
			return
		}

		// 5. Проверяем, что студент принадлежит той же школе, что и директор
		if studentSchoolID != directorSchoolID {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Student does not belong to your school"})
			return
		}

		// 6. Присваиваем баллы в зависимости от места
		switch olympiad.OlympiadPlace {
		case 1:
			olympiad.Score = 50
		case 2:
			olympiad.Score = 30
		case 3:
			olympiad.Score = 20
		default:
			olympiad.Score = 0
		}

		// 7. Присваиваем уровень олимпиады
		var level string
		switch olympiad.Level {
		case "city", "region", "republican":
			level = olympiad.Level
		default:
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid level"})
			return
		}

		// 8. Вставляем запись с именем олимпиады
		query := `INSERT INTO Olympiads (student_id, olympiad_place, score, school_id, level, olympiad_name) 
                  VALUES (?, ?, ?, ?, ?, ?)`
		_, err = db.Exec(query, olympiad.StudentID, olympiad.OlympiadPlace, olympiad.Score, directorSchoolID, level, olympiad.OlympiadName)
		if err != nil {
			log.Printf("Error inserting Olympiad: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create Olympiad record"})
			return
		}

		utils.ResponseJSON(w, "Olympiad record created successfully")
	}
}
func (oc *OlympiadController) GetOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentID := r.URL.Query().Get("student_id")
		level := r.URL.Query().Get("level")

		var query string
		var rows *sql.Rows
		var err error

		if studentID != "" {
			query = `SELECT 
						Olympiads.olympiad_id, Olympiads.student_id, Olympiads.olympiad_place, Olympiads.score, 
						Olympiads.school_id, Olympiads.level, Olympiads.olympiad_name,
						student.first_name, student.last_name, student.patronymic, 
						student.grade, student.letter
					FROM Olympiads
					JOIN student ON Olympiads.student_id = student.student_id
					WHERE Olympiads.student_id = ?`
			rows, err = db.Query(query, studentID)
		} else if level != "" {
			query = `SELECT 
						Olympiads.olympiad_id, Olympiads.student_id, Olympiads.olympiad_place, Olympiads.score, 
						Olympiads.school_id, Olympiads.level, Olympiads.olympiad_name,
						student.first_name, student.last_name, student.patronymic,
						student.grade, student.letter
					FROM Olympiads
					JOIN student ON Olympiads.student_id = student.student_id
					WHERE Olympiads.level = ?`
			rows, err = db.Query(query, level)
		} else {
			query = `SELECT 
						Olympiads.olympiad_id, Olympiads.student_id, Olympiads.olympiad_place, Olympiads.score, 
						Olympiads.school_id, Olympiads.level, Olympiads.olympiad_name,
						student.first_name, student.last_name, student.patronymic,
						student.grade, student.letter
					FROM Olympiads
					JOIN student ON Olympiads.student_id = student.student_id`
			rows, err = db.Query(query)
		}

		if err != nil {
			log.Printf("Error fetching Olympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch Olympiad records"})
			return
		}
		defer rows.Close()

		var olympiads []models.Olympiads

		for rows.Next() {
			var olympiad models.Olympiads
			var olympiadName sql.NullString // Use sql.NullString to handle NULL values

			err := rows.Scan(
				&olympiad.OlympiadID,
				&olympiad.StudentID,
				&olympiad.OlympiadPlace,
				&olympiad.Score,
				&olympiad.SchoolID,
				&olympiad.Level,
				&olympiadName, // Scan into NullString
				&olympiad.FirstName,
				&olympiad.LastName,
				&olympiad.Patronymic,
				&olympiad.Grade,
				&olympiad.Letter,
			)
			if err != nil {
				log.Printf("Error scanning Olympiad record: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing Olympiad data"})
				return
			}

			// Convert NullString to string
			if olympiadName.Valid {
				olympiad.OlympiadName = olympiadName.String
			} else {
				olympiad.OlympiadName = "" // Empty string for NULL values
			}

			olympiads = append(olympiads, olympiad)
		}

		utils.ResponseJSON(w, olympiads)
	}
}

func (oc *OlympiadController) DeleteOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Извлекаем параметры запроса
		olympiadID := mux.Vars(r)["olympiad_id"] // Пример: ?olympiad_id=5

		if olympiadID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Olympiad ID is required"})
			return
		}

		// 2. Получаем userID из токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// 3. Получаем роль пользователя
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user role"})
			return
		}

		// 4. Проверка, если пользователь имеет роль "schooladmin" или "superadmin"
		if userRole != "schooladmin" && userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Insufficient permissions to delete Olympiad"})
			return
		}

		// 5. Если роль "schooladmin", получаем school_id из данных пользователя
		var directorSchoolID int
		if userRole == "schooladmin" {
			err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", userID).Scan(&directorSchoolID)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user school"})
				return
			}
		}

		// 6. Проверяем существование олимпиады с таким olympiad_id
		var olympiadSchoolID int
		err = db.QueryRow("SELECT school_id FROM Olympiads WHERE olympiad_id = ?", olympiadID).Scan(&olympiadSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking Olympiad existence"})
			return
		}

		// 7. Если роль "schooladmin", проверяем, что олимпиада принадлежит той же школе
		if userRole == "schooladmin" && olympiadSchoolID != directorSchoolID {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Olympiad does not belong to your school"})
			return
		}

		// 8. Удаляем олимпиаду
		query := `DELETE FROM Olympiads WHERE olympiad_id = ?`
		_, err = db.Exec(query, olympiadID)
		if err != nil {
			log.Printf("Error deleting Olympiad: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete Olympiad record"})
			return
		}

		// 9. Отправляем успешный ответ
		utils.ResponseJSON(w, map[string]string{"message": "Olympiad deleted successfully"})
	}
}
func (oc *OlympiadController) UpdateOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверяем токен и получаем userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// 2. Получаем school_id и role из данных пользователя
		var userSchoolID int
		var userRole string
		err = db.QueryRow("SELECT school_id, role FROM users WHERE id = ?", userID).Scan(&userSchoolID, &userRole)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user information"})
			return
		}

		// Проверяем, имеет ли пользователь достаточные права (director, schooladmin или superadmin)
		isAuthorized := userRole == "director" || userRole == "schooladmin" || userRole == "superadmin"
		if !isAuthorized {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Insufficient permissions"})
			return
		}

		// 3. Получаем olympiad_id из URL параметров
		vars := mux.Vars(r)
		olympiadID, err := strconv.Atoi(vars["olympiad_id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID"})
			return
		}

		// 4. Декодируем тело запроса
		var olympiad models.Olympiads
		if err := json.NewDecoder(r.Body).Decode(&olympiad); err != nil {
			log.Printf("Error decoding request body: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// 5. Проверяем, существует ли олимпиада с таким ID
		var existingOlympiadSchoolID int
		var studentID int
		// Используем olympiad_id вместо id как имя столбца
		err = db.QueryRow("SELECT school_id, student_id FROM Olympiads WHERE olympiad_id = ?", olympiadID).Scan(&existingOlympiadSchoolID, &studentID)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad record not found"})
			} else {
				log.Printf("Error checking olympiad: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking olympiad"})
			}
			return
		}

		// 6. Проверяем права доступа пользователя к этой записи
		// Если пользователь не superadmin, то он может редактировать только записи своей школы
		if userRole != "superadmin" && existingOlympiadSchoolID != userSchoolID {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You don't have permission to update this olympiad record"})
			return
		}

		// 7. Если в запросе указан student_id и он отличается от текущего, проверяем его принадлежность к школе
		if olympiad.StudentID != 0 && olympiad.StudentID != studentID {
			var studentSchoolID int
			err = db.QueryRow("SELECT school_id FROM student WHERE student_id = ?", olympiad.StudentID).Scan(&studentSchoolID)
			if err != nil {
				if err == sql.ErrNoRows {
					utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
				} else {
					log.Printf("Error checking student school: %v", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking student school"})
				}
				return
			}

			// Superadmin может менять на любого студента, остальные - только на студента из своей школы
			if userRole != "superadmin" && studentSchoolID != userSchoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Student does not belong to your school"})
				return
			}
		} else if olympiad.StudentID == 0 {
			// Если student_id не указан, используем существующий
			olympiad.StudentID = studentID
		}

		// 8. Присваиваем баллы в зависимости от места
		switch olympiad.OlympiadPlace {
		case 1:
			olympiad.Score = 50
		case 2:
			olympiad.Score = 30
		case 3:
			olympiad.Score = 20
		default:
			olympiad.Score = 0 // Для всех других мест 0 баллов
		}

		// 9. Проверяем и присваиваем уровень олимпиады
		var level string
		switch olympiad.Level {
		case "city":
			level = "city"
		case "region":
			level = "region"
		case "republican":
			level = "republican"
		default:
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid level"})
			return
		}

		// 10. Обновляем запись в таблице Olympiads
		// Если пользователь не superadmin, сохраняем школу студента как школу олимпиады
		var schoolIDToSave int
		if userRole == "superadmin" && olympiad.SchoolID != 0 {
			// Суперадмин может изменить school_id
			schoolIDToSave = olympiad.SchoolID
		} else {
			// Для всех других случаев сохраняем школу в зависимости от student_id
			err = db.QueryRow("SELECT school_id FROM student WHERE student_id = ?", olympiad.StudentID).Scan(&schoolIDToSave)
			if err != nil {
				log.Printf("Error getting student school: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get student school"})
				return
			}
		}

		// 11. Включаем olympiad_name в SQL запрос для обновления
		query := `UPDATE Olympiads 
				 SET student_id = ?, olympiad_place = ?, score = ?, level = ?, school_id = ?, olympiad_name = ?
				 WHERE olympiad_id = ?`
		_, err = db.Exec(query, olympiad.StudentID, olympiad.OlympiadPlace, olympiad.Score, level, schoolIDToSave,
			olympiad.OlympiadName, olympiadID)
		if err != nil {
			log.Printf("Error updating Olympiad: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update Olympiad record"})
			return
		}

		utils.ResponseJSON(w, "Olympiad record updated successfully")
	}
}
func (oc *OlympiadController) GetOlympiadBySchoolId(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем school_id из URL параметров
		vars := mux.Vars(r)
		schoolID, err := strconv.Atoi(vars["school_id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// 2. Проверяем существование школы
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", schoolID).Scan(&exists)
		if err != nil {
			log.Printf("Error checking school existence: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking school"})
			return
		}
		if !exists {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
			return
		}

		// 3. Получаем список олимпиад для указанной школы
		// Присоединяем данные о студентах, чтобы получить их имена и добавляем olympiad_name
		query := `
			SELECT o.olympiad_id, o.student_id, s.first_name, s.last_name, o.olympiad_place, 
			       o.score, o.level, o.school_id, o.olympiad_name
			FROM Olympiads o
			JOIN student s ON o.student_id = s.student_id
			WHERE o.school_id = ?
			ORDER BY o.olympiad_id DESC
		`
		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Printf("Error querying olympiads: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch olympiads"})
			return
		}
		defer rows.Close()

		// 4. Создаем расширенную структуру для возвращаемых данных с информацией о студенте
		type OlympiadWithStudent struct {
			OlympiadID    int    `json:"olympiad_id"`
			StudentID     int    `json:"student_id"`
			FirstName     string `json:"first_name"`
			LastName      string `json:"last_name"`
			OlympiadPlace int    `json:"olympiad_place"`
			Score         int    `json:"score"`
			Level         string `json:"level"`
			SchoolID      int    `json:"school_id"`
			OlympiadName  string `json:"olympiad_name"` // Добавляем поле olympiad_name
		}

		// 5. Считываем данные из результата запроса
		olympiads := []OlympiadWithStudent{}
		for rows.Next() {
			var olympiad OlympiadWithStudent
			var olympiadName sql.NullString // Используем NullString для обработки NULL значений

			err := rows.Scan(
				&olympiad.OlympiadID,
				&olympiad.StudentID,
				&olympiad.FirstName,
				&olympiad.LastName,
				&olympiad.OlympiadPlace,
				&olympiad.Score,
				&olympiad.Level,
				&olympiad.SchoolID,
				&olympiadName, // Сканируем в NullString
			)
			if err != nil {
				log.Printf("Error scanning olympiad row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing olympiad data"})
				return
			}

			// Преобразуем NullString в обычную строку
			if olympiadName.Valid {
				olympiad.OlympiadName = olympiadName.String
			} else {
				olympiad.OlympiadName = "" // Пустая строка для NULL значений
			}

			olympiads = append(olympiads, olympiad)
		}

		// 6. Проверяем ошибки при обработке результатов
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating olympiad rows: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing olympiad data"})
			return
		}

		// 7. Возвращаем результат
		if len(olympiads) == 0 {
			// Если олимпиад нет, возвращаем пустой массив, а не null
			utils.ResponseJSON(w, []OlympiadWithStudent{})
		} else {
			utils.ResponseJSON(w, olympiads)
		}
	}
}
func (oc *OlympiadController) CalculateCityOlympiadRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID := vars["school_id"]

		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Запрос для получения всех записей о городской олимпиаде для этой школы, только для level = 'city'
		query := `
            SELECT o.olympiad_place, o.student_id, o.score, o.school_id, o.level, s.first_name, s.last_name, s.patronymic
            FROM Olympiads o
            JOIN student s ON o.student_id = s.student_id
            WHERE o.school_id = ? AND o.level = 'city'
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
		var olympiads []models.Olympiads // Срез для хранения олимпиады

		// Пройдем по всем записям и вычислим баллы для 1, 2, и 3 мест
		for rows.Next() {
			var olympiad models.Olympiads
			err := rows.Scan(&olympiad.OlympiadPlace, &olympiad.StudentID, &olympiad.Score, &olympiad.SchoolID, &olympiad.Level, &olympiad.FirstName, &olympiad.LastName, &olympiad.Patronymic)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing Olympiad data"})
				return
			}

			olympiads = append(olympiads, olympiad) // Добавляем олимпиады в срез

			// Присваиваем баллы в зависимости от места
			if olympiad.OlympiadPlace == 1 {
				totalScore += 50
				prizeWinnersCount++
			} else if olympiad.OlympiadPlace == 2 {
				totalScore += 30
				prizeWinnersCount++
			} else if olympiad.OlympiadPlace == 3 {
				totalScore += 20
				prizeWinnersCount++
			}
		}

		// Проверяем, было ли хоть одно призовое место
		if prizeWinnersCount == 0 {
			utils.ResponseJSON(w, map[string]float64{"rating": 0})
			return
		}

		// Расчет среднего балла
		maxPossibleScore := prizeWinnersCount * 50
		averageScore := float64(totalScore) / float64(maxPossibleScore)

		// Расчет рейтинга, умножаем на коэффициент 0.2
		cityOlympiadRating := averageScore * 0.2

		// Возвращаем рейтинг и олимпиады
		utils.ResponseJSON(w, map[string]interface{}{
			"City olympiad rating": cityOlympiadRating,
			"olympiads":            olympiads,
		})
	}
}
func (oc *OlympiadController) CalculateRegionalOlympiadRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID := vars["school_id"]

		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Запрос для получения всех записей о региональной олимпиаде для этой школы, только для level = 'region'
		query := `
            SELECT o.olympiad_place, o.student_id, o.score, o.school_id, o.level, s.first_name, s.last_name, s.patronymic
            FROM Olympiads o
            JOIN student s ON o.student_id = s.student_id
            WHERE o.school_id = ? AND o.level = 'region'
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
		var olympiads []models.Olympiads // Срез для хранения олимпиады

		// Пройдем по всем записям и вычислим баллы для 1, 2, и 3 мест
		for rows.Next() {
			var olympiad models.Olympiads
			err := rows.Scan(&olympiad.OlympiadPlace, &olympiad.StudentID, &olympiad.Score, &olympiad.SchoolID, &olympiad.Level, &olympiad.FirstName, &olympiad.LastName, &olympiad.Patronymic)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing Olympiad data"})
				return
			}

			olympiads = append(olympiads, olympiad) // Добавляем олимпиады в срез

			// Присваиваем баллы в зависимости от места
			if olympiad.OlympiadPlace == 1 {
				totalScore += 50
				prizeWinnersCount++
			} else if olympiad.OlympiadPlace == 2 {
				totalScore += 30
				prizeWinnersCount++
			} else if olympiad.OlympiadPlace == 3 {
				totalScore += 20
				prizeWinnersCount++
			}
		}

		// Проверяем, было ли хоть одно призовое место
		if prizeWinnersCount == 0 {
			utils.ResponseJSON(w, map[string]float64{"rating": 0})
			return
		}

		// Расчет среднего балла
		maxPossibleScore := prizeWinnersCount * 50
		averageScore := float64(totalScore) / float64(maxPossibleScore)

		// Расчет рейтинга, умножаем на коэффициент 0.2
		regionalOlympiadRating := averageScore * 0.3

		// Возвращаем рейтинг и олимпиады
		utils.ResponseJSON(w, map[string]interface{}{
			"rating":                  regionalOlympiadRating,
			"Region olympiads rating": olympiads,
		})
	}
}
func (oc *OlympiadController) CalculateRepublicanOlympiadRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID := vars["school_id"]

		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Запрос для получения всех записей о республиканской олимпиаде для этой школы, только для level = 'republican'
		query := `
            SELECT o.olympiad_place, o.student_id, o.score, o.school_id, o.level, s.first_name, s.last_name, s.patronymic
            FROM Olympiads o
            JOIN student s ON o.student_id = s.student_id
            WHERE o.school_id = ? AND o.level = 'republican'
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
		var olympiads []models.Olympiads // Срез для хранения олимпиады

		// Пройдем по всем записям и вычислим баллы для 1, 2, и 3 мест
		for rows.Next() {
			var olympiad models.Olympiads
			err := rows.Scan(&olympiad.OlympiadPlace, &olympiad.StudentID, &olympiad.Score, &olympiad.SchoolID, &olympiad.Level, &olympiad.FirstName, &olympiad.LastName, &olympiad.Patronymic)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing Olympiad data"})
				return
			}

			olympiads = append(olympiads, olympiad) // Добавляем олимпиады в срез

			// Присваиваем баллы в зависимости от места
			if olympiad.OlympiadPlace == 1 {
				totalScore += 50
				prizeWinnersCount++
			} else if olympiad.OlympiadPlace == 2 {
				totalScore += 30
				prizeWinnersCount++
			} else if olympiad.OlympiadPlace == 3 {
				totalScore += 20
				prizeWinnersCount++
			}
		}

		// Проверяем, было ли хоть одно призовое место
		if prizeWinnersCount == 0 {
			utils.ResponseJSON(w, map[string]float64{"rating": 0})
			return
		}

		// Расчет среднего балла
		maxPossibleScore := prizeWinnersCount * 50
		averageScore := float64(totalScore) / float64(maxPossibleScore)

		// Расчет рейтинга, умножаем на коэффициент 0.2
		republicanOlympiadRating := averageScore * 0.5

		// Возвращаем рейтинг и олимпиады
		utils.ResponseJSON(w, map[string]interface{}{
			"Calculate republicans rating": republicanOlympiadRating,
			"olympiads":                    olympiads,
		})
	}
}
func (oc *OlympiadController) CalculateTotalOlympiadRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID := vars["school_id"]

		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
			return
		}

		// Запрос для получения всех записей об олимпиадах для этой школы, разделенных по уровням
		query := `
            SELECT o.olympiad_place, o.student_id, o.score, o.school_id, o.level, s.first_name, s.last_name, s.patronymic
            FROM Olympiads o
            JOIN student s ON o.student_id = s.student_id
            WHERE o.school_id = ?
        `

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Printf("Error fetching Olympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch Olympiad records"})
			return
		}
		defer rows.Close()

		// Структуры для хранения данных по каждому уровню
		type LevelData struct {
			TotalScore        int
			PrizeWinnersCount int
			Olympiads         []models.Olympiads
		}

		levelData := map[string]*LevelData{
			"city":       {TotalScore: 0, PrizeWinnersCount: 0, Olympiads: []models.Olympiads{}},
			"region":     {TotalScore: 0, PrizeWinnersCount: 0, Olympiads: []models.Olympiads{}},
			"republican": {TotalScore: 0, PrizeWinnersCount: 0, Olympiads: []models.Olympiads{}},
		}

		// Проходим по всем записям и разделяем их по уровням
		for rows.Next() {
			var olympiad models.Olympiads
			err := rows.Scan(&olympiad.OlympiadPlace, &olympiad.StudentID, &olympiad.Score, &olympiad.SchoolID, &olympiad.Level, &olympiad.FirstName, &olympiad.LastName, &olympiad.Patronymic)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing Olympiad data"})
				return
			}

			// Проверяем, что уровень является одним из ожидаемых
			if data, exists := levelData[olympiad.Level]; exists {
				data.Olympiads = append(data.Olympiads, olympiad)

				// Присваиваем баллы в зависимости от места
				if olympiad.OlympiadPlace == 1 {
					data.TotalScore += 50
					data.PrizeWinnersCount++
				} else if olympiad.OlympiadPlace == 2 {
					data.TotalScore += 30
					data.PrizeWinnersCount++
				} else if olympiad.OlympiadPlace == 3 {
					data.TotalScore += 20
					data.PrizeWinnersCount++
				}
			}
		}

		// Коэффициенты для каждого уровня
		coefficients := map[string]float64{
			"city":       0.2,
			"region":     0.3,
			"republican": 0.5,
		}

		// Рассчитываем рейтинг для каждого уровня
		ratings := map[string]float64{
			"city":       0.0,
			"region":     0.0,
			"republican": 0.0,
		}

		// Расчет рейтингов по уровням
		for level, data := range levelData {
			if data.PrizeWinnersCount > 0 {
				maxPossibleScore := data.PrizeWinnersCount * 50
				averageScore := float64(data.TotalScore) / float64(maxPossibleScore)
				ratings[level] = averageScore * coefficients[level]
			}
		}

		// Вычисляем общий рейтинг как сумму рейтингов по уровням
		totalRating := ratings["city"] + ratings["region"] + ratings["republican"]

		// Возвращаем только общий рейтинг
		utils.ResponseJSON(w, map[string]float64{"total_rating": totalRating})
	}
}
