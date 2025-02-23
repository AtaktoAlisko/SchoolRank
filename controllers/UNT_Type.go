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
func (sc UNTTypeController) CreateUNTType(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var untType models.UNTType
        if err := json.NewDecoder(r.Body).Decode(&untType); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Ensure that first_type and second_type are linked and valid
        // Create the UNT_Type entry by inserting first_type and second_type references
        query := `INSERT INTO UNT_Type (first_type_id, second_type_id, history_of_kazakhstan, mathematical_literacy, reading_literacy) 
                  VALUES(?, ?, ?, ?, ?)`

        // Executing the query to insert data into UNT_Type table
        _, err := db.Exec(query, untType.FirstTypeID, untType.SecondTypeID, untType.HistoryKazakh, untType.MathLiteracy, untType.ReadingLiteracy)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT Type"})
            return
        }

        utils.ResponseJSON(w, "UNT Type created successfully")
    }
}
func (sc UNTTypeController) GetUNTTypes(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT unt_type_id, first_type_id, second_type_id, history_of_kazakhstan, mathematical_literacy, reading_literacy FROM UNT_Type")
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get UNT Types"})
            return
        }
        defer rows.Close()

        var types []models.UNTType
        for rows.Next() {
            var untType models.UNTType
            if err := rows.Scan(&untType.UNTTypeID, &untType.FirstTypeID, &untType.SecondTypeID, &untType.HistoryKazakh, &untType.MathLiteracy, &untType.ReadingLiteracy); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT Types"})
                return
            }
            types = append(types, untType)
        }

        utils.ResponseJSON(w, types)
    }
}


