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
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Error fetching user role and school ID:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user details"})
			return
		}

		// Step 3: Ensure the user is a schooladmin and has a school assigned
		if userRole != "schooladmin" {
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

		// Step 5: Set the school ID based on director's user
		student.SchoolID = int(userSchoolID.Int64)

		// Step 6: Generate login and email for the student
		randomString := generateRandomString(4)
		student.Login = student.FirstName + student.LastName + randomString
		student.Email = student.Login + "@school.com"

		// Step 7: Generate the student's password
		student.Password = student.FirstName + student.LastName

		// Step 8: Set role to student
		student.Role = "student"

		// Step 9: Insert the student into the database
		query := `INSERT INTO Student (first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email, password, role, login) 
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN, student.SchoolID, student.DateOfBirth, student.Grade, student.Letter, student.Gender, student.Phone, student.Email, student.Password, student.Role, student.Login)
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

		utils.ResponseJSON(w, student)
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
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email, login, password, role FROM Student")
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
func (sc StudentController) GetStudentsBySchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем school_id из URL параметров
		vars := mux.Vars(r)
		schoolIDParam := vars["school_id"]
		schoolID, err := strconv.Atoi(schoolIDParam) // Преобразуем school_id из строки в целое число
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid school ID"})
			return
		}

		// Шаг 1: Запрос к базе данных для получения студентов по school_id
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM Student WHERE school_id = ?", schoolID)
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
        rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM Student WHERE school_id = ? AND grade = ?", schoolID, grade)
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
        // Извлекаем student_id из URL параметров
        vars := mux.Vars(r)
        studentIDParam := vars["student_id"]
        studentID, err := strconv.Atoi(studentIDParam)
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
            return
        }

        // Step 1: Check if the student exists
        var existingStudent models.Student
        err = db.QueryRow("SELECT student_id FROM Student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
        if err != nil {
            if err == sql.ErrNoRows {
                utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
            } else {
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
            }
            return
        }

        // Step 2: Decode the updated student data from the request body
        var updatedStudent models.Student
        if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Step 3: Prepare the update query
        query := `UPDATE Student 
                  SET first_name = ?, last_name = ?, patronymic = ?, iin = ?, date_of_birth = ?, grade = ?, school_id = ?, 
                  letter = ?, gender = ?, phone = ?, email = ?
                  WHERE student_id = ?`

        // Step 4: Execute the update query
        _, err = db.Exec(query, updatedStudent.FirstName, updatedStudent.LastName, updatedStudent.Patronymic, 
                          updatedStudent.IIN, updatedStudent.DateOfBirth, updatedStudent.Grade, updatedStudent.SchoolID, 
                          updatedStudent.Letter, updatedStudent.Gender, updatedStudent.Phone, updatedStudent.Email, studentID)
        if err != nil {
            log.Println("Error updating student:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update student"})
            return
        }

        // Step 5: Respond with the updated student
        updatedStudent.ID = studentID // Ensure the student ID is set in the response
        utils.ResponseJSON(w, updatedStudent)
    }
}
func (sc StudentController) DeleteStudent(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Извлекаем student_id из URL параметров
        vars := mux.Vars(r)
        studentIDParam := vars["student_id"]
        studentID, err := strconv.Atoi(studentIDParam)
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid student ID"})
            return
        }

        // Шаг 1: Проверка существования студента
        var existingStudent models.Student
        err = db.QueryRow("SELECT student_id FROM Student WHERE student_id = ?", studentID).Scan(&existingStudent.ID)
        if err != nil {
            if err == sql.ErrNoRows {
                utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
            } else {
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
            }
            return
        }

        // Шаг 2: Удаление студента из базы данных
        _, err = db.Exec("DELETE FROM Student WHERE student_id = ?", studentID)
        if err != nil {
            log.Println("Error deleting student:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete student"})
            return
        }

        // Шаг 3: Ответ с успешным сообщением
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




