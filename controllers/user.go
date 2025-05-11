package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func (c *Controller) Signup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		var error models.Error

		// Декодируем запрос
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем формат даты рождения и вычисляем возраст
		if user.DateOfBirth != "" {
			// Проверка формата даты
			_, err := time.Parse("2006-01-02", user.DateOfBirth)
			if err != nil {
				error.Message = "Invalid date format. Please use YYYY-MM-DD format."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}

			// Вычисляем возраст, используя метод контроллера
			age, err := c.CalculateAge(&user)
			if err != nil {
				error.Message = "Error calculating age."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// Устанавливаем вычисленный возраст
			user.Age = age

			// Logging for debugging
			log.Printf("Date of birth: %s, Calculated age: %d", user.DateOfBirth, user.Age)
		} else {
			// If date of birth is not provided, explicitly set empty values
			user.DateOfBirth = ""
			user.Age = 0
		}

		// Проверка аутентификации суперадмина
		isCreatedBySuperAdmin := false

		// Извлекаем токен авторизации
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.Split(authHeader, " ")[1]
			token, err := utils.ParseToken(tokenString)

			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					// Проверяем роль пользователя в токене
					if role, exists := claims["role"].(string); exists && role == "superadmin" {
						isCreatedBySuperAdmin = true

						// Дополнительная проверка ID суперадмина в базе
						if userID, exists := claims["user_id"].(float64); exists {
							var exists bool
							err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND role = 'superadmin')", int(userID)).Scan(&exists)
							if err != nil || !exists {
								isCreatedBySuperAdmin = false
							}
						}
					}
				}
			}
		}

		// Если пользователь создается суперадмином и указан тип schooladmin
		if isCreatedBySuperAdmin && user.Role == "schooladmin" {

			// Проверяем наличие email, так как для админа школы обязателен email
			if user.Email == "" {
				error.Message = "Email is required for school administrator accounts."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}

			// Проверяем формат email
			if !strings.Contains(user.Email, "@") {
				error.Message = "Invalid email format."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}

			// Проверяем, существует ли уже email в базе
			var existingID int
			err = db.QueryRow("SELECT id FROM users WHERE email = ?", user.Email).Scan(&existingID)
			if err == nil {
				error.Message = "Email already exists."
				utils.RespondWithError(w, http.StatusConflict, error)
				return
			} else if err != sql.ErrNoRows {
				log.Printf("Error checking existing user: %v", err)
				error.Message = "Server error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// Если пароль не указан, генерируем случайный пароль
			var plainPassword string
			if user.Password == "" {
				// Генерируем случайный пароль для школьного администратора (12 символов)
				randomBytes := make([]byte, 9) // 9 байтов дадут 12 символов в base64
				_, err := rand.Read(randomBytes)
				if err != nil {
					log.Printf("Error generating random bytes: %v", err)
					error.Message = "Failed to generate password."
					utils.RespondWithError(w, http.StatusInternalServerError, error)
					return
				}

				// Используем функцию для генерации пароля из случайных байтов
				plainPassword = generateRandomPassword(12)
			} else {
				plainPassword = user.Password
			}

			// Хешируем пароль для сохранения в базе
			hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("Error hashing password: %v", err)
				error.Message = "Server error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}
			user.Password = string(hash)

			// Устанавливаем дефолтный аватар, если не указан
			if !user.AvatarURL.Valid || user.AvatarURL.String == "" {
				user.AvatarURL = sql.NullString{String: "", Valid: false}
			}

			// Генерируем токен для верификации (для совместимости со схемой БД)
			verificationToken, err := utils.GenerateVerificationToken(user.Email)
			if err != nil {
				log.Printf("Error generating verification token: %v", err)
				error.Message = "Failed to generate verification token."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// ВАЖНО: Устанавливаем verified = true для schooladmin
			// Вставка данных в базу (учетная запись школьного администратора создается уже верифицированной)
			query := "INSERT INTO users (email, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)"
			result, err := db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, verificationToken)
			if err != nil {
				log.Printf("Error inserting user: %v", err)
				error.Message = "Server error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// Получаем ID созданного пользователя
			userID, _ := result.LastInsertId()

			// Проверяем, что пользователь действительно создан и установлен verified = true
			var verified int
			err = db.QueryRow("SELECT verified FROM users WHERE id = ?", userID).Scan(&verified)
			if err != nil || verified != 1 {
				log.Printf("Failed to verify user creation or verification status: %v", err)
				// Если не удалось проверить или verified = false, исправляем это
				_, updateErr := db.Exec("UPDATE users SET verified = 1 WHERE id = ?", userID)
				if updateErr != nil {
					log.Printf("Failed to update verification status: %v", updateErr)
				}
			}

			// Отправляем email администратору школы с данными для входа
			if plainPassword != user.Password {
				subject := "Ваши учетные данные для входа в систему"
				body := fmt.Sprintf(
					"Уважаемый %s %s,\n\n"+
						"Для вас был создан аккаунт школьного администратора.\n\n"+
						"Логин: %s\n"+
						"Пароль: %s\n\n"+
						"Рекомендуем сменить пароль после первого входа в систему.\n\n"+
						"С уважением,\n"+
						"Администрация системы",
					user.FirstName, user.LastName, user.Email, plainPassword,
				)

				utils.SendEmail(user.Email, subject, body)
			}

			// Формируем ответ суперадмину
			response := map[string]interface{}{
				"message": "School administrator account created successfully. Login credentials sent to email.",
				"email":   user.Email,
			}

			utils.ResponseJSON(w, response)
			return
		}

		// Проверяем, если регистрируется суперадмин
		if user.Role == "superadmin" {
			// Проверяем наличие email для суперадмина
			if user.Email == "" {
				error.Message = "Email is required for superadmin accounts."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}

			// Проверяем формат email
			if !strings.Contains(user.Email, "@") {
				error.Message = "Invalid email format."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}

			// Проверяем, существует ли уже email в базе
			var existingID int
			err = db.QueryRow("SELECT id FROM users WHERE email = ?", user.Email).Scan(&existingID)
			if err == nil {
				error.Message = "Email already exists."
				utils.RespondWithError(w, http.StatusConflict, error)
				return
			} else if err != sql.ErrNoRows {
				log.Printf("Error checking existing user: %v", err)
				error.Message = "Server error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// Проверяем наличие пароля
			if user.Password == "" {
				error.Message = "Password is required for superadmin accounts."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}

			// Хешируем пароль
			hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("Error hashing password: %v", err)
				error.Message = "Server error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}
			user.Password = string(hash)

			// Устанавливаем дефолтный аватар, если не указан
			if !user.AvatarURL.Valid || user.AvatarURL.String == "" {
				user.AvatarURL = sql.NullString{String: "", Valid: false}
			}

			// Генерируем токен для верификации (для совместимости со схемой БД)
			verificationToken, err := utils.GenerateVerificationToken(user.Email)
			if err != nil {
				log.Printf("Error generating verification token: %v", err)
				error.Message = "Failed to generate verification token."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// ВАЖНО: Устанавливаем verified = 1 для superadmin
			// MySQL/SQL использует 1 для true и 0 для false в булевых полях
			query := "INSERT INTO users (email, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)"
			result, err := db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, verificationToken)
			if err != nil {
				log.Printf("Error inserting superadmin user: %v", err)
				error.Message = "Server error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// Получаем ID созданного пользователя
			userID, _ := result.LastInsertId()

			// Дополнительная проверка и принудительное обновление статуса verified если нужно
			var actualVerified int
			err = db.QueryRow("SELECT verified FROM users WHERE id = ?", userID).Scan(&actualVerified)
			if err == nil && actualVerified != 1 {
				// Если по какой-то причине verified не установлен в 1, исправляем это
				_, updateErr := db.Exec("UPDATE users SET verified = 1 WHERE id = ?", userID)
				if updateErr != nil {
					log.Printf("Failed to update verification status for superadmin: %v", updateErr)
				} else {
					log.Printf("Fixed verification status for superadmin: %s", user.Email)
				}
			}

			// Формируем ответ для создания суперадмина
			response := map[string]interface{}{
				"message": "Superadmin account created successfully. Account is already verified.",
				"email":   user.Email,
			}

			utils.ResponseJSON(w, response)
			return
		}

		// Устанавливаем роль "user" по умолчанию
		if user.Role == "" {
			user.Role = "user"
		}

		// Устанавливаем дефолтный аватар, если не указан
		if !user.AvatarURL.Valid || user.AvatarURL.String == "" {
			user.AvatarURL = sql.NullString{String: "", Valid: false} // Если аватар не указан, записываем NULL в базе данных
		}

		// Проверяем, что email или телефон предоставлены
		if user.Email == "" && user.Phone == "" {
			error.Message = "Email or phone is required."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, что формат email или телефона правильный
		var isEmail bool
		if user.Email != "" && strings.Contains(user.Email, "@") {
			isEmail = true
		} else if user.Phone != "" && utils.IsPhoneNumber(user.Phone) {
			isEmail = false
		} else {
			error.Message = "Invalid email or phone format."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, что пароль не пустой
		if user.Password == "" {
			error.Message = "Password is required."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, существует ли уже email или телефон в базе
		var existingID int
		var query string
		var identifier string

		if isEmail {
			query = "SELECT id FROM users WHERE email = ?"
			identifier = user.Email
		} else {
			query = "SELECT id FROM users WHERE phone = ?"
			identifier = user.Phone
		}

		err = db.QueryRow(query, identifier).Scan(&existingID)
		if err == nil {
			error.Message = "Email or phone already exists."
			utils.RespondWithError(w, http.StatusConflict, error)
			return
		} else if err != sql.ErrNoRows {
			log.Printf("Error checking existing user: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Хэшируем пароль
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}
		user.Password = string(hash)

		// Генерация OTP кода для верификации
		otpCode, err := utils.GenerateOTP()
		if err != nil {
			log.Printf("Error generating OTP: %v", err)
			error.Message = "Failed to generate OTP."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Генерация токена для верификации
		verificationToken, err := utils.GenerateVerificationToken(user.Email)
		if err != nil {
			log.Printf("Error generating verification token: %v", err)
			error.Message = "Failed to generate verification token."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Logging before insert
		log.Printf("Inserting user with date_of_birth: %s, age: %d", user.DateOfBirth, user.Age)

		// Вставка данных в базу
		if isEmail {
			query = "INSERT INTO users (email, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, otp_code, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, false, ?, ?)"
			_, err = db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, otpCode, verificationToken)
		} else {
			query = "INSERT INTO users (phone, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, otp_code, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, true, NULL, ?)"
			_, err = db.Exec(query, user.Phone, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, verificationToken)
		}

		if err != nil {
			log.Printf("Error inserting user: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Verify the data was inserted correctly - grab the user ID and check the inserted values
		if isEmail {
			var insertedID int
			var insertedDOB string
			var insertedAge int
			errVerify := db.QueryRow("SELECT id, date_of_birth, age FROM users WHERE email = ?", user.Email).
				Scan(&insertedID, &insertedDOB, &insertedAge)
			if errVerify == nil {
				log.Printf("Verified user insertion: ID=%d, DOB=%s, Age=%d", insertedID, insertedDOB, insertedAge)
			} else {
				log.Printf("Could not verify user insertion: %v", errVerify)
			}
		}

		// Отправка email с OTP
		if isEmail {
			utils.SendVerificationEmail(user.Email, verificationToken, otpCode)
		}

		user.Password = "" // Убираем пароль из ответа

		// Формируем сообщение для пользователя в зависимости от типа регистрации
		var message string
		if isEmail {
			message = "User registered successfully. Please verify your email with the OTP code."
		} else {
			message = "User registered successfully."
		}

		// Создаем ответ без date_of_birth и age
		response := map[string]interface{}{
			"message": message,
		}

		// Добавляем OTP код только для пользователей с email, которым нужна верификация
		if isEmail {
			response["otp_code"] = otpCode
		}

		// Добавляем avatar_url в ответ, только если оно задано
		if user.AvatarURL.Valid && user.AvatarURL.String != "" {
			response["avatar_url"] = user.AvatarURL.String
		}

		utils.ResponseJSON(w, response)
	}
}
func (c *Controller) GetMe(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Authorization header is required"})
			return
		}

		// Typically, the token is prefixed with "Bearer "
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// Parse the token
		token, err := utils.ParseToken(tokenString)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Verify token is valid
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Extract user ID and role from token
		id := int(claims["user_id"].(float64))
		role := claims["role"].(string)

		// Initialize variables for user/student data
		var userOrStudent interface{}
		var found bool = false

		// Try to check in the student table first if role is "student"
		if role == "student" {
			userOrStudent, found = c.getStudentData(db, id)
		}

		// If not found as a student or role is "user", check in the users table
		if !found {
			userOrStudent, found = c.getUserData(db, id)
		}

		if !found {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "User not found"})
			return
		}

		// Return the appropriate user or student data
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userOrStudent)
	}
}
func (c *Controller) getStudentData(db *sql.DB, id int) (interface{}, bool) {
	var email, phone, login sql.NullString
	var avatarURL, dateOfBirth sql.NullString
	var age sql.NullInt64
	var passwordHash sql.NullString
	var patronymic, iin, letter, gender sql.NullString
	var grade, schoolID sql.NullInt64

	var student models.Student

	err := db.QueryRow(`
		SELECT student_id, first_name, last_name, email, avatar_url, date_of_birth, 
		       age, password, patronymic, iin, grade, school_id, letter, gender, phone, login 
		FROM student WHERE student_id = ?`, id).
		Scan(&student.ID, &student.FirstName, &student.LastName, &email, &avatarURL,
			&dateOfBirth, &age, &passwordHash, &patronymic, &iin, &grade, &schoolID,
			&letter, &gender, &phone, &login)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		log.Printf("Error fetching student data: %v", err)
		return nil, false
	}

	// Заполняем student
	if email.Valid {
		student.Email = email.String
	}
	if phone.Valid {
		student.Phone = phone.String
	}
	if login.Valid {
		student.Login = login.String
	}
	if patronymic.Valid {
		student.Patronymic = patronymic.String
	}
	if iin.Valid {
		student.IIN = iin.String
	}
	if letter.Valid {
		student.Letter = letter.String
	}
	if gender.Valid {
		student.Gender = gender.String
	}
	if grade.Valid {
		student.Grade = int(grade.Int64)
	}
	if schoolID.Valid {
		student.SchoolID = int(schoolID.Int64)
	}

	// Вычисляем возраст
	var ageValue int
	if age.Valid {
		ageValue = int(age.Int64)
	} else {
		ageValue = 0
		if dateOfBirth.Valid && dateOfBirth.String != "" {
			dob, err := time.Parse("2006-01-02", dateOfBirth.String)
			if err == nil {
				now := time.Now()
				ageValue = now.Year() - dob.Year()
				if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
					ageValue--
				}
				_, updateErr := db.Exec("UPDATE student SET age = ? WHERE student_id = ?", ageValue, id)
				if updateErr != nil {
					log.Printf("Failed to update age in database: %v", updateErr)
				}
			}
		}
	}

	type StudentResponse struct {
		ID          int     `json:"id"`
		FirstName   string  `json:"first_name"`
		LastName    string  `json:"last_name"`
		Patronymic  string  `json:"patronymic"`
		IIN         string  `json:"iin"`
		Gender      string  `json:"gender"`
		Age         int     `json:"age"`
		AvatarURL   *string `json:"avatar_url"`
		DateOfBirth string  `json:"date_of_birth"`
		Email       string  `json:"email"`
		Role        string  `json:"role"`
		SchoolID    int     `json:"school_id"`
	}

	studentData := StudentResponse{
		Age:         ageValue,
		AvatarURL:   nil,
		DateOfBirth: "",
		Email:       student.Email,
		FirstName:   student.FirstName,
		Gender:      student.Gender,
		ID:          student.ID,
		IIN:         student.IIN,
		LastName:    student.LastName,
		Patronymic:  student.Patronymic,
		Role:        "student",
		SchoolID:    student.SchoolID,
	}

	if avatarURL.Valid {
		studentData.AvatarURL = &avatarURL.String
	}
	if dateOfBirth.Valid {
		studentData.DateOfBirth = dateOfBirth.String
	}

	// Возвращаем результат
	return studentData, true
}
func (c *Controller) Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		var error models.Error

		// Decode request body
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		var query string
		var identifier string
		var hashedPassword string
		var email sql.NullString
		var phone sql.NullString
		var role string
		var verified bool
		var userFound bool = false
		var login sql.NullString

		// For debugging
		log.Println("Login attempt with:", user.Email, user.Phone, user.Login, user.Password)

		// Check that email, phone, or login are provided
		if user.Email != "" {
			query = "SELECT id, email, phone, password, first_name, last_name, age, role, verified, login FROM users WHERE email = ?"
			identifier = user.Email
		} else if user.Phone != "" {
			query = "SELECT id, email, phone, password, first_name, last_name, age, role, verified, login FROM users WHERE phone = ?"
			identifier = user.Phone
		} else if user.Login != "" { // If login is provided
			query = "SELECT id, email, phone, password, first_name, last_name, age, role, verified, login FROM users WHERE login = ?"
			identifier = user.Login
		} else {
			error.Message = "Email, phone, or login is required."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Try to find the user in users table
		row := db.QueryRow(query, identifier)
		err = row.Scan(&user.ID, &email, &phone, &hashedPassword, &user.FirstName, &user.LastName, &user.Age, &role, &verified, &login)

		// If user is not found in users table, check students table
		if err == sql.ErrNoRows {
			log.Println("User not found in users table, checking Student table...")

			// Check if student
			var studentQuery string
			if user.Email != "" {
				studentQuery = "SELECT student_id, email, phone, password, first_name, last_name, grade, role, login FROM student WHERE email = ?"
			} else if user.Phone != "" {
				studentQuery = "SELECT student_id, email, phone, password, first_name, last_name, grade, role, login FROM student WHERE phone = ?"
			} else if user.Login != "" {
				studentQuery = "SELECT student_id, email, phone, password, first_name, last_name, grade, role, login FROM student WHERE login = ?"
			}

			var studentGrade int
			var studentID int
			var studentEmail, studentPhone, studentPassword, studentFirstName, studentLastName, studentRole, studentLogin string

			log.Println("Executing student query:", studentQuery, "with identifier:", identifier)
			studentRow := db.QueryRow(studentQuery, identifier)
			err = studentRow.Scan(&studentID, &studentEmail, &studentPhone, &studentPassword, &studentFirstName, &studentLastName, &studentGrade, &studentRole, &studentLogin)

			if err == nil {
				log.Println("Student found:", studentID, studentEmail, studentRole)
				userFound = true
				user.ID = studentID
				email = sql.NullString{String: studentEmail, Valid: studentEmail != ""}
				phone = sql.NullString{String: studentPhone, Valid: studentPhone != ""}
				hashedPassword = studentPassword // For students, password may not be hashed
				user.FirstName = studentFirstName
				user.LastName = studentLastName
				user.Age = 0 // Age might not be available for students, using grade instead
				role = studentRole
				if login.Valid {
					user.Login = login.String
				}

				verified = true // Students are considered verified by default

				log.Println("Student password from DB:", hashedPassword)
				log.Println("Password provided:", user.Password)
			} else {
				log.Println("Error finding student:", err)
			}
		} else if err == nil {
			// User found in users table
			userFound = true
			if login.Valid {
				user.Login = login.String
			}
			user.Role = role // ✅ Добавь ЭТО
		} else {
			log.Println("Error querying users table:", err)
		}

		if !userFound {
			error.Message = "User not found."
			utils.RespondWithError(w, http.StatusNotFound, error)
			return
		}

		// Check if the user is verified
		if !verified && role == "user" {
			error.Message = "Email not verified. Please verify your email before logging in."
			utils.RespondWithError(w, http.StatusForbidden, error)
			return
		}

		// Validate password
		var passwordValid bool = false
		if role == "student" {
			// Direct comparison for student passwords as they may not be hashed
			passwordValid = (hashedPassword == user.Password)
			log.Println("Student password validation:", passwordValid, hashedPassword, user.Password)
		} else {
			err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
			passwordValid = (err == nil)
		}

		if !passwordValid {
			error.Message = "Invalid password."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Set user properties before token generation
		if email.Valid {
			user.Email = email.String
		}
		if phone.Valid {
			user.Phone = phone.String
		}
		user.Role = role

		// Generate access token with 15 minutes expiration
		accessToken, err := utils.GenerateToken(user, 15*time.Minute)
		if err != nil {
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Generate refresh token with 7 days expiration
		refreshToken, err := utils.GenerateRefreshToken(user, 7*24*time.Hour)
		if err != nil {
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Send response with the tokens
		response := map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		}

		utils.ResponseJSON(w, response)
	}
}
func (c *Controller) GetAllUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Запрос для получения всех пользователей из базы
		rows, err := db.Query("SELECT id, email, first_name, last_name, date_of_birth, role, password FROM users")
		if err != nil {
			log.Printf("Error fetching users: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Server error"})
			return
		}
		defer rows.Close()

		var users []map[string]interface{}

		// Проходим по всем строкам в результатах запроса
		for rows.Next() {
			var user models.User
			var password string
			var dateOfBirth sql.NullString // Используем sql.NullString для работы с NULL значениями

			// Извлекаем данные пользователя
			err := rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &dateOfBirth, &user.Role, &password)
			if err != nil {
				log.Printf("Error scanning user: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning user data"})
				return
			}

			// Преобразуем date_of_birth в строку, если оно не NULL
			var dateOfBirthStr string
			if dateOfBirth.Valid {
				dateOfBirthStr = dateOfBirth.String
			} else {
				dateOfBirthStr = "" // Если дата рождения NULL, то оставляем пустую строку
			}

			// Создаем карту для каждого пользователя, которую будем добавлять в ответ
			userMap := map[string]interface{}{
				"id":            user.ID,
				"email":         user.Email,
				"first_name":    user.FirstName,
				"last_name":     user.LastName,
				"date_of_birth": dateOfBirthStr,
				"role":          user.Role,
				"password":      password, // Возможно, вы хотите хранить хеш пароля или не включать его в ответ
			}

			users = append(users, userMap)
		}

		// Проверяем на ошибки после итерации
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating over users: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing users"})
			return
		}

		// Отправляем список пользователей в формате JSON
		utils.ResponseJSON(w, users)
	}
}
func (c *Controller) CreateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			FirstName   string `json:"first_name"`
			LastName    string `json:"last_name"`
			DateOfBirth string `json:"date_of_birth"`
			Email       string `json:"email"`
			Password    string `json:"password"`
			Role        string `json:"role"`
		}

		// Проверка токена
		adminID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Проверка роли суперадмина
		var adminRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", adminID).Scan(&adminRole)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching role"})
			return
		}

		if adminRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can create users"})
			return
		}

		// Читаем тело запроса
		err = json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Валидация роли
		if requestData.Role != "user" && requestData.Role != "schooladmin" && requestData.Role != "superadmin" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid role"})
			return
		}

		// Хешируем пароль
		hashedPassword, err := utils.HashPassword(requestData.Password)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Password hashing failed"})
			return
		}

		// Проверка на существование email
		var existingID int
		err = db.QueryRow("SELECT id FROM users WHERE email = ?", requestData.Email).Scan(&existingID)
		if err == nil && existingID > 0 {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Email already exists"})
			return
		}

		// Вставляем нового пользователя в базу
		query := `INSERT INTO users (first_name, last_name, date_of_birth, email, password, role) 
		          VALUES (?, ?, ?, ?, ?, ?)`

		_, err = db.Exec(query, requestData.FirstName, requestData.LastName, requestData.DateOfBirth,
			requestData.Email, hashedPassword, requestData.Role)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create user"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "User created successfully"})
	}
}
func (c Controller) Logout(w http.ResponseWriter, r *http.Request) {
	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	bearerToken := strings.Split(authHeader, " ")

	if len(bearerToken) == 2 {
		tokenString := bearerToken[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("there was an error")
			}
			return []byte(os.Getenv("SECRET")), nil
		})

		if err != nil || !token.Valid {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid or expired token"})
			return
		}

		// Continue with logging out, e.g., clearing session or token
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Expires:  time.Unix(0, 0), // Set expiration time
			HttpOnly: true,
		})

		utils.ResponseJSON(w, map[string]string{"message": "Successfully logged out"})
		return
	} else {
		utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
		return
	}
}
func (c Controller) DeleteAccount(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errorObject models.Error

		// Извлекаем user_id из URL
		vars := mux.Vars(r)
		userID := vars["user_id"]

		// Проверяем, что user_id валидный
		if userID == "" {
			errorObject.Message = "User ID is required"
			utils.RespondWithError(w, http.StatusBadRequest, errorObject)
			return
		}

		// Проверка, существует ли пользователь с таким ID в базе данных
		var existingUserID int
		err := db.QueryRow("SELECT id FROM users WHERE id = ?", userID).Scan(&existingUserID)
		if err != nil {
			if err == sql.ErrNoRows {
				errorObject.Message = "User not found"
				utils.RespondWithError(w, http.StatusNotFound, errorObject)
				return
			}
			errorObject.Message = "Error querying user"
			utils.RespondWithError(w, http.StatusInternalServerError, errorObject)
			return
		}

		// Удаление пользователя из базы данных
		_, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
		if err != nil {
			errorObject.Message = "Failed to delete user"
			utils.RespondWithError(w, http.StatusInternalServerError, errorObject)
			return
		}

		// Ответ о успешном удалении
		utils.ResponseJSON(w, map[string]string{"message": "Account deleted successfully"})
	}
}
func (c *Controller) EditProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userUpdate models.User
		var error models.Error

		// Get user ID from token using VerifyToken
		currentUserID, err := utils.VerifyToken(r)
		if err != nil {
			error.Message = "Unauthorized access."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Parse target user ID from URL if provided (for superadmins)
		targetUserID := currentUserID
		targetUserIDParam := r.URL.Query().Get("id")
		if targetUserIDParam != "" {
			parsedID, err := strconv.Atoi(targetUserIDParam)
			if err != nil {
				error.Message = "Invalid user ID format."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}
			targetUserID = parsedID
		}

		// Decode request body
		err = json.NewDecoder(r.Body).Decode(&userUpdate)
		if err != nil {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Log update attempt
		log.Printf("Profile update attempt for userID: %d by userID: %d", targetUserID, currentUserID)

		// Check if target user exists and get current role
		var role string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", targetUserID).Scan(&role)
		if err != nil {
			if err == sql.ErrNoRows {
				error.Message = "User not found."
				utils.RespondWithError(w, http.StatusNotFound, error)
			} else {
				log.Printf("Database error: %v", err)
				error.Message = "Server error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
			}
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Transaction error: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}
		defer tx.Rollback()

		// Initialize query parts
		setClause := []string{}
		args := []interface{}{}

		// Process first_name update
		if userUpdate.FirstName != "" {
			setClause = append(setClause, "first_name = ?")
			args = append(args, userUpdate.FirstName)
		}

		// Process last_name update
		if userUpdate.LastName != "" {
			setClause = append(setClause, "last_name = ?")
			args = append(args, userUpdate.LastName)
		}

		// Process date of birth update
		if userUpdate.DateOfBirth != "" {
			birthDate, err := time.Parse("2006-01-02", userUpdate.DateOfBirth)
			if err != nil {
				error.Message = "Invalid date format. Please use YYYY-MM-DD format."
				utils.RespondWithError(w, http.StatusBadRequest, error)
				return
			}

			now := time.Now()
			age := now.Year() - birthDate.Year()
			if now.YearDay() < birthDate.YearDay() {
				age--
			}

			setClause = append(setClause, "date_of_birth = ?, age = ?")
			args = append(args, userUpdate.DateOfBirth, age)
		}

		// If no fields to update
		if len(setClause) == 0 {
			error.Message = "No valid fields to update."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Build and execute query
		query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(setClause, ", "))
		args = append(args, targetUserID)

		result, err := tx.Exec(query, args...)
		if err != nil {
			log.Printf("Update error: %v", err)
			error.Message = "Failed to update profile."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			error.Message = "No changes made."
			utils.RespondWithError(w, http.StatusNotModified, error)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Printf("Transaction commit error: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Return only the success message
		response := map[string]interface{}{
			"message": "Profile updated successfully",
		}
		utils.ResponseJSON(w, response)
	}
}

func (c Controller) UpdatePassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Structure to capture the request data
		var requestData struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
			ConfirmPassword string `json:"confirm_password"`
		}

		// Decode the request body into the password change struct
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			// If there is an error in decoding, respond with JSON error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "Invalid request body."})
			return
		}

		// Verify the token to get the user ID
		userID, err := utils.VerifyToken(r)
		if err != nil {
			// If token verification fails, respond with JSON error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}

		// Check if the new password matches the confirm password
		if requestData.NewPassword != requestData.ConfirmPassword {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "New password and confirm password do not match."})
			return
		}

		// Retrieve the current hashed password from the database
		var hashedPassword string
		query := "SELECT password FROM users WHERE id = ?"
		err = db.QueryRow(query, userID).Scan(&hashedPassword)
		if err != nil {
			// If there is an issue with retrieving the password, respond with JSON error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "Error retrieving user password."})
			return
		}

		// Compare the current password with the stored hashed password
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(requestData.CurrentPassword))
		if err != nil {
			// If the current password does not match, respond with JSON error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Incorrect current password."})
			return
		}

		// Hash the new password
		hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			// If there is an error hashing the new password, respond with JSON error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "Error hashing the new password."})
			return
		}

		// Update the password in the database
		_, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", hashedNewPassword, userID)
		if err != nil {
			// If there is an error updating the password, respond with JSON error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "Error updating password."})
			return
		}

		// Send a success response with JSON formatted message
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully."})
	}
}
func (c *Controller) TokenVerifyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errorObject models.Error
		authHeader := r.Header.Get("Authorization")
		bearerToken := strings.Split(authHeader, " ")

		// Check if the token exists
		if len(bearerToken) == 2 {
			authToken := bearerToken[1]

			// Parse and validate the token
			token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("there was an error with the token")
				}
				return []byte(os.Getenv("SECRET")), nil
			})

			// If there's an error in token parsing or it's invalid
			if err != nil {
				errorObject.Message = err.Error()
				utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}

			// If the token is valid, proceed to the next handler
			if token.Valid {
				next.ServeHTTP(w, r)
			} else {
				errorObject.Message = "Invalid token."
				utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}
		} else {
			errorObject.Message = "Invalid token format."
			utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
			return
		}
	})
}
func (c *Controller) RefreshTokenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jwtToken models.JWT

		// Parse refresh token from request body
		err := json.NewDecoder(r.Body).Decode(&jwtToken)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request format"})
			return
		}

		// Parse refresh token
		token, err := utils.ParseToken(jwtToken.RefreshToken)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid refresh token"})
			return
		}

		// Validate token
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Extract user_id from token
			userID := int(claims["user_id"].(float64))

			// Check expiration (should be valid for 7 days from creation)
			expTime := time.Unix(int64(claims["exp"].(float64)), 0)
			if time.Now().After(expTime) {
				utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Refresh token expired"})
				return
			}

			// Get user from database
			user, err := utils.GetUserByID(db, userID)
			if err != nil {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "User not found"})
				return
			}

			// Generate new access token with 15-minute expiration
			accessToken, err := utils.GenerateToken(user, 15*time.Minute)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to generate access token"})
				return
			}

			// Return new access token with ORIGINAL refresh token
			utils.ResponseJSON(w, map[string]string{
				"access_token":  accessToken,
				"refresh_token": jwtToken.RefreshToken, // Keep the original refresh token
			})
		} else {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid refresh token"})
		}
	}
}
func (c Controller) VerifyResetToken(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.FormValue("token")
	if tokenStr == "" {
		// Return the new access token in the response
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	// Разбор токена
	parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})

	if err != nil || !parsedToken.Valid {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || claims["email"] == nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Если токен валиден, вернуть успешный ответ
	email := claims["email"].(string)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Token valid", "email": email})
}
func (c *Controller) VerifyEmail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Email   string `json:"email"`
			OTPCode string `json:"otp_code"`
		}
		var error models.Error

		// Decode request body
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.Email == "" || requestData.OTPCode == "" {
			error.Message = "Email or OTP code is missing"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Определяем переменные
		var storedOTP sql.NullString
		var userID int // Инициализация переменной userID

		// Проверяем наличие OTP для сброса пароля в таблице password_resets
		err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ?", requestData.Email).Scan(&storedOTP)
		if err == nil && storedOTP.Valid {
			// Если OTP найден в таблице сброса пароля
			if storedOTP.String != requestData.OTPCode {
				error.Message = "Invalid OTP code"
				utils.RespondWithError(w, http.StatusUnauthorized, error)
				return
			}

			// Если OTP для сброса пароля действителен, удаляем старый
			_, err = db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)
			if err != nil {
				error.Message = "Error processing password reset"
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			// Ответ для сброса пароля
			utils.ResponseJSON(w, map[string]string{
				"message": "Password reset code verified successfully",
			})
			return
		}

		// Если OTP не найден в password_resets, проверяем в таблице пользователей
		err = db.QueryRow("SELECT id, otp_code FROM users WHERE email = ?", requestData.Email).Scan(&userID, &storedOTP)
		if err != nil {
			if err == sql.ErrNoRows {
				error.Message = "User not found"
				utils.RespondWithError(w, http.StatusNotFound, error)
			} else {
				log.Printf("Database error: %v", err)
				error.Message = "Server error"
				utils.RespondWithError(w, http.StatusInternalServerError, error)
			}
			return
		}

		// Проверяем, совпадает ли OTP
		if !storedOTP.Valid || storedOTP.String != requestData.OTPCode {
			error.Message = "Invalid OTP code"
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Обновляем статус верификации и очищаем OTP
		_, err = db.Exec("UPDATE users SET verified = true, otp_code = NULL WHERE email = ?", requestData.Email)
		if err != nil {
			log.Printf("Error updating verification status: %v", err)
			error.Message = "Failed to verify email"
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Ответ для успешной верификации
		utils.ResponseJSON(w, map[string]string{
			"message": "Email verified successfully",
		})
	}
}
func sendVerificationEmail(email, verificationLink string) {
	fmt.Println("Verification email sent to", email)
	fmt.Println("Verification Link:", verificationLink)
}
func (c Controller) ResetPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Email    string `json:"email"`
			OTPCode  string `json:"otp_code"`
			Password string `json:"password"`
		}
		var error models.Error

		// Декодируем JSON-запрос
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.Email == "" || requestData.OTPCode == "" || requestData.Password == "" {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, существует ли email в базе данных
		var storedOTP string
		err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ? ORDER BY created_at DESC LIMIT 1", requestData.Email).Scan(&storedOTP)
		if err != nil {
			error.Message = "Invalid email or OTP expired."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Проверяем, совпадает ли введенный OTP
		if storedOTP != requestData.OTPCode {
			error.Message = "Invalid OTP code."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Получаем текущий хешированный пароль пользователя
		var currentHashedPassword string
		err = db.QueryRow("SELECT password FROM users WHERE email = ?", requestData.Email).Scan(&currentHashedPassword)
		if err != nil {
			error.Message = "Failed to retrieve current password."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Проверка нового пароля с текущим хешированным
		err = bcrypt.CompareHashAndPassword([]byte(currentHashedPassword), []byte(requestData.Password))
		if err == nil { // если ошибки нет, значит пароли совпадают
			error.Message = "New password cannot be the same as the current password."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Хешируем новый пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
		if err != nil {
			error.Message = "Failed to hash password."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Обновляем пароль в базе данных
		_, err = db.Exec("UPDATE users SET password = ? WHERE email = ?", string(hashedPassword), requestData.Email)
		if err != nil {
			error.Message = "Failed to update password."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Обновляем статус верификации пользователя на true, чтобы он мог сразу войти
		_, err = db.Exec("UPDATE users SET is_verified = true WHERE email = ?", requestData.Email)
		if err != nil {
			error.Message = "Failed to verify email."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Удаляем OTP после успешного сброса пароля
		_, err = db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)
		if err != nil {
			log.Printf("Error deleting reset token: %v", err)
		}

		// Ответ успешный
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password reset and email verified successfully"})
	}
}
func (c *Controller) ResendCode(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Email string `json:"email"`
			Type  string `json:"type"` // "reset" or "verify"
		}
		var error models.Error

		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.Email == "" || (requestData.Type != "reset" && requestData.Type != "verify") {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		log.Printf("Received resend code request for email: %s, type: %s", requestData.Email, requestData.Type)

		// Generate new OTP
		otpCode := fmt.Sprintf("%04d", rand.Intn(10000))
		log.Printf("Generated OTP: %s", otpCode)

		// If request type is "reset" - reset password
		if requestData.Type == "reset" {
			// Delete old OTP entries for password reset
			res, err := db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)
			if err != nil {
				error.Message = "Failed to clear old OTP."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}
			affected, _ := res.RowsAffected()
			log.Printf("Deleted %d old password reset entries", affected)

			// Insert new OTP into password_resets table with timestamp
			res, err = db.Exec("INSERT INTO password_resets (email, otp_code, created_at) VALUES (?, ?, ?)",
				requestData.Email, otpCode, time.Now())
			if err != nil {
				error.Message = "Failed to insert new OTP."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}
			affected, _ = res.RowsAffected()
			log.Printf("Inserted %d password reset entries", affected)

			// Send OTP for password reset
			utils.SendVerificationEmail(requestData.Email, "", otpCode)
		}

		// If request type is "verify" - verify email
		if requestData.Type == "verify" {
			// Check if user exists
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", requestData.Email).Scan(&exists)
			if err != nil {
				error.Message = "Database error."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}

			if !exists {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No user found with this email"})
				return
			}

			// Update OTP in users table
			res, err := db.Exec("UPDATE users SET otp_code = ?, created_at = ? WHERE email = ?",
				otpCode, time.Now(), requestData.Email)
			if err != nil {
				error.Message = "Failed to update OTP."
				utils.RespondWithError(w, http.StatusInternalServerError, error)
				return
			}
			affected, _ := res.RowsAffected()
			log.Printf("Updated %d user records with new OTP", affected)

			// Send OTP for email verification
			utils.SendVerificationEmail(requestData.Email, "", otpCode)
		}

		// Form response with message and OTP
		response := map[string]interface{}{
			"message":  "OTP resent successfully",
			"otp_code": otpCode, // Send OTP in response
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
func ChangeAdminPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.ChangePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var hashedPassword string
		err := db.QueryRow("SELECT Password FROM User WHERE Email = ?", req.Email).Scan(&hashedPassword)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Проверяем старый пароль
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.OldPassword)); err != nil {
			http.Error(w, "Incorrect password", http.StatusUnauthorized)
			return
		}

		// Хешируем новый пароль
		hashedNewPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)

		// Обновляем пароль и активируем аккаунт
		_, err = db.Exec("UPDATE User SET Password = ?, is_active = TRUE WHERE Email = ?", string(hashedNewPassword), req.Email)
		if err != nil {
			http.Error(w, "Failed to update password", http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, "Password updated successfully")
	}
}
func (c Controller) ChangePassword(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	tokenStr := r.FormValue("token")
	newPassword := r.FormValue("new_password")
	if tokenStr == "" || newPassword == "" {
		http.Error(w, "Token and new password are required", http.StatusBadRequest)
		return
	}

	// Разбор токена
	parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})

	if err != nil || !parsedToken.Valid {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || claims["email"] == nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	email := claims["email"].(string)

	// Хеширование пароля перед сохранением в базе данных
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Обновление пароля в базе данных
	query := "UPDATE users SET password = ? WHERE email = ?"
	_, err = db.Exec(query, hashedPassword, email)
	if err != nil {
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}

	// Ответ пользователю
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
}
func (c Controller) ForgotPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Email string `json:"email"`
		}
		var error models.Error

		// Декодируем запрос
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.Email == "" {
			error.Message = "Invalid request body or missing email."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, существует ли email в базе данных
		var userID int
		err = db.QueryRow("SELECT id FROM users WHERE email = ?", requestData.Email).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				error.Message = "Email not found."
				utils.RespondWithError(w, http.StatusNotFound, error)
				return
			}
			log.Printf("Error checking email: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Генерируем случайный OTP код
		otpCode := fmt.Sprintf("%04d", rand.Intn(10000))

		// Генерируем уникальный токен для сброса пароля
		token := utils.GenerateResetToken(requestData.Email)

		// Сохраняем OTP и токен в базе данных
		_, err = db.Exec("INSERT INTO password_resets (email, otp_code, reset_token) VALUES (?, ?, ?)", requestData.Email, otpCode, token)
		if err != nil {
			log.Printf("Error saving reset token: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Отправляем email с кодом OTP и ссылкой для сброса пароля
		resetLink := fmt.Sprintf("http://localhost:8000/reset-password?token=%s", token)
		utils.SendEmail(requestData.Email, "Reset your password", fmt.Sprintf("Your OTP: %s\nReset link: %s", otpCode, resetLink))

		// Устанавливаем заголовок ответа, чтобы указать, что это JSON
		w.Header().Set("Content-Type", "application/json")

		// Возвращаем ответ с OTP для проверки на фронтенде
		response := map[string]interface{}{
			"message":  "Reset email sent",
			"otp_code": otpCode, // Отправляем OTP в ответе
		}

		// Отправляем JSON в ответ
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
func (c Controller) ConfirmResetPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Email    string `json:"email"`
			OTPCode  string `json:"otp_code"`
			Password string `json:"password"`
		}
		var error models.Error

		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.Email == "" || requestData.OTPCode == "" || requestData.Password == "" {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем код OTP
		var storedOTP string
		err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ? ORDER BY created_at DESC LIMIT 1", requestData.Email).Scan(&storedOTP)
		if err != nil || storedOTP != requestData.OTPCode {
			error.Message = "Invalid or expired OTP."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Хешируем новый пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
		if err != nil {
			error.Message = "Failed to hash password."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Обновляем пароль в БД
		_, err = db.Exec("UPDATE users SET password = ? WHERE email = ?", hashedPassword, requestData.Email)
		if err != nil {
			error.Message = "Failed to update password."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Удаляем OTP после успешного сброса
		db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successfully"})
	}
}
func (c *Controller) RegisterUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Generate unique verification token
		verificationToken := uuid.New().String()

		// Save user to DB with 'is_verified' as false
		query := `INSERT INTO users (email, password, first_name, last_name, is_verified, verification_token) 
		          VALUES(?, ?, ?, ?, false, ?)`
		_, err := db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, verificationToken)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create user"})
			return
		}

		// Send verification email
		verificationLink := fmt.Sprintf("http://localhost:8000/verify-email?token=%s", verificationToken)
		go sendVerificationEmail(user.Email, verificationLink)

		utils.ResponseJSON(w, "User registered successfully. Please verify your email.")
	}
}
func GenerateRandomCode() (string, error) {
	code := make([]byte, 6) // генерируем 6-значный код
	_, err := rand.Read(code)
	if err != nil {
		log.Println("Error generating random code:", err)
		return "", err
	}
	return fmt.Sprintf("%x", code[:6]), nil
}
func (c Controller) ChangeUserRole(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			UserID int    `json:"user_id"`
			Role   string `json:"role"`
		}

		// Декодируем тело запроса для получения user_id и новой роли
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.UserID == 0 || requestData.Role == "" {
			log.Printf("Invalid request body: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Проверяем, что роль правильная (например, "user", "schooladmin", "superadmin")
		if requestData.Role != "user" && requestData.Role != "schooladmin" && requestData.Role != "superadmin" {
			log.Printf("Invalid role provided: %s", requestData.Role)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid role"})
			return
		}

		// Получаем ID пользователя из токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			log.Printf("Unauthorized access: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Получаем роль пользователя, который отправил запрос
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
		if err != nil {
			log.Printf("Error fetching user role: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
			return
		}

		// Проверяем, что роль пользователя - superadmin
		if userRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to change roles"})
			return
		}

		// Обновляем роль в базе данных
		_, err = db.Exec("UPDATE users SET role = ? WHERE id = ?", requestData.Role, requestData.UserID)
		if err != nil {
			log.Printf("Failed to update role for user %d: %v", requestData.UserID, err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update role"})
			return
		}

		// Отправляем успешный ответ
		utils.ResponseJSON(w, map[string]string{"message": "User role updated successfully"})
	}
}
func (c Controller) UploadAvatar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем userID из токена
		userID, err := utils.VerifyToken(r) // Возвращает только userID (int)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Чтение файла аватара
		file, _, err := r.FormFile("avatar")
		if err != nil {
			log.Println("Error reading file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
			return
		}
		defer file.Close()

		// Генерация уникального имени файла для аватара
		uniqueFileName := fmt.Sprintf("avatar-%d-%d.jpg", userID, time.Now().Unix())

		// Fix: Changed boolean true to string "avatar"
		photoURL, err := utils.UploadFileToS3(file, uniqueFileName, "avatar") // передаем "avatar" для использования второго набора ключей
		if err != nil {
			log.Println("Error uploading file:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload avatar"})
			return
		}

		// Обновление URL аватара в базе данных
		query := "UPDATE users SET avatar_url = ? WHERE id = ?"
		_, err = db.Exec(query, photoURL, userID) // Теперь передаем только userID
		if err != nil {
			log.Println("Error updating avatar URL:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update avatar URL"})
			return
		}

		// Ответ с подтверждением
		utils.ResponseJSON(w, map[string]string{"message": "Avatar uploaded successfully", "avatar_url": photoURL})
	}
}

func (c Controller) UpdateAvatar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем userID из токена
		userID, err := utils.VerifyToken(r) // Возвращает только userID (int)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Логируем userID для отладки
		log.Println("UserID from token:", userID)

		// Получаем данные о старом аватаре
		var currentAvatarURL sql.NullString
		query := "SELECT avatar_url FROM users WHERE id = ?"
		err = db.QueryRow(query, userID).Scan(&currentAvatarURL)
		if err != nil {
			log.Println("Error fetching current avatar:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch current avatar"})
			return
		}

		// Логируем URL текущего аватара для отладки
		log.Println("Current avatar URL:", currentAvatarURL.String)

		// Удаление старого аватара с S3, если он существует
		if currentAvatarURL.Valid && currentAvatarURL.String != "" && currentAvatarURL.String != "https://your-bucket-name.s3.amazonaws.com/default-avatar.jpg" {
			err := utils.DeleteFileFromS3(currentAvatarURL.String)
			if err != nil {
				log.Println("Error deleting old avatar from S3:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete old avatar"})
				return
			}
		}

		// Чтение нового аватара
		file, _, err := r.FormFile("avatar")
		if err != nil {
			log.Println("Error reading file:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
			return
		}
		defer file.Close()

		// Генерация уникального имени файла для аватара
		uniqueFileName := fmt.Sprintf("avatar-%d-%d.jpg", userID, time.Now().Unix())

		// Fix: Changed boolean true to string "avatar"
		newAvatarURL, err := utils.UploadFileToS3(file, uniqueFileName, "avatar") // Передаем "avatar", чтобы использовать второй набор ключей для аватарок
		if err != nil {
			log.Println("Error uploading new avatar:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload new avatar"})
			return
		}

		// Обновление URL аватара в базе данных
		query = "UPDATE users SET avatar_url = ? WHERE id = ?"
		_, err = db.Exec(query, newAvatarURL, userID) // Теперь передаем только userID
		if err != nil {
			log.Println("Error updating avatar URL:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update avatar URL"})
			return
		}

		// Ответ с подтверждением
		utils.ResponseJSON(w, map[string]string{"message": "Avatar updated successfully", "avatar_url": newAvatarURL})
	}
}
func (c Controller) DeleteAvatar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем userID из токена
		userID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Получаем текущий URL аватара из базы данных
		var currentAvatarURL string
		query := "SELECT avatar_url FROM users WHERE id = ?"
		err = db.QueryRow(query, userID).Scan(&currentAvatarURL)
		if err != nil {
			log.Println("Error fetching current avatar:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch current avatar"})
			return
		}

		// Проверяем, что у пользователя есть аватар, который можно удалить
		if currentAvatarURL == "" || currentAvatarURL == "https://your-bucket-name.s3.amazonaws.com/default-avatar.jpg" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "No avatar to delete"})
			return
		}

		// Удаляем старое изображение с S3
		err = utils.DeleteFileFromS3(currentAvatarURL) // Здесь больше не передаем параметр bool
		if err != nil {
			log.Println("Error deleting avatar from S3:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete avatar"})
			return
		}

		// Сбрасываем URL аватара на NULL в базе данных
		query = "UPDATE users SET avatar_url = NULL WHERE id = ?"
		_, err = db.Exec(query, userID)
		if err != nil {
			log.Println("Error resetting avatar URL:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to reset avatar URL"})
			return
		}

		// Ответ с подтверждением
		utils.ResponseJSON(w, map[string]string{"message": "Avatar deleted successfully"})
	}
}
func (c Controller) ResetPasswordConfirm(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Email    string `json:"email"`
			OTPCode  string `json:"otp_code"`
			Password string `json:"password"`
		}
		var error models.Error

		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.Email == "" || requestData.OTPCode == "" || requestData.Password == "" {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем код OTP
		var storedOTP string
		err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ? ORDER BY created_at DESC LIMIT 1", requestData.Email).Scan(&storedOTP)
		if err != nil || storedOTP != requestData.OTPCode {
			error.Message = "Invalid or expired OTP."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Хешируем новый пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
		if err != nil {
			error.Message = "Failed to hash password."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Обновляем пароль в БД
		_, err = db.Exec("UPDATE users SET password = ? WHERE email = ?", hashedPassword, requestData.Email)
		if err != nil {
			error.Message = "Failed to update password."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Удаляем OTP после успешного сброса
		db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successfully"})
	}
}
func (c *Controller) VerifyResetCode(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Email   string `json:"email"`
			OTPCode string `json:"otp_code"`
		}
		var error models.Error

		// Decode request body
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil || requestData.Email == "" || requestData.OTPCode == "" {
			error.Message = "Email or OTP code is missing"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		log.Printf("Verifying reset code for email: %s with OTP: %s", requestData.Email, requestData.OTPCode)

		// Check that OTP matches what's stored in the password_resets table
		var storedOTP string
		var createdAt time.Time
		err = db.QueryRow("SELECT otp_code, created_at FROM password_resets WHERE email = ?",
			requestData.Email).Scan(&storedOTP, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Reset code not found for email: %s", requestData.Email)
				error.Message = "Invalid or expired reset code"
				utils.RespondWithError(w, http.StatusNotFound, error)
			} else {
				log.Printf("Database error: %v", err)
				error.Message = "Server error"
				utils.RespondWithError(w, http.StatusInternalServerError, error)
			}
			return
		}

		// Check if OTP has expired (15 minutes)
		if time.Now().Sub(createdAt).Minutes() > 15 {
			error.Message = "OTP has expired. Please request a new one."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Compare OTP codes
		if storedOTP != requestData.OTPCode {
			log.Printf("Reset code mismatch. Stored: %s, Received: %s", storedOTP, requestData.OTPCode)
			error.Message = "Invalid reset code"
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		utils.ResponseJSON(w, map[string]string{
			"message": "Reset code verified successfully",
		})
	}
}
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
	rand.Seed(time.Now().UnixNano())

	password := make([]byte, length)
	for i := range password {
		password[i] = charset[rand.Intn(len(charset))]
	}

	return string(password)
}
func (c *Controller) CalculateAge(user *models.User) (int, error) {
	dob, err := time.Parse("2006-01-02", user.DateOfBirth)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	age := now.Year() - dob.Year()

	// Adjust age if birthday hasn't occurred yet this year
	if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
		age--
	}

	return age, nil
}
func (c *Controller) GetAllSchoolAdmins(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, email, first_name, last_name, role FROM users WHERE role = 'schooladmin'")
		if err != nil {
			log.Printf("Error fetching schooladmins: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Server error"})
			return
		}
		defer rows.Close()

		var schoolAdmins []models.User

		// Проходим по всем строкам в результатах запроса
		for rows.Next() {
			var user models.User
			err := rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role)
			if err != nil {
				log.Printf("Error scanning user: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning user data"})
				return
			}

			schoolAdmins = append(schoolAdmins, user)
		}

		// Проверка на ошибки после итерации
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating over schooladmins: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing users"})
			return
		}

		// Отправляем список schooladmins в формате JSON
		utils.ResponseJSON(w, schoolAdmins)
	}
}
func (c *Controller) GetAllSuperAdmins(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Запрос для получения всех пользователей с ролью superadmin
		rows, err := db.Query("SELECT id, email, first_name, last_name, role FROM users WHERE role = 'superadmin'")
		if err != nil {
			log.Printf("Error fetching superadmins: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Server error"})
			return
		}
		defer rows.Close()

		var superAdmins []models.User

		// Проходим по всем строкам в результатах запроса
		for rows.Next() {
			var user models.User
			err := rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role)
			if err != nil {
				log.Printf("Error scanning user: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning user data"})
				return
			}

			superAdmins = append(superAdmins, user)
		}

		// Проверка на ошибки после итерации
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating over superadmins: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing users"})
			return
		}

		// Отправляем список superadmins в формате JSON
		utils.ResponseJSON(w, superAdmins)
	}
}
func (c *Controller) GetSchoolAdminsWithoutSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Запрос для получения всех пользователей с ролью schooladmin, у которых нет привязанных школ
		query := `
            SELECT u.id, u.email, u.first_name, u.last_name, u.role 
            FROM users u 
            LEFT JOIN Schools s ON u.email = s.school_admin_login
            WHERE u.role = 'schooladmin' AND s.school_id IS NULL
        `
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Error fetching schooladmins without schools: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Server error"})
			return
		}
		defer rows.Close()

		var schoolAdminsWithoutSchools []models.User

		// Проходим по всем строкам в результатах запроса
		for rows.Next() {
			var user models.User
			err := rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role)
			if err != nil {
				log.Printf("Error scanning user: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning user data"})
				return
			}

			schoolAdminsWithoutSchools = append(schoolAdminsWithoutSchools, user)
		}

		// Проверка на ошибки после итерации
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating over schooladmins without schools: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing users"})
			return
		}

		// Отправляем список schooladmins без школ в формате JSON
		utils.ResponseJSON(w, schoolAdminsWithoutSchools)
	}
}
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Authorization header is required"})
			return
		}

		// Typically, the token is prefixed with "Bearer "
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// Parse and validate the token
		token, err := utils.ParseToken(tokenString)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
			return
		}

		// Check if token is valid
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Check expiration time explicitly to ensure it's enforced
			expTime := time.Unix(int64(claims["exp"].(float64)), 0)
			if time.Now().After(expTime) {
				utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Token expired"})
				return
			}

			// Add user information to request context for further handlers
			userID := int(claims["user_id"].(float64))
			role := claims["role"].(string)

			// Create a context with user information
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "role", role)

			// Continue with the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
		}
	})
}
func (c *Controller) getUserData(db *sql.DB, id int) (interface{}, bool) {
	var email, phone, login sql.NullString
	var avatarURL, dateOfBirth sql.NullString
	var age sql.NullInt64
	var passwordHash sql.NullString
	var schoolID sql.NullInt64
	var isVerified sql.NullBool

	var user models.User

	err := db.QueryRow(`
		SELECT id, first_name, last_name, email, phone, login, avatar_url, date_of_birth, 
		       age, password, role, school_id, is_verified 
		FROM users WHERE id = ?`, id).
		Scan(&user.ID, &user.FirstName, &user.LastName, &email, &phone, &login, &avatarURL,
			&dateOfBirth, &age, &passwordHash, &user.Role, &schoolID, &isVerified)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		log.Printf("Error fetching user data: %v", err)
		return nil, false
	}

	// Обрабатываем значения
	if email.Valid {
		user.Email = email.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if login.Valid {
		user.Login = login.String
	}
	if isVerified.Valid {
		user.IsVerified = isVerified.Bool
	}
	if schoolID.Valid {
		user.SchoolID = int(schoolID.Int64)
	}

	// Вычисляем возраст
	var ageValue int
	if age.Valid {
		ageValue = int(age.Int64)
	} else {
		ageValue = 0
		if dateOfBirth.Valid && dateOfBirth.String != "" {
			dob, err := time.Parse("2006-01-02", dateOfBirth.String)
			if err == nil {
				now := time.Now()
				ageValue = now.Year() - dob.Year()
				if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
					ageValue--
				}
				_, updateErr := db.Exec("UPDATE users SET age = ? WHERE id = ?", ageValue, id)
				if updateErr != nil {
					log.Printf("Failed to update age in database: %v", updateErr)
				}
			}
		}
	}

	// === ❗ Создаём структуру ответа ===
	type UserResponse struct {
		ID          int     `json:"id"`
		FirstName   string  `json:"first_name"`
		LastName    string  `json:"last_name"`
		Age         int     `json:"age"`
		DateOfBirth string  `json:"date_of_birth"`
		Email       string  `json:"email"`
		Role        string  `json:"role"`
		AvatarURL   *string `json:"avatar_url"`
		SchoolID    int     `json:"school_id"`
		IsVerified  bool    `json:"is_verified"`
	}
	userData := UserResponse{
		Age:         ageValue,
		AvatarURL:   nil,
		DateOfBirth: "",
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Role:        user.Role,
		ID:          user.ID,
		SchoolID:    user.SchoolID,
		IsVerified:  user.IsVerified,
	}

	if avatarURL.Valid {
		userData.AvatarURL = &avatarURL.String
	}
	if dateOfBirth.Valid {
		userData.DateOfBirth = dateOfBirth.String
	}

	// Возвращаем
	return userData, true
}

func (c *Controller) GetTotalUsersWithRoleCount(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Query to get total number of users and the count per role
		query := `
			SELECT role, COUNT(*) as count
			FROM users
			GROUP BY role
		`

		// Step 2: Execute the query to get the counts for each role
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Error querying total users by role:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user counts by role"})
			return
		}
		defer rows.Close()

		// Step 3: Prepare a map to hold the counts for each role
		roleCounts := make(map[string]int)

		// Step 4: Iterate over the result set and populate the map
		for rows.Next() {
			var role string
			var count int
			err := rows.Scan(&role, &count)
			if err != nil {
				log.Println("Error scanning result:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning user count by role"})
				return
			}
			roleCounts[role] = count
		}

		// Step 5: Handle errors from iterating over rows
		if err := rows.Err(); err != nil {
			log.Println("Error processing rows:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing user count by role"})
			return
		}

		// Step 6: Query to get total number of users (regardless of role)
		var totalUsers int
		err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
		if err != nil {
			log.Println("Error fetching total user count:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching total user count"})
			return
		}

		// Step 7: Create a response with the total count and the breakdown by role
		response := map[string]interface{}{
			"total_users": totalUsers,
			"role_counts": roleCounts,
		}

		// Step 8: Send the response
		utils.ResponseJSON(w, response)
	}
}
