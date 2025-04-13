package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"

	"github.com/gorilla/mux"
)

type TypeController struct{}


func (c *TypeController) CreateFirstType(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Получаем userID из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Проверка на роль и школу
        var userRole string
        var userSchoolID sql.NullInt64
        err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
        if err != nil || userRole != "schooladmin" || !userSchoolID.Valid {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create First Type"})
            return
        }

        // Чтение данных первого типа
        var firstType models.FirstType
        if err := json.NewDecoder(r.Body).Decode(&firstType); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Вычисление total_score
        totalScore := firstType.FirstSubjectScore + firstType.SecondSubjectScore + firstType.HistoryOfKazakhstan +
            firstType.MathematicalLiteracy + firstType.ReadingLiteracy

        // Вставляем First Type в БД с привязкой к школе директора и новым полем type и student_id
        query := `INSERT INTO First_Type (first_subject, first_subject_score, second_subject, second_subject_score, 
                      history_of_kazakhstan, mathematical_literacy, reading_literacy, type, student_id, school_id, total_score) 
                  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
        _, err = db.Exec(query, 
            firstType.FirstSubject, 
            firstType.FirstSubjectScore, 
            firstType.SecondSubject, 
            firstType.SecondSubjectScore, 
            firstType.HistoryOfKazakhstan, 
            firstType.MathematicalLiteracy, 
            firstType.ReadingLiteracy,
            firstType.Type, 
            firstType.StudentID,
            userSchoolID.Int64, // Добавляем school_id из токена
            totalScore) // Добавляем вычисленный total_score

        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create First Type"})
            return
        }

        utils.ResponseJSON(w, map[string]string{"message": "First Type created successfully"})
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
            var schoolID sql.NullInt64 // Для обработки значения school_id

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

        utils.ResponseJSON(w, types)  // Возвращаем результат в формате JSON
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
        WHERE ft.school_id = ?`  /* Фильтрация по school_id */

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

        utils.ResponseJSON(w, types)  // Возвращаем результат в формате JSON
    }
}
func (c *TypeController) CreateSecondType(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var secondType models.SecondType
        if err := json.NewDecoder(r.Body).Decode(&secondType); err != nil {
            log.Println("Error decoding the request:", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Получаем userID из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Проверка роли пользователя
        var userRole string
        var userSchoolID sql.NullInt64
        err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
        if err != nil || userRole != "schooladmin" || !userSchoolID.Valid {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create second type"})
            return
        }

        // Рассчитываем total_score_creative с учетом всех компонентов
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

        // Устанавливаем type как 'type-2' для второго типа
        secondType.Type = "type-2" // Добавляем тип

        // Вставляем Second Type в таблицу базы данных с привязкой к школе и total_score_creative
        query := `INSERT INTO Second_Type (history_of_kazakhstan_creative, reading_literacy_creative, creative_exam1, creative_exam2, school_id, total_score_creative, type, student_id) 
                  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
        _, err = db.Exec(query, 
            secondType.HistoryOfKazakhstanCreative, 
            secondType.ReadingLiteracyCreative, 
            secondType.CreativeExam1, 
            secondType.CreativeExam2,
            userSchoolID.Int64, 
            totalScoreCreative,
            secondType.Type,
            secondType.StudentID) // Добавляем student_id

        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create Second Type"})
            return
        }

        // Возвращаем объект второго типа с вычисленным баллом
        secondType.TotalScoreCreative = &totalScoreCreative
        utils.ResponseJSON(w, map[string]interface{}{
            "message":       "Second Type created successfully",
            "second_type":   secondType,
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
            var typeColumn sql.NullString  // Используем sql.NullString для обработки type как строки
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
            "avg_first_subject_score":      avgFirstSubjectScore,
            "avg_second_subject_score":     avgSecondSubjectScore,
            "avg_history_of_kazakhstan":    avgHistoryOfKazakhstan,
            "avg_mathematical_literacy":    avgMathematicalLiteracy,
            "avg_reading_literacy":         avgReadingLiteracy,
            "avg_total_score":              avgTotalScore,
        }

        utils.ResponseJSON(w, result)  // Return result in JSON format
        utils.ResponseJSON(w, result)  // Возвращаем результат в формате JSON
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










