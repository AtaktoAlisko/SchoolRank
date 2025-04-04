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

// Функция для обработки NULL значений
func nullableValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	return value
}

func (c *TypeController) CreateFirstType(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Получаем userID из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Проверяем роль и школу директора
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

        // Проверяем существование First_Subject и Second_Subject
        var firstSubjectExists, secondSubjectExists bool
        err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM First_Subject WHERE first_subject_id = ?)", firstType.FirstSubjectID).Scan(&firstSubjectExists)
        if err != nil || !firstSubjectExists {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "First Subject ID does not exist"})
            return
        }

        err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Second_Subject WHERE second_subject_id = ?)", firstType.SecondSubjectID).Scan(&secondSubjectExists)
        if err != nil || !secondSubjectExists {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Second Subject ID does not exist"})
            return
        }

        // Вставляем First Type в БД с привязкой к школе директора и новым полем type
        query := `INSERT INTO First_Type (first_subject_id, second_subject_id, history_of_kazakhstan, mathematical_literacy, reading_literacy, school_id, type) 
                  VALUES (?, ?, ?, ?, ?, ?, ?)`
        _, err = db.Exec(query, 
            firstType.FirstSubjectID, 
            firstType.SecondSubjectID, 
            firstType.HistoryOfKazakhstan, 
            firstType.MathematicalLiteracy, 
            firstType.ReadingLiteracy,
            userSchoolID.Int64, 
            firstType.Type) // Указываем type

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
        // Запрос с JOIN, который соединяет несколько таблиц
        query := `
        SELECT 
            ft.first_type_id, 
            ft.first_subject_id, 
            fs.subject AS first_subject_name, 
            COALESCE(fs.score, 0) AS first_subject_score,
            ft.second_subject_id, 
            ss.subject AS second_subject_name, 
            COALESCE(ss.score, 0) AS second_subject_score,
            COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan, 
            COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy, 
            COALESCE(ft.reading_literacy, 0) AS reading_literacy,
            ft.type,
            (COALESCE(fs.score, 0) + COALESCE(ss.score, 0) + COALESCE(ft.history_of_kazakhstan, 0) + COALESCE(ft.mathematical_literacy, 0) + COALESCE(ft.reading_literacy, 0)) AS total_score
        FROM First_Type ft
        LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
        LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id`

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
            var typeColumn sql.NullString // Используем NullString для обработки NULL значений
            if err := rows.Scan(
                &firstType.ID,
                &firstType.FirstSubjectID, &firstType.FirstSubjectName, &firstType.FirstSubjectScore,
                &firstType.SecondSubjectID, &firstType.SecondSubjectName, &firstType.SecondSubjectScore,
                &firstType.HistoryOfKazakhstan, 
                &firstType.MathematicalLiteracy, 
                &firstType.ReadingLiteracy,
                &typeColumn, // Получаем тип как NullString
                &firstType.TotalScore, // Обновляем на total_score
            ); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Types"})
                return
            }

            // Преобразуем тип из NullString в обычный string, если значение не NULL
            if typeColumn.Valid {
                firstType.Type = typeColumn.String
            } else {
                firstType.Type = "" // Если значение NULL, то присваиваем пустую строку
            }

            types = append(types, firstType)
        }

        utils.ResponseJSON(w, types)
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
            ft.first_subject_id, 
            fs.subject AS first_subject_name, 
            COALESCE(fs.score, 0) AS first_subject_score,
            ft.second_subject_id, 
            ss.subject AS second_subject_name, 
            COALESCE(ss.score, 0) AS second_subject_score,
            COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan, 
            COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy, 
            COALESCE(ft.reading_literacy, 0) AS reading_literacy,
            ft.type,
            (COALESCE(fs.score, 0) + COALESCE(ss.score, 0) + COALESCE(ft.history_of_kazakhstan, 0) + COALESCE(ft.mathematical_literacy, 0) + COALESCE(ft.reading_literacy, 0)) AS total_score
        FROM First_Type ft
        LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
        LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
        WHERE ft.school_id = ?`  // Фильтрация по school_id

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
            var typeColumn sql.NullString
            if err := rows.Scan(
                &firstType.ID,
                &firstType.FirstSubjectID, &firstType.FirstSubjectName, &firstType.FirstSubjectScore,
                &firstType.SecondSubjectID, &firstType.SecondSubjectName, &firstType.SecondSubjectScore,
                &firstType.HistoryOfKazakhstan, 
                &firstType.MathematicalLiteracy, 
                &firstType.ReadingLiteracy,
                &typeColumn, 
                &firstType.TotalScore,
            ); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Types"})
                return
            }

            if typeColumn.Valid {
                firstType.Type = typeColumn.String
            } else {
                firstType.Type = ""
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
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Получаем userID из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Проверяем роль пользователя (директор)
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
        query := `INSERT INTO Second_Type (history_of_kazakhstan_creative, reading_literacy_creative, creative_exam1, creative_exam2, school_id, total_score_creative, type) 
                  VALUES (?, ?, ?, ?, ?, ?, ?)`
        _, err = db.Exec(query, 
            secondType.HistoryOfKazakhstanCreative, 
            secondType.ReadingLiteracyCreative, 
            secondType.CreativeExam1, 
            secondType.CreativeExam2,
            userSchoolID.Int64, 
            totalScoreCreative,
            secondType.Type) // Сохраняем тип как 'type-2'

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
            if err := rows.Scan(
                &secondType.ID, 
                &secondType.HistoryOfKazakhstanCreative, 
                &secondType.ReadingLiteracyCreative, 
                &secondType.CreativeExam1,
                &secondType.CreativeExam2,
                &secondType.TotalScoreCreative,
                &secondType.Type,
            ); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Types"})
                return
            }
            types = append(types, secondType)
        }

        utils.ResponseJSON(w, types)
    }
}




