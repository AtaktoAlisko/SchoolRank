package controllers

import (
	"database/sql"
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

        if userRole != "superadmin" {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create a school"})
            return
        }

        // 3. Считываем данные из form-data
        var school models.School
        school.SchoolName = r.FormValue("school_name")  // Название школы
        school.City = r.FormValue("city")               // Город
        school.SchoolAdminLogin = r.FormValue("school_admin_login") // Логин школьного администратора

        // Убедимся, что логин школьного администратора передан
        if school.SchoolAdminLogin == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "SchoolAdminLogin is required"})
            return
        }

        // 4. Загружаем фото школы (не обязательное поле)
        file, _, err := r.FormFile("photo")
        if err == nil { // Если фото загружено
            defer file.Close()
            uniqueFileName := fmt.Sprintf("school-%d-%d.jpg", userID, time.Now().Unix())
            photoURL, err := utils.UploadFileToS3(file, uniqueFileName, false)
            if err != nil {
                log.Println("Error uploading file:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload file"})
                return
            }
            school.PhotoURL = photoURL  // Используем обновленное поле PhotoURL
        }

        // 5. Сохраняем данные о школе в таблицу
        query := `
            INSERT INTO Schools (name, city, school_admin_login, photo_url, created_at, updated_at)
            VALUES (?, ?, ?, ?, NOW(), NOW())
        `
        result, err := db.Exec(query,
            school.SchoolName,
            school.City,
            school.SchoolAdminLogin, // Сохраняем почту (логин) школьного администратора
            school.PhotoURL,         // Сохраняем URL фотографии
        )
        if err != nil {
            log.Println("SQL Insert Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create school"})
            return
        }

        // Получаем ID вставленной записи
        id, _ := result.LastInsertId()
        school.SchoolID = int(id)

        // 6. Обновляем поле school_id в таблице users для текущего пользователя
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

        // 7. Возвращаем результат в JSON
        utils.ResponseJSON(w, school)
    }
}


// func (sc SchoolController) GetSchool(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         // ✅ Получаем school_id из path-параметра
//         vars := mux.Vars(r)
//         schoolID := vars["id"]
//         if schoolID == "" {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
//             return
//         }

//         // 📥 Получаем данные из базы
//         var school models.School
//         query := `
//             SELECT school_id, name, address, city, title, description, photo_url, email, phone, director_email,
//                    DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s'),
//                    DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s')
//             FROM Schools
//             WHERE school_id = ?
//         `
//         err := db.QueryRow(query, schoolID).Scan(
//             &school.SchoolID,
//             &school.Name,
//             &school.Address,
//             &school.City,
//             &school.Title,
//             &school.Description,
//             &school.PhotoURL,
//             &school.Email,
//             &school.Phone,
//             &school.DirectorEmail,
//             &school.CreatedAt,
//             &school.UpdatedAt,
//         )

//         if err != nil {
//             if err == sql.ErrNoRows {
//                 utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
//             } else {
//                 log.Println("SQL Select Error:", err)
//                 utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve school"})
//             }
//             return
//         }

//         // ✅ Успешный ответ
//         utils.ResponseJSON(w, school)
//     }
// }
// func (sc SchoolController) UpdateSchool(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         // 1. Проверяем токен
//         userID, err := utils.VerifyToken(r)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
//             return
//         }

//         // 2. Проверка роли
//         var userRole string
//         err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
//             return
//         }

//         if userRole != "superadmin" {
//             utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can update schools"})
//             return
//         }

//         // ✅ 3. Получаем school_id из path-параметра
//         vars := mux.Vars(r)
//         schoolID := vars["id"]
//         if schoolID == "" {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
//             return
//         }

//         // 4. Декодируем JSON из тела запроса
//         var school models.School
//         err = json.NewDecoder(r.Body).Decode(&school)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid JSON data"})
//             return
//         }

//         // 5. Обновляем школу
//         query := `
//             UPDATE Schools
//             SET 
//                 name = ?, city = ?, title = ?, description = ?, 
//                 address = ?, email = ?, phone = ?, director_email = ?, 
//                 photo_url = ?, updated_at = NOW()
//             WHERE school_id = ?
//         `
//         _, err = db.Exec(query,
//             school.Name,
//             school.City,
//             school.Title,
//             school.Description,
//             school.Address,
//             school.Email,
//             school.Phone,
//             school.DirectorEmail,
//             school.PhotoURL,
//             schoolID,
//         )
//         if err != nil {
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school"})
//             return
//         }

//         utils.ResponseJSON(w, map[string]interface{}{
//             "message":    "School updated successfully",
//             "school_id":  schoolID,
//             "updated_by": userID,
//         })
//     }
// }
// func (sc SchoolController) DeleteSchool(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         // 1. Проверка токена и роли
//         userID, err := utils.VerifyToken(r)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
//             return
//         }

//         var userRole string
//         err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
//         if err != nil {
//             log.Println("Error fetching user role:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
//             return
//         }

//         if userRole != "superadmin" {
//             utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can delete a school"})
//             return
//         }

//         // ✅ 2. Получаем school_id из path-параметра
//         vars := mux.Vars(r)
//         schoolID := vars["id"]
//         if schoolID == "" {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
//             return
//         }

//         // 3. Удаляем школу
//         result, err := db.Exec("DELETE FROM Schools WHERE school_id = ?", schoolID)
//         if err != nil {
//             log.Println("SQL Delete Error:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete school"})
//             return
//         }

//         // 4. Проверка затронутых строк
//         rowsAffected, _ := result.RowsAffected()
//         if rowsAffected == 0 {
//             utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
//             return
//         }

//         // 5. Успешный ответ
//         utils.ResponseJSON(w, map[string]interface{}{
//             "message":    "School deleted successfully",
//             "school_id":  schoolID,
//             "deleted_by": userID,
//         })
//     }
// }
// func (sc SchoolController) UpdateMySchool(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         // 1. Проверка токена
//         userID, err := utils.VerifyToken(r)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
//             return
//         }

//         // 2. Получение роли и email пользователя
//         var role, userEmail string
//         err = db.QueryRow("SELECT role, email FROM users WHERE id = ?", userID).Scan(&role, &userEmail)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get user info"})
//             return
//         }

//         // 3. Разрешить только schooladmin
//         if role != "schooladmin" {
//             utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only schooladmin can update school"})
//             return
//         }

//         // 4. Найти school_id, где director_email совпадает с email пользователя
//         var schoolID int
//         err = db.QueryRow("SELECT school_id FROM Schools WHERE director_email = ?", userEmail).Scan(&schoolID)
//         if err != nil || schoolID == 0 {
//             utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No school assigned to your email"})
//             return
//         }

//         // 5. Прочитать JSON из запроса
//         var input struct {
//             Address     string `json:"address"`
//             Title       string `json:"title"`
//             Description string `json:"description"`
//             Email       string `json:"email"`
//             Phone       string `json:"phone"`
//             PhotoURL    string `json:"photo_url"`
//         }

//         err = json.NewDecoder(r.Body).Decode(&input)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid JSON format"})
//             return
//         }

//         // 6. Обновить школу
//         query := `
//             UPDATE Schools
//             SET address = ?, title = ?, description = ?, email = ?, phone = ?, photo_url = ?, updated_at = NOW()
//             WHERE school_id = ?
//         `
//         _, err = db.Exec(query,
//             input.Address,
//             input.Title,
//             input.Description,
//             input.Email,
//             input.Phone,
//             input.PhotoURL,
//             schoolID,
//         )
//         if err != nil {
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school"})
//             return
//         }

//         // 7. Успешный ответ
//         utils.ResponseJSON(w, map[string]interface{}{
//             "message":    "School updated successfully",
//             "school_id":  schoolID,
//             "updated_by": userEmail,
//         })
//     }
// }











// func (sc SchoolController) GetSchools(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         rows, err := db.Query("SELECT school_id, name, address, title, description, photo_url, email, phone FROM Schools")
//         if err != nil {
//             log.Println("SQL Select Error:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get schools"})
//             return
//         }
//         defer rows.Close()

//         var schools []models.School
//         for rows.Next() {
//             var school models.School
//             if err := rows.Scan(&school.SchoolID, &school.Name, &school.Address, &school.Title, &school.Description, &school.PhotoURL, &school.Email, &school.Phone); err != nil {
//                 log.Println("SQL Scan Error:", err)
//                 utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse schools"})
//                 return
//             }

//             // Преобразуем sql.NullString в строку (если значение NULL, то пустая строка)
//             if !school.Email.Valid {
//                 school.Email.String = "" // Если значение NULL, то пустая строка
//             }
//             if !school.Phone.Valid {
//                 school.Phone.String = "" // Если значение NULL, то пустая строка
//             }

//             schools = append(schools, school)
//         }

//         // Преобразуем sql.NullString в обычные строки для вывода в JSON
//         var response []map[string]interface{}
//         for _, school := range schools {
//             schoolData := map[string]interface{}{
//                 "school_id":   school.SchoolID,
//                 "name":        school.Name,
//                 "address":     school.Address,
//                 "title":       school.Title,
//                 "description": school.Description,
//                 "photo_url":   school.PhotoURL,
//                 "email":       school.Email.String,  // Просто строковое значение
//                 "phone":       school.Phone.String,  // Просто строковое значение
//             }
//             response = append(response, schoolData)
//         }

//         utils.ResponseJSON(w, response)
//     }
// }
// func (sc SchoolController) GetSchoolForDirector(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         // Проверяем токен и получаем userID
//         userID, err := utils.VerifyToken(r)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
//             return
//         }

//         // Получаем роль пользователя
//         var userRole string
//         err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
//             return
//         }

//         // Проверяем, что пользователь имеет роль "director"
//         if userRole != "director" {
//             utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view this school"})
//             return
//         }

//         // Логируем userID
//         log.Printf("Fetching school for user ID: %d", userID)

//         // Получаем информацию о школе для директора
//         var school models.School
//         err = db.QueryRow(`
//             SELECT s.school_id, s.name, s.address, s.title, s.description, s.photo_url, s.email, s.phone
//             FROM schools s
//             INNER JOIN users u ON u.school_id = s.school_id
//             WHERE u.id = ?`, userID).Scan(
//             &school.SchoolID, &school.Name, &school.Address, &school.Title, &school.Description, &school.PhotoURL, &school.Email, &school.Phone,
//         )
//         if err != nil {
//             if err == sql.ErrNoRows {
//                 utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No school found for this director"})
//             } else {
//                 utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school"})
//             }
//             return
//         }

//         // Убираем поле Contacts из ответа
//         // school.Contacts = "" // Убираем поле contacts, так как оно больше не используется

//         // Возвращаем данные о школе без поля Contacts
//         utils.ResponseJSON(w, school)
//     }
// }
// func (sc SchoolController) CalculateScore(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		var score models.UNTScore
// 		if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
// 			return
// 		}

// 		totalScore := score.FirstSubjectScore + score.SecondSubjectScore + score.HistoryKazakhstan + score.MathematicalLiteracy + score.ReadingLiteracy
// 		score.TotalScore = totalScore

// 		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, first_subject_score, second_subject_score, history_of_kazakhstan, mathematical_literacy, reading_literacy, score) 
// 				VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`
// 		_, err := db.Exec(query, score.Year, score.UNTTypeID, score.StudentID, score.FirstSubjectScore, score.SecondSubjectScore, score.HistoryKazakhstan, score.MathematicalLiteracy, score.ReadingLiteracy, score.TotalScore)
// 		if err != nil {
// 			log.Println("SQL Error:", err)
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate and save score"})
// 			return
// 		}

// 		utils.ResponseJSON(w, "Score calculated and saved successfully")
// 	}
// }
// func (sc SchoolController) DeleteSchool(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         // 1. Получаем school_id из URL параметра
//         schoolID := r.URL.Query().Get("school_id")
//         if schoolID == "" {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
//             return
//         }

//         // 2. Преобразуем school_id в целое число
//         id, err := strconv.Atoi(schoolID)
//         if err != nil {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID format"})
//             return
//         }

//         // 3. Проверяем, есть ли пользователи, привязанные к этой школе
//         var userCount int
//         err = db.QueryRow("SELECT COUNT(*) FROM users WHERE school_id = ?", id).Scan(&userCount)
//         if err != nil {
//             log.Println("Error checking users:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to check users"})
//             return
//         }

//         // 4. Если есть пользователи, не разрешаем удаление
//         if userCount > 0 {
//             utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "This school is assigned to a user and cannot be deleted"})
//             return
//         }

//         // 5. Теперь удаляем школу
//         query := "DELETE FROM Schools WHERE school_id = ?"
//         result, err := db.Exec(query, id)
//         if err != nil {
//             log.Println("Error deleting school:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete school"})
//             return
//         }

//         // 6. Проверяем, сколько строк было затронуто
//         rowsAffected, err := result.RowsAffected()
//         if err != nil || rowsAffected == 0 {
//             utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
//             return
//         }

//         // 7. Успешное удаление
//         utils.ResponseJSON(w, map[string]string{"message": "School deleted successfully"})
//     }
// }

// func (sc SchoolController) CalculateAverageRating(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         // 1. Get the school ID from query parameters
//         schoolID := r.URL.Query().Get("school_id")
//         if schoolID == "" {
//             utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
//             return
//         }

//         // 2. Calculate the average rating from various sources

//         // Get the total score from UNT scores for this school
//         var totalUNTScore int
//         var totalStudents int
//         rows, err := db.Query(`
//             SELECT COUNT(*) AS student_count, SUM(total_score) AS total_score
//             FROM UNT_Score us
//             JOIN Student s ON us.student_id = s.student_id
//             WHERE s.school_id = ?
//         `, schoolID)

//         if err != nil {
//             log.Println("SQL Error:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate UNT score"})
//             return
//         }
//         defer rows.Close()

//         if rows.Next() {
//             err = rows.Scan(&totalStudents, &totalUNTScore)
//             if err != nil {
//                 log.Println("Error scanning UNT data:", err)
//                 utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning UNT data"})
//                 return
//             }
//         }

//         // Calculate average UNT score
//         var averageUNTScore float64
//         if totalStudents > 0 {
//             averageUNTScore = float64(totalUNTScore) / float64(totalStudents)
//         }

//         // 3. Get reviews score (e.g., average of reviews from a `reviews` table)
//         var averageReviewScore float64
//         err = db.QueryRow(`
//             SELECT AVG(review_score) 
//             FROM reviews 
//             WHERE school_id = ?
//         `, schoolID).Scan(&averageReviewScore)
//         if err != nil {
//             log.Println("SQL Error fetching reviews:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get reviews score"})
//             return
//         }

//         // 4. Get olympiad participation score
//         var olympiadScore int
//         err = db.QueryRow(`
//             SELECT SUM(olympiad_points) 
//             FROM olympiad_results
//             WHERE school_id = ?
//         `, schoolID).Scan(&olympiadScore)
//         if err != nil {
//             log.Println("SQL Error fetching olympiad results:", err)
//             utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get olympiad results"})
//             return
//         }

//         // 5. Calculate the final average rating based on all criteria
//         finalAverageRating := (averageUNTScore * 0.4) + (averageReviewScore * 0.3) + (float64(olympiadScore) * 0.3)

//         // Return the result as JSON
//         utils.ResponseJSON(w, map[string]interface{}{
//             "school_id":            schoolID,
//             "average_rating":       finalAverageRating,
//             "average_unt_score":    averageUNTScore,
//             "average_review_score": averageReviewScore,
//             "olympiad_score":       olympiadScore,
//         })
//     }
// }












