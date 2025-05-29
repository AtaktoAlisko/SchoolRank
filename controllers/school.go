package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"sort"
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

		// Обработка SchoolAdminLogin как sql.NullString
		adminLogin := r.FormValue("school_admin_login")
		school.SchoolAdminLogin = sql.NullString{
			String: adminLogin,
			Valid:  adminLogin != "",
		}

		// Заполняем nullable поля
		schoolAddress := r.FormValue("school_address")
		school.SchoolAddress = sql.NullString{
			String: schoolAddress,
			Valid:  schoolAddress != "",
		}

		aboutSchool := r.FormValue("about_school")
		school.AboutSchool = sql.NullString{
			String: aboutSchool,
			Valid:  aboutSchool != "",
		}

		schoolEmail := r.FormValue("school_email")
		school.SchoolEmail = sql.NullString{
			String: schoolEmail,
			Valid:  schoolEmail != "",
		}

		schoolPhone := r.FormValue("school_phone")
		school.SchoolPhone = sql.NullString{
			String: schoolPhone,
			Valid:  schoolPhone != "",
		}

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

		// Проверка обязательных полей
		if school.SchoolName == "" || school.City == "" || !school.SchoolAdminLogin.Valid || school.SchoolAdminLogin.String == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Required fields: school_name, city, school_admin_login"})
			return
		}

		// Шаг 4: Проверка, существует ли уже школа для этого admin
		var schoolAdminID int
		err = db.QueryRow("SELECT id FROM users WHERE email = ? AND role = 'schooladmin'", school.SchoolAdminLogin.String).Scan(&schoolAdminID)
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
		err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", school.SchoolAdminLogin.String).Scan(&existingSchoolID)
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

		// Устанавливаем CreatedAt и UpdatedAt как текущее время в формате SQL
		currentTime := time.Now().Format(time.RFC3339)
		school.CreatedAt = sql.NullString{String: currentTime, Valid: true}
		school.UpdatedAt = sql.NullString{String: currentTime, Valid: true}

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
		school.UserID = schoolAdminID

		// Шаг 7: Обновление user записи с добавлением school_id
		_, err = db.Exec("UPDATE users SET school_id = ? WHERE id = ?", school.SchoolID, schoolAdminID)
		if err != nil {
			log.Println("Error updating school_id for admin:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school_id for admin"})
			return
		}

		// Подготовка ответа с учетом nullable полей
		responseSchool := struct {
			SchoolID         int      `json:"school_id"`
			UserID           int      `json:"user_id"`
			SchoolName       string   `json:"school_name"`
			SchoolAddress    *string  `json:"school_address"`
			City             string   `json:"city"`
			AboutSchool      *string  `json:"about_school"`
			PhotoURL         *string  `json:"photo_url"`
			SchoolEmail      *string  `json:"school_email"`
			SchoolPhone      *string  `json:"school_phone"`
			SchoolAdminLogin *string  `json:"school_admin_login"`
			CreatedAt        *string  `json:"created_at"`
			UpdatedAt        *string  `json:"updated_at"`
			Specializations  []string `json:"specializations"`
		}{
			SchoolID:        school.SchoolID,
			UserID:          school.UserID,
			SchoolName:      school.SchoolName,
			City:            school.City,
			Specializations: school.Specializations,
		}

		// Конвертируем sql.NullString в указатели для JSON
		if school.SchoolAddress.Valid {
			responseSchool.SchoolAddress = &school.SchoolAddress.String
		}
		if school.AboutSchool.Valid {
			responseSchool.AboutSchool = &school.AboutSchool.String
		}
		if school.PhotoURL.Valid {
			responseSchool.PhotoURL = &school.PhotoURL.String
		}
		if school.SchoolEmail.Valid {
			responseSchool.SchoolEmail = &school.SchoolEmail.String
		}
		if school.SchoolPhone.Valid {
			responseSchool.SchoolPhone = &school.SchoolPhone.String
		}
		if school.SchoolAdminLogin.Valid {
			responseSchool.SchoolAdminLogin = &school.SchoolAdminLogin.String
		}
		if school.CreatedAt.Valid {
			responseSchool.CreatedAt = &school.CreatedAt.String
		}
		if school.UpdatedAt.Valid {
			responseSchool.UpdatedAt = &school.UpdatedAt.String
		}

		utils.ResponseJSON(w, responseSchool)
	}
}
func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
func (sc *SchoolController) UpdateSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Token verification
		requesterID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		log.Println("DEBUG: Starting school update request")
		log.Println("DEBUG: Request method:", r.Method)
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
		log.Println("DEBUG: Updating school ID:", schoolID)

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
                user_id, 
                school_name, 
                school_address, 
                city, 
                about_school, 
                photo_url, 
                school_email, 
                school_phone, 
                school_admin_login, 
                created_at, 
                updated_at, 
                specializations 
            FROM Schools 
            WHERE school_id = ?`, schoolID).Scan(
			&existingSchool.SchoolID,
			&existingSchool.UserID,
			&existingSchool.SchoolName,
			&existingSchool.SchoolAddress,
			&existingSchool.City,
			&existingSchool.AboutSchool,
			&existingSchool.PhotoURL,
			&existingSchool.SchoolEmail,
			&existingSchool.SchoolPhone,
			&existingSchool.SchoolAdminLogin,
			&existingSchool.CreatedAt,
			&existingSchool.UpdatedAt,
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

		log.Println("DEBUG: Existing photo URL:", existingSchool.PhotoURL)

		// Unmarshal existing specializations
		if specializationsJSON != "" {
			err = json.Unmarshal([]byte(specializationsJSON), &existingSchool.Specializations)
			if err != nil {
				log.Println("Error unmarshaling existing specializations:", err)
			}
		}

		// Step 5: Parse the form data
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			log.Println("DEBUG: Error parsing form:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Failed to parse form"})
			return
		}

		// Debug: Log form keys
		log.Println("DEBUG: Form keys:")
		for key := range r.MultipartForm.Value {
			log.Printf("DEBUG: Key: %s, Value: %s\n", key, r.FormValue(key))
		}
		for key := range r.MultipartForm.File {
			log.Printf("DEBUG: File key: %s\n", key)
		}

		// Create a new school object based on the existing school
		updatedSchool := existingSchool

		// Step 6: Update fields that were provided in the form and are not empty
		if name := r.FormValue("school_name"); name != "" {
			updatedSchool.SchoolName = name
		}
		if login := r.FormValue("school_admin_login"); login != "" {
			updatedSchool.SchoolAdminLogin = toNullString(login)
		}
		if city := r.FormValue("city"); city != "" {
			updatedSchool.City = city // City is string, not sql.NullString
		}
		if address := r.FormValue("school_address"); address != "" {
			updatedSchool.SchoolAddress = toNullString(address)
		}
		if about := r.FormValue("about_school"); about != "" {
			updatedSchool.AboutSchool = toNullString(about)
		}
		if email := r.FormValue("school_email"); email != "" {
			updatedSchool.SchoolEmail = toNullString(email)
		}
		if phone := r.FormValue("schol_phone"); phone != "" {
			log.Println("DEBUG: Detected schol_phone (typo), value:", phone)
			updatedSchool.SchoolPhone = toNullString(phone)
		}
		if phone := r.FormValue("school_phone"); phone != "" {
			updatedSchool.SchoolPhone = toNullString(phone)
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

		// Step 8: Handle photo upload (check both photo_url and photo fields)
		for _, field := range []string{"photo_url", "photo"} {
			if fileHeaders := r.MultipartForm.File[field]; len(fileHeaders) > 0 {
				fileHeader := fileHeaders[0]
				log.Printf("DEBUG: Found file in '%s': %s, size: %d bytes", field, fileHeader.Filename, fileHeader.Size)
				if fileHeader.Size > 0 {
					file, err := fileHeader.Open()
					if err != nil {
						log.Println("DEBUG: Error opening file:", err)
						utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to open uploaded file"})
						return
					}
					defer file.Close()

					uniqueFileName := fmt.Sprintf("school-%d-%d.jpg", schoolID, time.Now().Unix())
					log.Println("DEBUG: Uploading file to S3 with name:", uniqueFileName)

					photoURL, err := utils.UploadFileToS3(file, uniqueFileName, "schoolphoto")
					if err != nil {
						log.Println("DEBUG: S3 upload failed:", err)
						utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Photo upload failed: " + err.Error()})
						return
					}

					updatedSchool.PhotoURL = sql.NullString{String: photoURL, Valid: true}
					log.Println("DEBUG: Photo uploaded successfully, URL:", photoURL)
				}
			}
		}

		// Check for explicit photo removal
		if removePhoto := r.FormValue("remove_photo"); removePhoto == "true" {
			updatedSchool.PhotoURL = sql.NullString{Valid: false}
			log.Println("DEBUG: Photo removed by request")
		}

		// Set default photo if needed
		if (!updatedSchool.PhotoURL.Valid && r.FormValue("use_default_if_empty") == "true") || r.FormValue("force_default_photo") == "true" {
			defaultURL := "https://schoolrank-schoolphotos.s3.eu-central-1.amazonaws.com/default-school.jpg"
			updatedSchool.PhotoURL = sql.NullString{String: defaultURL, Valid: true}
			log.Println("DEBUG: Set default photo:", defaultURL)
		}

		// Step 9: Convert specializations to JSON
		updatedSpecializationsJSON, err := json.Marshal(updatedSchool.Specializations)
		if err != nil {
			log.Println("Error marshaling specializations:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error marshaling specializations"})
			return
		}

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
			log.Println("DEBUG: Error updating database:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Println("DEBUG: Rows affected:", rowsAffected)

		// Step 11: Return the updated school data as a response
		responseSchool := struct {
			SchoolID         int      `json:"school_id"`
			UserID           int      `json:"user_id"`
			SchoolName       string   `json:"school_name"`
			SchoolAddress    string   `json:"school_address"`
			City             string   `json:"city"`
			AboutSchool      string   `json:"about_school"`
			PhotoURL         *string  `json:"photo_url"`
			SchoolEmail      string   `json:"school_email"`
			SchoolPhone      string   `json:"school_phone"`
			SchoolAdminLogin string   `json:"school_admin_login"`
			CreatedAt        string   `json:"created_at"`
			UpdatedAt        string   `json:"updated_at"`
			Specializations  []string `json:"specializations"`
		}{
			SchoolID:         updatedSchool.SchoolID,
			UserID:           updatedSchool.UserID,
			SchoolName:       updatedSchool.SchoolName,
			SchoolAddress:    updatedSchool.SchoolAddress.String,
			City:             updatedSchool.City,
			AboutSchool:      updatedSchool.AboutSchool.String,
			SchoolEmail:      updatedSchool.SchoolEmail.String,
			SchoolPhone:      updatedSchool.SchoolPhone.String,
			SchoolAdminLogin: updatedSchool.SchoolAdminLogin.String,
			CreatedAt:        updatedSchool.CreatedAt.String,
			UpdatedAt:        updatedSchool.UpdatedAt.String,
			Specializations:  updatedSchool.Specializations,
		}

		if updatedSchool.PhotoURL.Valid {
			responseSchool.PhotoURL = &updatedSchool.PhotoURL.String
		}

		log.Println("DEBUG: Request processing completed successfully")

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
		query := `SELECT school_id, user_id, school_name, school_address, city, about_school, photo_url, 
                         school_email, school_phone, school_admin_login, specializations, 
                         created_at, updated_at FROM Schools`
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error fetching schools:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve schools"})
			return
		}
		defer rows.Close()

		var schools []struct {
			SchoolID          int      `json:"school_id"`
			UserID            int      `json:"user_id"`
			SchoolName        string   `json:"school_name"`
			SchoolAddress     *string  `json:"school_address"`
			City              string   `json:"city"`
			AboutSchool       *string  `json:"about_school"`
			PhotoURL          *string  `json:"photo_url"`
			SchoolPhone       *string  `json:"school_phone"`
			SchoolAdminLogin  *string  `json:"school_admin_login"`
			Specializations   []string `json:"specializations"`
			CreatedAt         *string  `json:"created_at"`
			UpdatedAt         *string  `json:"updated_at"`
			Rating            *float64 `json:"rating"`
			UntRank           float64  `json:"unt_rank"`
			EventScore        float64  `json:"event_score"`
			ParticipantPoints float64  `json:"participant_points"`
			AverageRatingRank float64  `json:"average_rating_rank"`
			OlympiadRank      float64  `json:"olympiad_rank"`
			TotalRating       float64  `json:"total_rating"`
		}

		src := &SchoolRatingController{}

		for rows.Next() {
			var school models.School
			var specializationsJSON sql.NullString

			err := rows.Scan(
				&school.SchoolID,
				&school.UserID,
				&school.SchoolName,
				&school.SchoolAddress,
				&school.City,
				&school.AboutSchool,
				&school.PhotoURL,
				&school.SchoolEmail,
				&school.SchoolPhone,
				&school.SchoolAdminLogin,
				&specializationsJSON,
				&school.CreatedAt,
				&school.UpdatedAt,
			)
			if err != nil {
				log.Println("Error scanning school data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning school data"})
				return
			}

			var specializations []string
			if specializationsJSON.Valid {
				err = json.Unmarshal([]byte(specializationsJSON.String), &specializations)
				if err != nil {
					specializations = []string{}
				}
			}

			var schoolPhone *string
			if school.SchoolPhone.Valid {
				schoolPhone = &school.SchoolPhone.String
			}

			var adminLogin *string
			if school.SchoolAdminLogin.Valid {
				adminLogin = &school.SchoolAdminLogin.String
			}

			var schoolAddress *string
			if school.SchoolAddress.Valid {
				schoolAddress = &school.SchoolAddress.String
			}

			var aboutSchool *string
			if school.AboutSchool.Valid {
				aboutSchool = &school.AboutSchool.String
			}

			var photoURL *string
			if school.PhotoURL.Valid && school.PhotoURL.String != "" {
				photoURL = &school.PhotoURL.String
			}

			var createdAt *string
			if school.CreatedAt.Valid {
				createdAt = &school.CreatedAt.String
			}
			var updatedAt *string
			if school.UpdatedAt.Valid {
				updatedAt = &school.UpdatedAt.String
			}

			// Rating
			var avgRating sql.NullFloat64
			ratingQuery := `SELECT AVG(rating) FROM Reviews WHERE school_id = ?`
			err = db.QueryRow(ratingQuery, school.SchoolID).Scan(&avgRating)
			if err != nil && err != sql.ErrNoRows {
				log.Println("Error fetching average rating for school", school.SchoolID, ":", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching average rating"})
				return
			}

			var rating *float64
			if avgRating.Valid {
				roundedRating := math.Round(avgRating.Float64*100) / 100
				rating = &roundedRating
			}

			// UNT Rank
			untRank, err := src.getUNTRank(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении UNT рейтинга для школы %d: %v", school.SchoolID, err)
				untRank = 0.0
			}
			untRank = math.Round(untRank*100) / 100

			// Event Score
			eventScore, err := src.getEventScore(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении счета событий для школы %d: %v", school.SchoolID, err)
				eventScore = 0.0
			}
			eventScore = math.Round(eventScore*100) / 100

			// Participant Points
			participantPoints, err := src.getParticipantPoints(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении очков участников для школы %d: %v", school.SchoolID, err)
				participantPoints = 0.0
			}
			participantPoints = math.Round(participantPoints*100) / 100

			// Average Rating Rank
			averageRatingRank, err := src.getAverageRatingRank(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении рейтинга отзывов для школы %d: %v", school.SchoolID, err)
				averageRatingRank = 0.0
			}
			averageRatingRank = math.Round(averageRatingRank*100) / 100

			// Olympiad Rank
			olympiadRank, err := src.getOlympiadRank(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении олимпиадного рейтинга для школы %d: %v", school.SchoolID, err)
				olympiadRank = 0.0
			}
			olympiadRank = math.Round(olympiadRank*100) / 100

			// Total Rating
			totalRating := eventScore + participantPoints + averageRatingRank + untRank + olympiadRank
			totalRating = math.Round(totalRating*100) / 100

			schools = append(schools, struct {
				SchoolID          int      `json:"school_id"`
				UserID            int      `json:"user_id"`
				SchoolName        string   `json:"school_name"`
				SchoolAddress     *string  `json:"school_address"`
				City              string   `json:"city"`
				AboutSchool       *string  `json:"about_school"`
				PhotoURL          *string  `json:"photo_url"`
				SchoolPhone       *string  `json:"school_phone"`
				SchoolAdminLogin  *string  `json:"school_admin_login"`
				Specializations   []string `json:"specializations"`
				CreatedAt         *string  `json:"created_at"`
				UpdatedAt         *string  `json:"updated_at"`
				Rating            *float64 `json:"rating"`
				UntRank           float64  `json:"unt_rank"`
				EventScore        float64  `json:"event_score"`
				ParticipantPoints float64  `json:"participant_points"`
				AverageRatingRank float64  `json:"average_rating_rank"`
				OlympiadRank      float64  `json:"olympiad_rank"`
				TotalRating       float64  `json:"total_rating"`
			}{
				SchoolID:          school.SchoolID,
				UserID:            school.UserID,
				SchoolName:        school.SchoolName,
				SchoolAddress:     schoolAddress,
				City:              school.City,
				AboutSchool:       aboutSchool,
				PhotoURL:          photoURL,
				SchoolPhone:       schoolPhone,
				SchoolAdminLogin:  adminLogin,
				Specializations:   specializations,
				CreatedAt:         createdAt,
				UpdatedAt:         updatedAt,
				Rating:            rating,
				UntRank:           untRank,
				EventScore:        eventScore,
				ParticipantPoints: participantPoints,
				AverageRatingRank: averageRatingRank,
				OlympiadRank:      olympiadRank,
				TotalRating:       totalRating,
			})
		}

		if err = rows.Err(); err != nil {
			log.Println("Error during iteration:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error during iteration"})
			return
		}

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
		var specializationsJSON sql.NullString
		var aboutSchool sql.NullString

		query := `
            SELECT 
                school_id, 
                user_id, 
                school_name, 
                school_address, 
                city, 
                photo_url, 
                school_email, 
                school_phone, 
                school_admin_login, 
                specializations,
                about_school
            FROM Schools 
            WHERE school_id = ?`

		err = db.QueryRow(query, schoolID).Scan(
			&school.SchoolID,
			&school.UserID,
			&school.SchoolName,
			&school.SchoolAddress,
			&school.City,
			&school.PhotoURL,
			&school.SchoolEmail,
			&school.SchoolPhone,
			&school.SchoolAdminLogin,
			&specializationsJSON,
			&aboutSchool,
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
		if specializationsJSON.Valid && specializationsJSON.String != "" {
			err = json.Unmarshal([]byte(specializationsJSON.String), &school.Specializations)
			if err != nil {
				log.Println("Error unmarshaling specializations:", err)
				school.Specializations = []string{}
			}
		} else {
			school.Specializations = []string{}
		}

		// Prepare response struct for JSON serialization
		responseSchool := struct {
			SchoolID         int      `json:"school_id"`
			UserID           int      `json:"user_id"`
			SchoolName       string   `json:"school_name"`
			SchoolAddress    *string  `json:"school_address"`
			City             string   `json:"city"`
			PhotoURL         *string  `json:"photo_url"`
			SchoolEmail      *string  `json:"school_email"`
			SchoolPhone      *string  `json:"school_phone"`
			SchoolAdminLogin *string  `json:"school_admin_login"`
			Specializations  []string `json:"specializations"`
			AboutSchool      *string  `json:"about_school"`
		}{
			SchoolID:        school.SchoolID,
			UserID:          school.UserID,
			SchoolName:      school.SchoolName,
			City:            school.City,
			Specializations: school.Specializations,
		}

		// Assign nullable fields
		if school.SchoolAddress.Valid {
			responseSchool.SchoolAddress = &school.SchoolAddress.String
		}
		if school.PhotoURL.Valid && school.PhotoURL.String != "" {
			responseSchool.PhotoURL = &school.PhotoURL.String
		}
		if school.SchoolEmail.Valid {
			responseSchool.SchoolEmail = &school.SchoolEmail.String
		}
		if school.SchoolPhone.Valid {
			responseSchool.SchoolPhone = &school.SchoolPhone.String
		}
		if school.SchoolAdminLogin.Valid {
			responseSchool.SchoolAdminLogin = &school.SchoolAdminLogin.String
		}
		if aboutSchool.Valid {
			responseSchool.AboutSchool = &aboutSchool.String
		}

		// Return response with school data
		response := struct {
			School interface{} `json:"school"`
		}{
			School: responseSchool,
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
func (sc *SchoolController) GetTopSchoolsByRating(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, models.Error{Message: "Method not allowed"})
			return
		}

		query := `SELECT school_id, user_id, school_name, school_address, city, about_school, photo_url, 
                         school_email, school_phone, school_admin_login, specializations, 
                         created_at, updated_at FROM Schools`
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Ошибка при получении списка школ: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при получении списка школ"})
			return
		}
		defer rows.Close()

		var schools []models.School
		for rows.Next() {
			var school models.School
			var specializationsJSON sql.NullString

			err := rows.Scan(
				&school.SchoolID,
				&school.UserID,
				&school.SchoolName,
				&school.SchoolAddress,
				&school.City,
				&school.AboutSchool,
				&school.PhotoURL,
				&school.SchoolEmail,
				&school.SchoolPhone,
				&school.SchoolAdminLogin,
				&specializationsJSON,
				&school.CreatedAt,
				&school.UpdatedAt,
			)
			if err != nil {
				log.Printf("Ошибка при сканировании данных школы: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при сканировании данных школы"})
				return
			}

			if specializationsJSON.Valid {
				err = json.Unmarshal([]byte(specializationsJSON.String), &school.Specializations)
				if err != nil {
					school.Specializations = []string{}
				}
			}

			schools = append(schools, school)
		}

		if err = rows.Err(); err != nil {
			log.Printf("Ошибка во время итерации: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка во время итерации"})
			return
		}

		type SchoolWithRating struct {
			SchoolID          int      `json:"school_id"`
			UserID            int      `json:"user_id"`
			SchoolName        string   `json:"school_name"`
			SchoolAddress     *string  `json:"school_address"`
			City              string   `json:"city"`
			AboutSchool       *string  `json:"about_school"`
			PhotoURL          *string  `json:"photo_url"`
			SchoolPhone       *string  `json:"school_phone"`
			SchoolAdminLogin  *string  `json:"school_admin_login"`
			Specializations   []string `json:"specializations"`
			CreatedAt         *string  `json:"created_at"`
			UpdatedAt         *string  `json:"updated_at"`
			Rating            *float64 `json:"rating"`
			UntRank           float64  `json:"unt_rank"`
			EventScore        float64  `json:"event_score"`
			ParticipantPoints float64  `json:"participant_points"`
			AverageRatingRank float64  `json:"average_rating_rank"`
			OlympiadRank      float64  `json:"olympiad_rank"`
			TotalRating       float64  `json:"total_rating"`
			Rank              int      `json:"rank"`
		}

		var schoolsWithRating []SchoolWithRating
		src := &SchoolRatingController{}

		for _, school := range schools {
			schoolWithRating := SchoolWithRating{
				SchoolID:   school.SchoolID,
				UserID:     school.UserID,
				SchoolName: school.SchoolName,
				City:       school.City,
			}

			var schoolPhone *string
			if school.SchoolPhone.Valid {
				schoolPhone = &school.SchoolPhone.String
			}
			schoolWithRating.SchoolPhone = schoolPhone

			var adminLogin *string
			if school.SchoolAdminLogin.Valid {
				adminLogin = &school.SchoolAdminLogin.String
			}
			schoolWithRating.SchoolAdminLogin = adminLogin

			var schoolAddress *string
			if school.SchoolAddress.Valid {
				schoolAddress = &school.SchoolAddress.String
			}
			schoolWithRating.SchoolAddress = schoolAddress

			var aboutSchool *string
			if school.AboutSchool.Valid {
				aboutSchool = &school.AboutSchool.String
			}
			schoolWithRating.AboutSchool = aboutSchool

			var photoURL *string
			if school.PhotoURL.Valid && school.PhotoURL.String != "" {
				photoURL = &school.PhotoURL.String
			}
			schoolWithRating.PhotoURL = photoURL

			var createdAt *string
			if school.CreatedAt.Valid {
				createdAt = &school.CreatedAt.String
			}
			schoolWithRating.CreatedAt = createdAt

			var updatedAt *string
			if school.UpdatedAt.Valid {
				updatedAt = &school.UpdatedAt.String
			}
			schoolWithRating.UpdatedAt = updatedAt

			schoolWithRating.Specializations = school.Specializations

			untRank, err := src.getUNTRank(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении UNT рейтинга для школы %d: %v", school.SchoolID, err)
				untRank = 0.0
			}
			schoolWithRating.UntRank = math.Round(untRank*100) / 100

			eventScore, err := src.getEventScore(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении счета событий для школы %d: %v", school.SchoolID, err)
				eventScore = 0.0
			}
			schoolWithRating.EventScore = math.Round(eventScore*100) / 100

			participantPoints, err := src.getParticipantPoints(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении очков участников для школы %d: %v", school.SchoolID, err)
				participantPoints = 0.0
			}
			schoolWithRating.ParticipantPoints = math.Round(participantPoints*100) / 100

			averageRatingRank, err := src.getAverageRatingRank(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении рейтинга отзывов для школы %d: %v", school.SchoolID, err)
				averageRatingRank = 0.0
			}
			schoolWithRating.AverageRatingRank = math.Round(averageRatingRank*100) / 100

			var avgRating sql.NullFloat64
			ratingQuery := `SELECT AVG(rating) FROM Reviews WHERE school_id = ?`
			err = db.QueryRow(ratingQuery, school.SchoolID).Scan(&avgRating)
			if err != nil && err != sql.ErrNoRows {
				log.Printf("Ошибка при получении среднего рейтинга для школы %d: %v", school.SchoolID, err)
			}

			var rating *float64
			if avgRating.Valid {
				roundedRating := math.Round(avgRating.Float64*100) / 100
				rating = &roundedRating
			}
			schoolWithRating.Rating = rating

			olympiadRank, err := src.getOlympiadRank(db, int64(school.SchoolID))
			if err != nil {
				log.Printf("Ошибка при получении олимпиадного рейтинга для школы %d: %v", school.SchoolID, err)
				olympiadRank = 0.0
			}
			schoolWithRating.OlympiadRank = math.Round(olympiadRank*100) / 100

			totalRating := eventScore + participantPoints + averageRatingRank + untRank + olympiadRank
			schoolWithRating.TotalRating = math.Round(totalRating*100) / 100

			schoolsWithRating = append(schoolsWithRating, schoolWithRating)
		}

		sort.Slice(schoolsWithRating, func(i, j int) bool {
			return schoolsWithRating[i].TotalRating > schoolsWithRating[j].TotalRating
		})

		topSchools := schoolsWithRating
		if len(topSchools) > 5 {
			topSchools = topSchools[:5]
		}

		for i := range topSchools {
			topSchools[i].Rank = i + 1
		}

		utils.ResponseJSON(w, topSchools)
	}
}
func getAllSchools(db *sql.DB) ([]models.School, error) {
	query := `
		SELECT school_id, school_name, city 
		FROM Schools 
		ORDER BY school_id
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schools []models.School

	for rows.Next() {
		var school models.School

		err := rows.Scan(&school.SchoolID, &school.SchoolName, &school.City)
		if err != nil {
			return nil, err
		}
		schools = append(schools, school)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return schools, nil
}
