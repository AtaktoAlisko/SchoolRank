package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
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
        var firstType models.FirstType
        if err := json.NewDecoder(r.Body).Decode(&firstType); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Проверяем существование First_Subject и Second_Subject
        var firstSubjectExists, secondSubjectExists bool
        err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM First_Subject WHERE first_subject_id = ?)", firstType.FirstSubjectID).Scan(&firstSubjectExists)
        if err != nil || !firstSubjectExists {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "First Subject ID does not exist"})
            return
        }

        err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Second_Subject WHERE second_subject_id = ?)", firstType.SecondSubjectID).Scan(&secondSubjectExists)
        if err != nil || !secondSubjectExists {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Second Subject ID does not exist"})
            return
        }

        // Вставляем First Type в БД
        query := `INSERT INTO First_Type (first_subject_id, second_subject_id, history_of_kazakhstan, mathematical_literacy, reading_literacy) 
                  VALUES (?, ?, ?, ?, ?)`
        _, err = db.Exec(query, 
            nullableValue(firstType.FirstSubjectID), 
            nullableValue(firstType.SecondSubjectID), 
            nullableValue(firstType.HistoryOfKazakhstan), 
            nullableValue(firstType.MathematicalLiteracy), 
            nullableValue(firstType.ReadingLiteracy))

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
        query := `SELECT 
                        ft.first_type_id, 
                        ft.first_subject_id, 
                        fs.subject AS first_subject_name, 
                        COALESCE(fs.score, 0) AS first_subject_score,
                        ft.second_subject_id, 
                        ss.subject AS second_subject_name, 
                        COALESCE(ss.score, 0) AS second_subject_score,
                        COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan, 
                        COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy, 
                        COALESCE(ft.reading_literacy, 0) AS reading_literacy
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
            if err := rows.Scan(
                &firstType.ID,
                &firstType.FirstSubjectID, &firstType.FirstSubjectName, &firstType.FirstSubjectScore,
                &firstType.SecondSubjectID, &firstType.SecondSubjectName, &firstType.SecondSubjectScore,
                &firstType.HistoryOfKazakhstan, 
                &firstType.MathematicalLiteracy, 
                &firstType.ReadingLiteracy,
            ); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Types"})
                return
            }
            types = append(types, firstType)
        }

        utils.ResponseJSON(w, types)
    }
}
// Создание SecondType
func (c *TypeController) CreateSecondType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var secondType models.SecondType
		if err := json.NewDecoder(r.Body).Decode(&secondType); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Вставляем Second Type в БД
		query := `INSERT INTO Second_Type (history_of_kazakhstan, reading_literacy) VALUES (?, ?)`
		_, err := db.Exec(query, secondType.HistoryOfKazakhstan, secondType.ReadingLiteracy)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create Second Type"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "Second Type created successfully"})
	}
}

// Получение всех Second Types
func (c *TypeController) GetSecondTypes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `SELECT second_type_id, 
						 COALESCE(history_of_kazakhstan, 0), 
						 COALESCE(reading_literacy, 0) 
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
			if err := rows.Scan(&secondType.ID, &secondType.HistoryOfKazakhstan, &secondType.ReadingLiteracy); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse Second Types"})
				return
			}
			types = append(types, secondType)
		}

		utils.ResponseJSON(w, types)
	}
}
