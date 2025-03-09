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

// Create UNT Type by selecting either First_Type or Second_Type
func (sc UNTTypeController) CreateUNTType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var untType models.UNTType
		if err := json.NewDecoder(r.Body).Decode(&untType); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Check if only one type is provided (either First_Type or Second_Type)
		if (untType.FirstTypeID == nil && untType.SecondTypeID == nil) || (untType.FirstTypeID != nil && untType.SecondTypeID != nil) {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "You must provide either First_Type or Second_Type, but not both"})
			return
		}

		// Check if the provided First_Type exists
		if untType.FirstTypeID != nil {
			var firstTypeExists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM First_Type WHERE first_type_id = ?)", *untType.FirstTypeID).Scan(&firstTypeExists)
			if err != nil || !firstTypeExists {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "First Type ID does not exist"})
				return
			}
		}

		// Check if the provided Second_Type exists
		if untType.SecondTypeID != nil {
			var secondTypeExists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM Second_Type WHERE second_type_id = ?)", *untType.SecondTypeID).Scan(&secondTypeExists)
			if err != nil || !secondTypeExists {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Second Type ID does not exist"})
				return
			}
		}

		// Insert into UNT_Type
		query := `INSERT INTO UNT_Type (first_type_id, second_type_id) VALUES (?, ?)`
		_, err := db.Exec(query, utils.NullableValue(untType.FirstTypeID), utils.NullableValue(untType.SecondTypeID))
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT Type"})
			return
		}

		utils.ResponseJSON(w, "UNT Type created successfully")
	}
}

// Get all UNT Types with their respective names
// Get all UNT Types with their respective names
func (sc UNTTypeController) GetUNTTypes(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        query := `
            SELECT 
                ft.first_type_id, 
                fs.first_subject_id, fs.subject AS first_subject_name, 
                COALESCE(fs.score, 0) AS first_subject_score,
                ss.second_subject_id, ss.subject AS second_subject_name, 
                COALESCE(ss.score, 0) AS second_subject_score,
                COALESCE(ft.history_of_kazakhstan, 0) AS history_of_kazakhstan, 
                COALESCE(ft.mathematical_literacy, 0) AS mathematical_literacy,
                COALESCE(ft.reading_literacy, 0) AS reading_literacy
            FROM First_Type ft
            LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
            LEFT JOIN Second_Subject ss ON ft.second_subject_id = ss.second_subject_id
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

            // Scan the values directly
            if err := rows.Scan(
                &untType.FirstTypeID,
                &firstSubjectID, &firstSubjectName, &firstSubjectScore,
                &secondSubjectID, &secondSubjectName, &secondSubjectScore,
                &historyKazakhstan, &mathematicalLiteracy, &readingLiteracy,
            ); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Types"})
                return
            }

            // Handle the scanned sql.Null* values
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

            // Add the populated UNTType to the result slice
            types = append(types, untType)
        }

        // Send the response
        utils.ResponseJSON(w, types)
    }
}

