package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type StudentController struct{}

func (sc *StudentController) CreateStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get user ID
		userID, err := utils.VerifyToken(r) // Return userID directly, not user struct
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role - we already have the role from the token,
		// but you might want to verify it from the database for extra security
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check if the user is a superadmin or schooladmin
		if userRole != "schooladmin" && userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to create a student"})
			return
		}

		// Step 4: Decode the student data from the request (only do this ONCE)
		var student models.Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			log.Printf("Error decoding student data: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student data"})
			return
		}

		// Step 5: Handle school ID based on user role
		if userRole == "schooladmin" {
			// For schooladmin, get the school ID associated with them
			var userEmail string
			err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
			if err != nil {
				log.Println("Error fetching user email:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user email"})
				return
			}

			var schoolID int
			err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", userEmail).Scan(&schoolID)
			if err != nil {
				log.Println("Error fetching school ID:", err)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Director does not have an assigned school. Please create a school first."})
				return
			}

			// Set the school ID for the student
			student.SchoolID = schoolID
		} else if userRole == "superadmin" {
			// For superadmin, use the school_id from the request body
			// Make sure it's provided
			if student.SchoolID <= 0 {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required for superadmin"})
				return
			}
		}

		// Validate required student fields
		if student.FirstName == "" || student.LastName == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "First name and last name are required"})
			return
		}

		// Step 6: Generate login and email for the student
		randomString := generateRandomString(4)
		student.Login = student.FirstName + student.LastName + randomString
		student.Email = student.Login + "@school.com"

		// Step 7: Generate the student's password
		student.Password = student.FirstName + student.LastName

		// Step 8: Set role to student
		student.Role = "student"

		// Step 9: Handle AvatarURL (set NULL if not provided)
		var avatarURL sql.NullString
		if student.AvatarURL != "" {
			// Set the avatar URL if provided
			avatarURL = sql.NullString{String: student.AvatarURL, Valid: true}
		} else {
			// Set it to NULL if not provided
			avatarURL = sql.NullString{Valid: false}
		}

		// Log the student data before insertion for debugging
		log.Printf("Inserting student: %+v with school_id: %d", student, student.SchoolID)

		// Step 10: Insert the student into the database
		query := `INSERT INTO student (first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email, password, role, login, avatar_url)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN, student.SchoolID, student.DateOfBirth, student.Grade, student.Letter, student.Gender, student.Phone, student.Email, student.Password, student.Role, student.Login, avatarURL)
		if err != nil {
			log.Println("Error inserting student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create student"})
			return
		}

		studentID, err := result.LastInsertId()
		if err != nil {
			log.Println("Error retrieving student ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to retrieve student ID"})
			return
		}

		student.ID = int(studentID)

		// Send response with the student data
		utils.ResponseJSON(w, student)
	}
}
func (sc *StudentController) EditStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role from the database
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check if the user is a superadmin or schooladmin
		if userRole != "schooladmin" && userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to edit a student"})
			return
		}

		// Step 4: Get student ID from URL parameters
		vars := mux.Vars(r)
		studentIDStr, ok := vars["id"]
		if !ok {
			log.Println("Student ID not found in URL")
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID not provided"})
			return
		}

		log.Printf("Attempting to edit student with ID string: %s", studentIDStr)

		studentID, err := strconv.Atoi(studentIDStr)
		if err != nil {
			log.Printf("Invalid student ID format: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
			return
		}

		log.Printf("Attempting to edit student with ID: %d", studentID)

		// Step 5: Verify that student exists before proceeding
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE id = ?)", studentID).Scan(&exists)
		if err != nil {
			log.Printf("Error checking if student exists: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}

		if !exists {
			log.Printf("Student with ID %d not found in database", studentID)
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			return
		}

		log.Printf("Student with ID %d exists, proceeding with edit", studentID)

		// Step 6: Decode the student data from the request
		var updatedStudent models.Student
		if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
			log.Printf("Error decoding student data: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student data format"})
			return
		}

		// Step 7: Handle permissions based on user role
		if userRole == "schooladmin" {
			// Fetch school ID for schooladmin
			var userEmail string
			err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
			if err != nil {
				log.Println("Error fetching user email:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user email"})
				return
			}

			var adminSchoolID int
			err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", userEmail).Scan(&adminSchoolID)
			if err != nil {
				log.Println("Error fetching school ID:", err)
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Director does not have an assigned school"})
				return
			}

			// Verify that student belongs to admin's school
			var studentSchoolID int
			err = db.QueryRow("SELECT school_id FROM student WHERE id = ?", studentID).Scan(&studentSchoolID)
			if err != nil {
				log.Printf("Error fetching student school ID: %v for student ID: %d", err, studentID)
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
				return
			}

			if studentSchoolID != adminSchoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to edit this student"})
				return
			}

			// Ensure school ID remains the same for schooladmin
			updatedStudent.SchoolID = adminSchoolID
		} else if userRole == "superadmin" {
			// Superadmin can edit any student and can change school ID
			if updatedStudent.SchoolID <= 0 {
				err = db.QueryRow("SELECT school_id FROM student WHERE id = ?", studentID).Scan(&updatedStudent.SchoolID)
				if err != nil {
					log.Printf("Error fetching student school ID: %v for student ID: %d", err, studentID)
					utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
					return
				}
			}
		}

		// Step 8: Validate required student fields
		if updatedStudent.FirstName == "" || updatedStudent.LastName == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "First name and last name are required"})
			return
		}

		// Step 9: Update student data in database
		query := `UPDATE student 
                  SET first_name = ?, last_name = ?, patronymic = ?, iin = ?, 
                      school_id = ?, date_of_birth = ?, grade = ?, letter = ?, 
                      gender = ?, phone = ?, email = ?, password = ?, role = ?, 
                      login = ?, avatar_url = ?
                  WHERE id = ?`

		_, err = db.Exec(query,
			updatedStudent.FirstName,
			updatedStudent.LastName,
			updatedStudent.Patronymic,
			updatedStudent.IIN,
			updatedStudent.SchoolID,
			updatedStudent.DateOfBirth,
			updatedStudent.Grade,
			updatedStudent.Letter,
			updatedStudent.Gender,
			updatedStudent.Phone,
			updatedStudent.Email,
			updatedStudent.Password,
			updatedStudent.Role,
			updatedStudent.Login,
			updatedStudent.AvatarURL,
			studentID)
		if err != nil {
			log.Printf("Error updating student: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update student"})
			return
		}

		// Step 10: Get updated student data to return in response
		var student models.Student
		err = db.QueryRow("SELECT id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email, role FROM student WHERE id = ?", studentID).
			Scan(&student.ID, &student.FirstName, &student.LastName, &student.Patronymic, &student.IIN, &student.SchoolID, &student.DateOfBirth, &student.Grade, &student.Letter, &student.Gender, &student.Phone, &student.Email, &student.Role)

		if err != nil {
			log.Printf("Error fetching updated student: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Student updated but failed to retrieve updated data"})
			return
		}

		// Send response with updated student data
		utils.ResponseJSON(w, student)
	}
}
func (sc *StudentController) GetAllStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set current date (hardcoded to May 25, 2025, as per context)
		currentDate := time.Date(2025, time.May, 25, 0, 0, 0, 0, time.UTC)

		// Step 1: Get all students and their associated school names from the database
		rows, err := db.Query(`
            SELECT s.student_id, s.first_name, s.last_name, s.patronymic, s.date_of_birth, s.iin, s.school_id, 
                   s.grade, s.letter, s.gender, s.phone, s.email, s.role, s.login, s.avatar_url, s.password,
                   Sc.school_name 
            FROM student s
            JOIN Schools Sc ON s.school_id = Sc.school_id`)
		if err != nil {
			log.Println("Error fetching students:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch students"})
			return
		}
		defer rows.Close()

		// Step 2: Create a slice to hold the students
		var students []models.Student

		// Step 3: Iterate through the rows and scan the student data
		for rows.Next() {
			var student models.Student
			var schoolID sql.NullInt64     // For handling NULL in school_id
			var dateOfBirth sql.NullString // For handling NULL in date_of_birth
			var grade sql.NullInt64        // For handling NULL in grade
			var letter sql.NullString      // For handling NULL in letter
			var gender sql.NullString      // For handling NULL in gender
			var phone sql.NullString       // For handling NULL in phone
			var email sql.NullString       // For handling NULL in email
			var avatarURL sql.NullString   // For handling NULL in avatar_url
			var password sql.NullString    // For handling NULL in password
			var schoolName sql.NullString  // For handling NULL in school_name

			if err := rows.Scan(
				&student.ID,
				&student.FirstName,
				&student.LastName,
				&student.Patronymic,
				&dateOfBirth, // Scan into NullString
				&student.IIN,
				&schoolID,
				&grade,
				&letter,
				&gender,
				&phone,
				&email,
				&student.Role,
				&student.Login,
				&avatarURL,
				&password,
				&schoolName,
			); err != nil {
				log.Println("Error scanning student:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to scan student data"})
				return
			}

			// Check if schoolID is valid (not NULL)
			if schoolID.Valid {
				student.SchoolID = int(schoolID.Int64)
			} else {
				student.SchoolID = 0
			}

			// Check if date_of_birth is valid (not NULL)
			if dateOfBirth.Valid {
				student.DateOfBirth = dateOfBirth.String
			} else {
				student.DateOfBirth = ""
			}

			// Calculate age based on date_of_birth
			student.Age = calculateAge(student.DateOfBirth, currentDate)

			// Check if grade is valid (not NULL)
			if grade.Valid {
				student.Grade = int(grade.Int64)
			} else {
				student.Grade = 0
			}

			// Check if letter is valid (not NULL)
			if letter.Valid {
				student.Letter = letter.String
			} else {
				student.Letter = ""
			}

			// Check if gender is valid (not NULL)
			if gender.Valid {
				student.Gender = gender.String
			} else {
				student.Gender = ""
			}

			// Check if phone is valid (not NULL)
			if phone.Valid {
				student.Phone = phone.String
			} else {
				student.Phone = ""
			}

			// Check if email is valid (not NULL)
			if email.Valid {
				student.Email = email.String
			} else {
				student.Email = ""
			}

			// Check if avatarURL is valid (not NULL)
			if avatarURL.Valid {
				student.AvatarURL = avatarURL.String
			} else {
				student.AvatarURL = ""
			}

			// Check if password is valid (not NULL)
			if password.Valid {
				student.Password = password.String
			} else {
				student.Password = ""
			}

			// Check if schoolName is valid (not NULL)
			if schoolName.Valid {
				student.SchoolName = schoolName.String
			} else {
				student.SchoolName = ""
			}

			// Append the student to the slice
			students = append(students, student)
		}

		// Step 4: Handle case where no students were found
		if len(students) == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No students found"})
			return
		}

		// Step 5: Return the list of students in the response
		utils.ResponseJSON(w, students)
	}
}
func (sc *StudentController) GetStudentByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set current date (hardcoded to May 25, 2025, as per context)
		currentDate := time.Date(2025, time.May, 25, 0, 0, 0, 0, time.UTC)

		// Step 1: Get studentID from URL parameters
		vars := mux.Vars(r)
		studentIDStr, ok := vars["id"]
		if !ok {
			log.Println("Student ID not found in URL")
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Student ID not provided"})
			return
		}

		studentID, err := strconv.Atoi(studentIDStr)
		if err != nil {
			log.Printf("Invalid student ID format: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
			return
		}

		// Step 2: Fetch student data with school name
		var student models.Student
		var schoolID sql.NullInt64
		var dateOfBirth sql.NullString
		var grade sql.NullInt64
		var letter sql.NullString
		var gender sql.NullString
		var phone sql.NullString
		var email sql.NullString
		var avatarURL sql.NullString
		var password sql.NullString
		var schoolName sql.NullString

		err = db.QueryRow(`
			SELECT s.student_id, s.first_name, s.last_name, s.patronymic, s.date_of_birth, s.iin, s.school_id, 
				   s.grade, s.letter, s.gender, s.phone, s.email, s.role, s.login, s.avatar_url, s.password,
				   Sc.school_name 
			FROM student s
			JOIN Schools Sc ON s.school_id = Sc.school_id
			WHERE s.student_id = ?`, studentID).
			Scan(
				&student.ID,
				&student.FirstName,
				&student.LastName,
				&student.Patronymic,
				&dateOfBirth,
				&student.IIN,
				&schoolID,
				&grade,
				&letter,
				&gender,
				&phone,
				&email,
				&student.Role,
				&student.Login,
				&avatarURL,
				&password,
				&schoolName,
			)

		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Student with ID %d not found", studentID)
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			} else {
				log.Printf("Error fetching student: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch student"})
			}
			return
		}

		// Step 3: Handle NULL values
		if schoolID.Valid {
			student.SchoolID = int(schoolID.Int64)
		} else {
			student.SchoolID = 0
		}

		if dateOfBirth.Valid {
			student.DateOfBirth = dateOfBirth.String
		} else {
			student.DateOfBirth = ""
		}

		// Calculate age
		student.Age = calculateAge(student.DateOfBirth, currentDate)

		if grade.Valid {
			student.Grade = int(grade.Int64)
		} else {
			student.Grade = 0
		}

		if letter.Valid {
			student.Letter = letter.String
		} else {
			student.Letter = ""
		}

		if gender.Valid {
			student.Gender = gender.String
		} else {
			student.Gender = ""
		}

		if phone.Valid {
			student.Phone = phone.String
		} else {
			student.Phone = ""
		}

		if email.Valid {
			student.Email = email.String
		} else {
			student.Email = ""
		}

		if avatarURL.Valid {
			student.AvatarURL = avatarURL.String
		} else {
			student.AvatarURL = ""
		}

		if password.Valid {
			student.Password = password.String
		} else {
			student.Password = ""
		}

		if schoolName.Valid {
			student.SchoolName = schoolName.String
		} else {
			student.SchoolName = ""
		}

		// Step 4: Return the student data in the response
		utils.ResponseJSON(w, student)
	}
}
func (c *StudentController) GetStudentsBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из URL параметров
		vars := mux.Vars(r)
		schoolIDParam := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDParam) // Преобразуем school_id из строки в целое число
		if err != nil {
			log.Println("Error converting school_id:", err) // Логируем ошибку преобразования
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Шаг 1: Запрос к базе данных для получения студентов по school_id
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email, login, password FROM student WHERE school_id = ?", schoolID)
		if err != nil {
			log.Println("SQL Error:", err) // Логируем ошибку SQL запроса
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get students"})
			return
		}
		defer rows.Close()

		// Шаг 2: Создаем срез для хранения студентов
		var students []models.Student
		for rows.Next() {
			var student models.Student

			// Шаг 3: Обработка возможных NULL значений с помощью sql.Null типов
			var firstName, lastName, patronymic, iin, letter, gender, phone, email, login, password sql.NullString
			var schoolID sql.NullInt64
			var dateOfBirth sql.NullString
			var grade sql.NullInt64

			// Шаг 4: Сканирование строки в объект student
			if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email, &login, &password); err != nil {
				log.Println("Scan Error:", err) // Логируем ошибку сканирования
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse students"})
				return
			}

			// Шаг 5: Присваиваем значения студенту, проверяя на NULL
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
			if letter.Valid {
				student.Letter = letter.String
			}
			if gender.Valid {
				student.Gender = gender.String
			}
			if phone.Valid {
				student.Phone = phone.String
			}
			if email.Valid {
				student.Email = email.String
			}
			if login.Valid {
				student.Login = login.String
			}
			if password.Valid {
				student.Password = password.String
			}

			// Шаг 6: Добавляем студента в срез
			students = append(students, student)
		}

		// Шаг 7: Проверяем на ошибки при обработке строк
		if err := rows.Err(); err != nil {
			log.Println("Rows Error:", err) // Логируем ошибку обработки строк
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing students"})
			return
		}

		// Шаг 8: Ответ с данным списком студентов
		utils.ResponseJSON(w, students)
	}
}
func (sc StudentController) GetAvailableLettersByGrade(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract grade and school_id from URL
		vars := mux.Vars(r)
		gradeParam := vars["grade"]
		schoolIDParam := vars["school_id"]

		// Convert grade and school_id to integers
		grade, err := strconv.Atoi(gradeParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade"})
			return
		}

		schoolID, err := strconv.Atoi(schoolIDParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Check if any student exists with this grade and school_id
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM student WHERE grade = ? AND school_id = ? LIMIT 1)", grade, schoolID).Scan(&exists)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database error"})
			return
		}

		if !exists {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No students found in this grade and school"})
			return
		}

		// Get distinct letters for this grade and school_id
		rows, err := db.Query("SELECT DISTINCT letter FROM student WHERE grade = ? AND school_id = ? ORDER BY letter", grade, schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get letters"})
			return
		}
		defer rows.Close()

		var letters []string
		for rows.Next() {
			var letter sql.NullString
			if err := rows.Scan(&letter); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse letters"})
				return
			}
			if letter.Valid && letter.String != "" {
				letters = append(letters, letter.String)
			}
		}

		if err := rows.Err(); err != nil {
			log.Println("Rows Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing letters"})
			return
		}

		type LettersResponse struct {
			AvailableLetters []string `json:"available_letters"`
			Grade            int      `json:"grade"`
			SchoolID         int      `json:"school_id"`
		}

		response := LettersResponse{
			AvailableLetters: letters,
			Grade:            grade,
			SchoolID:         schoolID,
		}

		utils.ResponseJSON(w, response)
	}
}
func (c *StudentController) GetTotalStudentsBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из URL параметров
		vars := mux.Vars(r)
		schoolIDParam := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDParam) // Преобразуем school_id из строки в целое число
		if err != nil {
			log.Println("Error converting school_id:", err) // Логируем ошибку преобразования
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Запрос к базе данных для получения количества студентов по school_id
		var totalStudents int
		err = db.QueryRow("SELECT COUNT(*) FROM student WHERE school_id = ?", schoolID).Scan(&totalStudents)
		if err != nil {
			log.Println("SQL Error:", err) // Логируем ошибку SQL запроса
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get total students"})
			return
		}

		// Возвращаем количество студентов в ответе
		response := map[string]interface{}{
			"school_id":      schoolID,
			"total_students": totalStudents,
		}

		utils.ResponseJSON(w, response)
	}
}
func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// generateRandomString generates a random string of length n
	result := make([]byte, n)
	// The string of characters to use for the random string
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))

		// Create a byte array of length n
		if err != nil {
			log.Fatal(err)
		}
		result[i] = letters[num.Int64()]
	}
	return string(result)
}
func (sc StudentController) GetStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Query the database for student details, including the email, login, and password
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email, login, password, role FROM student")
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
			var firstName, lastName, patronymic, iin, letter, gender, phone, email, login, password, role sql.NullString
			var schoolID sql.NullInt64
			var dateOfBirth sql.NullString
			var grade sql.NullInt64

			// Step 3: Scan the row into the student object
			if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email, &login, &password, &role); err != nil {
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
			if letter.Valid {
				student.Letter = letter.String
			}
			if gender.Valid {
				student.Gender = gender.String
			}
			if phone.Valid {
				student.Phone = phone.String
			}
			if email.Valid {
				student.Email = email.String
			}
			if login.Valid {
				student.Login = login.String
			}
			if password.Valid {
				student.Password = password.String // Adding password
			}
			if role.Valid {
				student.Role = role.String
			}

			// Step 5: Append the student to the students slice
			students = append(students, student)
		}

		// Step 6: Respond with the list of students
		utils.ResponseJSON(w, students)
	}
}
func (sc StudentController) GetAvailableGradesBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract school_id from URL
		vars := mux.Vars(r)
		schoolIDParam := vars["school_id"]

		// Convert parameter to required type
		schoolID, err := strconv.Atoi(schoolIDParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// No need to check if school exists - we'll just return available grades

		// Query to get distinct grades that exist for this school
		rows, err := db.Query("SELECT DISTINCT grade FROM student WHERE school_id = ? ORDER BY grade", schoolID)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get grades"})
			return
		}
		defer rows.Close()

		// Create a slice to store grades
		var grades []int

		for rows.Next() {
			var grade int
			if err := rows.Scan(&grade); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse grades"})
				return
			}
			grades = append(grades, grade)
		}

		// Check for errors in processing rows
		if err := rows.Err(); err != nil {
			log.Println("Rows Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing grades"})
			return
		}

		// Create a response structure that includes available grades
		type GradesResponse struct {
			AvailableGrades []int `json:"available_grades"`
			SchoolID        int   `json:"school_id"`
		}

		response := GradesResponse{
			AvailableGrades: grades,
			SchoolID:        schoolID,
		}

		// Return the list of available grades
		utils.ResponseJSON(w, response)
	}
}
func (sc StudentController) GetStudentsByGradeAndLetter(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем параметры grade и letter из URL
		vars := mux.Vars(r)
		gradeParam := vars["grade"]
		letterParam := vars["letter"]

		// Преобразуем grade в целое число
		grade, err := strconv.Atoi(gradeParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade"})
			return
		}

		// Шаг 1: Запрос к базе данных для получения студентов по grade и letter
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM student WHERE grade = ? AND letter = ?", grade, letterParam)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get students"})
			return
		}
		defer rows.Close()

		// Шаг 2: Создаем срез для хранения студентов
		var students []models.Student
		for rows.Next() {
			var student models.Student

			// Шаг 3: Обработка возможных NULL значений с помощью sql.Null типов
			var firstName, lastName, patronymic, iin, letter, gender, phone, email sql.NullString
			var schoolID sql.NullInt64
			var dateOfBirth sql.NullString
			var grade sql.NullInt64

			// Шаг 4: Сканирование строки в объект student
			if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse students"})
				return
			}

			// Шаг 5: Присваиваем значения студенту, проверяя на NULL
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
			if letter.Valid {
				student.Letter = letter.String
			}
			if gender.Valid {
				student.Gender = gender.String
			}
			if phone.Valid {
				student.Phone = phone.String
			}
			if email.Valid {
				student.Email = email.String
			}

			// Шаг 6: Добавляем студента в срез
			students = append(students, student)
		}

		// Шаг 7: Проверяем на ошибки при обработке строк
		if err := rows.Err(); err != nil {
			log.Println("Rows Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing students"})
			return
		}

		// Шаг 8: Ответ с данным списком студентов
		utils.ResponseJSON(w, students)
	}
}
func (sc *StudentController) UpdateStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract student_id from URL parameters
		vars := mux.Vars(r)
		studentIDParam, ok := vars["student_id"]
		if !ok {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing student ID"})
			return
		}

		studentID, err := strconv.Atoi(studentIDParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID format"})
			return
		}

		// Step 1: Verify the user's token and get userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Ensure the user is authorized (superadmin or schooladmin)
		// Superadmin can update any student from any school
		if userRole != "superadmin" && userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to update a student"})
			return
		}

		// If user is a schooladmin, check that the student belongs to the same school
		if userRole == "schooladmin" {
			var studentSchoolID int
			err = db.QueryRow("SELECT school_id FROM student WHERE student_id = ?", studentID).Scan(&studentSchoolID)
			if err != nil {
				log.Println("Error fetching student school ID:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch student details"})
				return
			}
			var userSchoolID int
			err = db.QueryRow("SELECT school_id FROM users WHERE id = ?", userID).Scan(&userSchoolID)
			if err != nil {
				log.Println("Error fetching user school ID:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
				return
			}

			if userSchoolID != studentSchoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You can only update students from your own school"})
				return
			}
		}
		// For superadmin, no additional checks needed - they can update any student

		// Step 4: Check if the student exists
		var existingStudent models.Student
		err = db.QueryRow("SELECT student_id FROM student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			} else {
				log.Println("Error checking student existence:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
			}
			return
		}

		// Step 5: Decode the updated student data from the request body
		var updatedStudent models.Student
		if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request payload"})
			return
		}

		// Step 6: Prepare and execute the update query
		query := `UPDATE student 
                  SET first_name = ?, last_name = ?, patronymic = ?, iin = ?, date_of_birth = ?, 
                  grade = ?, school_id = ?, letter = ?, gender = ?, phone = ?, email = ? 
                  WHERE student_id = ?`

		_, err = db.Exec(query,
			updatedStudent.FirstName, updatedStudent.LastName, updatedStudent.Patronymic,
			updatedStudent.IIN, updatedStudent.DateOfBirth, updatedStudent.Grade,
			updatedStudent.SchoolID, updatedStudent.Letter, updatedStudent.Gender,
			updatedStudent.Phone, updatedStudent.Email, studentID)

		if err != nil {
			log.Println("Error updating student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update student"})
			return
		}

		// Step 7: Fetch the updated student to return in the response
		query = `SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, 
                grade, letter, gender, phone, email, role, login 
                FROM student WHERE student_id = ?`

		err = db.QueryRow(query, studentID).Scan(
			&updatedStudent.ID, &updatedStudent.FirstName, &updatedStudent.LastName,
			&updatedStudent.Patronymic, &updatedStudent.IIN, &updatedStudent.SchoolID,
			&updatedStudent.DateOfBirth, &updatedStudent.Grade, &updatedStudent.Letter,
			&updatedStudent.Gender, &updatedStudent.Phone, &updatedStudent.Email,
			&updatedStudent.Role, &updatedStudent.Login)

		if err != nil {
			log.Println("Error fetching updated student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Student updated but failed to retrieve updated details"})
			return
		}

		// Step 8: Respond with the updated student
		utils.ResponseJSON(w, updatedStudent)
	}
}

// func (sc *StudentController) DeleteStudent(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Извлекаем student_id из URL параметров
// 		vars := mux.Vars(r)
// 		studentIDParam := vars["student_id"]
// 		studentID, err := strconv.Atoi(studentIDParam)
// 		if err != nil {
// 			log.Println("Invalid student ID:", studentIDParam) // Log invalid student ID
// 			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
// 			return
// 		}

// 		// Шаг 1: Проверка существования студента
// 		var existingStudent models.Student
// 		err = db.QueryRow("SELECT student_id FROM student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
// 		if err != nil {
// 			if err == sql.ErrNoRows {
// 				log.Println("No student found with ID:", studentID) // Log when student is not found
// 				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
// 			} else {
// 				log.Println("Error fetching student details:", err) // Log any other error while fetching
// 				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
// 			}
// 			return
// 		}

// 		// Шаг 2: Удаление студента из базы данных
// 		_, err = db.Exec("DELETE FROM student WHERE student_id = ?", studentID)
// 		if err != nil {
// 			log.Println("Error deleting student with ID:", studentID, err) // Log error while deleting student
// 			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete student"})
// 			return
// 		}

// 		// Шаг 3: Ответ с успешным сообщением
// 		log.Println("Student with ID:", studentID, "deleted successfully.") // Log successful deletion
// 		utils.ResponseJSON(w, map[string]string{"message": "Student deleted successfully"})
// 	}
// }

func (sc *StudentController) DeleteStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		studentIDParam := vars["student_id"]
		studentID, err := strconv.Atoi(studentIDParam)
		if err != nil {
			log.Println("Invalid student ID:", studentIDParam)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
			return
		}

		var existingStudent models.Student
		err = db.QueryRow("SELECT student_id FROM student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Println("No student found with ID:", studentID)
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			} else {
				log.Println("Error fetching student details:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
			}
			return
		}

		relatedTables := []string{
			"UNT_Exams",
			"EventRegistrations",
			"events_participants",
			"olympiad_registrations",
			"Olympiads",
			"student_ratings",
			"First_Type",
			"Second_Type",
			"UNT_Score",
			"student_types",
			"subject_olympiad_registrations",
		}

		for _, table := range relatedTables {
			query := fmt.Sprintf("DELETE FROM %s WHERE student_id = ?", table)
			_, err = db.Exec(query, studentID)
			if err != nil {
				log.Printf("Failed to delete from %s for student_id %d: %v\n", table, studentID, err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete related student data"})
				return
			}
		}

		_, err = db.Exec("DELETE FROM student WHERE student_id = ?", studentID)
		if err != nil {
			log.Println("Error deleting student with ID:", studentID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete student"})
			return
		}
		log.Println("Student with ID:", studentID, "deleted successfully.")
		utils.ResponseJSON(w, map[string]string{"message": "Student deleted successfully"})
	}
}

func (sc StudentController) SuperadminUpdateStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем student_id из URL параметров
		vars := mux.Vars(r)
		studentIDParam, ok := vars["student_id"]
		if !ok {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Missing student ID"})
			return
		}

		studentID, err := strconv.Atoi(studentIDParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID format"})
			return
		}

		// Step 1: Verify the user's token and get userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role
		var userRole string
		var userSchoolID int // To store school ID for schooladmin
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Ensure the user is authorized (superadmin or schooladmin)
		var studentSchoolID int
		err = db.QueryRow("SELECT school_id FROM Student WHERE student_id = ?", studentID).Scan(&studentSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			return
		}

		// If user is superadmin, allow updating any student
		if userRole != "superadmin" && userSchoolID != studentSchoolID {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to update this student"})
			return
		}

		// Step 4: Decode the updated student data from the request body
		var updatedStudent models.Student
		if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request payload"})
			return
		}

		// Step 5: Prepare and execute the update query
		query := `UPDATE Student 
                  SET first_name = ?, last_name = ?, patronymic = ?, iin = ?, date_of_birth = ?, 
                  grade = ?, school_id = ?, letter = ?, gender = ?, phone = ?, email = ? 
                  WHERE student_id = ?`

		_, err = db.Exec(query,
			updatedStudent.FirstName, updatedStudent.LastName, updatedStudent.Patronymic,
			updatedStudent.IIN, updatedStudent.DateOfBirth, updatedStudent.Grade,
			updatedStudent.SchoolID, updatedStudent.Letter, updatedStudent.Gender,
			updatedStudent.Phone, updatedStudent.Email, studentID)

		if err != nil {
			log.Println("Error updating student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update student"})
			return
		}

		// Step 6: Fetch the updated student to return in the response
		query = `SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, 
                grade, letter, gender, phone, email, role, login 
                FROM Student WHERE student_id = ?`

		err = db.QueryRow(query, studentID).Scan(
			&updatedStudent.ID, &updatedStudent.FirstName, &updatedStudent.LastName,
			&updatedStudent.Patronymic, &updatedStudent.IIN, &updatedStudent.SchoolID,
			&updatedStudent.DateOfBirth, &updatedStudent.Grade, &updatedStudent.Letter,
			&updatedStudent.Gender, &updatedStudent.Phone, &updatedStudent.Email,
			&updatedStudent.Role, &updatedStudent.Login)

		if err != nil {
			log.Println("Error fetching updated student:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Student updated but failed to retrieve updated details"})
			return
		}

		// Step 7: Respond with the updated student
		utils.ResponseJSON(w, updatedStudent)
	}
}

func (sc StudentController) GetStudentData(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role and school ID
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Error fetching user role and school ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Ensure the user is a director and has a school assigned
		if userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to view student data"})
			return
		}

		if !userSchoolID.Valid {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Director does not have an assigned school"})
			return
		}

		// Step 4: Extract student_id from the URL
		vars := mux.Vars(r)
		studentID, err := strconv.Atoi(vars["student_id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
			return
		}

		// Step 5: Retrieve student data from the database
		var student models.Student
		query := `SELECT id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email
                  FROM student WHERE id = ? AND school_id = ?`
		err = db.QueryRow(query, studentID, userSchoolID.Int64).Scan(&student.ID, &student.FirstName, &student.LastName, &student.Patronymic, &student.IIN, &student.SchoolID, &student.DateOfBirth, &student.Grade, &student.Letter, &student.Gender, &student.Phone, &student.Email)

		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found or does not belong to this school"})
				return
			}
			log.Println("Error fetching student data:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student data"})
			return
		}

		// Step 6: Respond with student data
		utils.ResponseJSON(w, student)
	}
}
func (sc StudentController) SuperadminDeleteStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем student_id из URL параметров
		vars := mux.Vars(r)
		studentIDParam := vars["student_id"]
		studentID, err := strconv.Atoi(studentIDParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
			return
		}

		// Step 1: Verify the user's token and get userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role and school ID
		var userRole string
		var userSchoolID int
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Check if the student exists and get student's school ID
		var studentSchoolID int
		err = db.QueryRow("SELECT school_id FROM Student WHERE student_id = ?", studentID).Scan(&studentSchoolID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			return
		}

		// Step 4: Check permission to delete the student
		if userRole != "superadmin" && userSchoolID != studentSchoolID {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to delete this student"})
			return
		}

		// Step 5: Delete the student from the database
		_, err = db.Exec("DELETE FROM Student WHERE student_id = ?", studentID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete student"})
			return
		}

		// Step 6: Respond with a success message
		utils.ResponseJSON(w, map[string]string{"message": "Student deleted successfully"})
	}
}
func (sfc *StudentController) GetStudentFilters(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role from the database
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		response := map[string]interface{}{
			"role": userRole,
		}

		// Step 3: Add school list for superadmin
		if userRole == "superadmin" {
			// Fetch all schools for superadmin
			rows, err := db.Query("SELECT school_id, school_name FROM Schools")
			if err != nil {
				log.Println("Error fetching schools:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch schools"})
				return
			}
			defer rows.Close()

			var schools []map[string]interface{}
			for rows.Next() {
				var schoolID int
				var schoolName string
				if err := rows.Scan(&schoolID, &schoolName); err != nil {
					log.Println("Error scanning school data:", err)
					continue
				}
				schools = append(schools, map[string]interface{}{
					"id":   schoolID,
					"name": schoolName,
				})
			}
			response["schools"] = schools
		} else if userRole == "schooladmin" {
			// For schooladmin, get their associated school
			var userEmail string
			err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
			if err != nil {
				log.Println("Error fetching user email:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user email"})
				return
			}

			var schoolID int
			var schoolName string
			err = db.QueryRow("SELECT school_id, school_name FROM Schools WHERE school_admin_login = ?", userEmail).Scan(&schoolID, &schoolName)
			if err != nil {
				log.Println("Error fetching school:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school details"})
				return
			}

			response["school"] = map[string]interface{}{
				"id":   schoolID,
				"name": schoolName,
			}
		}

		utils.ResponseJSON(w, response)
	}
}

// GetAvailableGrades returns available grades for a specific school
func (sfc *StudentController) GetAvailableGrades(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		var schoolID int

		// Step 3: Determine school ID based on role
		if userRole == "superadmin" {
			// Get school_id from query parameters
			schoolIDParam := r.URL.Query().Get("schoolId")
			if schoolIDParam == "" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required for superadmin"})
				return
			}

			var err error
			schoolID, err = strconv.Atoi(schoolIDParam)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
				return
			}
		} else if userRole == "schooladmin" {
			// Get school ID associated with this schooladmin
			var userEmail string
			err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
			if err != nil {
				log.Println("Error fetching user email:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user email"})
				return
			}

			err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", userEmail).Scan(&schoolID)
			if err != nil {
				log.Println("Error fetching school ID:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school details"})
				return
			}
		} else {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Unauthorized access"})
			return
		}

		// Step 4: Get distinct grades for the school
		rows, err := db.Query("SELECT DISTINCT grade FROM student WHERE school_id = ? AND grade IS NOT NULL ORDER BY grade", schoolID)
		if err != nil {
			log.Println("Error fetching grades:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch grade data"})
			return
		}
		defer rows.Close()

		var grades []int
		for rows.Next() {
			var grade int
			if err := rows.Scan(&grade); err != nil {
				log.Println("Error scanning grade:", err)
				continue
			}
			grades = append(grades, grade)
		}

		response := map[string]interface{}{
			"school_id": schoolID,
			"grades":    grades,
		}

		utils.ResponseJSON(w, response)
	}
}
func (sfc *StudentController) GetAvailableLetters(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Verify the user's token and get user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Step 2: Get user role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		var schoolID int

		// Step 3: Determine school ID based on role
		if userRole == "superadmin" {
			// Get school_id from query parameters
			schoolIDParam := r.URL.Query().Get("schoolId")
			if schoolIDParam == "" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required for superadmin"})
				return
			}

			var err error
			schoolID, err = strconv.Atoi(schoolIDParam)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
				return
			}
		} else if userRole == "schooladmin" {
			// Get school ID associated with this schooladmin
			var userEmail string
			err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
			if err != nil {
				log.Println("Error fetching user email:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user email"})
				return
			}

			err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", userEmail).Scan(&schoolID)
			if err != nil {
				log.Println("Error fetching school ID:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school details"})
				return
			}
		} else {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Unauthorized access"})
			return
		}

		// Step 4: Get grade from query parameters
		gradeParam := r.URL.Query().Get("grade")
		if gradeParam == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Grade is required"})
			return
		}

		grade, err := strconv.Atoi(gradeParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade"})
			return
		}

		// Step 5: Get distinct letters for the school and grade
		rows, err := db.Query("SELECT DISTINCT letter FROM student WHERE school_id = ? AND grade = ? AND letter IS NOT NULL AND letter != '' ORDER BY letter", schoolID, grade)
		if err != nil {
			log.Println("Error fetching letters:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch letter data"})
			return
		}
		defer rows.Close()

		var letters []string
		for rows.Next() {
			var letter string
			if err := rows.Scan(&letter); err != nil {
				log.Println("Error scanning letter:", err)
				continue
			}
			letters = append(letters, letter)
		}

		response := map[string]interface{}{
			"school_id": schoolID,
			"grade":     grade,
			"letters":   letters,
		}

		utils.ResponseJSON(w, response)
	}
}
func (sfc *StudentController) GetFilteredStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Println("Error fetching user role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Extract path variables using gorilla/mux
		vars := mux.Vars(r)

		var schoolID int
		if userRole == "superadmin" {
			schoolIDParam := vars["schoolId"]
			if schoolIDParam == "" {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "School ID is required for superadmin"})
				return
			}
			schoolID, err = strconv.Atoi(schoolIDParam)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
				return
			}
		} else if userRole == "schooladmin" {
			var userEmail string
			err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
			if err != nil {
				log.Println("Error fetching user email:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user email"})
				return
			}
			err = db.QueryRow("SELECT school_id FROM Schools WHERE school_admin_login = ?", userEmail).Scan(&schoolID)
			if err != nil {
				log.Println("Error fetching school ID:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching school details"})
				return
			}

			// For school admin, verify they're accessing their own school
			pathSchoolID, err := strconv.Atoi(vars["schoolId"])
			if err != nil || pathSchoolID != schoolID {
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Unauthorized access to this school"})
				return
			}
		} else {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Unauthorized access"})
			return
		}

		gradeParam := vars["grade"]
		if gradeParam == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Grade is required"})
			return
		}
		grade, err := strconv.Atoi(gradeParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade"})
			return
		}

		letter := strings.ToUpper(vars["letter"])
		if letter == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Letter is required"})
			return
		}

		// Debug: print unicode codepoints
		for i, r := range letter {
			log.Printf("letter[%d] = %q | Unicode: U+%04X", i, r, r)
		}

		log.Printf("Fetching students with schoolID=%d, grade=%d, letter=%s", schoolID, grade, letter)

		rows, err := db.Query(`
			SELECT student_id, first_name, last_name, patronymic, letter
			FROM student
			WHERE school_id = ? AND grade = ?
		`, schoolID, grade)
		if err != nil {
			log.Println("Error querying students:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Database query failed"})
			return
		}
		defer rows.Close()

		var students []models.Student
		for rows.Next() {
			var student models.Student
			var patronymic sql.NullString
			var dbLetter string

			if err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &patronymic, &dbLetter); err != nil {
				log.Println("Error scanning row:", err)
				continue
			}

			log.Printf("Found student: %s %s (%s), letter in DB: %q", student.FirstName, student.LastName, student.ID, dbLetter)

			if patronymic.Valid {
				student.Patronymic = patronymic.String
			}

			// Compare letters explicitly
			if strings.ToUpper(strings.TrimSpace(dbLetter)) == letter {
				students = append(students, student)
			}
		}

		if err := rows.Err(); err != nil {
			log.Println("Error after iterating rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing rows"})
			return
		}

		if len(students) == 0 {
			utils.ResponseJSON(w, map[string]interface{}{
				"message":  "No students found matching these criteria",
				"students": []models.Student{},
			})
			return
		}

		utils.ResponseJSON(w, students)
	}
}
func (c *StudentController) GetStudentsCountBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract school_id from URL parameters
		vars := mux.Vars(r)
		schoolIDParam := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDParam) // Convert school_id from string to integer
		if err != nil {
			log.Printf("Error converting school_id: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Query to count students by school_id
		var studentCount int
		err = db.QueryRow("SELECT COUNT(*) FROM student WHERE school_id = ?", schoolID).Scan(&studentCount)
		if err != nil {
			log.Printf("SQL Error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to count students"})
			return
		}

		// Response structure
		type StudentCountResponse struct {
			TotalStudents int `json:"total_students"`
		}

		// Create response
		response := StudentCountResponse{
			TotalStudents: studentCount,
		}

		// Return the count
		log.Printf("Successfully counted %d students for school ID %d", studentCount, schoolID)
		utils.ResponseJSON(w, response)
	}
}
func calculateAge(dob string, currentDate time.Time) int {
	if dob == "" {
		return 0 // Return 0 if date_of_birth is empty
	}

	// Parse date_of_birth (assuming format "YYYY-MM-DD")
	birthDate, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return 0 // Return 0 if date_of_birth is invalid
	}

	// Calculate age
	age := currentDate.Year() - birthDate.Year()
	if currentDate.YearDay() < birthDate.YearDay() {
		age--
	}
	return age
}
