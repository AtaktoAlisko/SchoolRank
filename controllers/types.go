package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
            ft.first_subject AS first_subject_name, 
            COALESCE(ft.first_subject_score, 0) AS first_subject_score,
            ft.second_subject AS second_subject_name, 
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
