package controllers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type OlympiadController struct {
}

func (oc *OlympiadController) CreateOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		err = r.ParseMultipartForm(10 << 20) // до 10 MB
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Failed to parse form"})
			return
		}

		// Получение текстовых полей
		studentID, _ := strconv.Atoi(r.FormValue("student_id"))
		place, _ := strconv.Atoi(r.FormValue("olympiad_place"))
		level := r.FormValue("level")
		name := r.FormValue("olympiad_name")
		date := r.FormValue("date") // Get date from form data

		// Validate date format (YYYY-MM-DD)
		if date != "" {
			_, err := time.Parse("2006-01-02", date)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date format. Use YYYY-MM-DD"})
				return
			}
		} else {
			// Default to current date if not provided
			date = time.Now().Format("2006-01-02")
		}

		var score int
		switch place {
		case 1:
			score = 50
		case 2:
			score = 30
		case 3:
			score = 20
		default:
			score = 0
		}

		// Проверка уровня олимпиады
		if level != "city" && level != "region" && level != "republican" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid level"})
			return
		}

		// Проверка роли и принадлежности к школе
		var schoolID sql.NullInt64
		var role string
		err = db.QueryRow("SELECT school_id, role FROM users WHERE id = ?", userID).Scan(&schoolID, &role)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user info"})
			return
		}
		var studentSchoolID int
		err = db.QueryRow("SELECT school_id FROM student WHERE student_id = ?", studentID).Scan(&studentSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Student not found"})
			return
		}
		if role == "schooladmin" && (!schoolID.Valid || int(schoolID.Int64) != studentSchoolID) {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Student does not belong to your school"})
			return
		}

		// Загрузка файла
		file, handler, err := r.FormFile("document")
		var documentURL string
		if err == nil {
			defer file.Close()
			fileExt := filepath.Ext(handler.Filename)
			fileName := fmt.Sprintf("olympiads/%d_%s%s", studentID, time.Now().Format("20060102150405"), fileExt)

			// Загружаем файл в S3 вместо локального хранилища
			documentURL, err = utils.UploadFileToS3(file, fileName, "olympiaddoc") // Указываем полный путь к функции
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: fmt.Sprintf("Failed to upload document to S3: %v", err)})
				return
			}
		} else {
			documentURL = ""
		}

		// Вставка в БД с учетом поля date
		query := `INSERT INTO Olympiads (student_id, olympiad_place, score, school_id, level, olympiad_name, document_url, date) 
                   VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = db.Exec(query, studentID, place, score, studentSchoolID, level, name, documentURL, date)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to insert olympiad"})
			return
		}

		utils.ResponseJSON(w, "Olympiad created with document successfully")
	}
}
func (oc *OlympiadController) GetOlympiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем userID из токена для проверки прав доступа
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Получаем роль и школу пользователя
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user information"})
			return
		}

		studentID := r.URL.Query().Get("student_id")
		level := r.URL.Query().Get("level")
		schoolIDParam := r.URL.Query().Get("school_id")

		var queryArgs []interface{}

		// Базовая часть запроса с выборкой всех полей включая grade, letter и school_name
		baseQuery := `SELECT 
						Olympiads.olympiad_id, Olympiads.student_id, Olympiads.olympiad_place, Olympiads.score, 
						Olympiads.school_id, Olympiads.level, Olympiads.olympiad_name, Olympiads.document_url,
						Olympiads.date, 
						student.first_name, student.last_name, student.patronymic, 
						student.grade, student.letter,
						Schools.school_name
					FROM Olympiads
					JOIN student ON Olympiads.student_id = student.student_id
					LEFT JOIN Schools ON Olympiads.school_id = Schools.school_id`

		// Создаем условия запроса на основе параметров и роли пользователя
		whereConditions := []string{}

		// Для schooladmin показываем только данные из его школы
		if userRole == "schooladmin" && userSchoolID.Valid {
			whereConditions = append(whereConditions, "Olympiads.school_id = ?")
			queryArgs = append(queryArgs, userSchoolID.Int64)
		}

		// Добавляем фильтр по studentID, если указан
		if studentID != "" {
			whereConditions = append(whereConditions, "Olympiads.student_id = ?")
			queryArgs = append(queryArgs, studentID)
		}

		// Добавляем фильтр по level, если указан
		if level != "" {
			whereConditions = append(whereConditions, "Olympiads.level = ?")
			queryArgs = append(queryArgs, level)
		}

		// Добавляем фильтр по school_id, если указан и пользователь superadmin
		if schoolIDParam != "" && userRole == "superadmin" {
			schoolID, err := strconv.Atoi(schoolIDParam)
			if err == nil {
				whereConditions = append(whereConditions, "Olympiads.school_id = ?")
				queryArgs = append(queryArgs, schoolID)
			}
		}

		// Строим полный запрос с условиями WHERE
		query := baseQuery
		if len(whereConditions) > 0 {
			query = baseQuery + " WHERE " + strings.Join(whereConditions, " AND ")
		}

		// Выполняем запрос
		rows, err := db.Query(query, queryArgs...)
		if err != nil {
			log.Printf("Error fetching Olympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch Olympiad records"})
			return
		}
		defer rows.Close()

		// Расширенная структура для ответа с полями Grade, Letter и SchoolName
		type OlympiadWithExtendedInfo struct {
			models.Olympiads
			SchoolName string `json:"school_name"`
		}

		var olympiads []OlympiadWithExtendedInfo

		for rows.Next() {
			var olympiad OlympiadWithExtendedInfo
			var olympiadName, documentURL, date, schoolName sql.NullString
			var grade sql.NullInt64
			var letter sql.NullString

			err := rows.Scan(
				&olympiad.OlympiadID,
				&olympiad.StudentID,
				&olympiad.OlympiadPlace,
				&olympiad.Score,
				&olympiad.SchoolID,
				&olympiad.Level,
				&olympiadName,
				&documentURL,
				&date,
				&olympiad.FirstName,
				&olympiad.LastName,
				&olympiad.Patronymic,
				&grade,
				&letter,
				&schoolName,
			)
			if err != nil {
				log.Printf("Error scanning Olympiad record: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing Olympiad data"})
				return
			}

			if olympiadName.Valid {
				olympiad.OlympiadName = olympiadName.String
			}
			if documentURL.Valid {
				olympiad.DocumentURL = documentURL.String
			}
			if date.Valid {
				olympiad.Date = date.String
			}
			if grade.Valid {
				olympiad.Grade = int(grade.Int64)
			}
			if letter.Valid {
				olympiad.Letter = letter.String
			}
			if schoolName.Valid {
				olympiad.SchoolName = schoolName.String
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
		// Проверка авторизации
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Получаем ID олимпиады из тела запроса или параметров URL
		var olympiadID int

		// Сначала проверяем, есть ли ID в параметрах URL
		vars := mux.Vars(r)
		if idStr, ok := vars["olympiad_id"]; ok { // Изменено с "id" на "olympiad_id"
			var errAtoi error
			olympiadID, errAtoi = strconv.Atoi(idStr)
			if errAtoi != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID in URL"})
				return
			}
		} else {
			// Если ID нет в URL, проверяем в параметрах запроса
			idStr := r.URL.Query().Get("olympiad_id")
			if idStr != "" {
				var errAtoi error
				olympiadID, errAtoi = strconv.Atoi(idStr)
				if errAtoi != nil {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID in query parameter"})
					return
				}
			} else {
				// Проверяем, есть ли ID в form-data
				if err := r.ParseForm(); err == nil {
					if idStr := r.FormValue("olympiad_id"); idStr != "" {
						var errAtoi error
						olympiadID, errAtoi = strconv.Atoi(idStr)
						if errAtoi != nil {
							utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid olympiad ID in form data"})
							return
						}
					}
				}

				// Если ID олимпиады всё еще нет, пытаемся получить из JSON-данных
				if olympiadID == 0 {
					var olympiadIDFromBody struct {
						OlympiadID int `json:"olympiad_id"`
					}

					// Сначала считываем тело запроса в буфер, чтобы не "израсходовать" его
					bodyBytes, errRead := io.ReadAll(r.Body)
					if errRead != nil {
						utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Failed to read request body"})
						return
					}

					// Восстанавливаем тело запроса для дальнейшего использования
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					// Пытаемся извлечь olympiad_id из тела
					errUnmarshal := json.Unmarshal(bodyBytes, &olympiadIDFromBody)
					if errUnmarshal != nil || olympiadIDFromBody.OlympiadID == 0 {
						utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Olympiad ID is required"})
						return
					}

					olympiadID = olympiadIDFromBody.OlympiadID
				}
			}
		}

		// Финальная проверка ID олимпиады
		if olympiadID == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Olympiad ID is required"})
			return
		}

		// Разбор данных из запроса
		var updateData struct {
			OlympiadID    int    `json:"olympiad_id,omitempty"`
			StudentID     int    `json:"student_id"`
			OlympiadPlace int    `json:"olympiad_place"`
			Level         string `json:"level"`
			OlympiadName  string `json:"olympiad_name"`
			DocumentURL   string `json:"document_url"`
			Date          string `json:"date,omitempty"`
			SchoolID      int    `json:"school_id"`
			Grade         int    `json:"grade"`
			Letter        string `json:"letter"`
		}

		// Переменная для хранения URL документа
		var documentURL string

		// Для запросов с form-data
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			// Обрабатываем multipart/form-data
			if err := r.ParseMultipartForm(10 << 20); err == nil { // Поддержка до 10MB файлов
				// Получаем значения из формы
				studentIDStr := r.FormValue("student_id")
				if studentIDStr != "" {
					updateData.StudentID, _ = strconv.Atoi(studentIDStr)
				}

				olympiadPlaceStr := r.FormValue("olympiad_place")
				if olympiadPlaceStr != "" {
					updateData.OlympiadPlace, _ = strconv.Atoi(olympiadPlaceStr)
				}

				updateData.Level = r.FormValue("level")
				updateData.OlympiadName = r.FormValue("olympiad_name")
				updateData.DocumentURL = r.FormValue("document_url")
				updateData.Date = r.FormValue("date")

				schoolIDStr := r.FormValue("school_id")
				if schoolIDStr != "" {
					updateData.SchoolID, _ = strconv.Atoi(schoolIDStr)
				}

				gradeStr := r.FormValue("grade")
				if gradeStr != "" {
					updateData.Grade, _ = strconv.Atoi(gradeStr)
				}

				updateData.Letter = r.FormValue("letter")

				// Обработка загруженного файла
				file, handler, errFile := r.FormFile("document")
				if errFile == nil && file != nil {
					defer file.Close()

					// Генерируем уникальное имя для файла
					timestamp := time.Now().Unix()
					filename := fmt.Sprintf("olympiaddoc_%d_%s", timestamp, handler.Filename)

					// Загружаем файл в S3
					uploadedURL, errUpload := utils.UploadFileToS3(file, filename, "olympiaddoc")
					if errUpload != nil {
						utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload document: " + errUpload.Error()})
						return
					}

					documentURL = uploadedURL
				}
			}
		} else if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			// Обрабатываем application/x-www-form-urlencoded
			if err := r.ParseForm(); err == nil {
				studentIDStr := r.FormValue("student_id")
				if studentIDStr != "" {
					updateData.StudentID, _ = strconv.Atoi(studentIDStr)
				}

				olympiadPlaceStr := r.FormValue("olympiad_place")
				if olympiadPlaceStr != "" {
					updateData.OlympiadPlace, _ = strconv.Atoi(olympiadPlaceStr)
				}

				updateData.Level = r.FormValue("level")
				updateData.OlympiadName = r.FormValue("olympiad_name")
				updateData.DocumentURL = r.FormValue("document_url")
				updateData.Date = r.FormValue("date")

				schoolIDStr := r.FormValue("school_id")
				if schoolIDStr != "" {
					updateData.SchoolID, _ = strconv.Atoi(schoolIDStr)
				}

				gradeStr := r.FormValue("grade")
				if gradeStr != "" {
					updateData.Grade, _ = strconv.Atoi(gradeStr)
				}

				updateData.Letter = r.FormValue("letter")
			}
		} else {
			// Для запросов с JSON
			// Используем новый экземпляр Body, так как мы уже могли прочитать его выше
			errDecode := json.NewDecoder(r.Body).Decode(&updateData)
			if errDecode != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
				return
			}
		}

		// Если olympiadID еще не установлен и присутствует в теле запроса, используем его
		if olympiadID == 0 && updateData.OlympiadID > 0 {
			olympiadID = updateData.OlympiadID
		}

		// Проверка роли пользователя и школы
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user information"})
			return
		}

		// Проверяем существование олимпиады и получаем текущую школу
		var currentSchoolID int
		var currentStudentID int
		err = db.QueryRow("SELECT school_id, student_id FROM Olympiads WHERE olympiad_id = ?", olympiadID).Scan(&currentSchoolID, &currentStudentID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Olympiad not found"})
			return
		}

		// Проверка доступа: schooladmin может редактировать только своей школы
		if userRole == "schooladmin" && (!userSchoolID.Valid || int(userSchoolID.Int64) != currentSchoolID) {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only edit olympiads for your school"})
			return
		}

		// Если меняется ученик, проверяем принадлежность к школе
		if updateData.StudentID != 0 && updateData.StudentID != currentStudentID {
			var studentSchoolID int
			err = db.QueryRow("SELECT school_id FROM student WHERE student_id = ?", updateData.StudentID).Scan(&studentSchoolID)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Student not found"})
				return
			}

			// Школьный админ может назначать только учеников своей школы
			if userRole == "schooladmin" && (!userSchoolID.Valid || int(userSchoolID.Int64) != studentSchoolID) {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Student does not belong to your school"})
				return
			}
		}

		// Проверка уровня олимпиады
		if updateData.Level != "" && updateData.Level != "city" && updateData.Level != "region" && updateData.Level != "republican" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid level"})
			return
		}

		// Вычисление очков на основе места
		var score int
		switch updateData.OlympiadPlace {
		case 1:
			score = 50
		case 2:
			score = 30
		case 3:
			score = 20
		default:
			score = 0
		}

		// Проверка формата даты, если она указана
		if updateData.Date != "" {
			_, err := time.Parse("2006-01-02", updateData.Date)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid date format. Use YYYY-MM-DD"})
				return
			}
		}

		// Проверяем, какие поля нужно обновить
		var queryParts []string
		var queryParams []interface{}

		// Добавляем только те поля, которые были указаны в запросе
		if updateData.StudentID != 0 {
			queryParts = append(queryParts, "student_id = ?")
			queryParams = append(queryParams, updateData.StudentID)
		}

		if updateData.OlympiadPlace != 0 {
			queryParts = append(queryParts, "olympiad_place = ?")
			queryParams = append(queryParams, updateData.OlympiadPlace)

			queryParts = append(queryParts, "score = ?")
			queryParams = append(queryParams, score)
		}

		if updateData.Level != "" {
			queryParts = append(queryParts, "level = ?")
			queryParams = append(queryParams, updateData.Level)
		}

		if updateData.OlympiadName != "" {
			queryParts = append(queryParts, "olympiad_name = ?")
			queryParams = append(queryParams, updateData.OlympiadName)
		}

		// Обновляем document_url, если загружен новый файл или указан в данных
		if documentURL != "" {
			// Новый файл был загружен
			queryParts = append(queryParts, "document_url = ?")
			queryParams = append(queryParams, documentURL)
		} else if updateData.DocumentURL != "" {
			// URL документа был передан в данных
			queryParts = append(queryParts, "document_url = ?")
			queryParams = append(queryParams, updateData.DocumentURL)
		}

		if updateData.SchoolID != 0 {
			queryParts = append(queryParts, "school_id = ?")
			queryParams = append(queryParams, updateData.SchoolID)
		}

		if updateData.Date != "" {
			queryParts = append(queryParts, "date = ?")
			queryParams = append(queryParams, updateData.Date)
		}

		// Если нет полей для обновления, возвращаем ошибку
		if len(queryParts) == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "No fields to update"})
			return
		}

		// Формируем и выполняем запрос
		query := "UPDATE Olympiads SET " + strings.Join(queryParts, ", ") + " WHERE olympiad_id = ?"
		queryParams = append(queryParams, olympiadID)

		_, err = db.Exec(query, queryParams...)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update olympiad: " + err.Error()})
			return
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"message":     "Olympiad updated successfully",
			"olympiad_id": olympiadID,
		})
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

		// 3. Получаем список олимпиад для указанной школы с добавлением grade и letter
		query := `
			SELECT o.olympiad_id, o.student_id, s.first_name, s.last_name, o.olympiad_place, 
			       o.score, o.level, o.school_id, o.olympiad_name, o.document_url,
			       s.grade, s.letter
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

		// 4. Расширенная структура с полями grade и letter
		type OlympiadWithStudent struct {
			OlympiadID    int    `json:"olympiad_id"`
			StudentID     int    `json:"student_id"`
			FirstName     string `json:"first_name"`
			LastName      string `json:"last_name"`
			OlympiadPlace int    `json:"olympiad_place"`
			Score         int    `json:"score"`
			Level         string `json:"level"`
			SchoolID      int    `json:"school_id"`
			OlympiadName  string `json:"olympiad_name"`
			DocumentURL   string `json:"document_url"`
			Grade         int    `json:"grade"`
			Letter        string `json:"letter"`
		}

		// 5. Считываем данные из результата запроса
		olympiads := []OlympiadWithStudent{}
		for rows.Next() {
			var olympiad OlympiadWithStudent
			var olympiadName, documentURL, letter sql.NullString
			var grade sql.NullInt64

			err := rows.Scan(
				&olympiad.OlympiadID,
				&olympiad.StudentID,
				&olympiad.FirstName,
				&olympiad.LastName,
				&olympiad.OlympiadPlace,
				&olympiad.Score,
				&olympiad.Level,
				&olympiad.SchoolID,
				&olympiadName,
				&documentURL,
				&grade,
				&letter,
			)
			if err != nil {
				log.Printf("Error scanning olympiad row: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing olympiad data"})
				return
			}

			if olympiadName.Valid {
				olympiad.OlympiadName = olympiadName.String
			} else {
				olympiad.OlympiadName = ""
			}

			if documentURL.Valid {
				olympiad.DocumentURL = documentURL.String
			} else {
				olympiad.DocumentURL = ""
			}

			if grade.Valid {
				olympiad.Grade = int(grade.Int64)
			} else {
				olympiad.Grade = 0
			}

			if letter.Valid {
				olympiad.Letter = letter.String
			} else {
				olympiad.Letter = ""
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
		utils.ResponseJSON(w, olympiads)
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
func (soc *SubjectOlympiadController) GetAllSubOlypmiad(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем userID из токена для проверки прав доступа
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Получаем роль пользователя
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Printf("Error fetching user information: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user information"})
			return
		}

		// Получаем параметры запроса
		studentID := r.URL.Query().Get("student_id")
		schoolIDParam := r.URL.Query().Get("school_id")
		subject := r.URL.Query().Get("subject")
		level := r.URL.Query().Get("level")

		var queryArgs []interface{}
		// Базовый SQL-запрос
		baseQuery := `
			SELECT 
				o.olympiad_id, o.student_id, o.olympiad_place, o.score, 
				o.school_id, o.level, o.olympiad_name, o.document_url, o.date, o.subject,
				s.first_name, s.last_name, s.patronymic, s.grade, s.letter,
				sc.school_name
			FROM SubjectOlympiads o
			JOIN student s ON o.student_id = s.student_id
			LEFT JOIN Schools sc ON o.school_id = sc.school_id`

		// Условия фильтрации
		whereConditions := []string{}

		// Ограничения по роли
		if userRole == "student" {
			// Студент видит только свои олимпиады
			var studentIDFromDB int
			err = db.QueryRow("SELECT student_id FROM student WHERE user_id = ?", userID).Scan(&studentIDFromDB)
			if err != nil {
				log.Printf("Error fetching student ID: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch student information"})
				return
			}
			whereConditions = append(whereConditions, "o.student_id = ?")
			queryArgs = append(queryArgs, studentIDFromDB)
		} else if userRole != "superadmin" && userRole != "schooladmin" && userRole != "user" {
			// Проверка на допустимую роль
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Invalid user role"})
			return
		}

		// Добавляем фильтры из параметров запроса
		if studentID != "" && userRole != "student" { // Студент не может фильтровать по другим student_id
			whereConditions = append(whereConditions, "o.student_id = ?")
			queryArgs = append(queryArgs, studentID)
		}

		if schoolIDParam != "" { // Любой авторизованный пользователь может фильтровать по school_id
			whereConditions = append(whereConditions, "o.school_id = ?")
			queryArgs = append(queryArgs, schoolIDParam)
		}

		if subject != "" {
			whereConditions = append(whereConditions, "o.subject = ?")
			queryArgs = append(queryArgs, subject)
		}

		if level != "" {
			whereConditions = append(whereConditions, "o.level = ?")
			queryArgs = append(queryArgs, level)
		}

		// Формируем полный запрос
		query := baseQuery
		if len(whereConditions) > 0 {
			query += " WHERE " + strings.Join(whereConditions, " AND ")
		}

		// Выполняем запрос
		rows, err := db.Query(query, queryArgs...)
		if err != nil {
			log.Printf("Error fetching SubjectOlympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch SubjectOlympiad records"})
			return
		}
		defer rows.Close()

		// Структура для ответа
		type SubjectOlympiadWithInfo struct {
			OlympiadID    int    `json:"olympiad_id"`
			StudentID     int    `json:"student_id"`
			OlympiadPlace int    `json:"olympiad_place"`
			Score         int    `json:"score"`
			SchoolID      int    `json:"school_id"`
			Level         string `json:"level"`
			OlympiadName  string `json:"olympiad_name"`
			DocumentURL   string `json:"document_url"`
			Date          string `json:"date"`
			Subject       string `json:"subject"`
			FirstName     string `json:"first_name"`
			LastName      string `json:"last_name"`
			Patronymic    string `json:"patronymic"`
			Grade         int    `json:"grade"`
			Letter        string `json:"letter"`
			SchoolName    string `json:"school_name"`
		}

		var olympiads []SubjectOlympiadWithInfo

		// Обрабатываем результаты
		for rows.Next() {
			var olympiad SubjectOlympiadWithInfo
			var olympiadName, documentURL, date, subject, letter, schoolName sql.NullString
			var grade sql.NullInt64

			err := rows.Scan(
				&olympiad.OlympiadID,
				&olympiad.StudentID,
				&olympiad.OlympiadPlace,
				&olympiad.Score,
				&olympiad.SchoolID,
				&olympiad.Level,
				&olympiadName,
				&documentURL,
				&date,
				&subject,
				&olympiad.FirstName,
				&olympiad.LastName,
				&olympiad.Patronymic,
				&grade,
				&letter,
				&schoolName,
			)
			if err != nil {
				log.Printf("Error scanning SubjectOlympiad record: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing SubjectOlympiad data"})
				return
			}

			// Присваиваем значения, если они валидны
			if olympiadName.Valid {
				olympiad.OlympiadName = olympiadName.String
			}
			if documentURL.Valid {
				olympiad.DocumentURL = documentURL.String
			}
			if date.Valid {
				olympiad.Date = date.String
			}
			if subject.Valid {
				olympiad.Subject = subject.String
			}
			if grade.Valid {
				olympiad.Grade = int(grade.Int64)
			}
			if letter.Valid {
				olympiad.Letter = letter.String
			}
			if schoolName.Valid {
				olympiad.SchoolName = schoolName.String
			}

			olympiads = append(olympiads, olympiad)
		}

		// Проверяем ошибки после обработки строк
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating SubjectOlympiad rows: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing SubjectOlympiad data"})
			return
		}

		// Возвращаем результат
		utils.ResponseJSON(w, olympiads)
	}
}
func (soc *SubjectOlympiadController) GetAllSubOlypmiadNamePicture(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем userID из токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Получаем роль пользователя
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Printf("Error fetching user role: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch user role"})
			return
		}

		// Получаем параметры запроса
		studentID := r.URL.Query().Get("student_id")
		var queryArgs []interface{}

		// Базовый SQL-запрос
		baseQuery := `
			SELECT o.subject_name, o.photo_url
			FROM SubjectOlympiads o
			JOIN student s ON o.student_id = s.student_id`

		// Условия фильтрации
		whereConditions := []string{}

		// Ограничения по роли
		if userRole == "student" {
			var studentIDFromDB int
			err = db.QueryRow("SELECT student_id FROM student WHERE user_id = ?", userID).Scan(&studentIDFromDB)
			if err != nil {
				log.Printf("Error fetching student ID: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch student information"})
				return
			}
			whereConditions = append(whereConditions, "o.student_id = ?")
			queryArgs = append(queryArgs, studentIDFromDB)
		} else if userRole != "superadmin" && userRole != "schooladmin" && userRole != "user" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Invalid user role"})
			return
		}

		// Фильтрация по student_id (только для не-студентов)
		if studentID != "" && userRole != "student" {
			whereConditions = append(whereConditions, "o.student_id = ?")
			queryArgs = append(queryArgs, studentID)
		}

		// Формируем полный SQL-запрос
		query := baseQuery
		if len(whereConditions) > 0 {
			query += " WHERE " + strings.Join(whereConditions, " AND ")
		}

		// Выполняем запрос
		rows, err := db.Query(query, queryArgs...)
		if err != nil {
			log.Printf("Error fetching SubjectOlympiad records: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch SubjectOlympiad records"})
			return
		}
		defer rows.Close()

		type SubjectOlympiadNamePicture struct {
			SubjectName string `json:"subject_name"`
			PhotoURL    string `json:"photo_url"`
		}

		var olympiads []SubjectOlympiadNamePicture

		for rows.Next() {
			var olympiad SubjectOlympiadNamePicture
			var subjectName, photoURL sql.NullString

			err := rows.Scan(&subjectName, &photoURL)
			if err != nil {
				log.Printf("Error scanning SubjectOlympiad record: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing SubjectOlympiad data"})
				return
			}

			if subjectName.Valid {
				olympiad.SubjectName = subjectName.String
			}
			if photoURL.Valid {
				olympiad.PhotoURL = photoURL.String
			}

			olympiads = append(olympiads, olympiad)
		}

		if err = rows.Err(); err != nil {
			log.Printf("Error iterating SubjectOlympiad rows: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing SubjectOlympiad data"})
			return
		}

		utils.ResponseJSON(w, olympiads)
	}
}
