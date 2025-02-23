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
func (c *TypeController) GetFirstTypes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT first_type_id, first_subject_id, second_subject_id, history_of_kazakhstan, mathematical_literacy, reading_literacy FROM First_Type")
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get First Types"})
			return
		}
		defer rows.Close()

		var types []models.FirstType
		for rows.Next() {
			var firstType models.FirstType
			if err := rows.Scan(&firstType.ID, &firstType.FirstSubjectID, &firstType.SecondSubjectID, &firstType.HistoryOfKazakhstan, &firstType.MathematicalLiteracy, &firstType.ReadingLiteracy); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse First Types"})
				return
			}
			types = append(types, firstType)
		}

		utils.ResponseJSON(w, types)
	}
}
func (c *TypeController) CreateFirstType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var firstType models.FirstType
		if err := json.NewDecoder(r.Body).Decode(&firstType); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		query := `INSERT INTO First_Type (history_of_kazakhstan, mathematical_literacy, reading_literacy, first_subject_id, second_subject_id) 
                  VALUES(?, ?, ?, ?, ?)`
		_, err := db.Exec(query, firstType.HistoryOfKazakhstan, firstType.MathematicalLiteracy, firstType.ReadingLiteracy, firstType.FirstSubjectID, firstType.SecondSubjectID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create First Type"})
			return
		}

		utils.ResponseJSON(w, "First Type created successfully")
	}
}
func (c *TypeController) CreateSecondType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var secondType models.SecondType
		if err := json.NewDecoder(r.Body).Decode(&secondType); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		query := `INSERT INTO Second_Type (history_of_kazakhstan, reading_literacy) VALUES(?, ?)`
		_, err := db.Exec(query, secondType.HistoryOfKazakhstan, secondType.ReadingLiteracy)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create Second Type"})
			return
		}

		utils.ResponseJSON(w, "Second Type created successfully")
	}
}
func (c *TypeController) GetSecondTypes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT second_type_id, history_of_kazakhstan, reading_literacy FROM Second_Type")
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
