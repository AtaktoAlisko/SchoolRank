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
			photoURL, err := utils.UploadFileToS3(file, uniqueFileName, "schoolphoto")
			if err != nil {
				log.Println("S3 upload failed:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Photo upload failed"})
				return
			}
			school.PhotoURL = sql.NullString{String: photoURL, Valid: true}
		} else {
			school.PhotoURL = sql.NullString{Valid: false} // Set to NULL when no photo is uploaded
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
		result, err := db.Exec(query,
			school.SchoolName,
			school.SchoolAddress,
			school.City,
			school.AboutSchool,
			school.PhotoURL,
			school.SchoolEmail,
			school.SchoolPhone,
			school.SchoolAdminLogin,
			string(specializationsJSON),
			schoolAdminID,
		)
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

		// Подготовка ответа
		responseSchool := struct {
			SchoolID         int      `json:"school_id"`
			UserID           int      `json:"user_id"`
			SchoolName       string   `json:"school_name"`
			SchoolAddress    string   `json:"school_address"`
			City             string   `json:"city"`
			AboutSchool      string   `json:"about_school"`
			PhotoURL         *string  `json:"photo_url"` // Pointer to handle NULL
			SchoolEmail      string   `json:"school_email"`
			SchoolPhone      string   `json:"school_phone"`
			SchoolAdminLogin string   `json:"school_admin_login"`
			CreatedAt        string   `json:"created_at"`
			UpdatedAt        string   `json:"updated_at"`
			Specializations  []string `json:"specializations"`
		}{
			SchoolID:         school.SchoolID,
			UserID:           schoolAdminID,
			SchoolName:       school.SchoolName,
			SchoolAddress:    school.SchoolAddress,
			City:             school.City,
			AboutSchool:      school.AboutSchool,
			SchoolEmail:      school.SchoolEmail,
			SchoolPhone:      school.SchoolPhone,
			SchoolAdminLogin: school.SchoolAdminLogin,
			Specializations:  school.Specializations,
			CreatedAt:        time.Now().Format(time.RFC3339),
			UpdatedAt:        time.Now().Format(time.RFC3339),
		}

		// Преобразуем sql.NullString в указатель для JSON
		if school.PhotoURL.Valid {
			responseSchool.PhotoURL = &school.PhotoURL.String
		}

		utils.ResponseJSON(w, responseSchool)
	}
}
func (sc *SchoolController) UpdateSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Token verification
		requesterID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		log.Println("DEBUG: Начало обработки запроса на обновление школы")
		log.Println("DEBUG: Метод запроса:", r.Method)
		log.Println("DEBUG: Content-Type:", r.Header.Get("Content-Type"))

		// Step 2: Get user role
		var requesterRole string
		var requesterSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", requesterID).Scan(&requesterRole, &requesterSchoolID)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user role"})
			return
		}

		// Step 3: Get school ID from URL
		vars := mux.Vars(r)
		schoolID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}
		log.Println("DEBUG: Обновление школы ID:", schoolID)

		// Access control: ensure the user has permission to update this school
		if requesterRole != "superadmin" && (requesterRole != "schooladmin" || int(requesterSchoolID.Int64) != schoolID) {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only update your own school"})
			return
		}

		// Step 4: Fetch the existing school data
		var existingSchool models.School
		var specializationsJSON string
		err = db.QueryRow(`
            SELECT 
                school_id, 
                school_name, 
                school_address, 
                city, 
                about_school, 
                photo_url, 
                school_email, 
                school_phone, 
                school_admin_login, 
                specializations 
            FROM Schools 
            WHERE school_id = ?`, schoolID).Scan(
			&existingSchool.SchoolID,
			&existingSchool.SchoolName,
			&existingSchool.SchoolAddress,
			&existingSchool.City,
			&existingSchool.AboutSchool,
			&existingSchool.PhotoURL, // sql.NullString
			&existingSchool.SchoolEmail,
			&existingSchool.SchoolPhone,
			&existingSchool.SchoolAdminLogin,
			&specializationsJSON,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
			} else {
				log.Println("Error fetching school data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school data"})
			}
			return
		}

		log.Println("DEBUG: Существующее фото URL:", existingSchool.PhotoURL)

		// Unmarshal existing specializations
		if specializationsJSON != "" {
			err = json.Unmarshal([]byte(specializationsJSON), &existingSchool.Specializations)
			if err != nil {
				log.Println("Error unmarshaling existing specializations:", err)
			}
		}

		// Step 5: Parse the form data for other fields
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			log.Println("DEBUG: Ошибка при разборе формы:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Failed to parse form"})
			return
		}

		// Отладка: вывести все ключи формы
		log.Println("DEBUG: Ключи формы:")
		for key := range r.MultipartForm.Value {
			log.Printf("DEBUG: Ключ: %s, Значение: %s\n", key, r.FormValue(key))
		}
		for key := range r.MultipartForm.File {
			log.Printf("DEBUG: Файловый ключ: %s\n", key)
		}

		// Create a new school object based on the existing school
		updatedSchool := existingSchool

		// Step 6: Only update fields that were provided in the form and are not empty
		if name := r.FormValue("school_name"); name != "" {
			updatedSchool.SchoolName = name
		}
		if login := r.FormValue("school_admin_login"); login != "" {
			updatedSchool.SchoolAdminLogin = login
		}
		if city := r.FormValue("city"); city != "" {
			updatedSchool.City = city
		}
		if address := r.FormValue("school_address"); address != "" {
			updatedSchool.SchoolAddress = address
		}
		if about := r.FormValue("about_school"); about != "" {
			updatedSchool.AboutSchool = about
		}
		if email := r.FormValue("school_email"); email != "" {
			updatedSchool.SchoolEmail = email
		}
		// Исправляем проблему с опечаткой в форме
		if phone := r.FormValue("schol_phone"); phone != "" {
			log.Println("DEBUG: Обнаружено поле schol_phone (опечатка), значение:", phone)
			updatedSchool.SchoolPhone = phone
		}
		if phone := r.FormValue("school_phone"); phone != "" {
			updatedSchool.SchoolPhone = phone
		}

		// Step 7: Handle specializations if provided
		specializationsStr := r.FormValue("specializations")
		if specializationsStr != "" {
			err = json.Unmarshal([]byte(specializationsStr), &updatedSchool.Specializations)
			if err != nil {
				updatedSchool.Specializations = strings.Split(specializationsStr, ",")
				for i := range updatedSchool.Specializations {
					updatedSchool.Specializations[i] = strings.TrimSpace(updatedSchool.Specializations[i])
				}
			}
		}

		// Step 8: Handle photo upload (если загружена как photo или photo_url)
		photoUpdated := false

		// Проверяем наличие файла photo_url в форме (исправляем имя поля)
		if fileHeaders := r.MultipartForm.File["photo_url"]; len(fileHeaders) > 0 {
			fileHeader := fileHeaders[0]

			log.Printf("DEBUG: Найден файл в поле 'photo_url': %s, размер: %d байт",
				fileHeader.Filename, fileHeader.Size)

			if fileHeader.Size > 0 {
				file, err := fileHeader.Open()
				if err != nil {
					log.Println("DEBUG: Ошибка при открытии файла:", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to open uploaded file"})
					return
				}
				defer file.Close()

				uniqueFileName := fmt.Sprintf("school-%d-%d.jpg", schoolID, time.Now().Unix())
				log.Println("DEBUG: Загрузка файла в S3 с именем:", uniqueFileName)

				photoURL, err := utils.UploadFileToS3(file, uniqueFileName, "schoolphoto")
				if err != nil {
					log.Println("DEBUG: S3 upload failed:", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Photo upload failed: " + err.Error()})
					return
				}

				updatedSchool.PhotoURL = sql.NullString{String: photoURL, Valid: true}
				photoUpdated = true
				log.Println("DEBUG: Фото успешно загружено, URL:", photoURL)
			} else {
				log.Println("DEBUG: Файл имеет нулевой размер, игнорируем")
			}
		} else {
			// Проверяем также поле photo для обратной совместимости
			if fileHeaders := r.MultipartForm.File["photo"]; len(fileHeaders) > 0 {
				fileHeader := fileHeaders[0]

				log.Printf("DEBUG: Найден файл в поле 'photo': %s, размер: %d байт",
					fileHeader.Filename, fileHeader.Size)

				if fileHeader.Size > 0 {
					file, err := fileHeader.Open()
					if err != nil {
						log.Println("DEBUG: Ошибка при открытии файла:", err)
						utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to open uploaded file"})
						return
					}
					defer file.Close()

					uniqueFileName := fmt.Sprintf("school-%d-%d.jpg", schoolID, time.Now().Unix())
					log.Println("DEBUG: Загрузка файла в S3 с именем:", uniqueFileName)

					photoURL, err := utils.UploadFileToS3(file, uniqueFileName, "schoolphoto")
					if err != nil {
						log.Println("DEBUG: S3 upload failed:", err)
						utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Photo upload failed: " + err.Error()})
						return
					}

					updatedSchool.PhotoURL = sql.NullString{String: photoURL, Valid: true}
					photoUpdated = true
					log.Println("DEBUG: Фото успешно загружено, URL:", photoURL)
				} else {
					log.Println("DEBUG: Файл имеет нулевой размер, игнорируем")
				}
			} else {
				log.Println("DEBUG: Поля 'photo' и 'photo_url' не найдены в форме")
			}
		}

		// Проверяем явный запрос на удаление фото
		if removePhoto := r.FormValue("remove_photo"); removePhoto == "true" {
			updatedSchool.PhotoURL = sql.NullString{Valid: false}
			photoUpdated = true
			log.Println("DEBUG: Фото удалено по запросу")
		}

		// Если фото не было обновлено, сохраняем существующее
		if !photoUpdated {
			log.Println("DEBUG: Сохраняем существующее фото URL:", existingSchool.PhotoURL)
		}

		// Принудительная установка URL фото по умолчанию, если запрашивается
		// или если нет фото и включен флаг use_default_if_empty
		if (!updatedSchool.PhotoURL.Valid && r.FormValue("use_default_if_empty") == "true") ||
			r.FormValue("force_default_photo") == "true" {
			defaultURL := "https://schoolrank-schoolphotos.s3.eu-central-1.amazonaws.com/default-school.jpg"
			updatedSchool.PhotoURL = sql.NullString{String: defaultURL, Valid: true}
			log.Println("DEBUG: Установлено фото по умолчанию:", defaultURL)
		}

		// Step 9: Convert specializations to JSON
		updatedSpecializationsJSON, err := json.Marshal(updatedSchool.Specializations)
		if err != nil {
			log.Println("Error marshaling specializations:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error marshaling specializations"})
			return
		}

		// Отладка: выводим, что именно будем обновлять
		log.Printf("DEBUG: Обновляемые данные: PhotoURL=%v, Valid=%v",
			updatedSchool.PhotoURL.String, updatedSchool.PhotoURL.Valid)

		// Step 10: Update school data in the database
		query := `
            UPDATE Schools
            SET
                school_name = ?,
                school_address = ?,
                city = ?,
                about_school = ?,
                photo_url = ?,
                school_email = ?,
                school_phone = ?,
                school_admin_login = ?,
                specializations = ?,
                updated_at = NOW()
            WHERE school_id = ?
        `
		result, err := db.Exec(query,
			updatedSchool.SchoolName,
			updatedSchool.SchoolAddress,
			updatedSchool.City,
			updatedSchool.AboutSchool,
			updatedSchool.PhotoURL,
			updatedSchool.SchoolEmail,
			updatedSchool.SchoolPhone,
			updatedSchool.SchoolAdminLogin,
			string(updatedSpecializationsJSON),
			updatedSchool.SchoolID,
		)
		if err != nil {
			log.Println("DEBUG: Ошибка обновления в БД:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Println("DEBUG: Затронуто строк в БД:", rowsAffected)

		// Проверка после обновления
		var checkPhotoURL sql.NullString
		err = db.QueryRow("SELECT photo_url FROM Schools WHERE school_id = ?", schoolID).Scan(&checkPhotoURL)
		if err != nil {
			log.Println("DEBUG: Ошибка при проверке обновления:", err)
		} else {
			log.Printf("DEBUG: После обновления в БД: PhotoURL=%v, Valid=%v",
				checkPhotoURL.String, checkPhotoURL.Valid)
		}

		// Step 11: Return the updated school data as a response
		responseSchool := struct {
			SchoolID         int      `json:"school_id"`
			SchoolName       string   `json:"school_name"`
			SchoolAddress    string   `json:"school_address"`
			City             string   `json:"city"`
			AboutSchool      string   `json:"about_school"`
			PhotoURL         *string  `json:"photo_url"` // Pointer to handle NULL
			SchoolEmail      string   `json:"school_email"`
			SchoolPhone      string   `json:"school_phone"`
			SchoolAdminLogin string   `json:"school_admin_login"`
			Specializations  []string `json:"specializations"`
		}{
			SchoolID:         updatedSchool.SchoolID,
			SchoolName:       updatedSchool.SchoolName,
			SchoolAddress:    updatedSchool.SchoolAddress,
			City:             updatedSchool.City,
			AboutSchool:      updatedSchool.AboutSchool,
			SchoolEmail:      updatedSchool.SchoolEmail,
			SchoolPhone:      updatedSchool.SchoolPhone,
			SchoolAdminLogin: updatedSchool.SchoolAdminLogin,
			Specializations:  updatedSchool.Specializations,
		}

		// Handle PhotoURL for JSON response
		if updatedSchool.PhotoURL.Valid {
			responseSchool.PhotoURL = &updatedSchool.PhotoURL.String
		}

		log.Println("DEBUG: Обработка запроса завершена успешно")

		utils.ResponseJSON(w, map[string]interface{}{
			"message":    "School updated successfully",
			"school_id":  schoolID,
			"updated_by": requesterID,
			"school":     responseSchool,
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
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
func (sc SchoolController) GetAllSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Выполнение запроса для получения всех школ
		query := `SELECT school_id, school_name, school_address, city, about_school, photo_url, 
                         school_email, school_phone, school_admin_login, specializations, 
                         created_at, updated_at FROM Schools`
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error fetching schools:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve schools"})
			return
		}
		defer rows.Close()

		// Шаг 2: Создание среза для хранения данных о школах
		var schools []struct {
			models.School
			PhotoURL *string `json:"photo_url"` // Pointer for JSON null handling
		}

		// Шаг 3: Прохождение по результатам запроса и заполнение среза
		for rows.Next() {
			var school models.School
			var city sql.NullString
			var schoolAddress sql.NullString
			var aboutSchool sql.NullString
			var photoURL sql.NullString
			var schoolEmail sql.NullString
			var schoolPhone sql.NullString
			var schoolAdminLogin sql.NullString
			var specializationsJSON sql.NullString
			var createdAtStr sql.NullString // Use sql.NullString for created_at
			var updatedAtStr sql.NullString // Use sql.NullString for updated_at

			err := rows.Scan(
				&school.SchoolID,
				&school.SchoolName,
				&schoolAddress,
				&city,
				&aboutSchool,
				&photoURL,
				&schoolEmail,
				&schoolPhone,
				&schoolAdminLogin,
				&specializationsJSON,
				&createdAtStr,
				&updatedAtStr,
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

			// Обработка photo_url
			school.PhotoURL = photoURL // Assign sql.NullString directly

			// Обработка специализаций
			if specializationsJSON.Valid && specializationsJSON.String != "" {
				err = json.Unmarshal([]byte(specializationsJSON.String), &school.Specializations)
				if err != nil {
					school.Specializations = []string{specializationsJSON.String}
				}
			} else {
				school.Specializations = []string{}
			}

			// Обработка created_at
			if createdAtStr.Valid && createdAtStr.String != "" {
				parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAtStr.String)
				if err != nil {
					log.Printf("Error parsing created_at '%s': %v", createdAtStr.String, err)
					school.CreatedAt = "" // Fallback to empty string
				} else {
					school.CreatedAt = parsedTime.Format(time.RFC3339)
				}
			} else {
				school.CreatedAt = ""
			}

			// Обработка updated_at
			if updatedAtStr.Valid && updatedAtStr.String != "" {
				parsedTime, err := time.Parse("2006-01-02 15:04:05", updatedAtStr.String)
				if err != nil {
					log.Printf("Error parsing updated_at '%s': %v", updatedAtStr.String, err)
					school.UpdatedAt = "" // Fallback to empty string
				} else {
					school.UpdatedAt = parsedTime.Format(time.RFC3339)
				}
			} else {
				school.UpdatedAt = ""
			}

			// Подготовка структуры для ответа с правильной сериализацией photo_url
			responseSchool := struct {
				models.School
				PhotoURL *string `json:"photo_url"`
			}{School: school}
			if photoURL.Valid {
				responseSchool.PhotoURL = &photoURL.String
			}

			// Добавляем школу в срез
			schools = append(schools, responseSchool)
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
func (sc *SchoolController) GetTotalSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Выполняем запрос для получения общего количества школ
		var totalSchools int
		err := db.QueryRow("SELECT COUNT(*) FROM Schools").Scan(&totalSchools)
		if err != nil {
			log.Println("Error fetching total schools:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get total schools"})
			return
		}

		// Шаг 2: Формируем ответ с количеством школ
		response := map[string]interface{}{
			"total_schools": totalSchools,
		}

		utils.ResponseJSON(w, response)
	}
}
func (sc SchoolController) GetAllStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT s.student_id, s.first_name, s.school_id, sch.school_name
			FROM student s
			JOIN Schools sch ON s.school_id = sch.school_id
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error fetching students:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve students"})
			return
		}
		defer rows.Close()

		type StudentWithSchool struct {
			StudentID   int    `json:"student_id"`
			StudentName string `json:"student_name"`
			SchoolID    int    `json:"school_id"`
			SchoolName  string `json:"school_name"`
		}

		var students []StudentWithSchool
		for rows.Next() {
			var s StudentWithSchool
			err := rows.Scan(&s.StudentID, &s.StudentName, &s.SchoolID, &s.SchoolName)
			if err != nil {
				log.Println("Error scanning student data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning student data"})
				return
			}
			students = append(students, s)
		}

		if err = rows.Err(); err != nil {
			log.Println("Error during iteration:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error during iteration"})
			return
		}

		// Вернуть также общее количество школ
		var totalSchools int
		err = db.QueryRow(`SELECT COUNT(*) FROM Schools`).Scan(&totalSchools)
		if err != nil {
			log.Println("Error counting schools:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to count schools"})
			return
		}

		response := map[string]interface{}{
			"total_schools": totalSchools,
			"students":      students,
		}

		utils.ResponseJSON(w, response)
	}
}
func (sc *SchoolController) GetSchoolByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get school ID from URL
		vars := mux.Vars(r)
		idStr, ok := vars["id"]
		if !ok {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID not provided"})
			return
		}

		// Clean the ID string - trim any whitespace or newline characters
		idStr = strings.TrimSpace(idStr)
		log.Printf("ID parameter after trimming: '%s'", idStr)

		schoolID, err := strconv.Atoi(idStr)
		if err != nil {
			log.Printf("Error converting ID to int: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Query to fetch the school by ID
		var school models.School
		var specializationsJSON string
		query := `
			SELECT 
				school_id, 
				school_name, 
				school_address, 
				city, 
				about_school, 
				photo_url, 
				school_email, 
				school_phone, 
				school_admin_login, 
				specializations 
			FROM Schools 
			WHERE school_id = ?`

		err = db.QueryRow(query, schoolID).Scan(
			&school.SchoolID,
			&school.SchoolName,
			&school.SchoolAddress,
			&school.City,
			&school.AboutSchool,
			&school.PhotoURL,
			&school.SchoolEmail,
			&school.SchoolPhone,
			&school.SchoolAdminLogin,
			&specializationsJSON,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "School not found"})
			} else {
				log.Println("Error fetching school data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school data"})
			}
			return
		}

		// Parse specializations from JSON
		if specializationsJSON != "" {
			err = json.Unmarshal([]byte(specializationsJSON), &school.Specializations)
			if err != nil {
				log.Println("Error unmarshaling specializations:", err)
				// If there's an error parsing JSON, initialize as empty array
				school.Specializations = []string{}
			}
		} else {
			school.Specializations = []string{} // Empty array if no specializations
		}

		// Get total number of students for this school
		var totalStudents int
		err = db.QueryRow("SELECT COUNT(*) FROM student WHERE school_id = ?", schoolID).Scan(&totalStudents)
		if err != nil {
			log.Println("Error counting students:", err)
			// Continue even if we can't count students
			totalStudents = 0
		}

		// Return response with school data and student count
		response := map[string]interface{}{
			"school":         school,
			"total_students": totalStudents,
		}

		utils.ResponseJSON(w, response)
	}
}

func (sc *SchoolController) GetSchoolCount(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Println("GetSchoolCount endpoint called") // Debug log
        
        var count int
        var error models.Error

        // Query to count all schools
        err := db.QueryRow("SELECT COUNT(*) FROM Schools").Scan(&count)
        if err != nil {
            log.Printf("Error counting schools: %v", err)
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Prepare response
        response := map[string]interface{}{
            "count": count,
        }

        utils.ResponseJSON(w, response)
    }
}