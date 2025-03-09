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

func (sc StudentController) CreateStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var student models.Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Проверяем существование School перед вставкой студента
		var schoolExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM School WHERE school_id = ?)", student.SchoolID).Scan(&schoolExists)
		if err != nil || !schoolExists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID does not exist"})
			return
		}

		query := `INSERT INTO Student (first_name, last_name, patronymic, iin, school_id) VALUES(?, ?, ?, ?, ?)`
		result, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN, student.SchoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create student"})
			return
		}

		studentID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error retrieving last insert ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve student ID"})
			return
		}
		student.ID = int(studentID)

		utils.ResponseJSON(w, student)
	}
}

func (sc StudentController) GetStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id FROM Student")
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get students"})
			return
		}
		defer rows.Close()

		var students []models.Student
		for rows.Next() {
			var student models.Student
			if err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Patronymic, &student.IIN, &student.SchoolID); err != nil {
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
		var result models.UNTScore
		if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Проверяем существование Student и UNT_Type
		var studentExists, untTypeExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM Student WHERE student_id = ?)", result.StudentID).Scan(&studentExists)
		if err != nil || !studentExists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID does not exist"})
			return
		}

		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM UNT_Type WHERE unt_type_id = ?)", result.UNTTypeID).Scan(&untTypeExists)
		if err != nil || !untTypeExists {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "UNT Type ID does not exist"})
			return
		}

		// Считаем totalScore корректно по типу
		totalScore := 0
		if result.UNTTypeID == 1 {
			totalScore = result.FirstSubjectScore + result.SecondSubjectScore + result.HistoryKazakhstan + result.MathLiteracy + result.ReadingLiteracy
		} else if result.UNTTypeID == 2 {
			totalScore = result.HistoryKazakhstan + result.ReadingLiteracy
		}

		// Правильный запрос на вставку данных
		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, first_subject_score, second_subject_score, history_of_kazakhstan, math_literacy, reading_literacy, total_score) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = db.Exec(query, result.Year, result.UNTTypeID, result.StudentID, result.FirstSubjectScore, result.SecondSubjectScore, result.HistoryKazakhstan, result.MathLiteracy, result.ReadingLiteracy, totalScore)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT score"})
			return
		}

		utils.ResponseJSON(w, "UNT results saved successfully")
	}
}

func (sc StudentController) GetUNTResults(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentID := r.URL.Query().Get("student_id")
		if studentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "student_id is required"})
			return
		}

		rows, err := db.Query(`
			SELECT unt_score_id, year, unt_type_id, student_id, 
				   first_subject_score, second_subject_score, 
				   history_of_kazakhstan, math_literacy, 
				   reading_literacy, total_score
			FROM UNT_Score 
			WHERE student_id = ?`, studentID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get UNT results"})
			return
		}
		defer rows.Close()

		var results []models.UNTScore
		for rows.Next() {
			var result models.UNTScore
			if err := rows.Scan(&result.ID, &result.Year, &result.UNTTypeID, &result.StudentID,
				&result.FirstSubjectScore, &result.SecondSubjectScore, &result.HistoryKazakhstan,
				&result.MathLiteracy, &result.ReadingLiteracy, &result.TotalScore); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT results"})
				return
			}
			results = append(results, result)
		}

		utils.ResponseJSON(w, results)
	}
}


