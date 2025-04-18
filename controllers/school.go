package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"time"
)

type SchoolController struct{}


func (sc SchoolController) CreateSchool(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Проверяем токен и получаем userID
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // 2. Проверяем роль пользователя
        var userRole string
        err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
        if err != nil {
            log.Println("Error fetching user role:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
            return
        }
        if userRole != "schooladmin" && userRole != "superadmin" {
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

        // 4. Генерируем уникальное имя файла
        uniqueFileName := fmt.Sprintf("school-%d-%d.jpg", userID, time.Now().Unix())

        // 5. Загружаем файл в S3
        photoURL, err := utils.UploadFileToS3(file, uniqueFileName, false)
        if err != nil {
            log.Println("Error uploading file:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload file"})
            return
        }

        // 6. Считываем остальные поля из form-data
        var school models.School
        school.Name = r.FormValue("name")
        school.Address = r.FormValue("address")
        school.Title = r.FormValue("title")
        school.Description = r.FormValue("description")
        school.Contacts = r.FormValue("contacts")
        school.PhotoURL = photoURL
        
        // Преобразуем email и phone в sql.NullString
        email := r.FormValue("email")
        phone := r.FormValue("phone")
        
        if email == "" {
            school.Email = sql.NullString{String: "", Valid: false} // NULL значение
        } else {
            school.Email = sql.NullString{String: email, Valid: true} // Валидный email
        }

        if phone == "" {
            school.Phone = sql.NullString{String: "", Valid: false} // NULL значение
        } else {
            school.Phone = sql.NullString{String: phone, Valid: true} // Валидный phone
        }

        // 7. Сохраняем данные школы в таблицу
        query := `
            INSERT INTO Schools (name, address, title, description, contacts, photo_url, email, phone, user_id)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `
        result, err := db.Exec(query,
            school.Name,
            school.Address,
            school.Title,
            school.Description,
            school.Contacts,
            school.PhotoURL,
            school.Email,
            school.Phone,
            userID,
        )
        if err != nil {
            log.Println("SQL Insert Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create school"})
            return
        }

        // Получаем ID вставленной записи
        id, _ := result.LastInsertId()
        school.SchoolID = int(id)

        // 8. Обновляем поле school_id в таблице users для текущего пользователя
        updateQuery := `
            UPDATE users
            SET school_id = ?
            WHERE id = ?
        `
        _, err = db.Exec(updateQuery, school.SchoolID, userID)
        if err != nil {
            log.Println("SQL Update Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update user with school_id"})
            return
        }

        // 9. Возвращаем результат в JSON
        utils.ResponseJSON(w, school)
    }
}
func (sc SchoolController) GetSchools(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT school_id, name, address, title, description, contacts, photo_url, email, phone FROM Schools")
        if err != nil {
            log.Println("SQL Select Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get schools"})
            return
        }
        defer rows.Close()

        var schools []models.School
        for rows.Next() {
            var school models.School
            if err := rows.Scan(&school.SchoolID, &school.Name, &school.Address, &school.Title, &school.Description, &school.Contacts, &school.PhotoURL, &school.Email, &school.Phone); err != nil {
                log.Println("SQL Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse schools"})
                return
            }

            // Преобразуем sql.NullString в строку (если значение NULL, то пустая строка)
            if school.Email.Valid {
                school.Email.String = school.Email.String
            } else {
                school.Email.String = ""
            }

            if school.Phone.Valid {
                school.Phone.String = school.Phone.String
            } else {
                school.Phone.String = ""
            }

            schools = append(schools, school)
        }

        utils.ResponseJSON(w, schools)
    }
}
// GetSchoolForDirector - получение школы для директора
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

		// Логируем userID
		log.Printf("Fetching school for user ID: %d", userID)

		// Получаем информацию о школе для директора
		var school models.School
		err = db.QueryRow(`
			SELECT s.school_id, s.name, s.address, s.title, s.description, s.contacts, s.photo_url
			FROM schools s
			INNER JOIN users u ON u.school_id = s.school_id
			WHERE u.id = ?`, userID).Scan(
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
// CalculateScore - расчет и сохранение результата UNT
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
// Deleting school only if it is not linked to a user
func (sc SchoolController) DeleteSchool(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Получаем school_id из URL параметра
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
            return
        }

        // 2. Преобразуем school_id в целое число
        id, err := strconv.Atoi(schoolID)
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
            return
        }

        // 3. Проверяем, есть ли пользователи, привязанные к этой школе
        var userCount int
        err = db.QueryRow("SELECT COUNT(*) FROM users WHERE school_id = ?", id).Scan(&userCount)
        if err != nil {
            log.Println("Error checking users:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to check users"})
            return
        }

        // 4. Если есть пользователи, не разрешаем удаление
        if userCount > 0 {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "This school is assigned to a user and cannot be deleted"})
            return
        }

        // 5. Теперь удаляем школу
        query := "DELETE FROM Schools WHERE school_id = ?"
        result, err := db.Exec(query, id)
        if err != nil {
            log.Println("Error deleting school:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete school"})
            return
        }

        // 6. Проверяем, сколько строк было затронуто
        rowsAffected, err := result.RowsAffected()
        if err != nil || rowsAffected == 0 {
            utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
            return
        }

        // 7. Успешное удаление
        utils.ResponseJSON(w, map[string]string{"message": "School deleted successfully"})
    }
}

func (sc SchoolController) CalculateAverageRating(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Get the school ID from query parameters
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
            return
        }

        // 2. Calculate the average rating from various sources

        // Get the total score from UNT scores for this school
        var totalUNTScore int
        var totalStudents int
        rows, err := db.Query(`
            SELECT COUNT(*) AS student_count, SUM(total_score) AS total_score
            FROM UNT_Score us
            JOIN Student s ON us.student_id = s.student_id
            WHERE s.school_id = ?
        `, schoolID)

        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate UNT score"})
            return
        }
        defer rows.Close()

        if rows.Next() {
            err = rows.Scan(&totalStudents, &totalUNTScore)
            if err != nil {
                log.Println("Error scanning UNT data:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning UNT data"})
                return
            }
        }

        // Calculate average UNT score
        var averageUNTScore float64
        if totalStudents > 0 {
            averageUNTScore = float64(totalUNTScore) / float64(totalStudents)
        }

        // 3. Get reviews score (e.g., average of reviews from a `reviews` table)
        var averageReviewScore float64
        err = db.QueryRow(`
            SELECT AVG(review_score) 
            FROM reviews 
            WHERE school_id = ?
        `, schoolID).Scan(&averageReviewScore)
        if err != nil {
            log.Println("SQL Error fetching reviews:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get reviews score"})
            return
        }

        // 4. Get olympiad participation score
        var olympiadScore int
        err = db.QueryRow(`
            SELECT SUM(olympiad_points) 
            FROM olympiad_results
            WHERE school_id = ?
        `, schoolID).Scan(&olympiadScore)
        if err != nil {
            log.Println("SQL Error fetching olympiad results:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get olympiad results"})
            return
        }

        // 5. Calculate the final average rating based on all criteria
        finalAverageRating := (averageUNTScore * 0.4) + (averageReviewScore * 0.3) + (float64(olympiadScore) * 0.3)

        // Return the result as JSON
        utils.ResponseJSON(w, map[string]interface{}{
            "school_id":            schoolID,
            "average_rating":       finalAverageRating,
            "average_unt_score":    averageUNTScore,
            "average_review_score": averageReviewScore,
            "olympiad_score":       olympiadScore,
        })
    }
}












