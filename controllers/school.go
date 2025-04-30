package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type SchoolController struct{}

func (sc *SchoolController) CreateSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Проверка токена
		requesterID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Шаг 2: Получение роли пользователя
		var requesterRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", requesterID).Scan(&requesterRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user role"})
			return
		}

		if requesterRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmins can create schools"})
			return
		}

		// Шаг 3: Разбор формы
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Failed to parse form"})
			return
		}

		var school models.School
		school.SchoolName = r.FormValue("school_name")
		school.City = r.FormValue("city")
		school.SchoolAdminLogin = r.FormValue("school_admin_login")

		// Заполняем остальные поля по умолчанию, если они не переданы
		school.SchoolAddress = r.FormValue("school_address")
		school.AboutSchool = r.FormValue("about_school")
		school.SchoolEmail = r.FormValue("school_email")
		school.SchoolPhone = r.FormValue("school_phone")

		// Обработка специализаций как массива
		specializationsStr := r.FormValue("specializations")
		if specializationsStr != "" {
			err = json.Unmarshal([]byte(specializationsStr), &school.Specializations)
			if err != nil {
				school.Specializations = strings.Split(specializationsStr, ",")
				for i := range school.Specializations {
					school.Specializations[i] = strings.TrimSpace(school.Specializations[i])
				}
			}
		}

		if school.SchoolName == "" || school.City == "" || school.SchoolAdminLogin == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Required fields: school_name, city, school_admin_login"})
			return
		}

		// Шаг 4: Проверка, существует ли уже школа для этого admin
		var schoolAdminID int
		err = db.QueryRow("SELECT id FROM users WHERE email = ? AND role = 'schooladmin'", school.SchoolAdminLogin).Scan(&schoolAdminID)
		if err != nil {
			log.Println("School admin not found:", err)
			rows, err := db.Query("SELECT id, email FROM users WHERE role = 'schooladmin'")
			if err != nil {
				log.Println("Error fetching schooladmin users:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch schooladmins"})
				return
			}
			defer rows.Close()

			var admins []string
			for rows.Next() {
				var adminEmail string
				if err := rows.Scan(&schoolAdminID, &adminEmail); err != nil {
					log.Println("Error scanning schooladmin user:", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning schooladmin users"})
					return
				}
				admins = append(admins, adminEmail)
			}

			if len(admins) == 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "No schooladmins found, please create a user with 'schooladmin' role"})
				return
			}

			utils.ResponseJSON(w, map[string]interface{}{
				"message":               "School admin not found by provided email.",
				"existing_schooladmins": admins,
			})
			return
		}

		// Проверяем, есть ли уже школа для этого admin
		var existingSchoolID int
		err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", school.SchoolAdminLogin).Scan(&existingSchoolID)
		if err == nil {
			log.Println("Admin already has a school, not adding a new one.")
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Admin already has a school."})
			return
		} else if err != sql.ErrNoRows {
			log.Println("Error checking for existing school:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error checking for existing school"})
			return
		}

		// Шаг 5: Загрузка фото, если есть
		file, _, err := r.FormFile("photo")
		if err == nil {
			defer file.Close()
			uniqueFileName := fmt.Sprintf("school-%d-%d.jpg", requesterID, time.Now().Unix())
			photoURL, err := utils.UploadFileToS3(file, uniqueFileName, false)
			if err != nil {
				log.Println("S3 upload failed:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Photo upload failed"})
				return
			}
			school.PhotoURL = photoURL
		}

		// Преобразуем массив специализаций в JSON строку для сохранения в БД
		specializationsJSON, err := json.Marshal(school.Specializations)
		if err != nil {
			log.Println("Error marshaling specializations:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error marshaling specializations"})
			return
		}

		// Шаг 6: Вставка в таблицу Schools
		query := `
           INSERT INTO Schools (school_name, school_address, city, about_school, photo_url, school_email, school_phone, school_admin_login, specializations, created_at, updated_at, user_id)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW(), ?)
        `
		result, err := db.Exec(query, school.SchoolName, school.SchoolAddress, school.City, school.AboutSchool, school.PhotoURL, school.SchoolEmail, school.SchoolPhone, school.SchoolAdminLogin, string(specializationsJSON), schoolAdminID)
		if err != nil {
			log.Println("Insert error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create school"})
			return
		}

		schoolID, _ := result.LastInsertId()
		school.SchoolID = int(schoolID)

		// Шаг 7: Обновление user записи с добавлением school_id
		_, err = db.Exec("UPDATE users SET school_id = ? WHERE id = ?", school.SchoolID, schoolAdminID)
		if err != nil {
			log.Println("Error updating school_id for admin:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school_id for admin"})
			return
		}

		utils.ResponseJSON(w, school)
	}
}
func (sc SchoolController) UpdateMySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверка токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// 2. Получение роли и email пользователя
		var role, userEmail string
		err = db.QueryRow("SELECT role, email FROM users WHERE id = ?", userID).Scan(&role, &userEmail)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get user info"})
			return
		}

		// 3. Разрешить только schooladmin для изменения названия школы и города
		if role != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only schooladmin can update school name or city"})
			return
		}

		// 4. Найти school_id, где school_admin_login совпадает с email пользователя
		var schoolID int
		err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", userEmail).Scan(&schoolID)
		if err != nil || schoolID == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No school assigned to your email"})
			return
		}

		// 5. Прочитать JSON из запроса
		var input struct {
			SchoolName      string `json:"school_name"`
			SchoolAddress   string `json:"school_address"`
			City            string `json:"city"`
			AboutSchool     string `json:"about_school"`
			SchoolEmail     string `json:"school_email"`
			Phone           string `json:"phone"`
			PhotoURL        string `json:"photo_url"`
			Specializations string `json:"specializations"` // Добавлено
		}

		err = json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid JSON format"})
			return
		}

		// Проверка, если роль не schooladmin, то не разрешаем менять school_name или city
		if role != "schooladmin" {
			input.SchoolName = "" // Не разрешаем менять название
			input.City = ""       // Не разрешаем менять город
		}

		// 6. Обновить школу
		query := `
        UPDATE Schools
        SET 
            school_address = ?, 
            about_school = ?, 
            school_email = ?, 
            school_phone = ?, 
            photo_url = ?, 
            specializations = ?, 
            updated_at = NOW()
        WHERE school_id = ?
        `

		// Добавим отладочную информацию
		fmt.Printf("Executing query: %s\n", query)
		fmt.Printf("Input values: %v\n", input)

		_, err = db.Exec(query,
			input.SchoolAddress,
			input.AboutSchool,
			input.SchoolEmail,
			input.Phone,
			input.PhotoURL,
			input.Specializations, // Передаем специализации
			schoolID,
		)
		if err != nil {
			fmt.Printf("Error executing query: %v\n", err) // Дополнительный вывод ошибки
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school"})
			return
		}

		// 7. Успешный ответ
		utils.ResponseJSON(w, map[string]interface{}{
			"message":    "School updated successfully",
			"school_id":  schoolID,
			"updated_by": userEmail,
		})
	}
}
func (sc SchoolController) DeleteSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверка токена и роли
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
			return
		}

		if userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can delete a school"})
			return
		}

		// ✅ 2. Получаем school_id из path-параметра
		vars := mux.Vars(r)
		schoolID := vars["id"]
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
			return
		}

		// 3. Удаляем школу
		result, err := db.Exec("DELETE FROM Schools WHERE school_id = ?", schoolID)
		if err != nil {
			log.Println("SQL Delete Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete school"})
			return
		}

		// 4. Проверка затронутых строк
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
			return
		}

		// 5. Успешный ответ
		utils.ResponseJSON(w, map[string]interface{}{
			"message":    "School deleted successfully",
			"school_id":  schoolID,
			"deleted_by": userID,
		})
	}
}
func (sc SchoolController) UpdateSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверяем токен
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// 2. Проверка роли
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
			return
		}

		if userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can update schools"})
			return
		}

		// ✅ 3. Получаем school_id из path-параметра
		vars := mux.Vars(r)
		schoolID := vars["id"]
		if schoolID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required"})
			return
		}

		// 4. Декодируем JSON из тела запроса в структуру School
		var school models.School
		err = json.NewDecoder(r.Body).Decode(&school)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid JSON data"})
			return
		}

		// 5. Обновляем данные о школе в базе данных
		query := `
            UPDATE Schools
            SET 
                school_name = ?, school_address = ?, city = ?, about_school = ?, 
                photo_url = ?, school_email = ?, school_phone = ?, school_admin_login = ?, 
                specializations = ?, updated_at = NOW()
            WHERE school_id = ?
        `
		// Выполнение запроса на обновление
		_, err = db.Exec(query,
			school.SchoolName,
			school.SchoolAddress,
			school.City,
			school.AboutSchool,
			school.PhotoURL,
			school.SchoolEmail,
			school.SchoolPhone,
			school.SchoolAdminLogin,
			school.Specializations, // Передаем specializations как строку
			schoolID,
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school"})
			return
		}

		// Успешный ответ
		utils.ResponseJSON(w, map[string]interface{}{
			"message":    "School updated successfully",
			"school_id":  schoolID,
			"updated_by": userID,
		})
	}
}
func (sc SchoolController) GetAllSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Выполнение запроса для получения всех школ, включая поле specializations
		query := `SELECT school_id, school_name, school_address, city, about_school, school_email, 
                  school_phone, school_admin_login, specializations FROM Schools`
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error fetching schools:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve schools"})
			return
		}
		defer rows.Close()

		// Шаг 2: Создание среза для хранения данных о школах
		var schools []models.School

		// Шаг 3: Прохождение по результатам запроса и заполнение среза
		for rows.Next() {
			var school models.School
			var city sql.NullString                // Используем sql.NullString для обработки возможного NULL в поле city
			var schoolAddress sql.NullString       // Для адреса
			var aboutSchool sql.NullString         // Для описания школы
			var schoolEmail sql.NullString         // Для email
			var schoolPhone sql.NullString         // Для телефона
			var schoolAdminLogin sql.NullString    // Для логина администратора
			var specializationsJSON sql.NullString // Для специализаций в JSON формате

			err := rows.Scan(
				&school.SchoolID,
				&school.SchoolName,
				&schoolAddress,       // sql.NullString
				&city,                // sql.NullString
				&aboutSchool,         // sql.NullString
				&schoolEmail,         // sql.NullString
				&schoolPhone,         // sql.NullString
				&schoolAdminLogin,    // sql.NullString
				&specializationsJSON, // sql.NullString
			)
			if err != nil {
				log.Println("Error scanning school data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning school data"})
				return
			}

			// Присваиваем значения из sql.NullString в обычные строки, если значение существует
			if city.Valid {
				school.City = city.String
			}

			if schoolAddress.Valid {
				school.SchoolAddress = schoolAddress.String
			}

			if aboutSchool.Valid {
				school.AboutSchool = aboutSchool.String
			}

			if schoolEmail.Valid {
				school.SchoolEmail = schoolEmail.String
			}

			if schoolPhone.Valid {
				school.SchoolPhone = schoolPhone.String
			}

			if schoolAdminLogin.Valid {
				school.SchoolAdminLogin = schoolAdminLogin.String
			}

			// Обработка специализаций
			if specializationsJSON.Valid && specializationsJSON.String != "" {
				err = json.Unmarshal([]byte(specializationsJSON.String), &school.Specializations)
				if err != nil {
					// Если не удалось распарсить JSON, сохраняем как одну строку
					school.Specializations = []string{specializationsJSON.String}
				}
			} else {
				school.Specializations = []string{} // Пустой массив, если нет специализаций
			}

			// Добавляем школу в срез
			schools = append(schools, school)
		}

		// Шаг 4: Проверка ошибок после завершения итерации
		if err = rows.Err(); err != nil {
			log.Println("Error during iteration:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error during iteration"})
			return
		}

		// Шаг 5: Возвращаем список всех школ в формате JSON
		utils.ResponseJSON(w, schools)
	}
}
