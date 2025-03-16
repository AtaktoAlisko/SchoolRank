package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"time"
)

type SchoolController struct{}


func (sc SchoolController) CreateSchool(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Проверяем токен и получаем userID
        userID, err := utils.VerifyToken(r)
        if err != nil {
            // Если токен неверный/просрочен — 401 Unauthorized
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // 2. Проверяем роль пользователя (director или superadmin)
        var userRole string
        err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
        if err != nil {
            log.Println("Error fetching user role:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
            return
        }
        if userRole != "director" && userRole != "superadmin" {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create a school"})
            return
        }

        // 3. Считываем файл из form-data (поле "photo")
        file, _, err := r.FormFile("photo")
        if err != nil {
            log.Println("Error reading file:", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
            return
        }
        defer file.Close()

        // 4. Генерируем уникальное имя файла, чтобы фото не перезаписывались
        // Например, используем userID и текущий Unix-временной штамп
        uniqueFileName := fmt.Sprintf("school-%d-%d.jpg", userID, time.Now().Unix())

        // 5. Загружаем файл в S3
        photoURL, err := utils.UploadFileToS3(file, uniqueFileName)
        if err != nil {
            log.Println("Error uploading file:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload file"})
            return
        }

        // 6. Считываем остальные поля из form-data (так как JSON Decode не подойдёт для multipart/form-data)
        var school models.School
        school.Name = r.FormValue("name")
        school.Address = r.FormValue("address")
        school.Title = r.FormValue("title")
        school.Description = r.FormValue("description")
        school.Contacts = r.FormValue("contacts")
        school.PhotoURL = photoURL // URL к файлу в S3

        // 7. Сохраняем данные школы в таблицу `schools`
        query := `
            INSERT INTO schools (name, address, title, description, contacts, photo_url)
            VALUES (?, ?, ?, ?, ?, ?)
        `
        result, err := db.Exec(query,
            school.Name,
            school.Address,
            school.Title,
            school.Description,
            school.Contacts,
            school.PhotoURL,
        )
        if err != nil {
            log.Println("SQL Insert Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create school"})
            return
        }

        // Получаем ID вставленной записи
        id, _ := result.LastInsertId()
        school.SchoolID = int(id)

        // 8. Возвращаем результат в JSON
        utils.ResponseJSON(w, school)
    }
}
func (sc SchoolController) GetSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT school_id, name, address, title, description, contacts, photo_url FROM schools")
		if err != nil {
			log.Println("SQL Select Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get schools"})
			return
		}
		defer rows.Close()

		var schools []models.School
		for rows.Next() {
			var school models.School
			if err := rows.Scan(&school.SchoolID, &school.Name, &school.Address, &school.Title, &school.Description, &school.Contacts, &school.PhotoURL); err != nil {
				log.Println("SQL Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse schools"})
				return
			}
			schools = append(schools, school)
		}

		utils.ResponseJSON(w, schools)
	}
}
func (sc SchoolController) GetSchoolForDirector(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Проверяем токен и получаем userID
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Получаем роль пользователя
        var userRole string
        err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
            return
        }

        // Проверяем, что пользователь имеет роль "director"
        if userRole != "director" {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view this school"})
            return
        }

        // Получаем информацию о школе, связанной с директором
        var school models.School
        err = db.QueryRow("SELECT school_id, name, address, title, description, contacts, photo_url FROM schools WHERE user_id = ?", userID).Scan(
            &school.SchoolID, &school.Name, &school.Address, &school.Title, &school.Description, &school.Contacts, &school.PhotoURL,
        )
        if err != nil {
            if err == sql.ErrNoRows {
                utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No school found for this director"})
            } else {
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school"})
            }
            return
        }

        // Возвращаем данные о школе
        utils.ResponseJSON(w, school)
    }
}
func (sc SchoolController) CalculateScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var score models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		totalScore := score.FirstSubjectScore + score.SecondSubjectScore + score.HistoryKazakhstan + score.MathematicalLiteracy + score.ReadingLiteracy
		score.TotalScore = totalScore

		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, first_subject_score, second_subject_score, history_of_kazakhstan, mathematical_literacy, reading_literacy, score) 
				VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, score.Year, score.UNTTypeID, score.StudentID, score.FirstSubjectScore, score.SecondSubjectScore, score.HistoryKazakhstan, score.MathematicalLiteracy, score.ReadingLiteracy, score.TotalScore)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate and save score"})
			return
		}

		utils.ResponseJSON(w, "Score calculated and saved successfully")
	}
}