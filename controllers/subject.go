package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type SubjectController struct{}

// Метод для создания предметов первого типа
func (sc SubjectController) CreateFirstSubject(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка токена и получение userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Проверка роли пользователя (только директор может создавать предметы)
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil || userRole != "director" || !userSchoolID.Valid {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create subject"})
			return
		}

		// Чтение данных из тела запроса (предмет)
		var subject models.FirstSubject
		if err := json.NewDecoder(r.Body).Decode(&subject); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Вставка нового предмета в базу данных с привязкой к школе директора
		query := `INSERT INTO First_Subject(subject, score, school_id) VALUES(?, ?, ?)`
		_, err = db.Exec(query, subject.Subject, subject.Score, userSchoolID.Int64)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create First Subject"})
			return
		}

		utils.ResponseJSON(w, "First Subject created successfully")
	}
}

// Метод для создания предметов второго типа
func (sc SubjectController) CreateSecondSubject(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка токена и получение userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Проверка роли пользователя (только директор может создавать предметы)
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil || userRole != "director" || !userSchoolID.Valid {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create second subject"})
			return
		}

		// Чтение данных из тела запроса (предмет)
		var subject models.SecondSubject
		if err := json.NewDecoder(r.Body).Decode(&subject); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Вставка нового предмета в базу данных с привязкой к школе директора
		query := `INSERT INTO Second_Subject(subject, score, school_id) VALUES(?, ?, ?)`
		_, err = db.Exec(query, subject.Subject, subject.Score, userSchoolID.Int64)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create second subject"})
			return
		}

		// Ответ с успешным сообщением
		utils.ResponseJSON(w, "Second Subject created successfully")
	}
}

// Метод для получения всех предметов первого типа
func (sc SubjectController) GetFirstSubjects(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT first_subject_id, subject, score FROM First_Subject")
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get First Subjects"})
			return
		}
		defer rows.Close()

		var subjects []models.FirstSubject
		for rows.Next() {
			var subject models.FirstSubject
			if err := rows.Scan(&subject.ID, &subject.Subject, &subject.Score); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Subjects"})
				return
			}
			subjects = append(subjects, subject)
		}

		utils.ResponseJSON(w, subjects)
	}
}

// Метод для получения всех предметов второго типа
func (sc SubjectController) GetSecondSubjects(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT second_subject_id, subject, score FROM Second_Subject")
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get Second Subjects"})
            return
        }
        defer rows.Close()

        var subjects []models.SecondSubject
        for rows.Next() {
            var subject models.SecondSubject
            if err := rows.Scan(&subject.ID, &subject.Subject, &subject.Score); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Subjects"})
                return
            }
            subjects = append(subjects, subject)
        }

        utils.ResponseJSON(w, subjects)
    }
}
