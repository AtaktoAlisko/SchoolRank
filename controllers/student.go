package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"

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

		var schoolID int

		// Step 4: If the user is a schooladmin, get the school ID associated with them
		if userRole == "schooladmin" {
			// Get the school ID from the Schools table using the user's email
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
				utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Director does not have an assigned school. Please create a school first."})
				return
			}
		} else if userRole == "superadmin" {
			// If the user is superadmin, they can create students for any school, so we don't need to fetch school ID here.
			// The school ID will be provided in the request body (not URL).
			// Retrieve the school_id from the request body
			var studentData struct {
				SchoolID int `json:"school_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&studentData); err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID in request"})
				return
			}
			schoolID = studentData.SchoolID
		}

		// Step 5: Decode the student data from the request
		var student models.Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Step 6: Set the school ID based on the role of the user (schooladmin or superadmin)
		student.SchoolID = schoolID

		// Step 7: Generate login and email for the student
		randomString := generateRandomString(4)
		student.Login = student.FirstName + student.LastName + randomString
		student.Email = student.Login + "@school.com"

		// Step 8: Generate the student's password
		student.Password = student.FirstName + student.LastName

		// Step 9: Set role to student
		student.Role = "student"

		// Step 10: Handle AvatarURL (set NULL if not provided)
		var avatarURL sql.NullString
		if student.AvatarURL != "" {
			// Set the avatar URL if provided
			avatarURL = sql.NullString{String: student.AvatarURL, Valid: true}
		} else {
			// Set it to NULL if not provided
			avatarURL = sql.NullString{Valid: false}
		}

		// Step 11: Insert the student into the database
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
func (sc *StudentController) GetAllStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Get all students and their associated school names from the database
		rows, err := db.Query(`
			SELECT s.student_id, s.first_name, s.last_name, s.patronymic, s.iin, s.school_id, 
					s.grade, s.letter, s.gender, s.phone, s.email, s.role, s.login, s.avatar_url, 
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
			var schoolID sql.NullInt64    // For handling NULL in school_id
			var grade sql.NullInt64       // For handling NULL in grade
			var letter sql.NullString     // For handling NULL in letter
			var gender sql.NullString     // For handling NULL in gender
			var phone sql.NullString      // For handling NULL in phone
			var email sql.NullString      // For handling NULL in email
			var avatarURL sql.NullString  // For handling NULL in avatar_url
			var schoolName sql.NullString // For handling NULL in school_name

			if err := rows.Scan(
				&student.ID,
				&student.FirstName,
				&student.LastName,
				&student.Patronymic,
				&student.IIN,
				&schoolID, // Scan into NullInt64
				&grade,    // Scan into NullInt64
				&letter,   // Scan into NullString
				&gender,   // Scan into NullString
				&phone,    // Scan into NullString
				&email,    // Scan into NullString
				&student.Role,
				&student.Login,
				&avatarURL,  // Scan into NullString
				&schoolName, // Scan into NullString (school_name)
			); err != nil {
				log.Println("Error scanning student:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to scan student data"})
				return
			}

			// Check if schoolID is valid (not NULL)
			if schoolID.Valid {
				student.SchoolID = int(schoolID.Int64)
			} else {
				student.SchoolID = 0 // or set to NULL if necessary
			}

			// Check if grade is valid (not NULL)
			if grade.Valid {
				student.Grade = int(grade.Int64) // Convert to int if valid
			} else {
				student.Grade = 0 // If NULL, assign default value (0)
			}

			// Check if letter is valid (not NULL)
			if letter.Valid {
				student.Letter = letter.String // If valid, assign the string value
			} else {
				student.Letter = "" // If NULL, assign empty string
			}

			// Check if gender is valid (not NULL)
			if gender.Valid {
				student.Gender = gender.String // If valid, assign the string value
			} else {
				student.Gender = "" // If NULL, assign empty string
			}

			// Check if phone is valid (not NULL)
			if phone.Valid {
				student.Phone = phone.String // If valid, assign the string value
			} else {
				student.Phone = "" // If NULL, assign empty string
			}

			// Check if email is valid (not NULL)
			if email.Valid {
				student.Email = email.String // If valid, assign the string value
			} else {
				student.Email = "" // If NULL, assign empty string
			}

			// Check if avatarURL is valid (not NULL)
			if avatarURL.Valid {
				student.AvatarURL = avatarURL.String // If valid, assign the string value
			} else {
				student.AvatarURL = "" // If NULL, assign empty string
			}

			// Check if schoolName is valid (not NULL)
			if schoolName.Valid {
				student.SchoolName = schoolName.String // If valid, assign the string value
			} else {
				student.SchoolName = "" // If NULL, assign empty string
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
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM student WHERE school_id = ?", schoolID)
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
			var firstName, lastName, patronymic, iin, letter, gender, phone, email sql.NullString
			var schoolID sql.NullInt64
			var dateOfBirth sql.NullString
			var grade sql.NullInt64

			// Шаг 4: Сканирование строки в объект student
			if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email); err != nil {
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
func (sc StudentController) GetStudentsBySchoolAndGrade(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем параметры school_id и grade из URL
		vars := mux.Vars(r)
		schoolIDParam := vars["school_id"]
		gradeParam := vars["grade"]

		// Преобразуем параметры в нужные типы
		schoolID, err := strconv.Atoi(schoolIDParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		grade, err := strconv.Atoi(gradeParam)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid grade"})
			return
		}

		// Шаг 1: Запрос к базе данных для получения студентов по school_id и grade
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM student WHERE school_id = ? AND grade = ?", schoolID, grade)
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
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM Student WHERE grade = ? AND letter = ?", grade, letterParam)
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
func (sc StudentController) UpdateStudent(db *sql.DB) http.HandlerFunc {
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

		// Step 1: Verify the user's token and get userID (similar to CreateStudent)
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

		// Step 3: Ensure the user is authorized (schooladmin)
		if userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to update a student"})
			return
		}

		// Step 4: Check if the student exists
		var existingStudent models.Student
		err = db.QueryRow("SELECT student_id FROM Student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
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

		// Step 7: Fetch the updated student to return in the response
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

		// Step 8: Respond with the updated student
		utils.ResponseJSON(w, updatedStudent)
	}
}
func (sc *StudentController) DeleteStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем student_id из URL параметров
		vars := mux.Vars(r)
		studentIDParam := vars["student_id"]
		studentID, err := strconv.Atoi(studentIDParam)
		if err != nil {
			log.Println("Invalid student ID:", studentIDParam) // Log invalid student ID
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
			return
		}

		// Шаг 1: Проверка существования студента
		var existingStudent models.Student
		err = db.QueryRow("SELECT student_id FROM student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Println("No student found with ID:", studentID) // Log when student is not found
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			} else {
				log.Println("Error fetching student details:", err) // Log any other error while fetching
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
			}
			return
		}

		// Шаг 2: Удаление студента из базы данных
		_, err = db.Exec("DELETE FROM student WHERE student_id = ?", studentID)
		if err != nil {
			log.Println("Error deleting student with ID:", studentID, err) // Log error while deleting student
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete student"})
			return
		}

		// Шаг 3: Ответ с успешным сообщением
		log.Println("Student with ID:", studentID, "deleted successfully.") // Log successful deletion
		utils.ResponseJSON(w, map[string]string{"message": "Student deleted successfully"})
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
