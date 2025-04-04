package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type UNTTypeController struct{}

// Функция создания UNT типа
func (sc UNTTypeController) CreateUNTType(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var untType models.UNTType
        if err := json.NewDecoder(r.Body).Decode(&untType); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Проверяем, что предоставлен только один тип (либо First_Type, либо Second_Type)
        if (untType.FirstTypeID == nil && untType.SecondTypeID == nil) || (untType.FirstTypeID != nil && untType.SecondTypeID != nil) {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "You must provide either First_Type or Second_Type, but not both"})
            return
        }

        // Устанавливаем тип (type-1 или type-2) в зависимости от того, какой тип был передан
        if untType.FirstTypeID != nil {
            untType.Type = "type-1" // Первый тип
        } else if untType.SecondTypeID != nil {
            untType.Type = "type-2" // Второй тип
        }

        // Проверка существования First_Type, если передан first_type_id
        if untType.FirstTypeID != nil {
            var firstTypeExists bool
            err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM First_Type WHERE first_type_id = ?)", *untType.FirstTypeID).Scan(&firstTypeExists)
            if err != nil || !firstTypeExists {
                utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "First Type ID does not exist"})
                return
            }

            // Рассчитываем total_score для первого типа
            totalScore := 0
            if untType.FirstSubjectScore != nil {
                totalScore += *untType.FirstSubjectScore
            }
            if untType.SecondSubjectScore != nil {
                totalScore += *untType.SecondSubjectScore
            }
            if untType.HistoryKazakhstan != nil {
                totalScore += *untType.HistoryKazakhstan
            }
            if untType.MathematicalLiteracy != nil {
                totalScore += *untType.MathematicalLiteracy
            }
            if untType.ReadingLiteracy != nil {
                totalScore += *untType.ReadingLiteracy
            }

            untType.TotalScore = new(int)
            *untType.TotalScore = totalScore
        }

        // Проверка существования Second_Type, если передан second_type_id
        if untType.SecondTypeID != nil {
            var secondTypeExists bool
            err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM Second_Type WHERE second_type_id = ?)", *untType.SecondTypeID).Scan(&secondTypeExists)
            if err != nil || !secondTypeExists {
                utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Second Type ID does not exist"})
                return
            }

            // Рассчитываем total_score_creative для второго типа
            totalScoreCreative := 0
            if untType.SecondTypeHistoryKazakhstan != nil {
                totalScoreCreative += *untType.SecondTypeHistoryKazakhstan
            }
            if untType.SecondTypeReadingLiteracy != nil {
                totalScoreCreative += *untType.SecondTypeReadingLiteracy
            }
            if untType.CreativeExam1 != nil {
                totalScoreCreative += *untType.CreativeExam1
            }
            if untType.CreativeExam2 != nil {
                totalScoreCreative += *untType.CreativeExam2
            }

            untType.TotalScoreCreative = new(int)
            *untType.TotalScoreCreative = totalScoreCreative
        }

        // Вставка в UNT_Type таблицу
        query := `INSERT INTO UNT_Type (first_type_id, second_type_id, second_type_history_kazakhstan, second_type_reading_literacy, creative_exam1, creative_exam2, total_score, total_score_creative, type) 
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
        _, err := db.Exec(query, utils.NullableValue(untType.FirstTypeID), utils.NullableValue(untType.SecondTypeID),
            utils.NullableValue(untType.SecondTypeHistoryKazakhstan), utils.NullableValue(untType.SecondTypeReadingLiteracy),
            utils.NullableValue(untType.CreativeExam1), utils.NullableValue(untType.CreativeExam2),
            utils.NullableValue(untType.TotalScore), utils.NullableValue(untType.TotalScoreCreative), untType.Type)  // Добавляем тип в запрос
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT Type"})
            return
        }

        utils.ResponseJSON(w, "UNT Type created successfully")
    }
}
func (sc *UNTTypeController) GetUNTTypes(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        query := `
            SELECT 
                ut.unt_type_id, 
                ut.type AS unt_type, 
                ft.first_type_id, 
                fs.first_subject_id, fs.subject AS first_subject_name, 
                COALESCE(fs.score, 0) AS first_subject_score,
                ss.second_subject_id, ss.subject AS second_subject_name, 
                COALESCE(ss.score, 0) AS second_subject_score,
                COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan, 
                COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy,
                COALESCE(ft.reading_literacy, 0) AS reading_literacy,
                st.second_type_id,
                st.history_of_kazakhstan_creative,
                st.reading_literacy_creative,
                st.creative_exam1,
                st.creative_exam2,
                (COALESCE(fs.score, 0) + COALESCE(ss.score, 0) + COALESCE(ft.history_of_kazakhstan, 0) + COALESCE(ft.mathematical_literacy, 0) + COALESCE(ft.reading_literacy, 0)) AS total_score,
                (COALESCE(st.history_of_kazakhstan_creative, 0) + COALESCE(st.reading_literacy_creative, 0) + COALESCE(st.creative_exam1, 0) + COALESCE(st.creative_exam2, 0)) AS total_score_creative
            FROM UNT_Type ut
            LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
            LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
            LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
            LEFT JOIN Second_Type st ON ut.second_type_id = st.second_type_id
        `

        rows, err := db.Query(query)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get UNT Types"})
            return
        }
        defer rows.Close()

        var types []models.UNTType
        for rows.Next() {
            var untType models.UNTType
            var firstSubjectID, secondSubjectID, historyKazakhstan, mathematicalLiteracy, readingLiteracy sql.NullInt64
            var firstSubjectName, secondSubjectName sql.NullString
            var firstSubjectScore, secondSubjectScore sql.NullInt64
            var secondTypeID sql.NullInt64
            var secondTypeHistoryOfKazakhstan, secondTypeReadingLiteracy, creativeExam1, creativeExam2 sql.NullInt64

            if err := rows.Scan(
                &untType.UNTTypeID,
                &untType.Type,
                &untType.FirstTypeID,
                &firstSubjectID, &firstSubjectName, &firstSubjectScore,
                &secondSubjectID, &secondSubjectName, &secondSubjectScore,
                &historyKazakhstan, &mathematicalLiteracy, &readingLiteracy,
                &secondTypeID,
                &secondTypeHistoryOfKazakhstan, &secondTypeReadingLiteracy,
                &creativeExam1, &creativeExam2,
                &untType.TotalScore,
                &untType.TotalScoreCreative,
            ); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Types"})
                return
            }

            if firstSubjectID.Valid {
                untType.FirstSubjectID = new(int)
                *untType.FirstSubjectID = int(firstSubjectID.Int64)
            }
            if firstSubjectName.Valid {
                untType.FirstSubjectName = new(string)
                *untType.FirstSubjectName = firstSubjectName.String
            }
            if firstSubjectScore.Valid {
                untType.FirstSubjectScore = new(int)
                *untType.FirstSubjectScore = int(firstSubjectScore.Int64)
            }
            if secondSubjectName.Valid {
                untType.SecondSubjectName = new(string)
                *untType.SecondSubjectName = secondSubjectName.String
            }
            if secondSubjectScore.Valid {
                untType.SecondSubjectScore = new(int)
                *untType.SecondSubjectScore = int(secondSubjectScore.Int64)
            }
            if historyKazakhstan.Valid {
                untType.HistoryKazakhstan = new(int)
                *untType.HistoryKazakhstan = int(historyKazakhstan.Int64)
            }
            if mathematicalLiteracy.Valid {
                untType.MathematicalLiteracy = new(int)
                *untType.MathematicalLiteracy = int(mathematicalLiteracy.Int64)
            }
            if readingLiteracy.Valid {
                untType.ReadingLiteracy = new(int)
                *untType.ReadingLiteracy = int(readingLiteracy.Int64)
            }

            if secondTypeID.Valid {
                untType.SecondTypeID = new(int)
                *untType.SecondTypeID = int(secondTypeID.Int64)
            }
            if secondTypeHistoryOfKazakhstan.Valid {
                untType.SecondTypeHistoryKazakhstan = new(int)
                *untType.SecondTypeHistoryKazakhstan = int(secondTypeHistoryOfKazakhstan.Int64)
            }
            if secondTypeReadingLiteracy.Valid {
                untType.SecondTypeReadingLiteracy = new(int)
                *untType.SecondTypeReadingLiteracy = int(secondTypeReadingLiteracy.Int64)
            }
            if creativeExam1.Valid {
                untType.CreativeExam1 = new(int)
                *untType.CreativeExam1 = int(creativeExam1.Int64)
            }
            if creativeExam2.Valid {
                untType.CreativeExam2 = new(int)
                *untType.CreativeExam2 = int(creativeExam2.Int64)
            }

            // Добавляем тип в результирующий список
            types = append(types, untType)
        }

        utils.ResponseJSON(w, types)
    }
}
func (tc *TypeController) GetAverageUNTScore(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        schoolID := r.URL.Query().Get("school_id")
        if schoolID == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "school_id is required"})
            return
        }

        query := `
            SELECT 
                first_subject_score, 
                second_subject_score, 
                history_of_kazakhstan, 
                mathematical_literacy, 
                reading_literacy
            FROM UNT_Type
            WHERE school_id = ? AND type = 'type-1';
        `
        
        rows, err := db.Query(query, schoolID)
        if err != nil {
            log.Printf("Error fetching UNT Type records: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch UNT Type records"})
            return
        }
        defer rows.Close()

        var totalScore float64
        var count int

        for rows.Next() {
            var firstSubjectScore, secondSubjectScore, historyKazakhstan, mathematicalLiteracy, readingLiteracy sql.NullInt64
            err := rows.Scan(&firstSubjectScore, &secondSubjectScore, &historyKazakhstan, &mathematicalLiteracy, &readingLiteracy)
            if err != nil {
                log.Printf("Error scanning row: %v", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning UNT Type records"})
                return
            }

            // Суммируем баллы всех предметов
            totalScore += float64(firstSubjectScore.Int64 + secondSubjectScore.Int64 + historyKazakhstan.Int64 + mathematicalLiteracy.Int64 + readingLiteracy.Int64)
            count++
        }

        if count == 0 {
            utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No students found"})
            return
        }

        // Рассчитываем среднюю оценку
        averageScore := totalScore / float64(count)

        // Обновляем рейтинг школы
        updateQuery := `
            UPDATE Schools
            SET average_unt_score = ?
            WHERE school_id = ?
        `
        _, err = db.Exec(updateQuery, averageScore, schoolID)
        if err != nil {
            log.Printf("Error updating school rating: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update school rating"})
            return
        }

        // Возвращаем результат
        utils.ResponseJSON(w, map[string]interface{}{
            "average_unt_score": averageScore,
        })
    }
}



