package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

type SchoolController struct{}

func (sc SchoolController) CreateSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var school models.School

		if err := json.NewDecoder(r.Body).Decode(&school); err != nil {
			log.Println("JSON Decode Error:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Проверяем существование UNT Score
		var untExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM UNT_Score WHERE unt_score_id = ?)", school.UNTID).Scan(&untExists)
		if err != nil || !untExists {
			log.Println("UNT Score ID not found:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "UNT Score ID does not exist"})
			return
		}

		// Проверяем существование студента
		var studentExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Student WHERE student_id = ?)", school.ID).Scan(&studentExists)
		if err != nil || !studentExists {
			log.Println("Student ID not found:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID does not exist"})
			return
		}

		// Вставляем школу в БД
		query := "INSERT INTO School (unt_id, student_id, avg_unt_score) VALUES (?, ?, ?)"
		result, err := db.Exec(query, school.UNTID, school.ID, school.AvgUNTScore)
		if err != nil {
			log.Println("SQL Insert Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create school"})
			return
		}

		id, _ := result.LastInsertId()
		school.SchoolID = int(id)

		utils.ResponseJSON(w, school)
	}
}
func (sc SchoolController) GetSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT school_id, unt_id, student_id, avg_unt_score FROM School")
		if err != nil {
			log.Println("SQL Select Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get schools"})
			return
		}
		defer rows.Close()

		var schools []models.School
		for rows.Next() {
			var school models.School
			if err := rows.Scan(&school.SchoolID, &school.UNTID, &school.ID, &school.AvgUNTScore); err != nil {
				log.Println("SQL Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse schools"})
				return
			}
			schools = append(schools, school)
		}

		utils.ResponseJSON(w, schools)
	}
}
func (sc SchoolController) CreateStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var student models.Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		query := `INSERT INTO Student (first_name, last_name, patronymic, iin) VALUES(?, ?, ?, ?)`
		_, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create student"})
			return
		}

		utils.ResponseJSON(w, "Student created successfully")
	}
}
func (sc SchoolController) CreateUNTType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var untType models.UNTType
		if err := json.NewDecoder(r.Body).Decode(&untType); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		query := `INSERT INTO UNT_Type (first_type_id, second_type_id) VALUES(?, ?)`
		_, err := db.Exec(query, untType.FirstTypeID, untType.SecondTypeID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT Type"})
			return
		}

		utils.ResponseJSON(w, "UNT Type created successfully")
	}
}
func (sc SchoolController) CalculateScore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var score models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		totalScore := score.FirstSubjectScore + score.SecondSubjectScore + score.HistoryKazakhstan + score.MathLiteracy + score.ReadingLiteracy
		score.TotalScore = totalScore

		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, first_subject_score, second_subject_score, history_of_kazakhstan, mathematical_literacy, reading_literacy, score) 
				VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, score.Year, score.UNTTypeID, score.StudentID, score.FirstSubjectScore, score.SecondSubjectScore, score.HistoryKazakhstan, score.MathLiteracy, score.ReadingLiteracy, score.TotalScore)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate and save score"})
			return
		}

		utils.ResponseJSON(w, "Score calculated and saved successfully")
	}
}