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
		// Step 1: Verify the user's token and get userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role and school ID
		var userRole string
		var userSchoolID sql.NullInt64 // Using sql.NullInt64 to handle NULL values
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Error fetching user role and school ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Ensure the user is a director and has a school assigned
		if userRole != "director" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create a student"})
			return
		}

		if !userSchoolID.Valid {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Director does not have an assigned school"})
			return
		}

		// Step 4: Decode the student data from the request
		var student models.Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Step 5: Ensure the student's school ID matches the director's school ID
		if student.SchoolID != int(userSchoolID.Int64) {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only create students for your school"})
			return
		}

		// Step 6: Insert the student into the database
		query := `INSERT INTO Student (first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, parent_name, parent_phone_number) 
		          VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`

		// Execute the query
		result, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN, student.SchoolID, student.DateOfBirth, student.Grade, student.ParentName, student.ParentPhoneNumber)
		if err != nil {
			log.Println("Error inserting student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create student"})
			return
		}

		// Step 7: Retrieve the student's ID from the result of the insert query
		studentID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error retrieving student ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve student ID"})
			return
		}

		// Set the ID of the student object
		student.ID = int(studentID)

		// Step 8: Respond with the newly created student
		utils.ResponseJSON(w, student)
	}
}
func (sc StudentController) GetStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Query the database for student details, including parent info
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, parent_name, parent_phone_number FROM Student")
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get students"})
			return
		}
		defer rows.Close()

		var students []models.Student
		for rows.Next() {
			var student models.Student

			// Step 2: Handle the possibility of NULL values using sql.Null types
			var firstName, lastName, patronymic, iin, parentName, parentPhoneNumber sql.NullString
			var schoolID sql.NullInt64
			var dateOfBirth sql.NullString
			var grade sql.NullInt64

			// Step 3: Scan the row into the student object
			if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &parentName, &parentPhoneNumber); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse students"})
				return
			}

			// Step 4: Assign valid values to the student struct
			if firstName.Valid {
				student.FirstName = firstName.String
			}
			if lastName.Valid {
				student.LastName = lastName.String
			}
			if patronymic.Valid {
				student.Patronymic = patronymic.String
			}
			if iin.Valid {
				student.IIN = iin.String
			}
			if schoolID.Valid {
				student.SchoolID = int(schoolID.Int64)
			}
			if dateOfBirth.Valid {
				student.DateOfBirth = dateOfBirth.String
			}
			if grade.Valid {
				student.Grade = int(grade.Int64)
			}
			if parentName.Valid {
				student.ParentName = parentName.String
			}
			if parentPhoneNumber.Valid {
				student.ParentPhoneNumber = parentPhoneNumber.String
			}

			// Step 5: Append the student to the students slice
			students = append(students, student)
		}

		// Step 6: Respond with the list of students
		utils.ResponseJSON(w, students)
	}
}
func (sc StudentController) UpdateStudent(db *sql.DB) http.HandlerFunc {
// UpdateStudent is an HTTP handler for updating a student's details.
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract student_id from the query parameters
		studentID := r.URL.Query().Get("student_id")
		if studentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID is required"})
			return
		}

		// First, check if the student exists
		var existingStudent models.Student
		err := db.QueryRow("SELECT student_id FROM Student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
		if err != nil {
			if err == sql.ErrNoRows {
			// If the student does not exist, respond with a 404 error
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student not found"})
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			} else {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
				// If there was an error fetching the student, respond with a 500 error
			}
			return
		}

		// Decode the updated student data from the request body
		var updatedStudent models.Student
		if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			// If the request body is invalid, respond with a 400 error
			return
		}

		// Prepare the update query
		query := `UPDATE Student 
				  SET first_name = ?, last_name = ?, patronymic = ?, iin = ?, date_of_birth = ?, grade = ?, school_id = ?, parent_name = ?, parent_phone_number = ? 
				  WHERE student_id = ?`

		// Execute the update query
		_, err = db.Exec(query, updatedStudent.FirstName, updatedStudent.LastName, updatedStudent.Patronymic, updatedStudent.IIN, updatedStudent.DateOfBirth, updatedStudent.Grade, updatedStudent.SchoolID, updatedStudent.ParentName, updatedStudent.ParentPhoneNumber, studentID)
		if err != nil {
			log.Println("Error updating student:", err)
			// If there was an error updating the student, respond with a 500 error
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update student"})
			return
		}

		// Respond with the updated student
		utils.ResponseJSON(w, updatedStudent)
	}
}
func (sc StudentController) DeleteStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get student_id from the query parameters
		studentID := r.URL.Query().Get("student_id")
		if studentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID is required"})
			return
		}

		// Check if the student exists
		var existingStudent models.Student
		err := db.QueryRow("SELECT student_id FROM Student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student not found"})
			} else {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
			}
			return
		}

		// Delete the student from the database
		_, err = db.Exec("DELETE FROM Student WHERE student_id = ?", studentID)
		if err != nil {
			log.Println("Error deleting student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete student"})
			return
		}

		// Respond with a success message
		utils.ResponseJSON(w, map[string]string{"message": "Student deleted successfully"})
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
			totalScore = result.FirstSubjectScore + result.SecondSubjectScore + result.HistoryKazakhstan + result.MathematicalLiteracy + result.ReadingLiteracy
		} else if result.UNTTypeID == 2 {
			totalScore = result.HistoryKazakhstan + result.ReadingLiteracy
		}

		// Правильный запрос на вставку данных
		query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, first_subject_score, second_subject_score, history_of_kazakhstan, math_literacy, reading_literacy, total_score) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = db.Exec(query, result.Year, result.UNTTypeID, result.StudentID, result.FirstSubjectScore, result.SecondSubjectScore, result.HistoryKazakhstan, result.MathematicalLiteracy, result.ReadingLiteracy, totalScore)
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
				&result.MathematicalLiteracy, &result.ReadingLiteracy, &result.TotalScore); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse UNT results"})
				return
			}
			results = append(results, result)
		}

		utils.ResponseJSON(w, results)
	}
}


