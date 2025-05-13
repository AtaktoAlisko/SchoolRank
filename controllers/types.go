package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type TypeController struct{}
type UNTExam struct{}

func (c *UNTScoreController) CreateUNT(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Get userID from token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Check user role and school_id
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		// Step 3: Get school_id from URL parameters
		vars := mux.Vars(r)
		urlSchoolID, err := strconv.Atoi(vars["school_id"])
		if err != nil {
			log.Println("Ошибка при парсинге school_id из URL:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный school_id в URL"})
			return
		}

		// Step 4: Check permissions based on role
		if userRole == "superadmin" {
			// Superadmin can create UNT exams for any school
			log.Printf("Пользователь %d с ролью superadmin создаёт UNT для школы %d", userID, urlSchoolID)
		} else if userRole == "schooladmin" {
			// Schooladmin can only create UNT exams for their school
			if !userSchoolID.Valid || int(userSchoolID.Int64) != urlSchoolID {
				log.Printf("Пользователь %d с ролью schooladmin пытается создать UNT для не своей школы %d", userID, urlSchoolID)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав на создание UNT экзамена для этой школы"})
				return
			}
		} else {
			// Other roles don't have permission
			log.Printf("Пользователь %d с ролью %s пытается создать UNT", userID, userRole)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав на создание UNT экзамена"})
			return
		}

		// Step 5: Check if school exists
		var schoolExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", urlSchoolID).Scan(&schoolExists)
		if err != nil {
			log.Printf("Ошибка при проверке существования школы: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить наличие школы"})
			return
		}

		if !schoolExists {
			log.Printf("Школа с id %d не существует", urlSchoolID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Школа не найдена в нашей системе"})
			return
		}

		// Step 6: Process request based on Content-Type
		contentType := r.Header.Get("Content-Type")
		log.Printf("Получен запрос с Content-Type: %s", contentType)

		var untExam models.UNTExam
		untExam.SchoolID = urlSchoolID // Use the school_id from URL

		if strings.Contains(contentType, "multipart/form-data") {
			// Process multipart/form-data request
			err = r.ParseMultipartForm(10 << 20) // Max size 10MB
			if err != nil {
				log.Printf("Ошибка при парсинге multipart/form-data: %v", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Ошибка при обработке данных формы: " + err.Error()})
				return
			}

			// Debug info about received form data
			log.Printf("Получены поля формы: %v", r.MultipartForm.Value)
			if r.MultipartForm.File != nil {
				for k, v := range r.MultipartForm.File {
					if len(v) > 0 {
						log.Printf("Файл в форме: поле=%s, имя=%s, размер=%d", k, v[0].Filename, v[0].Size)
					} else {
						log.Printf("Файл в форме: поле=%s, пустой список файлов", k)
					}
				}
			} else {
				log.Println("MultipartForm.File является nil, файлы не были загружены")
			}

			// Get exam type
			untExam.ExamType = r.FormValue("exam_type")
			if untExam.ExamType == "" {
				untExam.ExamType = "regular" // Default value
			}

			// Convert to lowercase for consistency
			untExam.ExamType = strings.ToLower(untExam.ExamType)

			// Validate exam type
			if untExam.ExamType != "regular" && untExam.ExamType != "creative" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный тип экзамена. Допустимые значения: regular, creative"})
				return
			}

			// Get common fields
			untExam.Date = r.FormValue("date")
			untExam.StudentID, _ = strconv.Atoi(r.FormValue("student_id"))

			// Validate date
			if untExam.Date == "" {
				log.Println("Дата не указана")
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Необходимо указать дату сдачи экзамена"})
				return
			}

			// Check date format
			_, err = time.Parse("2006-01-02", untExam.Date)
			if err != nil {
				log.Printf("Некорректный формат даты: %v", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный формат даты. Используйте формат ГГГГ-ММ-ДД"})
				return
			}

			// Common fields for both exam types
			untExam.FirstSubject = r.FormValue("first_subject")
			untExam.SecondSubject = r.FormValue("second_subject")
			untExam.FirstSubjectScore, _ = strconv.Atoi(r.FormValue("first_subject_score"))
			untExam.SecondSubjectScore, _ = strconv.Atoi(r.FormValue("second_subject_score"))
			untExam.HistoryOfKazakhstan, _ = strconv.Atoi(r.FormValue("history_of_kazakhstan"))
			untExam.ReadingLiteracy, _ = strconv.Atoi(r.FormValue("reading_literacy"))

			// Get fields based on exam type
			if untExam.ExamType == "regular" {
				// Regular exam specific fields
				untExam.MathematicalLiteracy, _ = strconv.Atoi(r.FormValue("mathematical_literacy"))

				// Validate required fields for regular exam
				if untExam.FirstSubject == "" || untExam.SecondSubject == "" ||
					untExam.FirstSubjectScore < 0 || untExam.SecondSubjectScore < 0 ||
					untExam.HistoryOfKazakhstan < 0 || untExam.MathematicalLiteracy < 0 ||
					untExam.ReadingLiteracy < 0 || untExam.StudentID <= 0 {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректные или отсутствующие обязательные поля для Regular экзамена"})
					return
				}

				// Calculate total score for regular exam
				untExam.TotalScore = untExam.FirstSubjectScore + untExam.SecondSubjectScore +
					untExam.HistoryOfKazakhstan + untExam.MathematicalLiteracy +
					untExam.ReadingLiteracy
			} else {
				// Validate required fields for creative exam
				if untExam.FirstSubject == "" || untExam.SecondSubject == "" ||
					untExam.FirstSubjectScore < 0 || untExam.SecondSubjectScore < 0 ||
					untExam.HistoryOfKazakhstan < 0 || untExam.ReadingLiteracy < 0 ||
					untExam.StudentID <= 0 {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректные или отсутствующие обязательные поля для Creative экзамена"})
					return
				}

				// Calculate total score for creative exam
				untExam.TotalScore = untExam.FirstSubjectScore + untExam.SecondSubjectScore +
					untExam.HistoryOfKazakhstan + untExam.ReadingLiteracy
			}

			// Process file upload
			var documentURL string

			// Check if document_url was provided as a form field
			if urlFromForm := r.FormValue("document_url"); urlFromForm != "" {
				documentURL = urlFromForm
				log.Printf("Использую предоставленный URL документа: %s", documentURL)
			} else {
				// Try to find the file by specific field name
				file, handler, err := r.FormFile("document")

				// If not found, try alternative field names
				if err != nil {
					log.Printf("Не найден файл в поле 'document', пробуем альтернативные: %v", err)

					// List of possible file field names
					fileFieldNames := []string{"file", "document_file", "uploaded_file"}

					// Try all possible field names
					for _, fieldName := range fileFieldNames {
						file, handler, err = r.FormFile(fieldName)
						if err == nil {
							log.Printf("Файл найден в поле '%s': %s", fieldName, handler.Filename)
							break // Exit the loop if file found
						}
					}
				} else {
					log.Printf("Файл найден в поле 'document': %s", handler.Filename)
				}

				if err != nil {
					// Check all form keys directly
					foundFile := false
					if r.MultipartForm != nil && r.MultipartForm.File != nil {
						for fieldName, fileHeaders := range r.MultipartForm.File {
							if len(fileHeaders) > 0 {
								log.Printf("Нашли файл в неожиданном поле '%s': %s", fieldName, fileHeaders[0].Filename)
								file, err = fileHeaders[0].Open()
								if err == nil {
									handler = fileHeaders[0]
									foundFile = true
									break
								} else {
									log.Printf("Ошибка при открытии файла из поля '%s': %v", fieldName, err)
								}
							}
						}
					}

					if !foundFile {
						log.Printf("Файл не был загружен или не может быть прочитан: %v. Продолжаем без документа.", err)
						documentURL = ""
					}
				}

				// If file found, upload to S3
				if err == nil && file != nil && handler != nil {
					defer file.Close()

					// Create unique filename
					fileExt := filepath.Ext(handler.Filename)
					fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String(), fileExt)

					log.Printf("Подготовка к загрузке файла в S3 с именем %s", fileName)

					// Upload file to S3 bucket
					url, err := utils.UploadFileToS3(file, fileName, "olympiaddoc")
					if err != nil {
						log.Printf("Ошибка при загрузке файла в S3: %v", err)
						utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при загрузке файла в облачное хранилище: " + err.Error()})
						return
					}

					documentURL = url
					log.Printf("Файл успешно загружен в S3: %s", documentURL)
				}
			}

			untExam.DocumentURL = documentURL

		} else {
			// Process application/json request
			if err := json.NewDecoder(r.Body).Decode(&untExam); err != nil {
				log.Println("Ошибка декодирования запроса:", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный запрос"})
				return
			}

			// Convert exam_type to lowercase for consistency
			untExam.ExamType = strings.ToLower(untExam.ExamType)

			// Set default exam type if not provided
			if untExam.ExamType == "" {
				untExam.ExamType = "regular"
			}

			// Validate exam type
			if untExam.ExamType != "regular" && untExam.ExamType != "creative" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный тип экзамена. Допустимые значения: regular, creative"})
				return
			}

			// Check date presence
			if untExam.Date == "" {
				log.Println("Дата не указана в JSON запросе")
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Необходимо указать дату сдачи экзамена"})
				return
			}

			// Validate date format
			_, err = time.Parse("2006-01-02", untExam.Date)
			if err != nil {
				log.Printf("Некорректный формат даты в JSON запросе: %v", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный формат даты. Используйте формат ГГГГ-ММ-ДД"})
				return
			}

			// Override school_id from URL parameter
			untExam.SchoolID = urlSchoolID

			// Calculate total score based on exam type
			if untExam.ExamType == "regular" {
				untExam.TotalScore = untExam.FirstSubjectScore + untExam.SecondSubjectScore +
					untExam.HistoryOfKazakhstan + untExam.MathematicalLiteracy +
					untExam.ReadingLiteracy
			} else {
				untExam.TotalScore = untExam.FirstSubjectScore + untExam.SecondSubjectScore +
					untExam.HistoryOfKazakhstan + untExam.ReadingLiteracy
			}
		}

		// Step 7: Check if student exists
		if untExam.StudentID <= 0 {
			log.Printf("ID студента не указан или некорректен: %d", untExam.StudentID)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "ID студента не указан или некорректен"})
			return
		}

		var studentExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE student_id = ?)", untExam.StudentID).Scan(&studentExists)
		if err != nil {
			log.Printf("Ошибка при проверке существования студента: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить наличие студента"})
			return
		}

		if !studentExists {
			log.Printf("Студент с id %d не существует", untExam.StudentID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Студент не найден в нашей системе"})
			return
		}

		// Step 8: Insert record into database
		query := `INSERT INTO UNT_Exams (
			exam_type, first_subject, first_subject_score, second_subject, second_subject_score,
			history_of_kazakhstan, mathematical_literacy, reading_literacy, 
			total_score, student_id, school_id, document_url, date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := db.Exec(query,
			untExam.ExamType,
			untExam.FirstSubject,
			untExam.FirstSubjectScore,
			untExam.SecondSubject,
			untExam.SecondSubjectScore,
			untExam.HistoryOfKazakhstan,
			untExam.MathematicalLiteracy,
			untExam.ReadingLiteracy,
			untExam.TotalScore,
			untExam.StudentID,
			untExam.SchoolID,
			untExam.DocumentURL,
			untExam.Date,
		)

		if err != nil {
			log.Printf("Ошибка SQL: %v", err)

			// Specific check for foreign key violation
			if strings.Contains(err.Error(), "foreign key constraint fails") {
				if strings.Contains(err.Error(), "FK_School") {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный или отсутствующий school_id в таблице Schools"})
					return
				} else if strings.Contains(err.Error(), "student_id") {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный или отсутствующий student_id"})
					return
				}
			}

			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось создать UNT экзамен"})
			return
		}

		// Step 9: Get ID of the inserted record
		newID, _ := result.LastInsertId()
		untExam.ID = int(newID)

		// Step 10: Return success message
		utils.ResponseJSON(w, map[string]interface{}{
			"message":      fmt.Sprintf("UNT экзамен типа %s создан успешно", untExam.ExamType),
			"id":           newID,
			"exam_data":    untExam,
			"document_url": untExam.DocumentURL,
			"date":         untExam.Date,
		})
	}
}
func (c *UNTScoreController) GetUNTExams(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Get userID from token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Check user role and school_id
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		// Step 3: Get query parameters
		studentID := r.URL.Query().Get("student_id")
		examType := r.URL.Query().Get("exam_type")
		schoolID := r.URL.Query().Get("school_id")
		dateFrom := r.URL.Query().Get("date_from")
		dateTo := r.URL.Query().Get("date_to")

		// Step 4: Build query based on user role and parameters
		query := `SELECT id, exam_type, first_subject, first_subject_score, second_subject, 
				second_subject_score, history_of_kazakhstan, mathematical_literacy, 
				reading_literacy, total_score, student_id, school_id, document_url, date 
				FROM UNT_Exams WHERE 1=1`

		var args []interface{}

		// Apply user role restrictions
		if userRole == "schooladmin" {
			if !userSchoolID.Valid {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для просмотра экзаменов"})
				return
			}
			query += " AND school_id = ?"
			args = append(args, userSchoolID.Int64)
		}

		// Apply filters from query parameters
		if studentID != "" {
			query += " AND student_id = ?"
			args = append(args, studentID)
		}

		if examType != "" {
			query += " AND exam_type = ?"
			args = append(args, strings.ToLower(examType))
		}

		// School admins can't override their school_id
		if schoolID != "" && (userRole == "admin" || userRole == "moderator" || userRole == "superadmin") {
			query += " AND school_id = ?"
			args = append(args, schoolID)
		}

		if dateFrom != "" {
			query += " AND date >= ?"
			args = append(args, dateFrom)
		}

		if dateTo != "" {
			query += " AND date <= ?"
			args = append(args, dateTo)
		}

		// Add order by clause
		query += " ORDER BY date DESC"

		// Step 5: Execute query
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Ошибка при запросе экзаменов: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось получить список экзаменов"})
			return
		}
		defer rows.Close()

		// Step 6: Collect results
		var exams []models.UNTExam
		for rows.Next() {
			var exam models.UNTExam
			err := rows.Scan(
				&exam.ID, &exam.ExamType, &exam.FirstSubject, &exam.FirstSubjectScore,
				&exam.SecondSubject, &exam.SecondSubjectScore, &exam.HistoryOfKazakhstan,
				&exam.MathematicalLiteracy, &exam.ReadingLiteracy, &exam.TotalScore,
				&exam.StudentID, &exam.SchoolID, &exam.DocumentURL, &exam.Date,
			)
			if err != nil {
				log.Printf("Ошибка при сканировании строки: %v", err)
				continue
			}
			exams = append(exams, exam)
		}

		// Step 7: Return results
		utils.ResponseJSON(w, map[string]interface{}{
			"count": len(exams),
			"data":  exams,
		})
	}
}
func (c *UNTScoreController) UpdateUNTExam(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Get userID from token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get exam ID from URL parameter
		vars := mux.Vars(r)
		examID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный ID экзамена"})
			return
		}

		// Step 3: Check user role and school_id
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		// Step 4: Check if the user has permission to update this exam
		if userRole == "schooladmin" {
			if !userSchoolID.Valid {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для изменения этого экзамена"})
				return
			}

			// Check if exam belongs to this school
			var examSchoolID int
			err = db.QueryRow("SELECT school_id FROM UNT_Exams WHERE id = ?", examID).Scan(&examSchoolID)
			if err != nil {
				if err == sql.ErrNoRows {
					utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Экзамен не найден"})
				} else {
					log.Printf("Ошибка при проверке школы экзамена: %v", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить принадлежность экзамена к школе"})
				}
				return
			}

			if examSchoolID != int(userSchoolID.Int64) {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для изменения этого экзамена"})
				return
			}
		} else if userRole != "admin" && userRole != "moderator" && userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для изменения экзаменов"})
			return
		}

		// Step 5: Parse form data (instead of decoding JSON)
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max memory
			if err := r.ParseForm(); err != nil { // Try regular form if not multipart
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Ошибка при обработке формы: " + err.Error()})
				return
			}
		}

		// Step 6: Fetch existing exam for comparison and calculated fields
		var existingExam models.UNTExam
		err = db.QueryRow(`
			SELECT exam_type, first_subject, first_subject_score, second_subject, second_subject_score,
			history_of_kazakhstan, mathematical_literacy, reading_literacy, total_score,
			school_id, student_id, document_url, date
			FROM UNT_Exams WHERE id = ?`, examID).Scan(
			&existingExam.ExamType, &existingExam.FirstSubject, &existingExam.FirstSubjectScore,
			&existingExam.SecondSubject, &existingExam.SecondSubjectScore, &existingExam.HistoryOfKazakhstan,
			&existingExam.MathematicalLiteracy, &existingExam.ReadingLiteracy, &existingExam.TotalScore,
			&existingExam.SchoolID, &existingExam.StudentID, &existingExam.DocumentURL, &existingExam.Date,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Экзамен не найден"})
			} else {
				log.Printf("Ошибка при получении данных экзамена: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось получить данные экзамена"})
			}
			return
		}

		// Set exam ID
		existingExam.ID = examID

		// Step 7: Apply partial updates from form data

		// Track if any fields were changed
		fieldsChanged := false

		// Handle exam_type update
		if examType := r.FormValue("exam_type"); examType != "" {
			examType = strings.ToLower(examType)
			if examType != "regular" && examType != "creative" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный тип экзамена. Допустимые значения: regular, creative"})
				return
			}
			existingExam.ExamType = examType
			fieldsChanged = true
		}

		// Handle first_subject update
		if firstSubject := r.FormValue("first_subject"); firstSubject != "" {
			existingExam.FirstSubject = firstSubject
			fieldsChanged = true
		}

		// Handle first_subject_score update
		if firstSubjectScore := r.FormValue("first_subject_score"); firstSubjectScore != "" {
			score, err := strconv.Atoi(firstSubjectScore)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректное значение first_subject_score"})
				return
			}
			existingExam.FirstSubjectScore = score
			fieldsChanged = true
		}

		// Handle second_subject update
		if secondSubject := r.FormValue("second_subject"); secondSubject != "" {
			existingExam.SecondSubject = secondSubject
			fieldsChanged = true
		}

		// Handle second_subject_score update
		if secondSubjectScore := r.FormValue("second_subject_score"); secondSubjectScore != "" {
			score, err := strconv.Atoi(secondSubjectScore)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректное значение second_subject_score"})
				return
			}
			existingExam.SecondSubjectScore = score
			fieldsChanged = true
		}

		// Handle history_of_kazakhstan update
		if historyScore := r.FormValue("history_of_kazakhstan"); historyScore != "" {
			score, err := strconv.Atoi(historyScore)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректное значение history_of_kazakhstan"})
				return
			}
			existingExam.HistoryOfKazakhstan = score
			fieldsChanged = true
		}

		// Handle mathematical_literacy update
		if mathScore := r.FormValue("mathematical_literacy"); mathScore != "" {
			score, err := strconv.Atoi(mathScore)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректное значение mathematical_literacy"})
				return
			}
			existingExam.MathematicalLiteracy = score
			fieldsChanged = true
		}

		// Handle reading_literacy update
		if readingScore := r.FormValue("reading_literacy"); readingScore != "" {
			score, err := strconv.Atoi(readingScore)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректное значение reading_literacy"})
				return
			}
			existingExam.ReadingLiteracy = score
			fieldsChanged = true
		}

		// Handle document_url update (may come from file upload)
		file, handler, err := r.FormFile("document")
		if err == nil {
			defer file.Close()

			// Generate a unique filename or use a proper file storage service
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
			filepath := "./uploads/" + filename

			// Ensure directory exists
			os.MkdirAll("./uploads/", 0755)

			// Create the file
			dst, err := os.Create(filepath)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при сохранении файла"})
				return
			}
			defer dst.Close()

			// Copy the uploaded file to the created file on the filesystem
			if _, err := io.Copy(dst, file); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при копировании файла"})
				return
			}

			// Set the document URL to the uploaded file path
			existingExam.DocumentURL = filepath
			fieldsChanged = true
		} else if docURL := r.FormValue("document_url"); docURL != "" {
			// If no file uploaded, but URL provided
			existingExam.DocumentURL = docURL
			fieldsChanged = true
		}

		// Handle date update
		if date := r.FormValue("date"); date != "" {
			_, err = time.Parse("2006-01-02", date)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный формат даты. Используйте формат ГГГГ-ММ-ДД"})
				return
			}
			existingExam.Date = date
			fieldsChanged = true
		}

		// Return error if no fields were changed
		if !fieldsChanged {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Не указаны поля для обновления"})
			return
		}

		// Calculate total score based on exam type
		if existingExam.ExamType == "regular" {
			existingExam.TotalScore = existingExam.FirstSubjectScore + existingExam.SecondSubjectScore +
				existingExam.HistoryOfKazakhstan + existingExam.MathematicalLiteracy +
				existingExam.ReadingLiteracy
		} else { // creative
			existingExam.TotalScore = existingExam.FirstSubjectScore + existingExam.SecondSubjectScore +
				existingExam.HistoryOfKazakhstan + existingExam.ReadingLiteracy
		}

		// Step 8: Update record in database
		query := `UPDATE UNT_Exams SET 
			exam_type = ?, first_subject = ?, first_subject_score = ?, 
			second_subject = ?, second_subject_score = ?, history_of_kazakhstan = ?, 
			mathematical_literacy = ?, reading_literacy = ?, total_score = ?, 
			document_url = ?, date = ?
			WHERE id = ?`

		_, err = db.Exec(query,
			existingExam.ExamType,
			existingExam.FirstSubject,
			existingExam.FirstSubjectScore,
			existingExam.SecondSubject,
			existingExam.SecondSubjectScore,
			existingExam.HistoryOfKazakhstan,
			existingExam.MathematicalLiteracy,
			existingExam.ReadingLiteracy,
			existingExam.TotalScore,
			existingExam.DocumentURL,
			existingExam.Date,
			existingExam.ID,
		)

		if err != nil {
			log.Printf("Ошибка SQL при обновлении экзамена: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось обновить данные экзамена"})
			return
		}

		// Step 9: Return success message
		utils.ResponseJSON(w, map[string]interface{}{
			"message":   "Данные экзамена успешно обновлены",
			"id":        existingExam.ID,
			"exam_data": existingExam,
		})
	}
}
func (c *UNTScoreController) DeleteUNTExam(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Get userID from token
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get exam ID from URL parameter
		vars := mux.Vars(r)
		examID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный ID экзамена"})
			return
		}

		// Step 3: Check user role and school_id
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		// Step 4: Check if the user has permission to delete this exam
		if userRole == "schooladmin" {
			if !userSchoolID.Valid {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для удаления этого экзамена"})
				return
			}

			// Check if exam belongs to this school
			var examSchoolID int
			err = db.QueryRow("SELECT school_id FROM UNT_Exams WHERE id = ?", examID).Scan(&examSchoolID)
			if err != nil {
				if err == sql.ErrNoRows {
					utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Экзамен не найден"})
				} else {
					log.Printf("Ошибка при проверке школы экзамена: %v", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить принадлежность экзамена к школе"})
				}
				return
			}

			if examSchoolID != int(userSchoolID.Int64) {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для удаления этого экзамена"})
				return
			}
		} else if userRole != "admin" && userRole != "moderator" && userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для удаления экзаменов"})
			return
		}

		// Step 5: Check if exam exists
		var examExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM UNT_Exams WHERE id = ?)", examID).Scan(&examExists)
		if err != nil {
			log.Printf("Ошибка при проверке существования экзамена: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить наличие экзамена"})
			return
		}

		if !examExists {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Экзамен не найден"})
			return
		}

		// Step 6: Delete the exam
		_, err = db.Exec("DELETE FROM UNT_Exams WHERE id = ?", examID)
		if err != nil {
			log.Printf("Ошибка при удалении экзамена: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось удалить экзамен"})
			return
		}

		// Step 7: Return success message
		utils.ResponseJSON(w, map[string]interface{}{
			"message": "Экзамен успешно удален",
			"id":      examID,
		})
	}
}
func (c *UNTScoreController) GetUNTBySchoolID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		vars := mux.Vars(r)
		schoolIDStr := vars["school_id"]
		letter := r.URL.Query().Get("letter")
		studentID := r.URL.Query().Get("student_id")
		examType := r.URL.Query().Get("type")

		schoolID, err := strconv.ParseInt(schoolIDStr, 10, 64)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный ID школы"})
			return
		}

		switch userRole {
		case "schooladmin":
			if !userSchoolID.Valid {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет привязки к школе"})
				return
			}
			if userSchoolID.Int64 != schoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Вы не можете просматривать данные других школ"})
				return
			}
		case "admin", "moderator", "superadmin":
		default:
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав для просмотра результатов UNT"})
			return
		}

		query := `
			SELECT 
				e.id, 
				e.exam_type, 
				e.reading_literacy, 
				e.history_of_kazakhstan, 
				e.mathematical_literacy,
				e.first_subject,
				e.first_subject_score, 
				e.second_subject, 
				e.second_subject_score, 
				e.total_score,
				e.student_id,
				e.school_id,
				s.letter,
				e.date,
				e.document_url
			FROM 
				UNT_Exams e
			JOIN 
				student s ON e.student_id = s.student_id
			WHERE 
				e.school_id = ?`

		var args []interface{}
		args = append(args, schoolID)

		if letter != "" {
			query += " AND s.letter = ?"
			args = append(args, letter)
		}
		if studentID != "" {
			query += " AND e.student_id = ?"
			args = append(args, studentID)
		}
		if examType != "" {
			query += " AND e.exam_type = ?"
			args = append(args, strings.ToLower(examType))
		}

		query += " ORDER BY e.total_score DESC, s.letter, e.student_id"

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Ошибка при запросе результатов UNT: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось получить результаты UNT"})
			return
		}
		defer rows.Close()

		type UNTSchoolResult struct {
			ID                   int    `json:"id"`
			ExamType             string `json:"exam_type"`
			ReadingLiteracy      int    `json:"reading_literacy"`
			HistoryKazakhstan    int    `json:"history_of_kazakhstan"`
			MathematicalLiteracy int    `json:"mathematical_literacy"`
			FirstSubjectName     string `json:"first_subject"`
			FirstSubjectScore    int    `json:"first_subject_score"`
			SecondSubjectName    string `json:"second_subject"`
			SecondSubjectScore   int    `json:"second_subject_score"`
			TotalScore           int    `json:"total"`
			StudentID            int    `json:"student_id"`
			SchoolID             int    `json:"school_id"`
			Letter               string `json:"letter"`
			Date                 string `json:"date"`
			DocumentURL          string `json:"document_url"`
		}

		var results []UNTSchoolResult
		for rows.Next() {
			var result UNTSchoolResult
			err := rows.Scan(
				&result.ID,
				&result.ExamType,
				&result.ReadingLiteracy,
				&result.HistoryKazakhstan,
				&result.MathematicalLiteracy,
				&result.FirstSubjectName,
				&result.FirstSubjectScore,
				&result.SecondSubjectName,
				&result.SecondSubjectScore,
				&result.TotalScore,
				&result.StudentID,
				&result.SchoolID,
				&result.Letter,
				&result.Date,
				&result.DocumentURL,
			)
			if err != nil {
				log.Printf("Ошибка при сканировании строки: %v", err)
				continue
			}
			results = append(results, result)
		}

		utils.ResponseJSON(w, map[string]interface{}{
			"count": len(results),
			"data":  results,
		})
	}
}

func (c *TypeController) CreateFirstType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Получить userID из токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Шаг 2: Проверить роль и school_id пользователя
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		// Шаг 3: Проверить, что пользователь — schooladmin и у него есть валидный school_id
		if userRole != "schooladmin" || !userSchoolID.Valid || userSchoolID.Int64 == 0 {
			log.Printf("Пользователь %d имеет некорректную роль или school_id: роль=%s, school_id=%v", userID, userRole, userSchoolID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав на создание First Type"})
			return
		}

		// Шаг 4: Проверить, существует ли школа в таблице Schools, используя school_id
		var schoolExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", userSchoolID.Int64).Scan(&schoolExists)
		if err != nil {
			log.Printf("Ошибка при проверке существования школы: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить наличие школы"})
			return
		}

		if !schoolExists {
			log.Printf("Школа с id %d не существует", userSchoolID.Int64)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Школа не найдена в нашей системе"})
			return
		}

		// ИСПРАВЛЕНИЕ: Проверяем Content-Type и устанавливаем его, если необходимо
		contentType := r.Header.Get("Content-Type")
		log.Printf("Исходный Content-Type: %s", contentType)

		// Если Content-Type не содержит multipart/form-data, устанавливаем его
		if !strings.Contains(contentType, "multipart/form-data") {
			log.Println("Content-Type не определен как multipart/form-data, устанавливаем...")
			// Не меняем заголовок здесь, это может не сработать после отправки запроса
		}

		// Парсим форму перед доступом к данным
		err = r.ParseMultipartForm(10 << 20) // Максимальный размер 10MB
		if err != nil {
			log.Printf("Ошибка при парсинге multipart/form-data: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Ошибка при обработке данных формы: " + err.Error()})
			return
		}

		// Отладочная информация
		log.Printf("Получены поля формы: %v", r.MultipartForm.Value)
		if r.MultipartForm.File != nil {
			for k, v := range r.MultipartForm.File {
				if len(v) > 0 {
					log.Printf("Файл в форме: поле=%s, имя=%s, размер=%d", k, v[0].Filename, v[0].Size)
				} else {
					log.Printf("Файл в форме: поле=%s, пустой список файлов", k)
				}
			}
		} else {
			log.Println("MultipartForm.File является nil, файлы не были загружены")
		}

		// Получаем данные из формы
		firstType := models.FirstType{
			FirstSubject:  r.FormValue("first_subject"),
			SecondSubject: r.FormValue("second_subject"),
			Type:          r.FormValue("type"),
			Date:          r.FormValue("date"), // Добавляем получение даты из формы
		}

		// Проверка наличия даты
		if firstType.Date == "" {
			log.Println("Дата не указана")
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Необходимо указать дату сдачи ЕНТ"})
			return
		}

		// Проверка формата даты (опционально)
		_, err = time.Parse("2006-01-02", firstType.Date)
		if err != nil {
			log.Printf("Некорректный формат даты: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный формат даты. Используйте формат ГГГГ-ММ-ДД"})
			return
		}

		// Конвертация числовых значений
		firstSubjectScore, err := strconv.Atoi(r.FormValue("first_subject_score"))
		if err == nil {
			firstType.FirstSubjectScore = firstSubjectScore
		}

		secondSubjectScore, err := strconv.Atoi(r.FormValue("second_subject_score"))
		if err == nil {
			firstType.SecondSubjectScore = secondSubjectScore
		}

		historyOfKazakhstan, err := strconv.Atoi(r.FormValue("history_of_kazakhstan"))
		if err == nil {
			firstType.HistoryOfKazakhstan = historyOfKazakhstan
		}

		mathematicalLiteracy, err := strconv.Atoi(r.FormValue("mathematical_literacy"))
		if err == nil {
			firstType.MathematicalLiteracy = mathematicalLiteracy
		}

		readingLiteracy, err := strconv.Atoi(r.FormValue("reading_literacy"))
		if err == nil {
			firstType.ReadingLiteracy = readingLiteracy
		}

		studentID, err := strconv.Atoi(r.FormValue("student_id"))
		if err == nil {
			firstType.StudentID = studentID
		}

		// Шаг 5: Проверить, существует ли student_id в таблице student
		var studentExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE student_id = ?)", firstType.StudentID).Scan(&studentExists)
		if err != nil {
			log.Printf("Ошибка при проверке существования студента: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить наличие студента"})
			return
		}

		if !studentExists {
			log.Printf("Студент с id %d не существует", firstType.StudentID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Студент не найден в нашей системе"})
			return
		}

		// Шаг 6: Проверить обязательные поля
		if firstType.FirstSubject == "" || firstType.SecondSubject == "" ||
			firstType.FirstSubjectScore < 0 || firstType.SecondSubjectScore < 0 ||
			firstType.HistoryOfKazakhstan < 0 || firstType.StudentID <= 0 ||
			firstType.Date == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректные или отсутствующие обязательные поля"})
			return
		}

		// Обработка загрузки файла
		var documentURL string

		// Проверяем, был ли передан document_url как поле формы
		if urlFromForm := r.FormValue("document_url"); urlFromForm != "" {
			// Если пользователь предоставил URL вручную, используем его
			documentURL = urlFromForm
			log.Printf("Использую предоставленный URL документа: %s", documentURL)
		} else {
			// ИСПРАВЛЕНИЕ: Улучшенная обработка файла
			// Попытка найти файл по конкретному имени поля
			file, handler, err := r.FormFile("document")

			// Если не найден, проверяем другие возможные имена полей
			if err != nil {
				log.Printf("Не найден файл в поле 'document', пробуем альтернативные: %v", err)

				// Список возможных имен полей файла
				fileFieldNames := []string{"file", "document_file", "uploaded_file"}

				// Пробуем все возможные названия полей
				for _, fieldName := range fileFieldNames {
					file, handler, err = r.FormFile(fieldName)
					if err == nil {
						log.Printf("Файл найден в поле '%s': %s", fieldName, handler.Filename)
						break // Если файл найден, выходим из цикла
					}
				}
			} else {
				log.Printf("Файл найден в поле 'document': %s", handler.Filename)
			}

			if err != nil {
				// ИСПРАВЛЕНИЕ: Проверяем все ключи формы напрямую
				foundFile := false
				if r.MultipartForm != nil && r.MultipartForm.File != nil {
					for fieldName, fileHeaders := range r.MultipartForm.File {
						if len(fileHeaders) > 0 {
							log.Printf("Нашли файл в неожиданном поле '%s': %s", fieldName, fileHeaders[0].Filename)
							file, err = fileHeaders[0].Open()
							if err == nil {
								handler = fileHeaders[0]
								foundFile = true
								break
							} else {
								log.Printf("Ошибка при открытии файла из поля '%s': %v", fieldName, err)
							}
						}
					}
				}

				if !foundFile {
					log.Printf("Файл не был загружен или не может быть прочитан: %v. Продолжаем без документа.", err)
					documentURL = ""
				}
			}

			// Если файл найден, загружаем его в S3
			if err == nil && file != nil && handler != nil {
				defer file.Close()

				// Создаем уникальное имя файла
				fileExt := filepath.Ext(handler.Filename)
				fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String(), fileExt)

				log.Printf("Подготовка к загрузке файла в S3 с именем %s", fileName)

				// Загружаем файл в S3 бакет для олимпиадных документов
				url, err := utils.UploadFileToS3(file, fileName, "olympiaddoc")
				if err != nil {
					log.Printf("Ошибка при загрузке файла в S3: %v", err)
					utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при загрузке файла в облачное хранилище: " + err.Error()})
					return
				}

				documentURL = url
				log.Printf("Файл успешно загружен в S3: %s", documentURL)
			}
		}

		// Шаг 8: Вычислить total_score
		totalScore := firstType.FirstSubjectScore + firstType.SecondSubjectScore + firstType.HistoryOfKazakhstan +
			firstType.MathematicalLiteracy + firstType.ReadingLiteracy

		// Шаг 9: Вставить данные FirstType в базу данных с использованием school_id из токена
		query := `INSERT INTO First_Type (
			first_subject, first_subject_score, second_subject, second_subject_score, 
			history_of_kazakhstan, mathematical_literacy, reading_literacy, type, student_id, 
			school_id, total_score, document_url, date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := db.Exec(query,
			firstType.FirstSubject,
			firstType.FirstSubjectScore,
			firstType.SecondSubject,
			firstType.SecondSubjectScore,
			firstType.HistoryOfKazakhstan,
			firstType.MathematicalLiteracy,
			firstType.ReadingLiteracy,
			firstType.Type,
			firstType.StudentID,
			userSchoolID.Int64,
			totalScore,
			documentURL,
			firstType.Date,
		)

		if err != nil {
			log.Printf("Ошибка SQL: %v", err)

			// Специфическая проверка на нарушение внешнего ключа
			if strings.Contains(err.Error(), "foreign key constraint fails") {
				if strings.Contains(err.Error(), "FK_School") {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный или отсутствующий school_id в таблице Schools"})
					return
				} else if strings.Contains(err.Error(), "student_id") {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный или отсутствующий student_id"})
					return
				}
			}

			// Общая ошибка
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось создать First Type"})
			return
		}

		// Получить ID только что вставленной записи
		newID, _ := result.LastInsertId()

		// Шаг 10: Вернуть сообщение об успешном создании
		utils.ResponseJSON(w, map[string]interface{}{
			"message":      "First Type создан успешно",
			"id":           newID,
			"document_url": documentURL,
			"date":         firstType.Date,
		})
	}
}
func (c *TypeController) GetFirstTypes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
        SELECT 
            ft.first_type_id, 
            ft.first_subject AS first_subject, 
            COALESCE(ft.first_subject_score, 0) AS first_subject_score,
            ft.second_subject AS second_subject, 
            COALESCE(ft.second_subject_score, 0) AS second_subject_score,
            COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan, 
            COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy, 
            COALESCE(ft.reading_literacy, 0) AS reading_literacy,
            ft.type,
            COALESCE(ft.student_id, 0) AS student_id, 
            ft.school_id,
            (COALESCE(ft.first_subject_score, 0) + COALESCE(ft.second_subject_score, 0) + 
             COALESCE(ft.history_of_kazakhstan, 0) + COALESCE(ft.mathematical_literacy, 0) + 
             COALESCE(ft.reading_literacy, 0)) AS total_score
        FROM First_Type ft`

		rows, err := db.Query(query)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get First Types"})
			return
		}
		defer rows.Close()

		var types []models.FirstType
		for rows.Next() {
			var firstType models.FirstType
			var firstSubjectName sql.NullString
			var secondSubjectName sql.NullString
			var typeColumn sql.NullString // Для обработки значения NULL в поле type
			var schoolID sql.NullInt64    // Для обработки значения school_id

			if err := rows.Scan(
				&firstType.ID,
				&firstSubjectName, &firstType.FirstSubjectScore,
				&secondSubjectName, &firstType.SecondSubjectScore,
				&firstType.HistoryOfKazakhstan,
				&firstType.MathematicalLiteracy,
				&firstType.ReadingLiteracy,
				&typeColumn, // Добавляем sql.NullString для поля type
				&firstType.StudentID,
				&schoolID, // Добавляем поле school_id
				&firstType.TotalScore,
			); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Types"})
				return
			}

			// Преобразуем sql.NullString в обычные строки
			if firstSubjectName.Valid {
				firstType.FirstSubject = firstSubjectName.String
			} else {
				firstType.FirstSubject = ""
			}

			if secondSubjectName.Valid {
				firstType.SecondSubject = secondSubjectName.String
			} else {
				firstType.SecondSubject = ""
			}

			if typeColumn.Valid {
				firstType.Type = typeColumn.String
			} else {
				firstType.Type = "" // Если type равно NULL, присваиваем пустую строку
			}

			if schoolID.Valid {
				firstType.SchoolID = int(schoolID.Int64) // Преобразуем school_id из sql.NullInt64
			} else {
				firstType.SchoolID = 0 // Если school_id равно NULL, присваиваем 0
			}

			types = append(types, firstType)
		}

		utils.ResponseJSON(w, types) // Возвращаем результат в формате JSON
	}
}
func (c *TypeController) GetFirstTypesBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID, err := strconv.Atoi(vars["school_id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Запрос для получения данных для конкретной школы
		query := `
        SELECT 
            ft.first_type_id, 
            ft.first_subject,  -- Используем first_subject
            COALESCE(ft.first_subject_score, 0) AS first_subject_score,
            ft.second_subject, -- Используем second_subject
            COALESCE(ft.second_subject_score, 0) AS second_subject_score,
            COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan, 
            COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy, 
            COALESCE(ft.reading_literacy, 0) AS reading_literacy,
            ft.type,
            COALESCE(ft.student_id, 0) AS student_id, 
            (COALESCE(ft.first_subject_score, 0) + COALESCE(ft.second_subject_score, 0) + 
             COALESCE(ft.history_of_kazakhstan, 0) + COALESCE(ft.mathematical_literacy, 0) + 
             COALESCE(ft.reading_literacy, 0)) AS total_score
        FROM First_Type ft
        WHERE ft.school_id = ?` /* Фильтрация по school_id */

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get First Types by School"})
			return
		}
		defer rows.Close()

		var types []models.FirstType
		for rows.Next() {
			var firstType models.FirstType
			if err := rows.Scan(
				&firstType.ID,
				&firstType.FirstSubject, &firstType.FirstSubjectScore,
				&firstType.SecondSubject, &firstType.SecondSubjectScore,
				&firstType.HistoryOfKazakhstan,
				&firstType.MathematicalLiteracy,
				&firstType.ReadingLiteracy,
				&firstType.Type,
				&firstType.StudentID,
				&firstType.TotalScore,
			); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Types"})
				return
			}

			types = append(types, firstType)
		}

		utils.ResponseJSON(w, types) // Возвращаем результат в формате JSON
	}
}
func (c *TypeController) CreateSecondType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем userID из токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Проверка роли пользователя и school_id
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя или school_id:", err)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Не удалось получить роль или school_id пользователя"})
			return
		}

		// Проверяем, что пользователь — schooladmin и у него есть валидный school_id
		if userRole != "schooladmin" || !userSchoolID.Valid || userSchoolID.Int64 == 0 {
			log.Printf("Пользователь %d имеет некорректную роль или school_id: роль=%s, school_id=%v", userID, userRole, userSchoolID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "У вас нет прав на создание Second Type"})
			return
		}

		// Проверяем, существует ли школа в таблице Schools
		var schoolExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Schools WHERE school_id = ?)", userSchoolID.Int64).Scan(&schoolExists)
		if err != nil {
			log.Printf("Ошибка при проверке существования школы: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить наличие школы"})
			return
		}

		if !schoolExists {
			log.Printf("Школа с id %d не существует", userSchoolID.Int64)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Школа не найдена в нашей системе"})
			return
		}

		// Проверяем Content-Type запроса и обрабатываем соответственно
		contentType := r.Header.Get("Content-Type")
		log.Printf("Получен запрос с Content-Type: %s", contentType)

		var secondType models.SecondType

		if strings.Contains(contentType, "multipart/form-data") {
			// Обработка multipart/form-data запроса
			err = r.ParseMultipartForm(10 << 20) // Максимальный размер 10MB
			if err != nil {
				log.Printf("Ошибка при парсинге multipart/form-data: %v", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Ошибка при обработке данных формы: " + err.Error()})
				return
			}

			// Отладочная информация о полученных данных формы
			log.Printf("Получены поля формы: %v", r.MultipartForm.Value)
			if r.MultipartForm.File != nil {
				for k, v := range r.MultipartForm.File {
					if len(v) > 0 {
						log.Printf("Файл в форме: поле=%s, имя=%s, размер=%d", k, v[0].Filename, v[0].Size)
					} else {
						log.Printf("Файл в форме: поле=%s, пустой список файлов", k)
					}
				}
			} else {
				log.Println("MultipartForm.File является nil, файлы не были загружены")
			}

			// Заполняем данные из формы
			secondType.Type = "type-2"
			secondType.StudentID, _ = strconv.Atoi(r.FormValue("student_id"))
			secondType.Date = r.FormValue("date") // Получаем дату из формы

			// Проверка наличия даты
			if secondType.Date == "" {
				log.Println("Дата не указана")
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Необходимо указать дату сдачи творческого экзамена"})
				return
			}

			// Проверка формата даты
			_, err = time.Parse("2006-01-02", secondType.Date)
			if err != nil {
				log.Printf("Некорректный формат даты: %v", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный формат даты. Используйте формат ГГГГ-ММ-ДД"})
				return
			}

			// История Казахстана (творческий)
			if historyValue := r.FormValue("history_of_kazakhstan_creative"); historyValue != "" {
				historyScore, err := strconv.Atoi(historyValue)
				if err == nil {
					secondType.HistoryOfKazakhstanCreative = &historyScore
				}
			}

			// Читательская грамотность (творческий)
			if readingValue := r.FormValue("reading_literacy_creative"); readingValue != "" {
				readingScore, err := strconv.Atoi(readingValue)
				if err == nil {
					secondType.ReadingLiteracyCreative = &readingScore
				}
			}

			// Творческий экзамен 1
			if creativeExam1Value := r.FormValue("creative_exam1"); creativeExam1Value != "" {
				creativeExam1Score, err := strconv.Atoi(creativeExam1Value)
				if err == nil {
					secondType.CreativeExam1 = &creativeExam1Score
				}
			}

			// Творческий экзамен 2
			if creativeExam2Value := r.FormValue("creative_exam2"); creativeExam2Value != "" {
				creativeExam2Score, err := strconv.Atoi(creativeExam2Value)
				if err == nil {
					secondType.CreativeExam2 = &creativeExam2Score
				}
			}

			// Обработка загрузки файла
			var documentURL string

			// Проверяем, был ли передан document_url как поле формы
			if urlFromForm := r.FormValue("document_url"); urlFromForm != "" {
				documentURL = urlFromForm
				log.Printf("Использую предоставленный URL документа: %s", documentURL)
			} else {
				// Попытка найти файл по конкретному имени поля
				file, handler, err := r.FormFile("document")

				// Если не найден, проверяем другие возможные имена полей
				if err != nil {
					log.Printf("Не найден файл в поле 'document', пробуем альтернативные: %v", err)

					// Список возможных имен полей файла
					fileFieldNames := []string{"file", "document_file", "uploaded_file"}

					// Пробуем все возможные названия полей
					for _, fieldName := range fileFieldNames {
						file, handler, err = r.FormFile(fieldName)
						if err == nil {
							log.Printf("Файл найден в поле '%s': %s", fieldName, handler.Filename)
							break // Если файл найден, выходим из цикла
						}
					}
				} else {
					log.Printf("Файл найден в поле 'document': %s", handler.Filename)
				}

				if err != nil {
					// Проверяем все ключи формы напрямую
					foundFile := false
					if r.MultipartForm != nil && r.MultipartForm.File != nil {
						for fieldName, fileHeaders := range r.MultipartForm.File {
							if len(fileHeaders) > 0 {
								log.Printf("Нашли файл в неожиданном поле '%s': %s", fieldName, fileHeaders[0].Filename)
								file, err = fileHeaders[0].Open()
								if err == nil {
									handler = fileHeaders[0]
									foundFile = true
									break
								} else {
									log.Printf("Ошибка при открытии файла из поля '%s': %v", fieldName, err)
								}
							}
						}
					}

					if !foundFile {
						log.Printf("Файл не был загружен или не может быть прочитан: %v. Продолжаем без документа.", err)
						documentURL = ""
					}
				}

				// Если файл найден, загружаем его в S3
				if err == nil && file != nil && handler != nil {
					defer file.Close()

					// Создаем уникальное имя файла
					fileExt := filepath.Ext(handler.Filename)
					fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String(), fileExt)

					log.Printf("Подготовка к загрузке файла в S3 с именем %s", fileName)

					// Загружаем файл в S3 бакет для олимпиадных документов
					url, err := utils.UploadFileToS3(file, fileName, "olympiaddoc")
					if err != nil {
						log.Printf("Ошибка при загрузке файла в S3: %v", err)
						utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при загрузке файла в облачное хранилище: " + err.Error()})
						return
					}

					documentURL = url
					log.Printf("Файл успешно загружен в S3: %s", documentURL)
				}
			}

			secondType.DocumentURL = documentURL
		} else {
			// Обработка application/json запроса
			if err := json.NewDecoder(r.Body).Decode(&secondType); err != nil {
				log.Println("Ошибка декодирования запроса:", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный запрос"})
				return
			}

			// Проверяем наличие даты для JSON запроса
			if secondType.Date == "" {
				log.Println("Дата не указана в JSON запросе")
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Необходимо указать дату сдачи творческого экзамена"})
				return
			}

			// Проверка формата даты для JSON запроса
			_, err = time.Parse("2006-01-02", secondType.Date)
			if err != nil {
				log.Printf("Некорректный формат даты в JSON запросе: %v", err)
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный формат даты. Используйте формат ГГГГ-ММ-ДД"})
				return
			}

			// Проверяем, что тип был установлен, иначе устанавливаем default
			if secondType.Type == "" {
				secondType.Type = "type-2"
			}
		}

		// Проверяем, существует ли student_id в таблице student
		if secondType.StudentID <= 0 {
			log.Printf("ID студента не указан или некорректен: %d", secondType.StudentID)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "ID студента не указан или некорректен"})
			return
		}

		var studentExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE student_id = ?)", secondType.StudentID).Scan(&studentExists)
		if err != nil {
			log.Printf("Ошибка при проверке существования студента: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось проверить наличие студента"})
			return
		}

		if !studentExists {
			log.Printf("Студент с id %d не существует", secondType.StudentID)
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Студент не найден в нашей системе"})
			return
		}

		// Вычисляем общий балл
		totalScoreCreative := 0
		if secondType.HistoryOfKazakhstanCreative != nil {
			totalScoreCreative += *secondType.HistoryOfKazakhstanCreative
		}
		if secondType.ReadingLiteracyCreative != nil {
			totalScoreCreative += *secondType.ReadingLiteracyCreative
		}
		if secondType.CreativeExam1 != nil {
			totalScoreCreative += *secondType.CreativeExam1
		}
		if secondType.CreativeExam2 != nil {
			totalScoreCreative += *secondType.CreativeExam2
		}

		secondType.TotalScoreCreative = &totalScoreCreative

		// Вставка с document_url и date
		query := `INSERT INTO Second_Type (
			history_of_kazakhstan_creative, reading_literacy_creative, creative_exam1, 
			creative_exam2, school_id, total_score_creative, type, student_id, document_url, date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := db.Exec(query,
			secondType.HistoryOfKazakhstanCreative,
			secondType.ReadingLiteracyCreative,
			secondType.CreativeExam1,
			secondType.CreativeExam2,
			userSchoolID.Int64,
			totalScoreCreative,
			secondType.Type,
			secondType.StudentID,
			secondType.DocumentURL,
			secondType.Date,
		)
		if err != nil {
			log.Println("Ошибка SQL:", err)

			// Специфическая проверка на нарушение внешнего ключа
			if strings.Contains(err.Error(), "foreign key constraint fails") {
				if strings.Contains(err.Error(), "FK_School") {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный или отсутствующий school_id в таблице Schools"})
					return
				} else if strings.Contains(err.Error(), "student_id") {
					utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Некорректный или отсутствующий student_id"})
					return
				}
			}

			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось создать Second Type"})
			return
		}

		// Получить ID только что вставленной записи
		newID, _ := result.LastInsertId()
		secondType.ID = int(newID)

		utils.ResponseJSON(w, map[string]interface{}{
			"message":      "Second Type создан успешно",
			"id":           newID,
			"second_type":  secondType,
			"document_url": secondType.DocumentURL,
			"date":         secondType.Date,
		})
	}
}
func (c *TypeController) GetSecondTypes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
        SELECT 
            second_type_id, 
            COALESCE(history_of_kazakhstan_creative, 0) AS history_of_kazakhstan_creative, 
            COALESCE(reading_literacy_creative, 0) AS reading_literacy_creative,
            COALESCE(creative_exam1, 0) AS creative_exam1,
            COALESCE(creative_exam2, 0) AS creative_exam2,
            (COALESCE(history_of_kazakhstan_creative, 0) + 
             COALESCE(reading_literacy_creative, 0) + 
             COALESCE(creative_exam1, 0) + 
             COALESCE(creative_exam2, 0)) AS total_score_creative,
            type
        FROM Second_Type`

		rows, err := db.Query(query)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get Second Types"})
			return
		}
		defer rows.Close()

		var types []models.SecondType
		for rows.Next() {
			var secondType models.SecondType
			var typeColumn sql.NullString // Используем sql.NullString для обработки type как строки
			if err := rows.Scan(
				&secondType.ID,
				&secondType.HistoryOfKazakhstanCreative,
				&secondType.ReadingLiteracyCreative,
				&secondType.CreativeExam1,
				&secondType.CreativeExam2,
				&secondType.TotalScoreCreative,
				&typeColumn, // Сканируем поле type как строку
			); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Types"})
				return
			}

			// Преобразуем sql.NullString в обычную строку
			if typeColumn.Valid {
				secondType.Type = typeColumn.String
			} else {
				secondType.Type = ""
			}

			types = append(types, secondType)
		}

		utils.ResponseJSON(w, types)
	}
}
func (c *TypeController) GetSecondTypesBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID, err := strconv.Atoi(vars["school_id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Запрос для получения данных для конкретной школы
		query := `
        SELECT 
            second_type_id,
            history_of_kazakhstan_creative,
            reading_literacy_creative,
            creative_exam1,
            creative_exam2,
            total_score_creative,
            type,
            student_id
        FROM Second_Type
        WHERE school_id = ?`

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get Second Types by School"})
			return
		}
		defer rows.Close()

		var types []models.SecondType
		for rows.Next() {
			var secondType models.SecondType
			var studentID sql.NullInt64 // Для обработки NULL значений в student_id

			if err := rows.Scan(
				&secondType.ID,
				&secondType.HistoryOfKazakhstanCreative,
				&secondType.ReadingLiteracyCreative,
				&secondType.CreativeExam1,
				&secondType.CreativeExam2,
				&secondType.TotalScoreCreative,
				&secondType.Type,
				&studentID, // Убираем второй раз сканирование studentID
			); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Types"})
				return
			}

			// Проверка, если student_id действителен, то присваиваем его
			if studentID.Valid {
				secondType.StudentID = int(studentID.Int64) // Преобразуем int64 в int
			} else {
				secondType.StudentID = 0 // Значение по умолчанию, если NULL
			}

			types = append(types, secondType)
		}

		utils.ResponseJSON(w, types) // Возвращаем результат в формате JSON
	}
}
func (c *TypeController) GetAverageRatingBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID, err := strconv.Atoi(vars["school_id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Query to get average score by school
		// Запрос для получения среднего балла по всем предметам для школы
		query := `
        SELECT 
            AVG(CASE WHEN ft.first_subject_score IS NOT NULL THEN ft.first_subject_score ELSE 0 END) AS avg_first_subject_score,
            AVG(CASE WHEN ft.second_subject_score IS NOT NULL THEN ft.second_subject_score ELSE 0 END) AS avg_second_subject_score,
            AVG(CASE WHEN ft.history_of_kazakhstan IS NOT NULL THEN ft.history_of_kazakhstan ELSE 0 END) AS avg_history_of_kazakhstan,
            AVG(CASE WHEN ft.mathematical_literacy IS NOT NULL THEN ft.mathematical_literacy ELSE 0 END) AS avg_mathematical_literacy,
            AVG(CASE WHEN ft.reading_literacy IS NOT NULL THEN ft.reading_literacy ELSE 0 END) AS avg_reading_literacy,
            AVG(CASE WHEN ft.first_subject_score IS NOT NULL AND ft.second_subject_score IS NOT NULL AND 
                     ft.history_of_kazakhstan IS NOT NULL AND ft.mathematical_literacy IS NOT NULL AND 
                     ft.reading_literacy IS NOT NULL 
                     THEN (ft.first_subject_score + ft.second_subject_score + ft.history_of_kazakhstan + 
                           ft.mathematical_literacy + ft.reading_literacy) ELSE 0 END) AS avg_total_score
        FROM First_Type ft
        WHERE ft.school_id = ?`

		row := db.QueryRow(query, schoolID)

		var avgFirstSubjectScore, avgSecondSubjectScore, avgHistoryOfKazakhstan, avgMathematicalLiteracy, avgReadingLiteracy, avgTotalScore float64

		err = row.Scan(&avgFirstSubjectScore, &avgSecondSubjectScore, &avgHistoryOfKazakhstan, &avgMathematicalLiteracy, &avgReadingLiteracy, &avgTotalScore)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate average rating"})
			return
		}

		// Response with average score
		// Ответ с данными среднего балла
		result := map[string]float64{
			"avg_first_subject_score":   avgFirstSubjectScore,
			"avg_second_subject_score":  avgSecondSubjectScore,
			"avg_history_of_kazakhstan": avgHistoryOfKazakhstan,
			"avg_mathematical_literacy": avgMathematicalLiteracy,
			"avg_reading_literacy":      avgReadingLiteracy,
			"avg_total_score":           avgTotalScore,
		}

		utils.ResponseJSON(w, result) // Return result in JSON format
		utils.ResponseJSON(w, result) // Возвращаем результат в формате JSON
	}
}
func (c *TypeController) GetAverageRatingSecondBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из параметров URL
		vars := mux.Vars(r)
		schoolID, err := strconv.Atoi(vars["school_id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Запрос для получения всех оценок по конкретной школе
		query := `
        SELECT 
            history_of_kazakhstan_creative,
            reading_literacy_creative,
            creative_exam1,
            creative_exam2
        FROM Second_Type
        WHERE school_id = ?`

		rows, err := db.Query(query, schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get Second Types by School"})
			return
		}
		defer rows.Close()

		var totalScore float64
		var studentCount int

		for rows.Next() {
			var historyOfKazakhstanCreative, readingLiteracyCreative, creativeExam1, creativeExam2 sql.NullInt64

			// Считываем данные для каждого экзамена
			if err := rows.Scan(&historyOfKazakhstanCreative, &readingLiteracyCreative, &creativeExam1, &creativeExam2); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Types"})
				return
			}

			// Вычисляем сумму оценок для каждого студента
			totalScore += float64(historyOfKazakhstanCreative.Int64 + readingLiteracyCreative.Int64 + creativeExam1.Int64 + creativeExam2.Int64)
			studentCount++
		}

		if studentCount == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No students found for this school"})
			return
		}

		// Рассчитываем средний балл
		averageRating := totalScore / float64(studentCount)

		// Возвращаем результат в формате JSON
		utils.ResponseJSON(w, map[string]interface{}{
			"average_rating": averageRating,
		})
	}
}
