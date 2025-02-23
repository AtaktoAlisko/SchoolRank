package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)
type StudentController struct{}
func (sc StudentController) CreateStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var students []models.Student
		if err := json.NewDecoder(r.Body).Decode(&students); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Вставка каждого студента в базу данных
		for _, student := range students {
			query := `INSERT INTO Student (first_name, last_name, patronymic, iin, school_id) VALUES(?, ?, ?, ?, ?)`
			_, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN, student.SchoolID)
			if err != nil {
				log.Println("SQL Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create student"})
				return
			}
		}

		utils.ResponseJSON(w, "Students created successfully")
	}
}
func (sc StudentController) GetStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin FROM Student")
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get students"})
			return
		}
		defer rows.Close()

		var students []models.Student
		for rows.Next() {
			var student models.Student
			if err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Patronymic, &student.IIN); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse students"})
				return
			}
			students = append(students, student)
		}

		utils.ResponseJSON(w, students)
	}
}
func (sc StudentController) CreateUNTResults(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var results []models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&results); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Обработка каждого результата экзамена
		for _, result := range results {
			// Понимание типа теста: First_Type или Second_Type
			var totalScore int
			if result.UNTTypeID == 1 { // First_Type
				totalScore = result.FirstSubjectScore + result.SecondSubjectScore + result.HistoryKazakhstan + result.MathLiteracy + result.ReadingLiteracy
			} else if result.UNTTypeID == 2 { // Second_Type
				totalScore = result.HistoryKazakhstan + result.ReadingLiteracy
			}

			// Вставка результата в таблицу UNT_Score
			query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, score, total_score) VALUES(?, ?, ?, ?, ?)`
			_, err := db.Exec(query, result.Year, result.UNTTypeID, result.StudentID, totalScore, totalScore)
			if err != nil {
				log.Println("SQL Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
				return
			}
		}

		utils.ResponseJSON(w, "UNT results created successfully")
	}
}

