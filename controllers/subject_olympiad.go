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

type SubjectOlympiadController struct{}

// Метод для создания олимпиады по предмету
func (c *SubjectOlympiadController) CreateSubjectOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var error models.Error

		// Step 1: Verify user is a superadmin
		userID, err := utils.VerifyToken(r) // Возвращает только userID (int)
		if err != nil {
			error.Message = "Invalid token or not a superadmin."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Получаем роль пользователя из базы данных, используя userID
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil || userRole != "superadmin" {
			error.Message = "Only superadmin can create olympiads."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Step 2: Parse the multipart form with a reasonable size limit
		err = r.ParseMultipartForm(10 << 20) // 10MB limit
		if err != nil {
			error.Message = "Error parsing form data"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		// Get form values and convert types as needed
		schoolAdminID, err := strconv.Atoi(r.FormValue("school_admin_id"))
		if err != nil {
			error.Message = "Invalid school_admin_id format"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Error converting school_admin_id: %v", err)
			return
		}

		olympiad := models.SubjectOlympiad{
			SubjectName:   r.FormValue("subject_name"),
			EventName:     r.FormValue("event_name"),
			Date:          r.FormValue("date"),
			Duration:      r.FormValue("duration"),
			Description:   r.FormValue("description"),
			City:          r.FormValue("city"),
			SchoolAdminID: schoolAdminID, // Now using the converted integer
		}

		// Validate required fields
		if olympiad.SubjectName == "" || olympiad.EventName == "" || olympiad.Date == "" || olympiad.City == "" {
			error.Message = "Subject name, event name, date, and city are required fields."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			log.Printf("Missing required fields: %v", olympiad)
			return
		}

		// Get the file from form data
		file, _, err := r.FormFile("photo_url")
		if err != nil {
			log.Println("Error reading file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
			return
		}
		defer file.Close()

		// Generate unique filename and upload to S3
		uniqueFileName := fmt.Sprintf("olympiad-%d-%d.jpg", time.Now().Unix())
		photoURL, err := utils.UploadFileToS3(file, uniqueFileName, false)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload photo"})
			log.Println("Error uploading file:", err)
			return
		}

		olympiad.PhotoURL = photoURL

		// Insert into database
		query := `INSERT INTO subject_olympiads (subject_name, event_name, date, duration, description, photo_url, city, school_admin_id) 
                  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = db.Exec(query, olympiad.SubjectName, olympiad.EventName, olympiad.Date, olympiad.Duration, olympiad.Description, olympiad.PhotoURL, olympiad.City, olympiad.SchoolAdminID)
		if err != nil {
			log.Println("Error inserting olympiad:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create olympiad"})
			return
		}

		utils.ResponseJSON(w, olympiad)
	}
}
func (c *SubjectOlympiadController) RegisterStudentToOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var registrationData struct {
			OlympiadID int `json:"olympiad_id"` // ID олимпиады
		}

		var error models.Error

		// Декодируем запрос, исключая student_id, так как мы будем его получать из токена
		err := json.NewDecoder(r.Body).Decode(&registrationData)
		if err != nil {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Получаем userID (student_id) из токена
		userID, err := utils.VerifyToken(r) // Теперь вернулся только userID (тип int)
		if err != nil {
			error.Message = "Invalid token or not a student."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Проверка, что олимпиада существует
		var olympiadExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM subject_olympiads WHERE id = ?)", registrationData.OlympiadID).Scan(&olympiadExists)
		if err != nil || !olympiadExists {
			error.Message = "Olympiad not found."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверка, что студент уже зарегистрирован на олимпиаду
		var alreadyRegistered bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student_olympiads WHERE student_id = ? AND olympiad_id = ?)", userID, registrationData.OlympiadID).Scan(&alreadyRegistered)
		if err != nil || alreadyRegistered {
			error.Message = "Student is already registered for this olympiad."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Регистрируем студента на олимпиаду
		query := `INSERT INTO student_olympiads (student_id, olympiad_id) VALUES (?, ?)`
		_, err = db.Exec(query, userID, registrationData.OlympiadID)
		if err != nil {
			log.Println("Error registering student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to register student"})
			return
		}

		// Ответ с подтверждением
		utils.ResponseJSON(w, map[string]string{"message": "Student successfully registered for the olympiad."})
	}
}
// Метод для получения всех олимпиад для студентов
func (c *SubjectOlympiadController) GetAllOlympiadsForStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка токена и роль студента
		_, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Запрос для получения всех олимпиад
		rows, err := db.Query("SELECT id, subject_name, event_name, date, duration, description, photo_url, city FROM subject_olympiads")
		if err != nil {
			log.Println("Error fetching olympiads:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve olympiads"})
			return
		}
		defer rows.Close()

		var olympiads []models.SubjectOlympiad

		// Проходим по всем строкам в результатах запроса
		for rows.Next() {
			var olympiad models.SubjectOlympiad
			err := rows.Scan(&olympiad.ID, &olympiad.SubjectName, &olympiad.EventName, &olympiad.Date, &olympiad.Duration, &olympiad.Description, &olympiad.PhotoURL, &olympiad.City)
			if err != nil {
				log.Println("Error scanning olympiad data:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning olympiad data"})
				return
			}

			olympiads = append(olympiads, olympiad)
		}

		// Проверка на ошибки после итерации
		if err = rows.Err(); err != nil {
			log.Println("Error during iteration:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error during iteration"})
			return
		}

		// Возвращаем список всех олимпиад для студентов в формате JSON
		utils.ResponseJSON(w, olympiads)
	}
}
