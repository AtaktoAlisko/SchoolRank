package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type StudentController struct{}

func (sc StudentController) CreateStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Шаг 1: Проверить токен пользователя и получить userID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Шаг 2: Получить роль пользователя и school ID
		var userRole string
		var userSchoolID sql.NullInt64
		err = db.QueryRow("SELECT role, school_id FROM users WHERE id = ?", userID).Scan(&userRole, &userSchoolID)
		if err != nil {
			log.Println("Ошибка при получении роли пользователя и ID школы:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при получении данных пользователя"})
			return
		}

		// Шаг 3: Убедиться, что пользователь является директором и имеет привязанную школу
		if userRole != "schooladmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Вы не имеете прав для создания ученика"})
			return
		}

		if !userSchoolID.Valid {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Директор не имеет привязанной школы"})
			return
		}

		// Шаг 4: Декодировать данные студента из запроса
		var student models.Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Неверный запрос"})
			return
		}

		// Шаг 5: Убедиться, что школа студента соответствует школе директора
		if student.SchoolID != int(userSchoolID.Int64) {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Вы можете создавать студентов только для вашей школы"})
			return
		}

		// Шаг 6: Задать фиксированный email и пароль
		student.Email = student.FirstName + student.LastName + "school" // Фиксированный email
		student.Password = "password123" // Фиксированный пароль

		// Хэшируем пароль перед сохранением
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(student.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Ошибка при хэшировании пароля:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Ошибка при хэшировании пароля"})
			return
		}
		student.Password = string(hashedPassword) // Устанавливаем хэшированный пароль

		// Шаг 7: Вставить студента в базу данных
		query := `INSERT INTO student (first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email, password) 
		          VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		// Выполнить запрос
		result, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN, student.SchoolID, student.DateOfBirth, student.Grade, student.Letter, student.Gender, student.Phone, student.Email, student.Password)
		if err != nil {
			log.Println("Ошибка при вставке студента:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось создать студента"})
			return
		}

		// Шаг 8: Получить ID студента из результата запроса
		studentID, err := result.LastInsertId()
		if err != nil {
			log.Println("Ошибка при получении ID студента:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Не удалось получить ID студента"})
			return
		}

		// Устанавливаем ID студента в объект
		student.ID = int(studentID)

		// Шаг 9: Отправить ответ с созданным студентом
		utils.ResponseJSON(w, student)
	}
}
func (sc StudentController) GetStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Query the database for student details
		rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM Student")
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
			var firstName, lastName, patronymic, iin, letter, gender, phone, email sql.NullString
			var schoolID sql.NullInt64
			var dateOfBirth sql.NullString
			var grade sql.NullInt64

			// Step 3: Scan the row into the student object
			if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email); err != nil {
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
		schoolID, err := strconv.Atoi(schoolIDParam)
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

		var students []models.Student
		for rows.Next() {
			var student models.Student

			// Шаг 2: Обработка возможных NULL значений с помощью sql.Null типов
			var firstName, lastName, patronymic, iin, letter, gender, phone, email sql.NullString
			var schoolID sql.NullInt64
			var dateOfBirth sql.NullString
			var grade sql.NullInt64

			// Шаг 3: Сканирование строки в объект student
			if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email); err != nil {
				log.Println("Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse students"})
				return
			}

			// Шаг 4: Присваивание значений студенту
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

			// Шаг 5: Добавляем студента в срез
			students = append(students, student)
		}

		// Шаг 6: Ответ с данным списком студентов
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

        // Запрос к базе данных для получения студентов по school_id и grade
        rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM Student WHERE school_id = ? AND grade = ?", schoolID, grade)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get students"})
            return
        }
        defer rows.Close()

        var students []models.Student
        for rows.Next() {
            var student models.Student

            // Обработка возможных NULL значений с помощью sql.Null типов
            var firstName, lastName, patronymic, iin, letter, gender, phone, email sql.NullString
            var schoolID sql.NullInt64
            var dateOfBirth sql.NullString
            var grade sql.NullInt64

            // Сканирование строки в объект student
            if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse students"})
                return
            }

            // Присваивание значений студенту
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

            // Добавляем студента в срез
            students = append(students, student)
        }

        // Ответ с данным списком студентов
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

        // Запрос к базе данных для получения студентов по grade и letter
        rows, err := db.Query("SELECT student_id, first_name, last_name, patronymic, iin, school_id, date_of_birth, grade, letter, gender, phone, email FROM Student WHERE grade = ? AND letter = ?", grade, letterParam)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get students"})
            return
        }
        defer rows.Close()

        var students []models.Student
        for rows.Next() {
            var student models.Student

            // Обработка возможных NULL значений с помощью sql.Null типов
            var firstName, lastName, patronymic, iin, letter, gender, phone, email sql.NullString
            var schoolID sql.NullInt64
            var dateOfBirth sql.NullString
            var grade sql.NullInt64

            // Сканирование строки в объект student
            if err := rows.Scan(&student.ID, &firstName, &lastName, &patronymic, &iin, &schoolID, &dateOfBirth, &grade, &letter, &gender, &phone, &email); err != nil {
                log.Println("Scan Error:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse students"})
                return
            }

            // Присваивание значений студенту
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

            // Добавляем студента в срез
            students = append(students, student)
        }

        // Ответ с данным списком студентов
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
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
			} else {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching student details"})
			}
			return
		}

		// Decode the updated student data from the request body
		var updatedStudent models.Student
		if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Prepare the update query
		query := `UPDATE Student 
				  SET first_name = ?, last_name = ?, patronymic = ?, iin = ?, date_of_birth = ?, grade = ?, school_id = ?, 
				  letter = ?, gender = ?, phone = ?, email = ?
				  WHERE student_id = ?`

		// Execute the update query
		_, err = db.Exec(query, updatedStudent.FirstName, updatedStudent.LastName, updatedStudent.Patronymic, 
							updatedStudent.IIN, updatedStudent.DateOfBirth, updatedStudent.Grade, updatedStudent.SchoolID, 
							updatedStudent.Letter, updatedStudent.Gender, updatedStudent.Phone, updatedStudent.Email, studentID)
		if err != nil {
			log.Println("Error updating student:", err)
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
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "Student not found"})
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



