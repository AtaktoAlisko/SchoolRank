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
        _, err := db.Exec(query, untType.FirstTypeID, untType.SecondTypeID)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT Type"})
            return
        }

        utils.ResponseJSON(w, "UNT Type created successfully")
    }
}

// Get all UNT Types with their respective names
func (sc UNTTypeController) GetUNTTypes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT 
				ut.unt_type_id, 
				ft.first_type_id, fs.first_subject_id, fs.subject AS first_subject_name,
				st.second_type_id, st.history_of_kazakhstan, st.reading_literacy
			FROM UNT_Type ut
			LEFT JOIN First_Type ft ON ut.first_type_id = ft.first_type_id
			LEFT JOIN First_Subject fs ON ft.first_subject_id = fs.first_subject_id
			LEFT JOIN Second_Type st ON ut.second_type_id = st.second_type_id;
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
			var firstTypeID, secondTypeID, firstSubjectID, historyKazakhstan, readingLiteracy sql.NullInt64
			var firstSubjectName sql.NullString

			if err := rows.Scan(
				&untType.UNTTypeID, &firstTypeID, &firstSubjectID, &firstSubjectName,
				&secondTypeID, &historyKazakhstan, &readingLiteracy,
			); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Types"})
				return
			}
			if firstTypeID.Valid {
				untType.FirstTypeID = new(int)
				*untType.FirstTypeID = int(firstTypeID.Int64)
			}
			if firstSubjectID.Valid {
				untType.FirstSubjectID = new(int)
				*untType.FirstSubjectID = int(firstSubjectID.Int64)
			}
			if firstSubjectName.Valid {
				untType.FirstSubjectName = new(string)
				*untType.FirstSubjectName = firstSubjectName.String
			}
			if secondTypeID.Valid {
				untType.SecondTypeID = new(int)
				*untType.SecondTypeID = int(secondTypeID.Int64)
			}
			if historyKazakhstan.Valid {
				untType.HistoryKazakhstan = new(int)
				*untType.HistoryKazakhstan = int(historyKazakhstan.Int64)
			}
			if readingLiteracy.Valid {
				untType.ReadingLiteracy = new(int)
				*untType.ReadingLiteracy = int(readingLiteracy.Int64)
			}

			types = append(types, untType)
		}

		utils.ResponseJSON(w, types)
	}
}
